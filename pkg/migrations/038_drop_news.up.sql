-- Migration: 038_drop_news
-- Purpose: Remove the unused news module and its table.

DROP TABLE IF EXISTS news;
