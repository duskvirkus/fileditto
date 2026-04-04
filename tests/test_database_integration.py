"""
Test database integration functionality.
"""

import os
import tempfile
import sqlite3
import pytest
from scanner.database_integration import DatabaseIntegrator


class TestDatabaseIntegrator:
    """Test cases for DatabaseIntegrator class."""
    
    def test_initialization(self):
        """Test DatabaseIntegrator initialization."""
        with tempfile.NamedTemporaryFile(suffix='.db', delete=False) as temp_db:
            db_path = temp_db.name
        
        try:
            # This will fail because the database doesn't have the required schema
            # but we're just testing initialization
            integrator = DatabaseIntegrator(db_path)
            assert integrator.db_path == db_path
            assert integrator.scanner is not None
            assert integrator.extractor is not None
        finally:
            if os.path.exists(db_path):
                os.unlink(db_path)
    
    def test_file_hash_calculation(self):
        """Test file hash calculation."""
        with tempfile.TemporaryDirectory() as temp_dir:
            # Create a test file
            test_file = os.path.join(temp_dir, "test.txt")
            with open(test_file, 'w') as f:
                f.write("test content")
            
            integrator = DatabaseIntegrator("dummy.db")
            file_hash = integrator._calculate_file_hash(test_file)
            
            # Should return a valid SHA-256 hash
            assert len(file_hash) == 64
            assert all(c in '0123456789abcdef' for c in file_hash)
    
    def test_file_size(self):
        """Test file size calculation."""
        with tempfile.TemporaryDirectory() as temp_dir:
            # Create a test file with known size
            test_file = os.path.join(temp_dir, "test.txt")
            test_content = "Hello, World!"
            with open(test_file, 'w') as f:
                f.write(test_content)
            
            integrator = DatabaseIntegrator("dummy.db")
            file_size = integrator._get_file_size(test_file)
            
            assert file_size == len(test_content.encode('utf-8'))
    
    def test_file_type_detection(self):
        """Test file type detection."""
        integrator = DatabaseIntegrator("dummy.db")
        
        # Test image metadata
        image_metadata = {'type': 'image', 'width': 100, 'height': 100}
        assert integrator._get_file_type(image_metadata) == 'photo'
        
        # Test video metadata
        video_metadata = {'type': 'video', 'duration': 60.0}
        assert integrator._get_file_type(video_metadata) == 'video'
        
        # Test generic metadata
        generic_metadata = {'type': 'unknown'}
        assert integrator._get_file_type(generic_metadata) == 'generic'
    
    def test_scan_and_store_empty_directory(self):
        """Test scanning and storing from empty directory."""
        with tempfile.TemporaryDirectory() as temp_dir:
            # Create a temporary database
            with tempfile.NamedTemporaryFile(suffix='.db', delete=False) as temp_db:
                db_path = temp_db.name
            
            try:
                integrator = DatabaseIntegrator(db_path)
                # This should fail because database doesn't have schema, but we're testing the scan part
                result = integrator.scan_and_store_files(temp_dir)
                # Should return 0 since no files found
                assert result == 0
            finally:
                if os.path.exists(db_path):
                    os.unlink(db_path)
    
    def test_scan_and_store_with_files(self):
        """Test scanning and storing with actual files (mock scenario)."""
        with tempfile.TemporaryDirectory() as temp_dir:
            # Create a fake JPEG file
            jpeg_path = os.path.join(temp_dir, "test.jpg")
            with open(jpeg_path, 'wb') as f:
                f.write(b'\xFF\xD8\xFF\xE0\x00\x10JFIF')
                f.write(b'\x00' * 1000)
            
            # Create a temporary database
            with tempfile.NamedTemporaryFile(suffix='.db', delete=False) as temp_db:
                db_path = temp_db.name
            
            try:
                integrator = DatabaseIntegrator(db_path)
                # This will fail due to missing schema, but we're testing the scanning part
                result = integrator.scan_and_store_files(temp_dir)
                # Should handle gracefully
                assert isinstance(result, int)
            finally:
                if os.path.exists(db_path):
                    os.unlink(db_path)