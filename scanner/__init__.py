"""
File Scanner and Ingestion System (P2)
=======================================

A comprehensive file scanning and metadata extraction system for photos and videos.

This module provides functionality for:
- Discovering media files in directories and network shares
- Extracting metadata from images and videos
- Managing ingestion queues for processing
- Integrating with the P1 database schema

Main Components:
- FileScanner: Discovers files in directories and network shares
- MetadataExtractor: Extracts metadata from image and video files
- IngestionQueue: Manages queues of files for processing
- DatabaseIntegrator: Integrates scanner results with database

Usage:
    from scanner import FileScanner, MetadataExtractor
    scanner = FileScanner()
    files = scanner.scan_directory('/path/to/photos')
    extractor = MetadataExtractor()
    metadata = extractor.extract_metadata(files[0])
"""

from .file_scanner import FileScanner
from .metadata_extractor import MetadataExtractor
from .ingestion_queue import IngestionQueue
from .database_integration import DatabaseIntegrator

__all__ = ['FileScanner', 'MetadataExtractor', 'IngestionQueue', 'DatabaseIntegrator']