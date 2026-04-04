"""
Database Integration Module

This module integrates the file scanner with the P1 database schema.
"""

import os
import hashlib
import json
from datetime import datetime, timezone
from typing import Dict, List, Optional
import sqlite3

from db.connection import open_db
from db.uuid_util import file_uuid
from scanner.file_scanner import FileScanner
from scanner.metadata_extractor import MetadataExtractor


class DatabaseIntegrator:
    """Integrates file scanner results with the database schema."""

    def __init__(self, db_path: str):
        """
        Initialize the database integrator.

        Args:
            db_path: Path to the SQLite database file
        """
        self.db_path = db_path
        self.scanner = FileScanner()
        self.extractor = MetadataExtractor()

    def _calculate_file_hash(self, file_path: str) -> str:
        """
        Calculate SHA-256 hash of a file.

        Args:
            file_path: Path to the file

        Returns:
            SHA-256 hash as hex string
        """
        sha256_hash = hashlib.sha256()
        try:
            with open(file_path, "rb") as f:
                # Read file in chunks to handle large files
                for chunk in iter(lambda: f.read(4096), b""):
                    sha256_hash.update(chunk)
            return sha256_hash.hexdigest()
        except Exception as e:
            raise IOError(f"Failed to calculate hash for {file_path}: {e}")

    def _get_file_size(self, file_path: str) -> int:
        """
        Get file size in bytes.

        Args:
            file_path: Path to the file

        Returns:
            File size in bytes
        """
        try:
            return os.path.getsize(file_path)
        except Exception as e:
            raise IOError(f"Failed to get size for {file_path}: {e}")

    def _get_file_type(self, metadata: Dict) -> str:
        """
        Determine file type based on metadata.

        Args:
            metadata: Extracted metadata dictionary

        Returns:
            File type ('photo', 'video', or 'generic')
        """
        if metadata.get("type") == "image":
            return "photo"
        elif metadata.get("type") == "video":
            return "video"
        else:
            return "generic"

    def _insert_file_record(
        self, conn: sqlite3.Connection, file_hash: str, file_size: int, file_type: str
    ) -> Optional[str]:
        """
        Insert a file record into the Files table.

        Args:
            conn: Database connection
            file_hash: SHA-256 hash of the file
            file_size: File size in bytes
            file_type: File type ('photo', 'video', or 'generic')

        Returns:
            Generated file UUID
        """
        # Generate deterministic UUID from file hash
        file_uuid_value = file_uuid(file_hash)

        # Get current timestamp
        now = datetime.now(timezone.utc).isoformat()

        try:
            conn.execute(
                """
                INSERT INTO Files (id, sha256, size_bytes, file_type, created_at, updated_at)
                VALUES (?, ?, ?, ?, ?, ?)
                """,
                (str(file_uuid_value), file_hash, file_size, file_type, now, now),
            )
            return str(file_uuid_value)
        except sqlite3.IntegrityError as e:
            # Handle duplicate file (same hash already exists)
            if "UNIQUE constraint failed: Files.sha256" in str(e):
                # Return existing file UUID
                row = conn.execute(
                    "SELECT id FROM Files WHERE sha256 = ?", (file_hash,)
                ).fetchone()
                return row[0] if row else None
            raise

    def _insert_photo_metadata(
        self, conn: sqlite3.Connection, file_uuid: str, metadata: Dict
    ) -> None:
        """
        Insert photo metadata into the Photo table.

        Args:
            conn: Database connection
            file_uuid: File UUID
            metadata: Extracted metadata dictionary
        """
        try:
            conn.execute(
                """
                INSERT INTO Photo (id, width_px, height_px, color_profile, metadata_blob)
                VALUES (?, ?, ?, ?, ?)
                """,
                (
                    file_uuid,
                    metadata.get("width"),
                    metadata.get("height"),
                    metadata.get("mode"),  # Color profile
                    json.dumps(metadata.get("exif", {})).encode(
                        "utf-8"
                    ),  # EXIF data as blob
                ),
            )
        except sqlite3.IntegrityError:
            # Photo record already exists, update it
            conn.execute(
                """
                UPDATE Photo
                SET width_px = ?, height_px = ?, color_profile = ?, metadata_blob = ?
                WHERE id = ?
                """,
                (
                    metadata.get("width"),
                    metadata.get("height"),
                    metadata.get("mode"),
                    json.dumps(metadata.get("exif", {})).encode("utf-8"),
                    file_uuid,
                ),
            )

    def _insert_video_metadata(
        self, conn: sqlite3.Connection, file_uuid: str, metadata: Dict
    ) -> None:
        """
        Insert video metadata into the Video table.

        Args:
            conn: Database connection
            file_uuid: File UUID
            metadata: Extracted metadata dictionary
        """
        try:
            conn.execute(
                """
                INSERT INTO Video (id, duration_seconds, width_px, height_px, frame_rate, codec, metadata_blob)
                VALUES (?, ?, ?, ?, ?, ?, ?)
                """,
                (
                    file_uuid,
                    metadata.get("duration"),
                    metadata.get("width"),
                    metadata.get("height"),
                    None,  # frame_rate not extracted yet
                    metadata.get("codec"),
                    b"",  # metadata_blob not implemented yet
                ),
            )
        except sqlite3.IntegrityError:
            # Video record already exists, update it
            conn.execute(
                """
                UPDATE Video
                SET duration_seconds = ?, width_px = ?, height_px = ?, frame_rate = ?, codec = ?, metadata_blob = ?
                WHERE id = ?
                """,
                (
                    metadata.get("duration"),
                    metadata.get("width"),
                    metadata.get("height"),
                    None,  # frame_rate
                    metadata.get("codec"),
                    b"",  # metadata_blob
                    file_uuid,
                ),
            )

    def scan_and_store_files(
        self, directory_path: str, extensions: Optional[List[str]] = None
    ) -> int:
        """
        Scan a directory and store discovered files in the database.

        Args:
            directory_path: Path to scan
            extensions: Optional list of file extensions to filter by

        Returns:
            Number of files successfully stored in the database
        """
        # Scan directory for media files
        try:
            file_paths = self.scanner.scan_directory(directory_path, extensions)
            print(f"Found {len(file_paths)} media files")
        except Exception as e:
            print(f"Error scanning directory: {e}")
            return 0

        if not file_paths:
            print("No media files found")
            return 0

        # Open database connection
        try:
            conn = open_db(self.db_path)
        except Exception as e:
            print(f"Error opening database: {e}")
            return 0

        files_stored = 0

        try:
            # Process each file
            for file_path in file_paths:
                try:
                    # Extract metadata
                    metadata = self.extractor.extract_metadata(file_path)
                    if not metadata:
                        print(f"Skipping {file_path} - no metadata extracted")
                        continue

                    # Calculate file hash
                    file_hash = self._calculate_file_hash(file_path)

                    # Get file size
                    file_size = self._get_file_size(file_path)

                    # Determine file type
                    file_type = self._get_file_type(metadata)

                    # Insert file record
                    inserted_uuid = self._insert_file_record(
                        conn, file_hash, file_size, file_type
                    )
                    if not inserted_uuid:
                        print(f"Skipping {file_path} - file already exists in database")
                        continue

                    # Insert type-specific metadata
                    if file_type == "photo":
                        self._insert_photo_metadata(conn, inserted_uuid, metadata)
                    elif file_type == "video":
                        self._insert_video_metadata(conn, inserted_uuid, metadata)

                    files_stored += 1
                    print(f"Stored: {file_path} (UUID: {inserted_uuid})")

                except (IOError, OSError) as e:
                    print(f"File system error processing {file_path}: {e}")
                    continue
                except sqlite3.Error as e:
                    print(f"Database error processing {file_path}: {e}")
                    continue
                except Exception as e:
                    print(f"Unexpected error processing {file_path}: {e}")
                    continue

            conn.commit()
            print(f"Successfully stored {files_stored} files in database")
            return files_stored

        except Exception as e:
            conn.rollback()
            print(f"Error during database transaction: {e}")
            return 0

        finally:
            conn.close()
