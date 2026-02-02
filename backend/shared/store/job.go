package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

func (j *JobStore) Create(ctx context.Context, job Job) error {
	if job.Status == "" {
		job.Status = JobQueued
	}
	if job.Progress < 0 {
		job.Progress = 0
	}
	if job.Progress > 100 {
		job.Progress = 100
	}

	rendsJSON, _ := json.Marshal(job.AvailableRenditions)

	const q = `
		INSERT INTO jobs
			(id, video_id, input_key, pipeline, status, error_msg,
			 output_master_key, playback_ready, available_renditions, progress)
		VALUES
			($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,$10)
	`

	_, err := j.db.ExecContext(ctx, q,
		job.ID,
		job.VideoID,
		job.InputKey,
		job.Pipeline,
		string(job.Status),
		job.ErrorMsg,
		job.OutputMasterKey,
		job.PlaybackReady,
		string(rendsJSON),
		job.Progress,
	)
	return err
}

func (j *JobStore) Get(ctx context.Context, id string) (Job, error) {
	const q = `
		SELECT
			id, video_id, input_key, pipeline,
			status, error_msg,
			output_master_key, playback_ready,
			available_renditions, progress,
			created_at, updated_at
		FROM jobs
		WHERE id=$1
	`

	var out Job
	var status string
	var errMsg sql.NullString
	var master sql.NullString
	var rendsRaw []byte

	err := j.db.QueryRowContext(ctx, q, id).Scan(
		&out.ID,
		&out.VideoID,
		&out.InputKey,
		&out.Pipeline,
		&status,
		&errMsg,
		&master,
		&out.PlaybackReady,
		&rendsRaw,
		&out.Progress,
		&out.CreatedAt,
		&out.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Job{}, ErrNotFound
		}
		return Job{}, err
	}

	out.Status = JobStatus(status)

	if errMsg.Valid {
		out.ErrorMsg = &errMsg.String
	}
	if master.Valid {
		out.OutputMasterKey = &master.String
	}

	_ = json.Unmarshal(rendsRaw, &out.AvailableRenditions)
	return out, nil
}

func (j *JobStore) MarkProcessing(ctx context.Context, id string) error {
	return j.setStatus(ctx, id, JobProcessing, nil)
}

func (j *JobStore) MarkFailed(ctx context.Context, id, msg string) error {
	return j.setStatus(ctx, id, JobFailed, &msg)
}

func (j *JobStore) MarkCompleted(ctx context.Context, id string) error {
	// mark completed and progress=100, keep output fields as-is
	const q = `
		UPDATE jobs
		SET status='completed',
		    progress=100,
		    updated_at=now()
		WHERE id=$1
	`
	res, err := j.db.ExecContext(ctx, q, id)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return ErrNotFound
	}
	return nil
}

func (j *JobStore) UpdateProgress(ctx context.Context, id string, progress int, renditions []string, masterKey *string, playable bool) error {
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}

	rendsJSON, _ := json.Marshal(renditions)

	const q = `
		UPDATE jobs
		SET progress=$2,
		    available_renditions=$3::jsonb,
		    output_master_key=COALESCE($4, output_master_key),
		    playback_ready=$5,
		    updated_at=now()
		WHERE id=$1
	`
	res, err := j.db.ExecContext(ctx, q, id, progress, string(rendsJSON), masterKey, playable)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return ErrNotFound
	}
	return nil
}

func (j *JobStore) setStatus(ctx context.Context, id string, status JobStatus, errMsg *string) error {
	const q = `
		UPDATE jobs
		SET status=$2,
		    error_msg=$3,
		    updated_at=now()
		WHERE id=$1
	`
	res, err := j.db.ExecContext(ctx, q, id, string(status), errMsg)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return ErrNotFound
	}
	return nil
}
