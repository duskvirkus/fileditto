# file-registry Specification

## Purpose
TBD - created by archiving change database-schema-core-models. Update Purpose after archive.
## Requirements
### Requirement: Files table stores canonical file records
The database SHALL contain a `Files` table with columns: `id` (UUID, primary key), `sha256` (TEXT, unique, not null), `size_bytes` (INTEGER, not null), `file_type` (TEXT, not null), `created_at` (TEXT, ISO-8601 UTC), `updated_at` (TEXT, ISO-8601 UTC).

#### Scenario: File record inserted with all required fields
- **WHEN** a new file record is inserted with a valid UUID, SHA-256 hash, size, and file type
- **THEN** the row SHALL be persisted and retrievable by its `id`

#### Scenario: Duplicate SHA-256 rejected
- **WHEN** an insert is attempted with a `sha256` value that already exists in `Files`
- **THEN** the database SHALL reject the insert with a unique constraint violation

#### Scenario: Missing required field rejected
- **WHEN** an insert is attempted with a NULL value for `sha256`, `size_bytes`, or `file_type`
- **THEN** the database SHALL reject the insert with a not-null constraint violation

### Requirement: Deterministic UUID v5 for file identity
File UUIDs SHALL be generated using UUID v5 with a fixed project namespace UUID and the file's SHA-256 content hash as the name input. The same SHA-256 hash SHALL always produce the same UUID.

#### Scenario: Same hash produces same UUID
- **WHEN** UUID v5 generation is called twice with the same SHA-256 hash
- **THEN** both calls SHALL return the identical UUID string

#### Scenario: Different hashes produce different UUIDs
- **WHEN** UUID v5 generation is called with two different SHA-256 hashes
- **THEN** the resulting UUIDs SHALL be different

#### Scenario: UUID generated in both Go and Python is identical
- **WHEN** Go and Python both generate a UUID v5 for the same SHA-256 hash using the same namespace
- **THEN** both SHALL produce the same UUID string

### Requirement: Photo table stores photo-specific metadata
The database SHALL contain a `Photo` table with columns: `id` (UUID, primary key, foreign key → `Files.id`), `width_px` (INTEGER), `height_px` (INTEGER), `color_profile` (TEXT), `metadata_blob` (BLOB). The `id` column is both the primary key and a foreign key referencing `Files.id`.

#### Scenario: Photo record linked to Files entry
- **WHEN** a Photo record is inserted with an `id` that exists in `Files`
- **THEN** the insert SHALL succeed and the row SHALL be retrievable via JOIN on `Files.id`

#### Scenario: Photo record rejected for unknown file
- **WHEN** a Photo record is inserted with an `id` that does not exist in `Files`
- **THEN** the database SHALL reject the insert with a foreign key constraint violation

#### Scenario: Photo deleted when parent file deleted
- **WHEN** a `Files` row is deleted
- **THEN** the corresponding `Photo` row SHALL be automatically deleted (CASCADE)

### Requirement: Video table stores video-specific metadata
The database SHALL contain a `Video` table with columns: `id` (UUID, primary key, foreign key → `Files.id`), `duration_seconds` (REAL), `width_px` (INTEGER), `height_px` (INTEGER), `frame_rate` (REAL), `codec` (TEXT), `metadata_blob` (BLOB). The `id` column is both the primary key and a foreign key referencing `Files.id`.

#### Scenario: Video record linked to Files entry
- **WHEN** a Video record is inserted with an `id` that exists in `Files`
- **THEN** the insert SHALL succeed and the row SHALL be retrievable via JOIN on `Files.id`

#### Scenario: Video record rejected for unknown file
- **WHEN** a Video record is inserted with an `id` that does not exist in `Files`
- **THEN** the database SHALL reject the insert with a foreign key constraint violation

#### Scenario: Video deleted when parent file deleted
- **WHEN** a `Files` row is deleted
- **THEN** the corresponding `Video` row SHALL be automatically deleted (CASCADE)

### Requirement: File type discriminator is queryable
The `file_type` column on `Files` SHALL use a constrained set of values (`'generic'`, `'photo'`, `'video'`) to identify whether a subtype record exists. This column SHALL have a CHECK constraint enforcing valid values.

#### Scenario: Valid file type accepted
- **WHEN** a Files record is inserted with `file_type` of `'generic'`, `'photo'`, or `'video'`
- **THEN** the insert SHALL succeed

#### Scenario: Invalid file type rejected
- **WHEN** a Files record is inserted with an unrecognized `file_type` value
- **THEN** the database SHALL reject the insert with a CHECK constraint violation

