-- +migrate Up
ALTER TABLE actors ADD COLUMN IF NOT EXISTS spec TEXT;
