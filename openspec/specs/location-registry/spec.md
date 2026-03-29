# location-registry Specification

## Purpose
TBD - created by archiving change database-schema-core-models. Update Purpose after archive.
## Requirements
### Requirement: Locations table maps files to physical media
The database SHALL contain a `Locations` table with columns: `id` (UUID, primary key), `file_id` (UUID, foreign key → `Files.id`, not null), `media_id` (UUID, foreign key → `Media.id`, not null), `path_on_media` (TEXT, not null), `skip_for_counting` (INTEGER, boolean, default 0), `status` (TEXT, not null), `created_at` (TEXT, ISO-8601 UTC), `updated_at` (TEXT, ISO-8601 UTC). The `status` column SHALL be constrained to `'readable'`, `'unreadable'`, or `'lost'`.

#### Scenario: Location record inserted with valid fields
- **WHEN** a Locations record is inserted with valid `file_id`, `media_id`, `path_on_media`, and `status`
- **THEN** the insert SHALL succeed and the row SHALL be retrievable by `id`

#### Scenario: Location rejected for unknown file
- **WHEN** a Locations record is inserted with a `file_id` that does not exist in `Files`
- **THEN** the database SHALL reject the insert with a foreign key constraint violation

#### Scenario: Location rejected for unknown media
- **WHEN** a Locations record is inserted with a `media_id` that does not exist in `Media`
- **THEN** the database SHALL reject the insert with a foreign key constraint violation

#### Scenario: Invalid location status rejected
- **WHEN** a Locations record is inserted with a `status` value other than `'readable'`, `'unreadable'`, or `'lost'`
- **THEN** the database SHALL reject the insert with a CHECK constraint violation

### Requirement: skip_for_counting flag excludes locations from 3-2-1 compliance
The `skip_for_counting` flag on `Locations` SHALL allow specific file-media pairings to be excluded from 3-2-1 backup compliance counts. The flag SHALL default to `0` (false).

#### Scenario: skip_for_counting defaults to false
- **WHEN** a Locations record is inserted without specifying `skip_for_counting`
- **THEN** `skip_for_counting` SHALL be `0`

#### Scenario: Location excluded from compliance count when flagged
- **WHEN** a Locations record has `skip_for_counting = 1`
- **THEN** a query filtering `WHERE skip_for_counting = 0` SHALL NOT include that location in its results

#### Scenario: Location included in compliance count when not flagged
- **WHEN** a Locations record has `skip_for_counting = 0`
- **THEN** a query filtering `WHERE skip_for_counting = 0` SHALL include that location in its results

### Requirement: Location lifecycle status
The `status` field on `Locations` SHALL represent whether the file copy at that location is currently readable, known to be unreadable, or considered lost. Status changes SHALL be persisted with an updated `updated_at` timestamp.

#### Scenario: Location marked as unreadable
- **WHEN** a Locations record's `status` is updated to `'unreadable'`
- **THEN** the row SHALL reflect `status = 'unreadable'` and `updated_at` SHALL be updated to the current UTC time

#### Scenario: Location marked as lost
- **WHEN** a Locations record's `status` is updated to `'lost'`
- **THEN** the row SHALL reflect `status = 'lost'`

### Requirement: A file may have multiple locations
Multiple `Locations` rows SHALL be allowed with the same `file_id`, representing copies of the same file on different media. There is no unique constraint on `file_id` in the `Locations` table.

#### Scenario: Multiple locations for the same file
- **WHEN** two Locations records are inserted with the same `file_id` but different `media_id` values
- **THEN** both inserts SHALL succeed and both rows SHALL be retrievable by querying `WHERE file_id = ?`

### Requirement: Cascade delete cleans up locations
When a `Files` or `Media` row is deleted, all `Locations` rows that reference it SHALL be automatically deleted.

#### Scenario: Locations deleted when file is deleted
- **WHEN** a `Files` row is deleted
- **THEN** all `Locations` rows with that `file_id` SHALL be automatically deleted (CASCADE)

#### Scenario: Locations deleted when media is deleted
- **WHEN** a `Media` row is deleted
- **THEN** all `Locations` rows with that `media_id` SHALL be automatically deleted (CASCADE)

