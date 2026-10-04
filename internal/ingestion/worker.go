// Package ingestion processes queue entries: hashing files, extracting
// metadata, and writing records to the database.
package ingestion

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/duskvirkus/fileditto/internal/db"
	"github.com/gabriel-vasile/mimetype"
	"github.com/rwcarlsen/goexif/exif"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// ProcessNext dequeues and processes one pending entry.
// Returns (true, nil) if an entry was processed, (false, nil) if the queue is empty.
func ProcessNext(d db.DB, mediaID string) (bool, error) {
	entry, err := d.Queue().DequeueNext()
	if err != nil {
		return false, err
	}
	if entry == nil {
		return false, nil
	}

	if err := processEntry(d, entry, mediaID); err != nil {
		errStr := err.Error()
		if statusErr := d.Queue().SetStatus(entry.ID, db.QueueStatusFailed, &errStr); statusErr != nil {
			return true, fmt.Errorf("process failed (%w); also failed to record failure: %v", err, statusErr)
		}
		return true, fmt.Errorf("process %s: %w", entry.FilePath, err)
	}
	return true, nil
}

func processEntry(d db.DB, entry *db.QueueEntry, mediaID string) error {
	mime, err := mimetype.DetectFile(entry.FilePath)
	if err != nil {
		return fmt.Errorf("detect mime: %w", err)
	}

	fileType := classifyMIME(mime.String(), entry.FilePath)
	if fileType == "unsupported" {
		return d.Queue().SetStatus(entry.ID, db.QueueStatusUnsupported, nil)
	}

	sha, size, err := hashFile(entry.FilePath)
	if err != nil {
		return err
	}

	fileUUID, err := db.FileUUID(sha)
	if err != nil {
		return fmt.Errorf("derive uuid: %w", err)
	}

	meta, err := extractMetadata(entry.FilePath, fileType)
	if err != nil {
		return fmt.Errorf("extract metadata: %w", err)
	}

	absPath, err := filepath.Abs(entry.FilePath)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	f := &db.File{
		ID:        fileUUID.String(),
		SHA256:    sha,
		SizeBytes: size,
		FileType:  fileType,
	}

	metaJSON, _ := json.Marshal(meta)

	var photo *db.Photo
	var video *db.Video
	switch fileType {
	case "photo":
		width, _ := meta["width"].(int)
		height, _ := meta["height"].(int)
		photo = &db.Photo{
			WidthPx:      width,
			HeightPx:     height,
			MetadataJSON: metaJSON,
		}
	case "video":
		width, _ := meta["width"].(int)
		height, _ := meta["height"].(int)
		codec, _ := meta["codec"].(string)
		durSec, _ := meta["duration_seconds"].(float64)
		video = &db.Video{
			DurationSeconds: durSec,
			WidthPx:         width,
			HeightPx:        height,
			Codec:           codec,
			MetadataJSON:    metaJSON,
		}
	}

	if _, _, err := d.Files().WriteFile(f, photo, video, mediaID, absPath); err != nil {
		return err
	}

	return d.Queue().SetStatus(entry.ID, db.QueueStatusDone, nil)
}

var rawExtensions = map[string]bool{
	".cr2": true, ".cr3": true,
	".nef": true,
	".arw": true,
	".raf": true,
	".orf": true,
	".rw2": true,
	".dng": true,
	".pef": true,
	".srw": true,
	".x3f": true,
}

func classifyMIME(mime, path string) string {
	switch {
	case strings.HasPrefix(mime, "image/"):
		return "photo"
	case strings.HasPrefix(mime, "video/"):
		return "video"
	default:
		ext := strings.ToLower(filepath.Ext(path))
		if rawExtensions[ext] {
			return "photo"
		}
		return "unsupported"
	}
}

func hashFile(path string) (sha256hex string, size int64, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, fmt.Errorf("open for hash: %w", err)
	}
	defer f.Close()
	h := sha256.New()
	size, err = io.Copy(h, f)
	if err != nil {
		return "", 0, fmt.Errorf("hash: %w", err)
	}
	return fmt.Sprintf("%x", h.Sum(nil)), size, nil
}

func extractMetadata(path, fileType string) (map[string]interface{}, error) {
	switch fileType {
	case "photo":
		return extractImageMetadata(path)
	case "video":
		return extractVideoMetadata(path)
	default:
		return map[string]interface{}{}, nil
	}
}

func extractImageMetadata(path string) (map[string]interface{}, error) {
	meta := map[string]interface{}{}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cfg, _, decodeErr := image.DecodeConfig(f)
	if decodeErr == nil {
		meta["width"] = cfg.Width
		meta["height"] = cfg.Height
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek for exif: %w", err)
	}

	x, err := exif.Decode(f)
	if err == nil {
		if decodeErr != nil {
			if tag, err := x.Get(exif.PixelXDimension); err == nil {
				if v, err := tag.Int(0); err == nil {
					meta["width"] = v
				}
			}
			if tag, err := x.Get(exif.PixelYDimension); err == nil {
				if v, err := tag.Int(0); err == nil {
					meta["height"] = v
				}
			}
		}
		if lat, long, err := x.LatLong(); err == nil {
			meta["latitude"] = lat
			meta["longitude"] = long
		}
		if tm, err := x.DateTime(); err == nil {
			meta["taken_at"] = tm.UTC().Format(time.RFC3339)
		}
	}

	if _, ok := meta["width"]; !ok {
		return nil, fmt.Errorf("could not determine image dimensions: %w", decodeErr)
	}
	return meta, nil
}

type ffprobeOut struct {
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
	Streams []struct {
		CodecType string `json:"codec_type"`
		CodecName string `json:"codec_name"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
	} `json:"streams"`
}

func extractVideoMetadata(path string) (map[string]interface{}, error) {
	out, err := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration:stream=codec_name,width,height,codec_type",
		"-of", "json",
		path,
	).Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe: %w", err)
	}
	var fp ffprobeOut
	if err := json.Unmarshal(out, &fp); err != nil {
		return nil, err
	}
	meta := map[string]interface{}{}
	if d, err := strconv.ParseFloat(fp.Format.Duration, 64); err == nil {
		meta["duration_seconds"] = d
	}
	for _, s := range fp.Streams {
		if s.CodecType == "video" {
			meta["codec"] = s.CodecName
			meta["width"] = s.Width
			meta["height"] = s.Height
			break
		}
	}
	return meta, nil
}
