CREATE DATABASE image_uploader;

\connect image_uploader;

CREATE TABLE IF NOT EXISTS images (
  id BIGSERIAL PRIMARY KEY,
  key VARCHAR(1024) NOT NULL,
  url TEXT NOT NULL,
  content_type VARCHAR(255),
  size BIGINT,
  etag VARCHAR(128),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_images_key ON images (key);
