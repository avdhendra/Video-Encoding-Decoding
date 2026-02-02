CREATE TABLE IF NOT EXISTS videos (
  id TEXT PRIMARY KEY,

  title TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  filename TEXT NOT NULL DEFAULT '',
  content_type TEXT NOT NULL DEFAULT 'video/mp4',

  input_key TEXT NOT NULL,
  thumbnail_key TEXT NOT NULL DEFAULT '',
  latest_job_id TEXT,

  status TEXT NOT NULL CHECK (status IN ('uploaded','processing','ready','failed')) DEFAULT 'uploaded',
  error_msg TEXT,

  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_videos_created_at ON videos(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_videos_status ON videos(status);
CREATE INDEX IF NOT EXISTS idx_videos_latest_job_id ON videos(latest_job_id);


-- -------------------------
-- jobs
-- -------------------------
CREATE TABLE IF NOT EXISTS jobs (
  id TEXT PRIMARY KEY,
  video_id TEXT NOT NULL REFERENCES videos(id) ON DELETE CASCADE,

  input_key TEXT NOT NULL,
  pipeline TEXT NOT NULL DEFAULT 'hls',

  status TEXT NOT NULL CHECK (status IN ('queued','processing','completed','failed')) DEFAULT 'queued',
  error_msg TEXT,

  -- HLS master playlist key (e.g. reels/outputs/<video>/<job>/master.m3u8)
  output_master_key TEXT,
  playback_ready BOOLEAN NOT NULL DEFAULT FALSE,

  -- ["480p","720p","1080p"]
  available_renditions JSONB NOT NULL DEFAULT '[]'::jsonb,
  progress INT NOT NULL DEFAULT 0 CHECK (progress >= 0 AND progress <= 100),

  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_jobs_video_id ON jobs(video_id);
CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
CREATE INDEX IF NOT EXISTS idx_jobs_created_at ON jobs(created_at DESC);
