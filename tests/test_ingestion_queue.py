"""
Test ingestion queue functionality.
"""

import os
import tempfile
from scanner.ingestion_queue import IngestionQueue


class TestIngestionQueue:
    """Test cases for IngestionQueue class."""
    
    def test_in_memory_queue(self):
        """Test in-memory queue operations."""
        queue = IngestionQueue(persistent=False)
        
        # Test initial state
        assert queue.get_queue_size() == 0
        assert queue.get_next_file() is None
        
        # Test adding files
        test_file = {'file_path': '/test/photo.jpg', 'type': 'image'}
        queue.add_file(test_file)
        assert queue.get_queue_size() == 1
        
        # Test getting next file
        next_file = queue.get_next_file()
        assert next_file == test_file
        assert queue.get_queue_size() == 0
    
    def test_persistent_queue(self):
        """Test persistent queue operations."""
        with tempfile.TemporaryDirectory() as temp_dir:
            storage_path = os.path.join(temp_dir, 'test_queue.json')
            
            # Create queue and add files
            queue1 = IngestionQueue(persistent=True, storage_path=storage_path)
            file1 = {'file_path': '/test/photo1.jpg', 'type': 'image'}
            file2 = {'file_path': '/test/photo2.jpg', 'type': 'image'}
            queue1.add_files([file1, file2])
            
            # Create new queue instance with same storage
            queue2 = IngestionQueue(persistent=True, storage_path=storage_path)
            assert queue2.get_queue_size() == 2
            
            # Test getting files
            retrieved_file1 = queue2.get_next_file()
            retrieved_file2 = queue2.get_next_file()
            assert retrieved_file1 == file1
            assert retrieved_file2 == file2
            assert queue2.get_queue_size() == 0
    
    def test_add_multiple_files(self):
        """Test adding multiple files at once."""
        queue = IngestionQueue()
        
        files = [
            {'file_path': '/test/photo1.jpg', 'type': 'image'},
            {'file_path': '/test/photo2.jpg', 'type': 'image'},
            {'file_path': '/test/video1.mp4', 'type': 'video'}
        ]
        
        queue.add_files(files)
        assert queue.get_queue_size() == 3
    
    def test_clear_queue(self):
        """Test clearing the queue."""
        with tempfile.TemporaryDirectory() as temp_dir:
            storage_path = os.path.join(temp_dir, 'test_queue.json')
            
            queue = IngestionQueue(persistent=True, storage_path=storage_path)
            queue.add_file({'file_path': '/test/photo.jpg', 'type': 'image'})
            assert queue.get_queue_size() == 1
            
            queue.clear_queue()
            assert queue.get_queue_size() == 0
            assert not os.path.exists(storage_path)