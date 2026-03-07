## Project concept

This is the photo, video, and artwork backup system I always wanted to make but never had the time to do so. The high level overview is as a 3-2-1 backup system that categorizes everything backed-up in an sql database to track where it exists and metadata, this is also backed up.

The place this is different from many other 3-2-1 backup software I've seen is that the 3-2-1 can be different for each file. Additionally we use a combination of timestamp, image/video hash, and metadata to identify duplicated files and they get marked in the database as having an additional location but don't get saved as a new entry in the database.

The concept is there's a central program that includes a ui, database, and scheduled backup. This software would run on a Network Attached Storage (NAS) or a person's personal computer but if permissions were setup correctly it can categorize and save all photos and videos found on the local network. In doing the backup on any media the sql database should also be backed up in it's current state but in a way where it's timestamped. This means we need ability to use the database at any level of migration in the event we need to rebuild the primary host database but only have one very old copy stored on something like an immutable disc.

Conceptually it would be nice to support other types of files like documents, cad files, etc. However, I really think the more complicated a backup system is the more room there is for failure so it makes sense to try to design the system in a way that can be reused in backup projects or in a way where plugins can be added. I'd like this project to be useable by others but wherever possible I want to narrow the scope of what it's trying to do. It's better to have a project that does one thing really well then to have a project that tries to do a bunch of things but doesn't do them very well.

A note on the failure of 3-2-1 method. The 3 backups, 2 types of media, 1 offsite backup method may have some limitations. The primary one I'm concerned about is the ransomware vector. To properly protect from ransomware one of your sources needs to be immutable which means it can't be changed by any software once it's written. Also a note on cloud storage here; the cloud is just someone else's computer and you don't know how they are backing up their data so it doesn't count towards 2 types of media. Some people may differ from me on this point and count cloud as a separate type so maybe this should be a configurable variable but by default cloud is not another type however it might be one of the best methods for offsite. Also a specific to me thought is I don't count the original file as a backup. The reason for this I don't treat my files with a ton of care I will backup my machine when doing an os reinstall but they are often disorganized and not well maintained so I don't count this as a good backup. This means it's worth including a flag in locations that allows skipping the particular location when counting towards 3-2-1.

Ideally this software should be cross platform (linux, window, and mac) docker is likely a good bed because it also can be run in on many NAS machines. The software is able to run both in headless and a ui mode. The ui constrains us somewhat because it needs to be cross platform and preferably not need additional dependencies. I'd like to use something close to the system to avoid additional bloat. For example C++ is preferable to Javascript. I think ideally this project would be a mix of a high level language, likely python to be able to easily run image and video hashing libraries, and a lower level language where it would yield performance benefits maybe go, rust, or modern c++.

The beginning of this project should support data blurays and external drives (both ssd and hard disk). However it's worth considering expanding to other media in the future. Namely magnetic tape, other optical disks like dvds, cloud storage, and anything else that might be reasonable.

Initial SQL structure

Files
uuid, file_created: timestamp, metadata: json, labels: json, file_type

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
