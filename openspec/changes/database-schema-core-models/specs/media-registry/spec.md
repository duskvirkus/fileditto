## ADDED Requirements

### Requirement: Media table tracks physical storage media
The database SHALL contain a `Media` table with columns: `id` (UUID, primary key), `media_type` (TEXT, not null), `label` (TEXT), `status` (TEXT, not null), `subtype_id` (UUID), `created_at` (TEXT, ISO-8601 UTC), `updated_at` (TEXT, ISO-8601 UTC). The `media_type` column SHALL be constrained to `'bluray'` or `'drive'`. The `status` column SHALL be constrained to `'active'`, `'inactive'`, or `'lost'`.

#### Scenario: Media record inserted with valid type and status
- **WHEN** a Media record is inserted with `media_type` of `'bluray'` or `'drive'` and `status` of `'active'`, `'inactive'`, or `'lost'`
- **THEN** the insert SHALL succeed

#### Scenario: Invalid media_type rejected
- **WHEN** a Media record is inserted with a `media_type` value other than `'bluray'` or `'drive'`
- **THEN** the database SHALL reject the insert with a CHECK constraint violation

#### Scenario: Invalid status rejected
- **WHEN** a Media record is inserted with a `status` value other than `'active'`, `'inactive'`, or `'lost'`
- **THEN** the database SHALL reject the insert with a CHECK constraint violation

### Requirement: BlurayMedia table stores Blu-ray-specific attributes
The database SHALL contain a `BlurayMedia` table with columns: `id` (UUID, primary key), `disc_label` (TEXT), `capacity_gb` (REAL), `burn_date` (TEXT, ISO-8601 date), `disc_set_number` (INTEGER), `verified` (INTEGER, boolean, default 0). The `id` of a `BlurayMedia` row SHALL match the `subtype_id` of the corresponding `Media` row.

#### Scenario: BlurayMedia record inserted with valid fields
- **WHEN** a BlurayMedia record is inserted with a valid UUID and numeric capacity
- **THEN** the insert SHALL succeed and the record SHALL be retrievable by `id`

#### Scenario: Media and BlurayMedia linked via subtype_id
- **WHEN** a Media row with `media_type = 'bluray'` has a `subtype_id` that matches a BlurayMedia `id`
- **THEN** a JOIN between `Media` and `BlurayMedia` on `Media.subtype_id = BlurayMedia.id` SHALL return the combined record

#### Scenario: BlurayMedia verified flag defaults to false
- **WHEN** a BlurayMedia record is inserted without specifying `verified`
- **THEN** `verified` SHALL default to `0`

### Requirement: DriveMedia table stores drive-specific attributes
The database SHALL contain a `DriveMedia` table with columns: `id` (UUID, primary key), `serial_number` (TEXT, unique), `make` (TEXT), `model` (TEXT), `capacity_gb` (REAL), `interface_type` (TEXT), `acquired_date` (TEXT, ISO-8601 date). The `id` of a `DriveMedia` row SHALL match the `subtype_id` of the corresponding `Media` row.

#### Scenario: DriveMedia record inserted with valid fields
- **WHEN** a DriveMedia record is inserted with a valid UUID and serial number
- **THEN** the insert SHALL succeed and the record SHALL be retrievable by `id`

#### Scenario: Duplicate serial number rejected
- **WHEN** a DriveMedia record is inserted with a `serial_number` that already exists
- **THEN** the database SHALL reject the insert with a unique constraint violation

#### Scenario: Media and DriveMedia linked via subtype_id
- **WHEN** a Media row with `media_type = 'drive'` has a `subtype_id` that matches a DriveMedia `id`
- **THEN** a JOIN between `Media` and `DriveMedia` on `Media.subtype_id = DriveMedia.id` SHALL return the combined record

### Requirement: Media lifecycle status transitions
The `status` field on `Media` SHALL represent the lifecycle state of the physical medium. Status changes SHALL be persisted by updating the `status` column and the `updated_at` timestamp.

#### Scenario: Media marked as lost
- **WHEN** a Media record's `status` is updated to `'lost'`
- **THEN** the updated row SHALL reflect `status = 'lost'` and `updated_at` SHALL be updated to the current UTC time

#### Scenario: Media marked as inactive
- **WHEN** a Media record's `status` is updated to `'inactive'`
- **THEN** the updated row SHALL reflect `status = 'inactive'`
