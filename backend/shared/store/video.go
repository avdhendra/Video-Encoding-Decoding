package store

import (
	"context"
	"database/sql"
	"errors"
)

var ErrNotFound = errors.New("video not found")

func (v *VideoStore) Create(ctx context.Context, video Video) error {
	const q = `
		INSERT INTO videos
			(id, title, description, filename, content_type, input_key, thumbnail_key, latest_job_id, status, error_msg)
		VALUES
			($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	`
	_, err := v.db.ExecContext(
		ctx,
		q,
		video.ID,
		video.Title,
		video.Description,
		video.Filename,
		video.ContentType,
		video.InputKey,
		video.ThumbnailKey,
		video.LatestJobID, // can be nil
		string(video.Status),
		video.ErrorMsg, // can be nil
	)
	return err
}

func (v *VideoStore) Get(ctx context.Context, id string) (Video, error) {
	const q = `
		SELECT
			id, title, description, filename, content_type, input_key,
			thumbnail_key, latest_job_id,
			status, error_msg,
			created_at, updated_at
		FROM videos
		WHERE id = $1
	`

	var out Video
	var latestJob sql.NullString
	var errMsg sql.NullString
	var status string

	err := v.db.QueryRowContext(ctx, q, id).Scan(
		&out.ID,
		&out.Title,
		&out.Description,
		&out.Filename,
		&out.ContentType,
		&out.InputKey,
		&out.ThumbnailKey,
		&latestJob,
		&status,
		&errMsg,
		&out.CreatedAt,
		&out.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Video{}, ErrNotFound
		}
		return Video{}, err
	}

	out.Status = Status(status)

	if latestJob.Valid {
		out.LatestJobID = &latestJob.String
	} else {
		out.LatestJobID = nil
	}

	if errMsg.Valid {
		out.ErrorMsg = &errMsg.String
	} else {
		out.ErrorMsg = nil
	}

	return out, nil
}

func (v *VideoStore) List(ctx context.Context, limit, offset int) ([]Video, int, error) {
	// Total count
	const qCount = `SELECT COUNT(*) FROM videos`
	var total int
	if err := v.db.QueryRowContext(ctx, qCount).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Items
	const q = `
		SELECT
			id, title, description, filename, content_type, input_key,
			thumbnail_key, latest_job_id,
			status, error_msg,
			created_at, updated_at
		FROM videos
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := v.db.QueryContext(ctx, q, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]Video, 0, limit)

	for rows.Next() {
		var item Video
		var latestJob sql.NullString
		var errMsg sql.NullString
		var status string

		if err := rows.Scan(
			&item.ID,
			&item.Title,
			&item.Description,
			&item.Filename,
			&item.ContentType,
			&item.InputKey,
			&item.ThumbnailKey,
			&latestJob,
			&status,
			&errMsg,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}

		item.Status = Status(status)

		if latestJob.Valid {
			item.LatestJobID = &latestJob.String
		}

		if errMsg.Valid {
			item.ErrorMsg = &errMsg.String
		}

		out = append(out, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return out, total, nil
}

func (v *VideoStore) SetLatestJob(ctx context.Context, videoID, jobID string) error {
	const q = `
		UPDATE videos
		SET latest_job_id = $2,
		    updated_at = now()
		WHERE id = $1
	`
	res, err := v.db.ExecContext(ctx, q, videoID, jobID)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return ErrNotFound
	}
	return nil
}

func (v *VideoStore) MarkProcessing(ctx context.Context, id string) error {
	return v.setStatus(ctx, id, Processing, nil)
}

func (v *VideoStore) MarkReady(ctx context.Context, id string) error {
	// clear error_msg on success
	empty := (*string)(nil)
	return v.setStatus(ctx, id, Ready, empty)
}

func (v *VideoStore) MarkFailed(ctx context.Context, id, msg string) error {
	return v.setStatus(ctx, id, Failed, &msg)
}

// ---- internal helper ----

func (v *VideoStore) setStatus(ctx context.Context, id string, status Status, errMsg *string) error {
	const q = `
		UPDATE videos
		SET status = $2,
		    error_msg = $3,
		    updated_at = now()
		WHERE id = $1
	`
	res, err := v.db.ExecContext(ctx, q, id, string(status), errMsg)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return ErrNotFound
	}
	return nil
}

