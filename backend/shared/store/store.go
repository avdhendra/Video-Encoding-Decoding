package store

import (
	"context"
	"database/sql"
	"time"
)



type JobStatus string

const (
	JobQueued     JobStatus = "queued"
	JobProcessing JobStatus = "processing"
	JobCompleted  JobStatus = "completed"
	JobFailed     JobStatus = "failed"
)

type Job struct {
	ID       string
	VideoID  string
	InputKey string
	Pipeline string

	Status   JobStatus
	ErrorMsg *string

	OutputMasterKey     *string
	PlaybackReady       bool
	AvailableRenditions []string
	Progress            int

	CreatedAt time.Time
	UpdatedAt time.Time
}

// -------------------------
// Video model
// -------------------------

type Status string

const (
	Uploaded   Status = "uploaded"
	Processing Status = "processing"
	Ready      Status = "ready"
	Failed     Status = "failed"
)

type Video struct {
	ID          string
	Title       string
	Description string
	Filename    string
	ContentType string
	InputKey    string

	ThumbnailKey string
	LatestJobID  *string

	Status   Status
	ErrorMsg *string

	CreatedAt time.Time
	UpdatedAt time.Time
}

// -------------------------
// Stores
// -------------------------

type VideoStore struct{ db *sql.DB }
type JobStore struct{ db *sql.DB }

type Storage struct {
	Video interface {
		Create(ctx context.Context, v Video) error
		Get(ctx context.Context, id string) (Video, error)
		List(ctx context.Context, limit, offset int) ([]Video, int, error)

		SetLatestJob(ctx context.Context, videoID, jobID string) error
		MarkProcessing(ctx context.Context, id string) error
		MarkReady(ctx context.Context, id string) error
		MarkFailed(ctx context.Context, id, msg string) error
	}
	Job interface {
		Create(ctx context.Context, j Job) error
		Get(ctx context.Context, id string) (Job, error)

		MarkProcessing(ctx context.Context, id string) error
		MarkFailed(ctx context.Context, id, msg string) error
		MarkCompleted(ctx context.Context, id string) error

		UpdateProgress(ctx context.Context, id string, progress int, renditions []string, masterKey *string, playable bool) error
	}
}

func NewStorage(db *sql.DB) Storage {
	return Storage{
		Video: &VideoStore{db: db},
		Job:   &JobStore{db: db},
	}
}
