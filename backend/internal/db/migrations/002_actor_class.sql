-- +migrate Up
ALTER TABLE actors ADD COLUMN IF NOT EXISTS class TEXT;
