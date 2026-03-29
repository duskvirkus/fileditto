## ADDED Requirements

### Requirement: Database file creation and opening
The system SHALL create a new SQLite database file at a specified path if it does not exist, or open the existing file if it does. The database SHALL be opened in WAL (Write-Ahead Logging) mode on every connection.

#### Scenario: New database created on first run
- **WHEN** the application starts and no database file exists at the configured path
- **THEN** the system SHALL create a new SQLite file at that path and enable WAL mode

#### Scenario: Existing database opened on subsequent runs
- **WHEN** the application starts and a database file already exists at the configured path
- **THEN** the system SHALL open the existing file without modifying existing data and enable WAL mode

#### Scenario: WAL mode enforced
- **WHEN** any connection is opened to the database
- **THEN** the system SHALL execute `PRAGMA journal_mode=WAL` and confirm the response is `wal`

### Requirement: Schema version table
The database SHALL contain a `schema_version` table that identifies the current schema version. This table SHALL always contain exactly one row.

#### Scenario: Schema version table exists after migration
- **WHEN** the migration runner has completed
- **THEN** `SELECT version FROM schema_version` SHALL return exactly one row with a positive integer

#### Scenario: Schema version reflects latest applied migration
- **WHEN** all pending migrations have been applied
- **THEN** the `version` value in `schema_version` SHALL equal the highest migration number applied

### Requirement: Migration history table
The database SHALL contain a `schema_migrations` table that records every applied migration with its number, applied timestamp, and SHA-256 checksum of the migration SQL.

#### Scenario: Migration recorded on application
- **WHEN** a migration is successfully applied
- **THEN** a row SHALL be inserted into `schema_migrations` with the migration number, a UTC timestamp, and the SHA-256 checksum of the migration file content

#### Scenario: Checksum mismatch detected
- **WHEN** the migration runner encounters a previously-applied migration whose file checksum does not match the stored checksum
- **THEN** the system SHALL halt with a hard error and SHALL NOT apply any further migrations

### Requirement: Forward-only migration runner
The Go binary SHALL embed all numbered SQL migration files and execute them in ascending order on startup, applying only migrations with numbers greater than the current `schema_version`.

#### Scenario: No pending migrations
- **WHEN** the database `schema_version` equals the highest embedded migration number
- **THEN** the migration runner SHALL complete immediately without executing any SQL

#### Scenario: Pending migrations applied in order
- **WHEN** the database `schema_version` is less than the highest embedded migration number
- **THEN** the migration runner SHALL apply each missing migration in ascending numeric order within a single transaction per migration

#### Scenario: Migration failure halts runner
- **WHEN** a migration SQL statement returns an error
- **THEN** the transaction for that migration SHALL be rolled back and the runner SHALL halt with an error before applying any subsequent migrations

### Requirement: Python version assertion
The Python DB access layer SHALL assert on startup that the database `schema_version` matches the version it was compiled against, raising a hard error if they differ.

#### Scenario: Version matches
- **WHEN** the Python layer opens the database and `schema_version` matches the expected constant
- **THEN** the Python layer SHALL proceed normally

#### Scenario: Version mismatch
- **WHEN** the Python layer opens the database and `schema_version` does not match the expected constant
- **THEN** the Python layer SHALL raise a hard error and SHALL NOT execute any further queries

### Requirement: Project namespace UUID stored in schema metadata
The database SHALL contain a `schema_metadata` table with a row storing the project namespace UUID used for UUID v5 generation. This acts as a sanity check against accidental namespace changes.

#### Scenario: Namespace UUID present after initial migration
- **WHEN** the initial migration has been applied
- **THEN** `SELECT value FROM schema_metadata WHERE key = 'uuid_namespace'` SHALL return the hardcoded project namespace UUID string

#### Scenario: Namespace UUID mismatch detected
- **WHEN** the application reads `schema_metadata` and the stored namespace UUID differs from the compiled-in constant
- **THEN** the application SHALL halt with a hard error
