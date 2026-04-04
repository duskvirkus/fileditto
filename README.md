# pxvault

Personal media vault system for organizing and backing up photos and videos.

## Requirements

- Python 3.13+
- `uv` (for dependency management)
- `openspec` CLI (for change management)

## Quick Start

```bash
# Install dependencies
uv add pillow python-magic

# Run the file scanner
python3 -c "
from scanner.file_scanner import FileScanner
scanner = FileScanner()
files = scanner.scan_directory('/path/to/photos')
print(f'Found {len(files)} media files')
"
```

## File Scanner Usage

The scanner module provides comprehensive file discovery and metadata extraction:

### Basic Directory Scanning

```python
from scanner.file_scanner import FileScanner

scanner = FileScanner()
files = scanner.scan_directory('/path/to/photos')
print(f'Found {len(files)} media files')
```

### Metadata Extraction

```python
from scanner.metadata_extractor import MetadataExtractor

extractor = MetadataExtractor()
metadata = extractor.extract_metadata('/path/to/photo.jpg')
print(f'Dimensions: {metadata["width"]}x{metadata["height"]}')
```

### Network Share Scanning

```python
from scanner.file_scanner import FileScanner

scanner = FileScanner()
# Scan SMB share
smb_files = scanner.scan_smb_share('//server/share', 'username', 'password')
# Scan NFS share
nfs_files = scanner.scan_nfs_share('/mnt/nfs/share')
```

### Ingestion Queue Management

```python
from scanner.ingestion_queue import IngestionQueue

queue = IngestionQueue()
queue.add_files(['/path/to/file1.jpg', '/path/to/file2.mp4'])
next_file = queue.get_next_file()
```

### Database Integration

```python
from scanner.database_integration import DatabaseIntegrator
from scanner.file_scanner import FileScanner
from scanner.metadata_extractor import MetadataExtractor

# Scan files
scanner = FileScanner()
files = scanner.scan_directory('/path/to/photos')

# Extract metadata
Extractor = MetadataExtractor()
metadata_list = [extractor.extract_metadata(file) for file in files]

# Integrate with database
integrator = DatabaseIntegrator('sqlite:///media.db')
integrator.integrate_files(files, metadata_list)
```

## Project Structure

- `scanner/` - File scanning and metadata extraction modules
- `db/` - Database modules (from P1)
- `openspec/` - Change management and specifications

## Development

This project uses OpenSpec for change management:

```bash
# List changes
openspec list

# View change status
openspec status --change "change-name"
```

See individual change directories in `openspec/changes/` for detailed specifications.

## Common Use Cases

### Batch Processing Media Collection

```python
from scanner.file_scanner import FileScanner
from scanner.metadata_extractor import MetadataExtractor
from scanner.database_integration import DatabaseIntegrator

# Step 1: Scan entire media collection
scanner = FileScanner()
all_files = []
for directory in ['/photos/2023', '/photos/2024', '/videos']:
    all_files.extend(scanner.scan_directory(directory))

# Step 2: Extract metadata for all files
Extractor = MetadataExtractor()
all_metadata = []
for file_path in all_files:
    try:
        metadata = extractor.extract_metadata(file_path)
        all_metadata.append(metadata)
    except Exception as e:
        print(f'Error processing {file_path}: {e}')

# Step 3: Store in database
integrator = DatabaseIntegrator('sqlite:///media.db')
integrator.integrate_files(all_files, all_metadata)
```

### Network Share Backup

```python
from scanner.file_scanner import FileScanner
from scanner.ingestion_queue import IngestionQueue

# Scan network shares
scanner = FileScanner()
queue = IngestionQueue()

# Add files from multiple network shares to queue
smb_files = scanner.scan_smb_share('//nas/photos', 'user', 'pass')
queue.add_files(smb_files)

nfs_files = scanner.scan_nfs_share('/mnt/backup/videos')
queue.add_files(nfs_files)

# Process queue
while queue.has_files():
    file_path = queue.get_next_file()
    # Process file (backup, metadata extraction, etc.)
    print(f'Processing: {file_path}')
```

### Metadata Analysis

```python
from scanner.file_scanner import FileScanner
from scanner.metadata_extractor import MetadataExtractor
import json

# Scan and analyze metadata
scanner = FileScanner()
extractor = MetadataExtractor()

files = scanner.scan_directory('/path/to/photos')
metadata_results = []

for file_path in files:
    metadata = extractor.extract_metadata(file_path)
    metadata_results.append({
        'file': file_path,
        'type': metadata.get('file_type'),
        'size': metadata.get('file_size'),
        'dimensions': f"{metadata.get('width')}x{metadata.get('height')}",
        'date_taken': metadata.get('date_taken')
    })

# Save analysis results
with open('metadata_analysis.json', 'w') as f:
    json.dump(metadata_results, f, indent=2)

print(f'Analyzed {len(metadata_results)} files')
```