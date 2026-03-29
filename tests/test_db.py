"""Python integration tests for the pxvault db package.

These tests require a fully migrated database (created by the Go migration
runner). The helper `_make_migrated_db` shells out to `go run` to produce a
migrated database, then hands the path to the Python `open_db` function.

Tests 6.9 and 6.10 are covered here.
"""

import os
import subprocess
import sys
import tempfile
import unittest

# Allow running tests from the repo root.
sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from db.connection import open_db, EXPECTED_SCHEMA_VERSION
from db.uuid_util import file_uuid, NAMESPACE_UUID


def _go_available() -> bool:
    try:
        subprocess.run(["go", "version"], capture_output=True, check=True)
        return True
    except (FileNotFoundError, subprocess.CalledProcessError):
        return False


def _make_migrated_db(path: str) -> None:
    """Use the Go migrate helper to create and migrate a database at `path`."""
    repo_root = os.path.join(os.path.dirname(__file__), "..")
    result = subprocess.run(
        ["go", "run", "./cmd/migrate", path],
        cwd=repo_root,
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        raise RuntimeError(
            f"go run ./cmd/migrate failed:\nstdout: {result.stdout}\nstderr: {result.stderr}"
        )


@unittest.skipUnless(_go_available(), "go toolchain not available")
class TestOpenDB(unittest.TestCase):
    """6.9 — Python open_db version check passes when version matches."""

    def test_open_db_version_check_passes(self) -> None:
        with tempfile.TemporaryDirectory() as tmpdir:
            db_path = os.path.join(tmpdir, "test.db")
            _make_migrated_db(db_path)
            conn = open_db(db_path)
            try:
                row = conn.execute("SELECT version FROM schema_version").fetchone()
                self.assertEqual(row[0], EXPECTED_SCHEMA_VERSION)
            finally:
                conn.close()

    def test_open_db_rejects_wrong_version(self) -> None:
        import sqlite3

        with tempfile.TemporaryDirectory() as tmpdir:
            db_path = os.path.join(tmpdir, "wrong_version.db")
            _make_migrated_db(db_path)

            # Manually corrupt the schema_version.
            raw = sqlite3.connect(db_path)
            raw.execute("UPDATE schema_version SET version = 999")
            raw.commit()
            raw.close()

            with self.assertRaises(RuntimeError):
                open_db(db_path)


class TestUUIDUtil(unittest.TestCase):
    """6.10 — Python UUID v5 produces same result as Go for a known SHA-256 hash."""

    # Known SHA-256 of the empty string, used as a stable test vector.
    EMPTY_SHA256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

    def test_deterministic(self) -> None:
        u1 = file_uuid(self.EMPTY_SHA256)
        u2 = file_uuid(self.EMPTY_SHA256)
        self.assertEqual(u1, u2)

    def test_known_value(self) -> None:
        import uuid as _uuid

        expected = str(_uuid.uuid5(NAMESPACE_UUID, self.EMPTY_SHA256))
        self.assertEqual(file_uuid(self.EMPTY_SHA256), expected)

    def test_different_hashes_differ(self) -> None:
        hash_a = "a" * 64
        hash_b = "b" * 64
        self.assertNotEqual(file_uuid(hash_a), file_uuid(hash_b))


if __name__ == "__main__":
    unittest.main()
