package main

import (
	
	"bytes"
	"context"
	"encoding/json"
	
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	consumerkafka "video-encoding/consumer/internal"
	"video-encoding/shared/store"
	"video-encoding/shared/types"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	ckafka "github.com/confluentinc/confluent-kafka-go/kafka"
	"go.uber.org/zap"
)

type Worker struct {
	log   *zap.SugaredLogger
	store store.Storage
	co    *consumerkafka.Consumer
	topic string

	s3       *s3.Client
	s3Bucket string
	s3Base   string // e.g. "reels/"
}

func NewWorker(
	log *zap.SugaredLogger,
	st store.Storage,
	brokers, groupID, topic string,
	s3Client *s3.Client,
	s3Bucket string,
	s3Base string,
) *Worker {
	co, err := consumerkafka.NewConsumer(brokers, groupID)
	if err != nil {
		log.Fatalw("kafka consumer init failed", "err", err)
	}
	if err := co.Subscribe(topic); err != nil {
		log.Fatalw("kafka subscribe failed", "err", err)
	}

	return &Worker{
		log:      log,
		store:    st,
		co:       co,
		topic:    topic,
		s3:       s3Client,
		s3Bucket: s3Bucket,
		s3Base:   s3Base,
	}
}

func (w *Worker) Run(ctx context.Context, pollEvery time.Duration) error {
	defer w.co.Close()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		msgAny, err := w.co.Poll(int(pollEvery / time.Millisecond))
		if err != nil {
			// transient error is okay; continue
			continue
		}

		m, ok := msgAny.(*ckafka.Message)
		if !ok || m == nil || len(m.Value) == 0 {
			continue
		}

		var job types.TranscodeJobMessage
		if err := json.Unmarshal(m.Value, &job); err != nil {
			w.log.Errorw("bad message json", "err", err, "payload", string(m.Value))
			continue
		}

		if strings.TrimSpace(job.JobID) == "" || strings.TrimSpace(job.VideoID) == "" || strings.TrimSpace(job.InputKey) == "" {
			w.log.Errorw("invalid job message", "job", job)
			continue
		}

		// process each message with a bounded timeout so workers don't hang forever
		jobCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
		w.processOne(jobCtx, job)
		cancel()
	}
}

func (w *Worker) processOne(ctx context.Context, msg types.TranscodeJobMessage) {
	log := w.log.With("jobId", msg.JobID, "videoId", msg.VideoID, "inputKey", msg.InputKey, "pipeline", msg.Pipeline)

	// Mark job/video processing (best-effort; do not stop pipeline if this fails)
	_ = w.store.Job.MarkProcessing(ctx, msg.JobID)
	_ = w.store.Video.MarkProcessing(ctx, msg.VideoID)

	// Create working directory
	workDir, err := os.MkdirTemp("", "transcode-"+msg.JobID+"-*")
	if err != nil {
		w.fail(ctx, msg, fmt.Errorf("mktemp: %w", err))
		return
	}
	defer os.RemoveAll(workDir)

	inputPath := filepath.Join(workDir, "input.mp4")

	// 1) Download input from S3
	if err := w.downloadFromS3(ctx, msg.InputKey, inputPath); err != nil {
		w.fail(ctx, msg, fmt.Errorf("download input from s3: %w", err))
		return
	}
	_ = w.store.Job.UpdateProgress(ctx, msg.JobID, 10, nil, nil, false)

	// 2) Run ffmpeg → produce HLS outputs in local dir
	outDir := filepath.Join(workDir, "hls")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		w.fail(ctx, msg, fmt.Errorf("mkdir outdir: %w", err))
		return
	}

	renditions := []string{"480p", "720p", "1080p"} // target qualities
	if err := w.transcodeToHLS(ctx, inputPath, outDir, log); err != nil {
		w.fail(ctx, msg, fmt.Errorf("ffmpeg transcode: %w", err))
		return
	}
	_ = w.store.Job.UpdateProgress(ctx, msg.JobID, 70, []string{"480p"}, nil, false) // conservative midpoint update

	// 3) Upload HLS folder to S3
	// S3 base: reels/outputs/<video>/<job>/
	outputBase := w.s3Base + "outputs/" + msg.VideoID + "/" + msg.JobID + "/"
	masterKey := outputBase + "master.m3u8"

	if err := w.uploadDirToS3(ctx, outDir, outputBase); err != nil {
		w.fail(ctx, msg, fmt.Errorf("upload outputs to s3: %w", err))
		return
	}

	// 4) Mark playable + completed
	if err := w.store.Job.UpdateProgress(ctx, msg.JobID, 100, renditions, &masterKey, true); err != nil {
		// even if this fails, we still try to mark failed so the system isn't stuck
		w.fail(ctx, msg, fmt.Errorf("db update progress final: %w", err))
		return
	}

	_ = w.store.Job.MarkCompleted(ctx, msg.JobID)
	_ = w.store.Video.MarkReady(ctx, msg.VideoID)

	log.Infow("job completed", "masterKey", masterKey)
}

func (w *Worker) fail(ctx context.Context, msg types.TranscodeJobMessage, err error) {
	w.log.Errorw("job failed", "jobId", msg.JobID, "videoId", msg.VideoID, "err", err)

	// Store failure in DB (best effort)
	_ = w.store.Job.MarkFailed(ctx, msg.JobID, err.Error())
	_ = w.store.Video.MarkFailed(ctx, msg.VideoID, err.Error())
}

// ------------------------
// S3 helpers
// ------------------------

func (w *Worker) downloadFromS3(ctx context.Context, key, dstPath string) error {
	out, err := w.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(w.s3Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return err
	}
	defer out.Body.Close()

	f, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, out.Body)
	return err
}

func (w *Worker) uploadDirToS3(ctx context.Context, dir string, s3Prefix string) error {
	// Upload all files in dir recursively
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		key := s3Prefix + filepath.ToSlash(rel)

		// content-type inference (basic)
		ct := "application/octet-stream"
		if strings.HasSuffix(key, ".m3u8") {
			ct = "application/vnd.apple.mpegurl"
		} else if strings.HasSuffix(key, ".ts") {
			ct = "video/mp2t"
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = w.s3.PutObject(ctx, &s3.PutObjectInput{
			Bucket:      aws.String(w.s3Bucket),
			Key:         aws.String(key),
			Body:        f,
			ContentType: aws.String(ct),
		})
		return err
	})
}


func (w *Worker) transcodeToHLS(ctx context.Context, inputPath, outDir string, log *zap.SugaredLogger) error {
	
	// master := filepath.Join(outDir, "master.m3u8")

	// args := []string{
	// 	"-y",
	// 	"-i", inputPath,

	// 	// create 3 video streams
	// 	"-filter_complex",
	// 	"[0:v]split=3[v480][v720][v1080];" +
	// 		"[v480]scale=w=854:h=480:force_original_aspect_ratio=decrease[v480out];" +
	// 		"[v720]scale=w=1280:h=720:force_original_aspect_ratio=decrease[v720out];" +
	// 		"[v1080]scale=w=1920:h=1080:force_original_aspect_ratio=decrease[v1080out]",

	// 	// map streams
	// 	"-map", "[v480out]", "-map", "0:a?",
	// 	"-map", "[v720out]", "-map", "0:a?",
	// 	"-map", "[v1080out]", "-map", "0:a?",

	// 	// video codecs per rendition (H.264)
	// 	"-c:v:0", "libx264", "-b:v:0", "800k", "-maxrate:v:0", "900k", "-bufsize:v:0", "1200k",
	// 	"-c:v:1", "libx264", "-b:v:1", "2000k", "-maxrate:v:1", "2200k", "-bufsize:v:1", "3000k",
	// 	"-c:v:2", "libx264", "-b:v:2", "4500k", "-maxrate:v:2", "5000k", "-bufsize:v:2", "6500k",

	// 	// audio codec (AAC). "a?" means optional audio; if no audio, it won’t fail.
	// 	"-c:a", "aac", "-b:a", "128k",

	// 	// HLS settings
	// 	"-f", "hls",
	// 	"-hls_time", "4",
	// 	"-hls_playlist_type", "vod",
	// 	"-hls_flags", "independent_segments",
	// 	"-hls_segment_filename", filepath.Join(outDir, "%v_%03d.ts"),

	// 	// variant playlists output pattern
	// 	"-master_pl_name", filepath.Base(master),
	// 	"-var_stream_map", "v:0,a:0 v:1,a:1 v:2,a:2",
	// 	filepath.Join(outDir, "%v.m3u8"),
	// }

	// cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	// cmd.Dir = outDir

	// // Capture stderr for debugging
	// var stderr bytes.Buffer
	// cmd.Stderr = &stderr
	// cmd.Stdout = &stderr // ffmpeg logs mostly on stderr anyway

	// if err := cmd.Start(); err != nil {
	// 	return fmt.Errorf("ffmpeg start: %w", err)
	// }

	// // stream logs to zap while running (optional)
	// go func() {
	// 	sc := bufio.NewScanner(bytes.NewReader(stderr.Bytes()))
	// 	for sc.Scan() {
	// 		log.Debug(sc.Text())
	// 	}
	// }()

	// err := cmd.Wait()
	// if err != nil {
	// 	// include ffmpeg output
	// 	out := strings.TrimSpace(stderr.String())
	// 	if out == "" {
	// 		out = "no ffmpeg output"
	// 	}
	// 	return fmt.Errorf("ffmpeg failed: %w | output: %s", err, out)
	// }

	// // sanity check: master must exist
	// if _, err := os.Stat(master); err != nil {
	// 	if errors.Is(err, os.ErrNotExist) {
	// 		return fmt.Errorf("master playlist missing after transcode")
	// 	}
	// 	return err
	// }

	// return nil
	master := filepath.Join(outDir, "master.m3u8")

	filter :=
		"[0:v]split=3[v480][v720][v1080];" +
			"[v480]scale=w=854:h=480:force_original_aspect_ratio=decrease," +
			"scale=trunc(iw/2)*2:trunc(ih/2)*2[v480out];" +
			"[v720]scale=w=1280:h=720:force_original_aspect_ratio=decrease," +
			"scale=trunc(iw/2)*2:trunc(ih/2)*2[v720out];" +
			"[v1080]scale=w=1920:h=1080:force_original_aspect_ratio=decrease," +
			"scale=trunc(iw/2)*2:trunc(ih/2)*2[v1080out]"

	args := []string{
		"-y",
		"-i", inputPath,

		"-filter_complex", filter,

		// map video+audio
		"-map", "[v480out]", "-map", "0:a?",
		"-map", "[v720out]", "-map", "0:a?",
		"-map", "[v1080out]", "-map", "0:a?",

		// video encode
		"-c:v:0", "libx264", "-profile:v:0", "high", "-pix_fmt", "yuv420p",
		"-b:v:0", "800k", "-maxrate:v:0", "900k", "-bufsize:v:0", "1200k",

		"-c:v:1", "libx264", "-profile:v:1", "high", "-pix_fmt", "yuv420p",
		"-b:v:1", "2000k", "-maxrate:v:1", "2200k", "-bufsize:v:1", "3000k",

		"-c:v:2", "libx264", "-profile:v:2", "high", "-pix_fmt", "yuv420p",
		"-b:v:2", "4500k", "-maxrate:v:2", "5000k", "-bufsize:v:2", "6500k",

		// audio
		"-c:a", "aac", "-b:a", "128k", "-ac", "2",

		// GOP alignment for HLS (30fps * 4s = 120)
		"-g", "120",
		"-keyint_min", "120",
		"-sc_threshold", "0",

		// HLS output
		"-f", "hls",
		"-hls_time", "4",
		"-hls_playlist_type", "vod",
		"-hls_flags", "independent_segments",
		"-hls_segment_filename", filepath.Join(outDir, "%v_%03d.ts"),

		"-master_pl_name", filepath.Base(master),
		"-var_stream_map", "v:0,a:0 v:1,a:1 v:2,a:2",

		filepath.Join(outDir, "%v.m3u8"),
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	cmd.Dir = outDir

	var stderr bytes.Buffer
	cmd.Stdout = &stderr
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("ffmpeg start: %w", err)
	}

	err := cmd.Wait()
	if err != nil {
		out := strings.TrimSpace(stderr.String())
		if out == "" {
			out = "no ffmpeg output"
		}
		return fmt.Errorf("ffmpeg failed: %w | output: %s", err, out)
	}

	if _, err := os.Stat(master); err != nil {
		return fmt.Errorf("master playlist missing after transcode: %w", err)
	}

	return nil
}
