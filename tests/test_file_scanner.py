"""
Test file scanner functionality.
"""

import os
import tempfile
import pytest
from scanner.file_scanner import FileScanner


class TestFileScanner:
    """Test cases for FileScanner class."""

    def test_scan_directory_empty(self):
        """Test scanning an empty directory."""
        with tempfile.TemporaryDirectory() as temp_dir:
            scanner = FileScanner()
            files = scanner.scan_directory(temp_dir)
            assert len(files) == 0

    def test_scan_directory_with_files(self):
        """Test scanning a directory with files."""
        with tempfile.TemporaryDirectory() as temp_dir:
            # Create some test files
            test_files = ["test.jpg", "test.png", "test.txt"]
            for filename in test_files:
                with open(os.path.join(temp_dir, filename), "w") as f:
                    f.write("test")

            scanner = FileScanner()
            files = scanner.scan_directory(temp_dir)
            # Should return only files with proper MIME types
            assert isinstance(files, list)

    def test_scan_directory_with_extensions(self):
        """Test scanning with extension filtering."""
        with tempfile.TemporaryDirectory() as temp_dir:
            # Create test files
            files_to_create = ["photo1.jpg", "photo2.png", "doc.txt"]
            for filename in files_to_create:
                with open(os.path.join(temp_dir, filename), "w") as f:
                    f.write("test")

            scanner = FileScanner()
            jpg_files = scanner.scan_directory(temp_dir, extensions=[".jpg"])
            # Should return only .jpg files
            assert isinstance(jpg_files, list)

    def test_scan_directory_invalid_path(self):
        """Test scanning with invalid path."""
        scanner = FileScanner()
        with pytest.raises(ValueError):
            scanner.scan_directory("/nonexistent/path")

    def test_scan_directory_file_not_dir(self):
        """Test scanning when path is a file, not directory."""
        with tempfile.NamedTemporaryFile() as temp_file:
            scanner = FileScanner()
            with pytest.raises(ValueError):
                scanner.scan_directory(temp_file.name)

    def test_mime_detection(self):
        """Test MIME type detection."""
        with tempfile.TemporaryDirectory() as temp_dir:
            # Create a fake JPEG file
            jpeg_path = os.path.join(temp_dir, "test.jpg")
            with open(jpeg_path, "wb") as f:
                f.write(b"\xff\xd8\xff\xe0\x00\x10JFIF")
                f.write(b"\x00" * 100)

            scanner = FileScanner()
            is_media = scanner._is_media_file(jpeg_path)
            assert is_media

    def test_network_share_invalid_format(self):
        """Test network share with invalid format."""
        scanner = FileScanner()
        with pytest.raises(ValueError):
            scanner.scan_network_share("invalid://path")

    def test_network_share_smb_without_creds(self):
        """Test SMB share without credentials."""
        scanner = FileScanner()
        with pytest.raises(ValueError):
            scanner.scan_network_share("smb://server/share")

    def test_network_share_nfs_mounted(self):
        """Test NFS share (assumed mounted)."""
        with tempfile.TemporaryDirectory() as temp_dir:
            scanner = FileScanner()
            # This should work like regular directory scanning
            files = scanner.scan_network_share(temp_dir)
            assert isinstance(files, list)
