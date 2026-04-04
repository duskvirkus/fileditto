## Context

This design builds on P1 (database schema) to implement file scanning and ingestion. The system needs to discover photos and videos from local directories and network shares, extract metadata, and prepare files for deduplication and backup. Python is chosen for this phase due to its strong ecosystem for media metadata extraction.

## Goals / Non-Goals

**Goals:**
- Discover photos and videos by MIME type and file extension
- Extract comprehensive metadata (EXIF, creation date, resolution, video duration)
- Create a queue system for files ready for deduplication
- Support both local directories and network shares
- Handle large file collections efficiently

**Non-Goals:**
- Actual file copying or backup operations (handled in P4)
- Deduplication logic (handled in P3)
- Database schema changes (defined in P1)
- User interface for scanning (handled in P8)

## Decisions

**Language Choice - Python:**
- Rationale: Python has mature libraries for media metadata extraction (Pillow for images, ffprobe for videos) and file system operations
- Alternatives considered: Go (chosen for core engine but lacks mature media libraries), Rust (overkill for this phase)

**Metadata Extraction Libraries:**
- Images: Pillow (PIL) for EXIF and basic image metadata
- Videos: ffprobe (via subprocess) for duration, resolution, codecs
- Rationale: These are industry-standard tools with wide adoption

**File Discovery Approach:**
- Recursive directory traversal using os.walk()
- Filter by MIME type detection (python-magic) and file extensions
- Rationale: Balances accuracy with performance

**Queue System:**
- In-memory queue for candidate files during scan
- Persistent queue option for large collections
- Rationale: Allows pausing/resuming scans and handles system interruptions

**Network Share Support:**
- Use SMB/CIFS protocols for Windows shares
- Use NFS for Unix-like shares
- Rationale: Covers most common network storage scenarios

## Risks / Trade-offs

**Performance with Large Collections:**
- Risk: Scanning very large directories could be slow
- Mitigation: Implement batch processing and progress reporting

**Metadata Extraction Failures:**
- Risk: Some files may have corrupted metadata
- Mitigation: Graceful error handling with logging, skip problematic files

**Network Share Permissions:**
- Risk: Permission issues accessing network shares
- Mitigation: Configurable credentials and clear error messages

**Python Dependency Management:**
- Risk: Version conflicts in media libraries
- Mitigation: Pin specific versions in requirements.txt