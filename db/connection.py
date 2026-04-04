import sqlite3
from db.uuid_util import NAMESPACE_UUID

# This must match schema_version after all migrations have been applied.
EXPECTED_SCHEMA_VERSION = 1


def open_db(path: str) -> sqlite3.Connection:
    """Open the pxvault SQLite database.

    Enables WAL mode and asserts that the schema version and namespace UUID
    match the compiled-in constants. Raises RuntimeError on any mismatch.
    """
    conn = sqlite3.connect(path)
    conn.row_factory = sqlite3.Row

    # Enable WAL mode for concurrent read access.
    result = conn.execute("PRAGMA journal_mode=WAL").fetchone()
    if result[0] != "wal":
        conn.close()
        raise RuntimeError(f"Failed to enable WAL mode, got: {result[0]!r}")

    # Enable foreign key enforcement.
    conn.execute("PRAGMA foreign_keys=ON")

    # Assert schema version matches the version this code was compiled against.
    row = conn.execute("SELECT version FROM schema_version").fetchone()
    if row is None:
        conn.close()
        raise RuntimeError("schema_version table is empty")
    version = row[0]
    if version != EXPECTED_SCHEMA_VERSION:
        conn.close()
        raise RuntimeError(
            f"Schema version mismatch: expected {EXPECTED_SCHEMA_VERSION}, got {version}"
        )

    # Assert namespace UUID stored in the database matches the compiled-in constant.
    row = conn.execute(
        "SELECT value FROM schema_metadata WHERE key = 'uuid_namespace'"
    ).fetchone()
    if row is None:
        conn.close()
        raise RuntimeError("uuid_namespace not found in schema_metadata")
    stored_ns = row[0]
    if stored_ns != str(NAMESPACE_UUID):
        conn.close()
        raise RuntimeError(
            f"Namespace UUID mismatch: expected {NAMESPACE_UUID!s}, got {stored_ns!r}"
        )

    return conn
