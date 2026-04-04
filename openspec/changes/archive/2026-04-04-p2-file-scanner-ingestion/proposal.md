## Why

The P2 phase focuses on implementing the file scanner and ingestion system, which is essential for discovering and processing photos and videos before backup. This phase builds on the database schema (P1) and enables subsequent phases like hashing, deduplication, and backup operations. Without a robust scanner, the system cannot identify files to back up.

## What Changes

- Implement a directory tree walker to discover photos and videos
- Extract metadata (EXIF, creation date, resolution, video duration) from discovered files
- Output a queue of candidate files with extracted metadata for further processing
- No files are written to backup destinations in this phase

## Capabilities

### New Capabilities
- `file-scanner`: Directory traversal and file discovery based on MIME types and extensions
- `metadata-extraction`: Extract EXIF and other metadata from photos and videos
- `ingestion-queue`: Queue system for candidate files ready for deduplication

### Modified Capabilities

None (this is a new feature set building on P1 database schema)

## Impact

- Adds Python dependencies for media metadata extraction (Pillow, ffprobe bindings)
- Creates new modules for file scanning and metadata extraction
- Database schema from P1 will be used to store file metadata
- No breaking changes to existing functionality