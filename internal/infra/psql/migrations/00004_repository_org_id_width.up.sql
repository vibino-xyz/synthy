-- repository.organization_id holds a nexy-owned organization id, which nexy
-- stores as a 36-char UUID (VARCHAR(36)). synthy's column was VARCHAR(32),
-- overflowing on insert ("value too long for type character varying(32)").
-- Widen it to match nexy so the external reference fits.
ALTER TABLE repository ALTER COLUMN organization_id TYPE VARCHAR(36);
