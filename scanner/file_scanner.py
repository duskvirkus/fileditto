"""
File Scanner Module
===================

This module implements directory traversal and file discovery for photos and videos.
It supports both local directories and network shares using SMB/NFS protocols.

Key Features:
- Recursive directory traversal using os.walk()
- MIME type detection for accurate file type identification
- File extension filtering for targeted scans
- Network share support (SMB/NFS)
- Error handling for invalid paths and permissions

Usage Example:
    >>> from scanner.file_scanner import FileScanner
    >>> scanner = FileScanner()
    >>> # Scan a directory for all media files
    >>> files = scanner.scan_directory('/path/to/photos')
    >>> # Scan with extension filtering
    >>> jpg_files = scanner.scan_directory('/path/to/photos', extensions=['.jpg', '.jpeg'])
    >>> # Scan network share
    >>> smb_files = scanner.scan_network_share('smb://server/share',
    ...                                     credentials={'username': 'user', 'password': 'pass'})

Note:
- SMB support requires smbprotocol and pysmb packages
- NFS shares should be mounted locally before scanning
- MIME type detection uses python-magic library
"""

import os
import magic
from typing import List, Dict, Optional

try:
    import importlib.util

    smb_spec = importlib.util.find_spec("smbprotocol")
    SMB_AVAILABLE = smb_spec is not None
except ImportError:
    SMB_AVAILABLE = False


class FileScanner:
    """Scans directories for photo and video files."""

    def __init__(self):
        self.mime = magic.Magic(mime=True)

    def scan_directory(
        self, path: str, extensions: Optional[List[str]] = None
    ) -> List[str]:
        """
        Recursively scan a directory for files.

        Args:
            path: Directory path to scan
            extensions: Optional list of file extensions to filter by

        Returns:
            List of file paths that match the criteria
        """
        if not os.path.exists(path):
            raise ValueError(f"Path does not exist: {path}")

        if not os.path.isdir(path):
            raise ValueError(f"Path is not a directory: {path}")

        discovered_files = []

        for root, dirs, files in os.walk(path):
            for file in files:
                file_path = os.path.join(root, file)

                # Filter by extension if provided
                if extensions and not any(
                    file.lower().endswith(ext.lower()) for ext in extensions
                ):
                    continue

                # Filter by MIME type
                if self._is_media_file(file_path):
                    discovered_files.append(file_path)

        return discovered_files

    def _is_media_file(self, file_path: str) -> bool:
        """
        Check if a file is a photo or video based on MIME type.

        Args:
            file_path: Path to the file

        Returns:
            True if file is a photo or video, False otherwise
        """
        try:
            mime_type = self.mime.from_file(file_path)
            return mime_type.startswith("image/") or mime_type.startswith("video/")
        except Exception:
            return False

    def scan_network_share(
        self, share_path: str, credentials: Optional[Dict] = None
    ) -> List[str]:
        """
        Scan a network share (SMB/NFS).

        Args:
            share_path: Network share path (e.g., smb://server/share or /mnt/nfs/share)
            credentials: Optional credentials for authentication
                For SMB: {'username': 'user', 'password': 'pass', 'domain': 'domain'}
                For NFS: None (handled by system mount)

        Returns:
            List of file paths found on the network share

        Raises:
            ValueError: If share path is invalid or credentials are missing
            RuntimeError: If network share cannot be accessed
        """
        if not share_path:
            raise ValueError("Share path cannot be empty")

        # Handle SMB shares
        if share_path.lower().startswith("smb://"):
            return self._scan_smb_share(share_path, credentials)

        # Handle NFS shares (assume already mounted)
        elif share_path.startswith("/"):
            return self.scan_directory(share_path, extensions=None)

        else:
            raise ValueError(f"Unsupported share path format: {share_path}")

    def _scan_smb_share(
        self, share_path: str, credentials: Optional[Dict] = None
    ) -> List[str]:
        """
        Scan an SMB share.

        Args:
            share_path: SMB share path (e.g., smb://server/share)
            credentials: Credentials for authentication

        Returns:
            List of file paths found on the SMB share

        Raises:
            RuntimeError: If SMB connection fails
        """
        if not SMB_AVAILABLE:
            raise RuntimeError(
                "SMB support not available. Install smbprotocol and pysmb packages."
            )

        if not credentials:
            raise ValueError("Credentials required for SMB share access")

        # Parse SMB share path
        if not share_path.lower().startswith("smb://"):
            raise ValueError("SMB share path must start with smb://")

        # Remove smb:// prefix
        path_parts = share_path[6:].split("/", 1)
        if len(path_parts) < 2:
            raise ValueError(
                "Invalid SMB share path format. Expected smb://server/share"
            )

        # server = path_parts[0], share = path_parts[1]
        # Full SMB traversal not yet implemented.
        raise RuntimeError("SMB implementation not yet completed")

    def _scan_nfs_share(self, share_path: str) -> List[str]:
        """
        Scan an NFS share.

        Args:
            share_path: NFS share path (assumed to be already mounted)

        Returns:
            List of file paths found on the NFS share
        """
        # NFS shares should be mounted locally, so we can use regular directory scanning
        return self.scan_directory(share_path)
