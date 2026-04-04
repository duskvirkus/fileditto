"""
Ingestion Queue Module
======================

This module manages the queue of candidate files with extracted metadata for
further processing in the ingestion pipeline.

Key Features:
- In-memory queue for fast processing
- Persistent storage for large collections
- Batch operations for efficiency
- Queue management (add, get, clear)

Usage Example:
    >>> from scanner.ingestion_queue import IngestionQueue
    >>> # In-memory queue
    >>> queue = IngestionQueue()
    >>> queue.add_file({'file_path': '/path/to/photo.jpg', 'type': 'image'})
    >>> next_file = queue.get_next_file()
    >>> # Persistent queue
    >>> persistent_queue = IngestionQueue(persistent=True, storage_path='queue.json')
    >>> persistent_queue.add_files([file1, file2, file3])
    >>> # Queue size
    >>> size = queue.get_queue_size()

Queue Persistence:
- Persistent queues are stored as JSON files
- Automatically loaded on initialization
- Updated after each modification
- Can be cleared completely

Note:
- Persistent queues are useful for large collections that can't fit in memory
- Queue files can be used to resume interrupted processing
- Each file in the queue should have consistent metadata structure
"""

import json
import os
from collections import deque
from typing import List, Dict, Optional


class IngestionQueue:
    """Manages a queue of candidate files with extracted metadata."""

    def __init__(self, persistent: bool = False, storage_path: Optional[str] = None):
        """
        Initialize the ingestion queue.

        Args:
            persistent: Whether to use persistent storage
            storage_path: Path for persistent storage file
        """
        self.persistent = persistent
        self.storage_path = storage_path or 'ingestion_queue.json'
        self.queue = deque()

        if self.persistent:
            self._load_queue()
            
    def add_file(self, file_metadata: Dict) -> None:
        """
        Add a file with its metadata to the queue.
        
        Args:
            file_metadata: Dictionary containing file metadata
        """
        self.queue.append(file_metadata)
        if self.persistent:
            self._save_queue()
            
    def add_files(self, files_metadata: List[Dict]) -> None:
        """
        Add multiple files with their metadata to the queue.
        
        Args:
            files_metadata: List of dictionaries containing file metadata
        """
        self.queue.extend(files_metadata)
        if self.persistent:
            self._save_queue()
            
    def get_next_file(self) -> Optional[Dict]:
        """
        Get the next file from the queue.
        
        Returns:
            Dictionary containing file metadata, or None if queue is empty
        """
        if not self.queue:
            return None
            
        file_metadata = self.queue.popleft()
        if self.persistent:
            self._save_queue()
            
        return file_metadata
        
    def get_queue_size(self) -> int:
        """
        Get the current size of the queue.
        
        Returns:
            Number of files in the queue
        """
        return len(self.queue)
        
    def clear_queue(self) -> None:
        """Clear all files from the queue."""
        self.queue.clear()
        if self.persistent and os.path.exists(self.storage_path):
            os.remove(self.storage_path)
            
    def _save_queue(self) -> None:
        """Save queue to persistent storage."""
        if self.persistent:
            with open(self.storage_path, 'w') as f:
                json.dump(list(self.queue), f, indent=2)

    def _load_queue(self) -> None:
        """Load queue from persistent storage."""
        if self.persistent and os.path.exists(self.storage_path):
            with open(self.storage_path, 'r') as f:
                self.queue = deque(json.load(f))