"""
Metadata Extractor Module
==========================

This module extracts metadata from photo and video files using industry-standard
libraries and tools.

Key Features:
- Image metadata extraction using Pillow (PIL)
- Video metadata extraction using ffprobe
- EXIF data extraction for photos
- Consistent JSON metadata structure
- Graceful error handling for corrupted files

Supported Formats:
- Images: JPEG, PNG, GIF, BMP, TIFF, WEBP, HEIF/HEIC, AVIF, SVG, ICO, RAW formats
- Videos: MP4, MOV, AVI, MKV, FLV, WMV, WEBM, 3GP, MPEG, OGV, RM, DIVX, H.264/H.265

Usage Example:
    >>> from scanner.metadata_extractor import MetadataExtractor
    >>> extractor = MetadataExtractor()
    >>> # Extract metadata from an image
    >>> metadata = extractor.extract_image_metadata('/path/to/photo.jpg')
    >>> # Extract metadata from a video
    >>> metadata = extractor.extract_video_metadata('/path/to/video.mp4')
    >>> # Auto-detect and extract metadata
    >>> metadata = extractor.extract_metadata('/path/to/file.jpg')

Metadata Structure:
- file_path: Full path to the file
- type: 'image' or 'video'
- For images: width, height, format, mode, exif data
- For videos: duration, width, height, codec information

Note:
- Requires Pillow for image processing
- Requires ffprobe (from FFmpeg) for video processing - install via system package manager
- Returns None for unsupported or corrupted files
"""

import json
import subprocess
from typing import Dict, Optional
from PIL import Image
from PIL.ExifTags import TAGS


class MetadataExtractor:
    """Extracts metadata from photo and video files."""

    def extract_image_metadata(self, file_path: str) -> Dict:
        """
        Extract metadata from an image file.

        Args:
            file_path: Path to the image file

        Returns:
            Dictionary containing extracted metadata

        Raises:
            IOError: If file cannot be opened or is corrupted
        """
        try:
            with Image.open(file_path) as img:
                metadata = {
                    "file_path": file_path,
                    "type": "image",
                    "width": img.width,
                    "height": img.height,
                    "format": img.format,
                    "mode": img.mode,
                    "exif": self._extract_exif_data(img),
                }
                return metadata
        except Exception as e:
            raise IOError(f"Failed to extract metadata from {file_path}: {e}")

    def _extract_exif_data(self, img) -> Dict:
        """
        Extract EXIF data from an image.

        Args:
            img: PIL Image object

        Returns:
            Dictionary containing EXIF data
        """
        exif_data = {}
        try:
            if hasattr(img, "_getexif"):
                exif = img._getexif()
                if exif:
                    for tag_id, value in exif.items():
                        tag_name = TAGS.get(tag_id, tag_id)
                        exif_data[tag_name] = value
        except Exception:
            pass

        return exif_data

    def extract_video_metadata(self, file_path: str) -> Dict:
        """
        Extract metadata from a video file using ffprobe.

        Args:
            file_path: Path to the video file

        Returns:
            Dictionary containing extracted metadata

        Raises:
            IOError: If file cannot be processed or ffprobe fails
        """
        try:
            result = subprocess.run(
                [
                    "ffprobe",
                    "-v",
                    "error",
                    "-show_entries",
                    "format=duration,filename:stream=codec_name,width,height",
                    "-of",
                    "json",
                    file_path,
                ],
                capture_output=True,
                text=True,
                check=True,
            )

            metadata = json.loads(result.stdout)

            # Parse ffprobe output
            video_metadata = {
                "file_path": file_path,
                "type": "video",
                "format": metadata.get("format", {}).get("format_name"),
                "duration": float(metadata.get("format", {}).get("duration", 0)),
            }

            # Extract video stream info
            streams = metadata.get("streams", [])
            for stream in streams:
                if stream.get("codec_type") == "video":
                    video_metadata["codec"] = stream.get("codec_name")
                    video_metadata["width"] = int(stream.get("width", 0))
                    video_metadata["height"] = int(stream.get("height", 0))
                    break

            return video_metadata

        except subprocess.CalledProcessError as e:
            raise IOError(f"ffprobe failed for {file_path}: {e.stderr}")
        except Exception as e:
            raise IOError(f"Failed to extract video metadata from {file_path}: {e}")

    def _get_supported_extensions(self) -> tuple:
        """
        Get all supported file extensions.

        Returns:
            Tuple of all supported file extensions
        """
        # Expanded list of image extensions
        image_extensions = (
            ".jpg",
            ".jpeg",
            ".jpe",
            ".jfif",
            ".pjpeg",
            ".pjpg",
            ".png",
            ".apng",
            ".gif",
            ".bmp",
            ".dib",
            ".tiff",
            ".tif",
            ".webp",
            ".heif",
            ".heic",
            ".avif",
            ".svg",
            ".ico",
            ".raw",
            ".cr2",
            ".nef",
            ".arw",
            ".dng",
            ".orf",
            ".rw2",
            ".raf",
            ".sr2",
        )

        # Expanded list of video extensions
        video_extensions = (
            ".mp4",
            ".m4v",
            ".mov",
            ".qt",
            ".avi",
            ".wmv",
            ".mkv",
            ".mk3d",
            ".webm",
            ".flv",
            ".f4v",
            ".m4p",
            ".m4b",
            ".3gp",
            ".3g2",
            ".asf",
            ".vob",
            ".mpg",
            ".mpeg",
            ".m2v",
            ".mpe",
            ".mpv",
            ".m2ts",
            ".ts",
            ".ogv",
            ".ogg",
            ".rm",
            ".rmvb",
            ".divx",
            ".div",
            ".xvid",
            ".h264",
            ".h265",
            ".av1",
        )

        return image_extensions + video_extensions

    def extract_metadata(self, file_path: str) -> Optional[Dict]:
        """
        Extract metadata from any file (image or video).

        Args:
            file_path: Path to the file

        Returns:
            Dictionary containing extracted metadata, or None if extraction fails
        """
        try:
            # Get supported extensions
            image_extensions = (
                ".jpg",
                ".jpeg",
                ".jpe",
                ".jfif",
                ".pjpeg",
                ".pjpg",
                ".png",
                ".apng",
                ".gif",
                ".bmp",
                ".dib",
                ".tiff",
                ".tif",
                ".webp",
                ".heif",
                ".heic",
                ".avif",
                ".svg",
                ".ico",
                ".raw",
                ".cr2",
                ".nef",
                ".arw",
                ".dng",
                ".orf",
                ".rw2",
                ".raf",
                ".sr2",
            )

            # Expanded list of video extensions
            video_extensions = (
                ".mp4",
                ".m4v",
                ".mov",
                ".qt",
                ".avi",
                ".wmv",
                ".mkv",
                ".mk3d",
                ".webm",
                ".flv",
                ".f4v",
                ".m4p",
                ".m4b",
                ".3gp",
                ".3g2",
                ".asf",
                ".vob",
                ".mpg",
                ".mpeg",
                ".m2v",
                ".mpe",
                ".mpv",
                ".m2ts",
                ".ts",
                ".ogv",
                ".ogg",
                ".rm",
                ".rmvb",
                ".divx",
                ".div",
                ".xvid",
                ".h264",
                ".h265",
                ".av1",
            )

            if file_path.lower().endswith(image_extensions):
                return self.extract_image_metadata(file_path)
            elif file_path.lower().endswith(video_extensions):
                return self.extract_video_metadata(file_path)
            else:
                return None
        except Exception:
            return None
