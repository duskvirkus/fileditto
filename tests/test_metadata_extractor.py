"""
Test metadata extractor functionality.
"""

import os
import tempfile
import pytest
from scanner.metadata_extractor import MetadataExtractor


class TestMetadataExtractor:
    """Test cases for MetadataExtractor class."""
    
    def test_extract_metadata_unsupported_file(self):
        """Test metadata extraction from unsupported file type."""
        with tempfile.TemporaryDirectory() as temp_dir:
            # Create a text file
            txt_path = os.path.join(temp_dir, "test.txt")
            with open(txt_path, 'w') as f:
                f.write("test content")
            
            extractor = MetadataExtractor()
            metadata = extractor.extract_metadata(txt_path)
            assert metadata is None
    
    def test_extract_image_metadata_invalid_file(self):
        """Test image metadata extraction from invalid image file."""
        with tempfile.TemporaryDirectory() as temp_dir:
            # Create an invalid image file
            invalid_jpeg = os.path.join(temp_dir, "invalid.jpg")
            with open(invalid_jpeg, 'wb') as f:
                f.write(b"not a real image")
            
            extractor = MetadataExtractor()
            with pytest.raises(IOError):
                extractor.extract_image_metadata(invalid_jpeg)
    
    def test_extract_video_metadata_invalid_file(self):
        """Test video metadata extraction from invalid video file."""
        with tempfile.TemporaryDirectory() as temp_dir:
            # Create an invalid video file
            invalid_mp4 = os.path.join(temp_dir, "invalid.mp4")
            with open(invalid_mp4, 'wb') as f:
                f.write(b"not a real video")
            
            extractor = MetadataExtractor()
            with pytest.raises(IOError):
                extractor.extract_video_metadata(invalid_mp4)
    
    def test_extract_metadata_fake_jpeg(self):
        """Test metadata extraction from fake JPEG file."""
        with tempfile.TemporaryDirectory() as temp_dir:
            # Create a fake JPEG file
            jpeg_path = os.path.join(temp_dir, "fake.jpg")
            with open(jpeg_path, 'wb') as f:
                f.write(b'\xFF\xD8\xFF\xE0\x00\x10JFIF')
                f.write(b'\x00' * 1000)
            
            extractor = MetadataExtractor()
            # This should handle the fake file gracefully
            metadata = extractor.extract_metadata(jpeg_path)
            # Should return None for invalid image
            assert metadata is None