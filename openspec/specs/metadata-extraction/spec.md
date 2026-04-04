# metadata-extraction Specification

## Purpose
TBD - created by archiving change p2-file-scanner-ingestion. Update Purpose after archive.
## Requirements
### Requirement: Image metadata extraction
The system SHALL extract EXIF and basic metadata from image files.

#### Scenario: JPEG with EXIF
- **WHEN** processing a JPEG file with EXIF data
- **THEN** system extracts creation date, camera model, resolution

#### Scenario: PNG without EXIF
- **WHEN** processing a PNG file without EXIF data
- **THEN** system extracts basic image dimensions

### Requirement: Video metadata extraction
The system SHALL extract duration, resolution, and codec information from video files.

#### Scenario: MP4 video file
- **WHEN** processing an MP4 video file
- **THEN** system extracts duration, resolution, codec information

#### Scenario: Corrupted video file
- **WHEN** processing a corrupted video file
- **THEN** system logs error and skips file

### Requirement: Metadata structure
The system SHALL return metadata in a consistent JSON structure.

#### Scenario: Standard metadata format
- **WHEN** extracting metadata from any file
- **THEN** system returns JSON with consistent field names
