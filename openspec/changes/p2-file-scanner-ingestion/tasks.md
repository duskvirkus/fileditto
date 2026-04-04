## 1. Setup and Dependencies

- [ ] 1.1 Create Python module structure for file scanner
- [ ] 1.2 Add required dependencies (Pillow, python-magic, ffprobe)
- [ ] 1.3 Set up development environment

## 2. File Scanner Implementation

- [ ] 2.1 Implement directory traversal using os.walk()
- [ ] 2.2 Add MIME type detection for file filtering
- [ ] 2.3 Implement file extension filtering
- [ ] 2.4 Add network share support (SMB/NFS)

## 3. Metadata Extraction

- [ ] 3.1 Implement image metadata extraction with Pillow
- [ ] 3.2 Implement video metadata extraction with ffprobe
- [ ] 3.3 Create consistent JSON metadata structure
- [ ] 3.4 Add error handling for corrupted files

## 4. Ingestion Queue

- [ ] 4.1 Implement in-memory queue for candidate files
- [ ] 4.2 Add persistent queue support for large collections
- [ ] 4.3 Create queue interface for next processing phase

## 5. Integration and Testing

- [ ] 5.1 Integrate scanner with database schema from P1
- [ ] 5.2 Write unit tests for file scanning
- [ ] 5.3 Write unit tests for metadata extraction
- [ ] 5.4 Write integration tests for full ingestion pipeline

## 6. Documentation

- [ ] 6.1 Add module documentation
- [ ] 6.2 Update README with scanner usage
- [ ] 6.3 Add examples for common use cases