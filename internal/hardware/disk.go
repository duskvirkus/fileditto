package hardware

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jaypipes/ghw"
	"github.com/jaypipes/ghw/pkg/block"
)

// DiskInfo holds detected attributes for a physical block device.
type DiskInfo struct {
	Serial        string
	Model         string
	Vendor        string
	CapacityGB    float64
	InterfaceType string
}

// DetectDiskForPath finds the physical disk backing the given path by
// matching mount points from ghw against the absolute path.
func DetectDiskForPath(path string) (*DiskInfo, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("abs path: %w", err)
	}

	b, err := ghw.Block()
	if err != nil {
		return nil, fmt.Errorf("ghw block: %w", err)
	}

	disk, err := findDiskForPath(b.Disks, abs)
	if err != nil {
		return nil, err
	}

	vendor := disk.Vendor
	if vendor == "unknown" {
		vendor = ""
	}

	iface := disk.StorageController.String()
	if iface == "unknown" {
		iface = ""
	}

	return &DiskInfo{
		Serial:        strings.TrimSpace(disk.SerialNumber),
		Model:         strings.TrimSpace(disk.Model),
		Vendor:        vendor,
		CapacityGB:    float64(disk.SizeBytes) / 1e9,
		InterfaceType: iface,
	}, nil
}

func findDiskForPath(disks []*block.Disk, absPath string) (*block.Disk, error) {
	var bestDisk *block.Disk
	bestLen := -1

	for _, disk := range disks {
		for _, part := range disk.Partitions {
			mp := part.MountPoint
			if mp == "" {
				continue
			}
			if mp != "/" && !strings.HasSuffix(mp, "/") {
				mp += "/"
			}
			candidate := absPath
			if !strings.HasSuffix(candidate, "/") {
				candidate += "/"
			}
			if strings.HasPrefix(candidate, mp) && len(mp) > bestLen {
				bestLen = len(mp)
				bestDisk = disk
			}
		}
	}

	if bestDisk == nil {
		return nil, fmt.Errorf("no disk found for path %s", absPath)
	}
	return bestDisk, nil
}
