## Project concept

This is the photo, video, and artwork backup system I always wanted to make but never had the time to do so. The high level overview is as a 3-2-1 backup system that categorizes everything backed-up in an sql database to track where it exists and metadata, this is also backed up.

The place this is different from many other 3-2-1 backup software I've seen is that the 3-2-1 can be different for each file. Additionally we use a combination of timestamp, image/video hash, and metadata to identify duplicated files and they get marked in the database as having an additional location but don't get saved as a new entry in the database.

The concept is there's a central program that includes a ui, database, and scheduled backup. This software would run on a Network Attached Storage (NAS) or a person's personal computer but if permissions were setup correctly it can categorize and save all photos and videos found on the local network. In doing the backup on any media the sql database should also be backed up in it's current state but in a way where it's timestamped. This means we need ability to use the database at any level of migration in the event we need to rebuild the primary host database but only have one very old copy stored on something like an immutable disc.

Conceptually it would be nice to support other types of files like documents, cad files, etc. However, I really think the more complicated a backup system is the more room there is for failure so it makes sense to try to design the system in a way that can be reused in backup projects or in a way where plugins can be added. I'd like this project to be useable by others but wherever possible I want to narrow the scope of what it's trying to do. It's better to have a project that does one thing really well then to have a project that tries to do a bunch of things but doesn't do them very well.

A note on the failure of 3-2-1 method. The 3 backups, 2 types of media, 1 offsite backup method may have some limitations. The primary one I'm concerned about is the ransomware vector. To properly protect from ransomware one of your sources needs to be immutable which means it can't be changed by any software once it's written. Also a note on cloud storage here; the cloud is just someone else's computer and you don't know how they are backing up their data so it doesn't count towards 2 types of media. Some people may differ from me on this point and count cloud as a separate type so maybe this should be a configurable variable but by default cloud is not another type however it might be one of the best methods for offsite. Also a specific to me thought is I don't count the original file as a backup. The reason for this I don't treat my files with a ton of care I will backup my machine when doing an os reinstall but they are often disorganized and not well maintained so I don't count this as a good backup. This means it's worth including a flag in locations that allows skipping the particular location when counting towards 3-2-1.

The beginning of this project should support data blurays and external drives (both ssd and hard disk). However it's worth considering expanding to other media in the future. Namely magnetic tape, other optical disks like dvds, cloud storage, and anything else that might be reasonable.

### System overview

```
  Sources                    Core                    Destinations
  ────────               ──────────────              ────────────
  ┌────────┐            ┌──────────────┐             ┌────────────┐
  │ Local  │──scan──▶   │  Ingestion   │──copy──▶    │  Ext Drive │
  │ files  │            │  Engine      │             │  (SSD/HDD) │
  └────────┘            └──────┬───────┘             └────────────┘
                               │                     ┌────────────┐
  ┌────────┐                   ▼                     │  Blu-ray   │
  │Network │──discover──▶ ┌──────────┐   ──burn──▶   │  Disc      │
  │ shares │              │ Hash &   │               └────────────┘
  └────────┘              │ Dedup    │
                          └──────┬───┘      Database (SQLite)
                                 │          ┌────────────────┐
                                 ▼          │ Files          │
                          ┌──────────┐      │ Locations      │
                          │  3-2-1   │─────▶│ Media          │
                          │ Checker  │      │ Photo / Video  │
                          └──────┬───┘      └───────┬────────┘
                                 │                  │
                                 ▼                  ▼
                          ┌──────────┐      ┌────────────────┐
                          │  TUI /   │      │   DB Backup    │
                          │  CLI     │      │  (versioned,   │
                          └──────────┘      │  timestamped)  │
                                            └────────────────┘
```

### Technology decisions

**Languages: Python + Go**
Python handles file scanning, media hashing, and metadata extraction where the library ecosystem (Pillow, ffprobe bindings, etc.) is strongest. Go handles the core backup engine, CLI, TUI, and any performance-sensitive coordination work. The two communicate over a defined internal interface (likely stdin/stdout or a local socket).

**UI: TUI (terminal user interface)**
The UI is a terminal user interface rather than a desktop GUI or web app. This keeps the application cross-platform without a browser dependency or Electron bloat, works naturally in headless/NAS environments, and stays close to the system. The TUI runs on Linux, macOS, and Windows. A fully headless/scripting mode (no TUI) is also supported.

**Deployment: Docker**
Docker is the primary packaging target, enabling the software to run on NAS devices, servers, and personal computers without environment setup. Native binaries are a secondary target for those who prefer them.

**Database: SQLite**
SQLite is used for all persistent state. The rationale: it produces a single compact file with no server process, has excellent support in both Python and Go, and the file format has been stable and backward-compatible since 2004. Since the database is backed up many times across many types of media, minimizing file size and maximizing longevity are priorities. SQLite satisfies both.

The database must be self-describing: it stores its own schema version and a full migration history. This allows a database snapshot recovered from old media to be identified and upgraded to the current schema without external tooling. Migrations are forward-only and bundled into the application binary.

**File UUIDs: deterministic from content**
The UUID for each entry in the `Files` table is derived deterministically from the file's SHA-256 content hash using UUID v5 (RFC 4122) with a project-specific namespace. This means the same file discovered independently on two different machines — or re-ingested after a database rebuild — produces the same UUID without coordination. The UUID is stable for the lifetime of the file content. If the hashing algorithm ever needs to change, the content hash column is updated but the UUID (being already assigned) remains unchanged, preserving all foreign key relationships.

### Initial SQL structure

Files
uuid (UUID v5, deterministic from SHA-256 content hash), file_created: timestamp, metadata: json, labels: json, file_type

files-locations junction table

Locations
media foreign key to metadata on media, path, metadata

Media
Follows inheritance type structure where there's a centralized Media table then a reference to a foreign key with any data specific to the type of media. For example optical discs would have a separate table for things like burn date.
Media name, media type, date created, date updated, metadata: json, child: uuid to tables like BlurayMedia or DriveMedia

BlurayMedia, DriveMedia

Photo, Video tables
uuid: same as the uuid in Files, hash information, other file type specific metadata like resolution or in the case of video playtime

Additional notes:
we need a way to flag media as inactive. Also we need a way to archive or mark whole media as lost. In addition we need an archive / lost flag for specific files and locations if files become unreadable on a particular media or if we lose enough copies to lose the whole file.

An additional note. I don't want to use llms as part of the function of this codebase just as a tool for development. And specific machine learning systems are permitted to be used for metadata creation such as hashing, image similarity, or image annotation. However any output that could be incorrect must be reviewed by a person for example generated code, duplicate images based on similarity score in a image embedding. Please include this in any contributor docs.

### MVP note on media types

HDD and SSD are both considered drives — they are not counted as two different media types for 3-2-1 compliance purposes. Blu-ray optical discs are the second required media type for MVP. This means Blu-ray burn support is required for a complete MVP, though it is the last feature implemented. The MVP is not complete without at least one working optical disc backup path.

---

## MVP roadmap

The following features are scoped for MVP, listed in implementation order. Each builds on the previous.

```
  P1 (DB Schema)
     │
     ├──▶ P2 (Scanner) ──▶ P3 (Hashing) ──▶ P4 (Drive Backup)
     │                                              │
     ├──▶ P6 (Lifecycle)                            ▼
     │         │                          P5 (3-2-1 Checker)
     │         └─────────────────────────────────▶ │
     │                                             │
     └──▶ P7 (DB Backup) ◀────────────────────────┘
                │
                ▼
           P8 (CLI/TUI) ◀──── wraps everything
                │
                ▼
           P9 (Blu-ray) ◀──── final MVP gate
```

### P1 — Database schema & core models
Define and implement the SQLite schema: `Files`, `Locations`, `Media` (with `BlurayMedia` and `DriveMedia` subtypes), `Photo`, `Video`. Implement the self-describing version table and forward-only migration runner. This is the foundation — get the data model right before anything else.

### P2 — File scanner & ingestion
Walk a directory tree, identify photos and videos by MIME type and extension, extract metadata (EXIF, creation date, resolution, video duration). Python handles this using available media libraries. Output is a queue of candidate files with extracted metadata, ready for deduplication. No files are written to backup destinations yet.

### P3 — Hash & deduplication engine
Compute content hash (xxHash for speed, SHA-256 for verification) plus timestamp and metadata fingerprint per file. Determine whether an incoming file is new, a known file appearing at a new source location, or a true duplicate. Duplicates get a new location record without a new file entry. Python handles hash computation; Go coordinates the dedup logic against the database.

### P4 — External drive backup
Register external drives by serial number and label. Copy files from the ingestion queue to an attached drive, write location records to the database, and verify copies by re-hashing after transfer. This is the point where the system becomes a working backup tool.

### P5 — 3-2-1 compliance tracker
Query the database to calculate per-file backup health: number of copies, number of distinct media types, number of offsite locations. Configurable globally and per-file. Respects the `skip_for_counting` flag on locations. Produces a compliance report showing which files are under-protected and why.

### P6 — Media lifecycle management
Commands and database flags to mark media as active, inactive, or lost. Mark individual file locations as unreadable or lost. These flags feed into the compliance tracker so degraded backups are surfaced automatically. Includes the ability to mark a file as fully lost if all known locations are gone.

### P7 — Database backup with versioning
At the end of each backup run, snapshot the current SQLite database to the destination media with a UTC timestamp in the filename. The snapshot includes the schema version so it is self-identifying. The application can open any historical snapshot and upgrade it to the current schema — this is the disaster recovery path when the primary host is lost and only an old snapshot survives.

### P8 — CLI & TUI interface
The Go-based interface that ties everything together. CLI mode supports scripting and headless operation (for NAS/cron use). TUI mode provides an interactive terminal interface for humans. Commands cover the full workflow: `scan`, `backup`, `status`, `report`, `media add/remove/mark`, `db snapshot`, `db migrate`. This is the last cross-cutting feature that makes the system usable end-to-end.

### P9 — Blu-ray burn support
Write backup sets to data Blu-ray discs (BD-R). Handles UDF filesystem creation, multi-session disc management, and post-burn verification. Registers burned discs in the database as `BlurayMedia` with burn date and disc identifier. This completes the two-media-type requirement for 3-2-1 compliance and is the final MVP feature.

---

## Post-MVP candidates

- Network share scanning (discover files on local network)
- Cloud storage as an offsite destination
- DVD and other optical formats
- Magnetic tape support
- Perceptual/similarity-based duplicate detection (requires human review of results)
- Plugin system for additional file types or storage backends
