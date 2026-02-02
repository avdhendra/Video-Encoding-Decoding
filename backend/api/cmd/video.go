package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	httpx "video-encoding/shared/response"
	"video-encoding/shared/store"
	"video-encoding/shared/types"
	"video-encoding/shared/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-chi/chi"
	"github.com/google/uuid"
	producerpb "video-encoding/shared/proto/job"
)

func (app *application) GetVideoPlayback(w http.ResponseWriter, r *http.Request) {
	videoID := chi.URLParam(r, "id")
	if strings.TrimSpace(videoID) == "" {
		httpx.Fail(w, 400, "VALIDATION_ERROR", "id is required")
		return
	}

	v, err := app.store.Video.Get(r.Context(), videoID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) || errors.Is(err, sql.ErrNoRows) {
			httpx.Fail(w, 404, "NOT_FOUND", "video not found")
			return
		}
		httpx.Fail(w, 500, "DB_ERROR", err.Error())
		return
	}

	if v.LatestJobID == nil || *v.LatestJobID == "" {
		httpx.Ok(w, "no job yet", types.PlaybackResp{
			VideoID:       videoID,
			Status:        string(v.Status),
			Progress:      0,
			PlaybackReady: false,
		})
		return
	}

	if app.store.Job == nil {
		httpx.Fail(w, 500, "NOT_IMPLEMENTED", "Job store not wired (app.store.Job is nil)")
		return
	}

	j, err := app.store.Job.Get(r.Context(), *v.LatestJobID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) || errors.Is(err, sql.ErrNoRows) {
			httpx.Fail(w, 404, "NOT_FOUND", "job not found")
			return
		}
		httpx.Fail(w, 500, "DB_ERROR", err.Error())
		return
	}

	masterURL := ""
	if j.OutputMasterKey != nil && *j.OutputMasterKey != "" && j.PlaybackReady {
		u, err := app.PresignGet(r.Context(), *j.OutputMasterKey)
		if err == nil {
			masterURL = u
		}
	}

	httpx.Ok(w, "playback", types.PlaybackResp{
		VideoID:             videoID,
		JobID:               v.LatestJobID,
		Status:              string(j.Status),
		Progress:            j.Progress,
		PlaybackReady:       j.PlaybackReady,
		AvailableRenditions: j.AvailableRenditions,
		MasterKey:           j.OutputMasterKey,
		MasterURL:           masterURL,
	})
}

func (app *application) PresignVideoUpload(w http.ResponseWriter, r *http.Request) {

	var req types.PresignVideoUploadReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Fail(w, 400, "INVALID_JSON", err.Error())
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)
	req.VideoFilename = strings.TrimSpace(req.VideoFilename)
	req.ThumbFilename = strings.TrimSpace(req.ThumbFilename)

	if req.VideoFilename == "" {
		httpx.Fail(w, 400, "VALIDATION_ERROR", "videoFilename is required")
		return
	}
	if req.VideoType == "" {
		req.VideoType = "video/mp4"
	}
	if req.ThumbFilename == "" {
		httpx.Fail(w, 400, "VALIDATION_ERROR", "thumbFilename is required")
		return
	}
	if req.ThumbType == "" {
		req.ThumbType = "image/jpeg"
	}

	videoID := uuid.NewString()

	videoKey := app.config.s3.basePath + "inputs/" + videoID + "-" + utils.SafeFilename(req.VideoFilename)
	thumbKey := app.config.s3.basePath + "thumbnails/" + videoID + "-" + utils.SafeFilename(req.ThumbFilename)

	// Insert DB row first
	if err := app.store.Video.Create(r.Context(), store.Video{
		ID:           videoID,
		Title:        req.Title,
		Description:  req.Description,
		Filename:     req.VideoFilename,
		ContentType:  req.VideoType,
		InputKey:     videoKey,
		ThumbnailKey: thumbKey,
		Status:       store.Uploaded,
	}); err != nil {
		app.logger.Errorw("video create failed", "err", err)
		httpx.Fail(w, 500, "DB_ERROR", err.Error())
		return
	}

	// Presign PUT URLs
	videoPutURL, err := app.PresignPut(r.Context(), videoKey, req.VideoType)
	if err != nil {
		app.logger.Errorw("presign video put failed", "err", err)
		httpx.Fail(w, 500, "PRESIGN_FAILED", err.Error())
		return
	}

	thumbPutURL, err := app.PresignPut(r.Context(), thumbKey, req.ThumbType)
	if err != nil {
		app.logger.Errorw("presign thumb put failed", "err", err)
		httpx.Fail(w, 500, "PRESIGN_FAILED", err.Error())
		return
	}

	httpx.Created(w, "upload created", types.PresignVideoUploadResp{
		VideoID:     videoID,
		VideoKey:    videoKey,
		VideoPutURL: videoPutURL,
		ThumbKey:    thumbKey,
		ThumbPutURL: thumbPutURL,
	})
}

func (app *application) ListVideos(w http.ResponseWriter, r *http.Request) {
	limit := utils.ParseInt(r.URL.Query().Get("limit"), 24)
	offset := utils.ParseInt(r.URL.Query().Get("offset"), 0)
	if limit <= 0 || limit > 100 {
		limit = 24
	}
	items, total, err := app.store.Video.List(r.Context(), limit, offset)
	if err != nil {
		httpx.Fail(w, 500, "DB_ERROR", err.Error())
		return
	}

	// attach thumbnailUrl via presigned GET (for the grid)
	out := make([]map[string]any, 0, len(items))
	for _, v := range items {
		var thumbURL string
		if v.ThumbnailKey != "" {
			
			u, err := app.PresignGet(r.Context(), v.ThumbnailKey)
			if err == nil {
				thumbURL = u
			}
		}

		out = append(out, map[string]any{
			"id":           v.ID,
			"title":        v.Title,
			"description":  v.Description,
			"filename":     v.Filename,
			"contentType":  v.ContentType,
			"inputKey":     v.InputKey,
			"thumbnailKey": v.ThumbnailKey,
			"thumbnailUrl": thumbURL,
			"latestJobId":  v.LatestJobID,
			"status":       v.Status,
			"errorMsg":     v.ErrorMsg,
			"createdAt":    v.CreatedAt,
			"updatedAt":    v.UpdatedAt,
		})
	}

	httpx.Ok(w, "videos listed", map[string]any{
		"items":  out,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (app *application) PresignPut(ctx context.Context, key, contentType string) (string, error) {
	ps, err := app.s3Presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(app.config.s3.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, func(po *s3.PresignOptions) {
		po.Expires = app.config.s3.presignPUTTTL
	})
	if err != nil {
		return "", err
	}
	return ps.URL, nil
}

func (app *application) PresignGet(ctx context.Context, key string) (string, error) {
	ps, err := app.s3Presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(app.config.s3.bucket),
		Key:    aws.String(key),
	}, func(po *s3.PresignOptions) {
		po.Expires = app.config.s3.presignGETTTL
	})
	if err != nil {
		return "", err
	}
	return ps.URL, nil
}

func (app *application) GetVideo(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if strings.TrimSpace(id) == "" {
		httpx.Fail(w, 400, "VALIDATION_ERROR", "id is required")
		return
	}

	v, err := app.store.Video.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) || errors.Is(err, sql.ErrNoRows) {
			httpx.Fail(w, 404, "NOT_FOUND", "video not found")
			return
		}
		httpx.Fail(w, 500, "DB_ERROR", err.Error())
		return
	}

	thumbURL := ""
	if v.ThumbnailKey != "" {
		u, err := app.PresignGet(r.Context(), v.ThumbnailKey)
		if err == nil {
			thumbURL = u
		}
	}

	httpx.Ok(w, "video fetched", map[string]any{
		"id":           v.ID,
		"title":        v.Title,
		"description":  v.Description,
		"filename":     v.Filename,
		"contentType":  v.ContentType,
		"inputKey":     v.InputKey,
		"thumbnailKey": v.ThumbnailKey,
		"thumbnailUrl": thumbURL,
		"latestJobId":  v.LatestJobID,
		"status":       v.Status,
		"errorMsg":     v.ErrorMsg,
		"createdAt":    v.CreatedAt,
		"updatedAt":    v.UpdatedAt,
	})
}

func (app *application) CreateVideoJob(w http.ResponseWriter, r *http.Request) {
	videoID := chi.URLParam(r, "id")
	if strings.TrimSpace(videoID) == "" {
		httpx.Fail(w, 400, "VALIDATION_ERROR", "id is required")
		return
	}

	// Ensure video exists (also gives us input_key)
	v, err := app.store.Video.Get(r.Context(), videoID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) || errors.Is(err, sql.ErrNoRows) {
			httpx.Fail(w, 404, "NOT_FOUND", "video not found")
			return
		}
		httpx.Fail(w, 500, "DB_ERROR", err.Error())
		return
	}

	var req types.CreateVideoJobReq
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Pipeline == "" {
		req.Pipeline = "hls"
	}

	
	jobID := uuid.NewString()

	if err := app.store.Job.Create(r.Context(), store.Job{
		ID:       jobID,
		VideoID:  videoID,
		InputKey: v.InputKey,
		Pipeline: req.Pipeline,
		Status:   store.JobQueued,
		Progress: 0,
	}); err != nil {
		httpx.Fail(w, 500, "DB_ERROR", err.Error())
		return
	}

	_ = app.store.Video.SetLatestJob(r.Context(), videoID, jobID)

	

	resp, err := app.producer.Enqueue(r.Context(), &producerpb.EnqueueTranscodeJobRequest{
		JobId:    jobID,
		VideoId:  videoID,
		InputKey: v.InputKey,
		Pipeline: req.Pipeline,
	})
	if err != nil {
		httpx.Fail(w, 502, "PRODUCER_UNAVAILABLE", err.Error())
		return
	}
	if !resp.GetAccepted() {
		httpx.Fail(w, 502, "PRODUCER_REJECTED", resp.GetMessage())
		return
	}

	httpx.Created(w, "job created", map[string]any{
		"videoId": videoID,
		"jobId":   jobID,
		"status":  "queued",
	})
}
