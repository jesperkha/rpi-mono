package health

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// Uptime returns the system uptime as a time.Duration.
func Uptime() (time.Duration, error) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, fmt.Errorf("reading /proc/uptime: %w", err)
	}

	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return 0, fmt.Errorf("unexpected /proc/uptime format")
	}

	seconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, fmt.Errorf("parsing uptime seconds: %w", err)
	}

	return time.Duration(seconds * float64(time.Second)), nil
}

// CPULoad represents the average CPU load per core.
type CPULoad struct {
	Load1  float64 // 1-minute load average per core
	Load5  float64 // 5-minute load average per core
	Load15 float64 // 15-minute load average per core
	Cores  int     // number of logical CPU cores
}

// AverageCPULoad returns the system load averages divided by the number of
// logical CPU cores.
func AverageCPULoad() (CPULoad, error) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return CPULoad{}, fmt.Errorf("reading /proc/loadavg: %w", err)
	}

	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return CPULoad{}, fmt.Errorf("unexpected /proc/loadavg format")
	}

	load1, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return CPULoad{}, fmt.Errorf("parsing load1: %w", err)
	}

	load5, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return CPULoad{}, fmt.Errorf("parsing load5: %w", err)
	}

	load15, err := strconv.ParseFloat(fields[2], 64)
	if err != nil {
		return CPULoad{}, fmt.Errorf("parsing load15: %w", err)
	}

	cores := runtime.NumCPU()

	return CPULoad{
		Load1:  load1 / float64(cores),
		Load5:  load5 / float64(cores),
		Load15: load15 / float64(cores),
		Cores:  cores,
	}, nil
}

// DockerContainer holds the name and status of a running Docker container.
type DockerContainer struct {
	ID     string
	Name   string
	Image  string
	Status string
	State  string
}

// DockerContainers returns a list of all running Docker containers with their
// current status.
func DockerContainers(ctx context.Context) ([]DockerContainer, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("creating docker client: %w", err)
	}
	defer cli.Close()

	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("listing docker containers: %w", err)
	}

	result := make([]DockerContainer, 0, len(containers))
	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}
		result = append(result, DockerContainer{
			ID:     c.ID[:12],
			Name:   name,
			Image:  c.Image,
			Status: c.Status,
			State:  c.State,
		})
	}

	return result, nil
}

// DiskUsage holds disk usage statistics in bytes for a given path.
type DiskUsage struct {
	TotalBytes     uint64
	UsedBytes      uint64
	AvailableBytes uint64
	Path           string
}

// String returns a human-readable representation like "120.3 GB / 500.0 GB".
func (d DiskUsage) String() string {
	return fmt.Sprintf("%s / %s", formatBytes(d.UsedBytes), formatBytes(d.TotalBytes))
}

// Disk returns the disk usage for the given filesystem path (e.g. "/").
func Disk(path string) (DiskUsage, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return DiskUsage{}, fmt.Errorf("statfs %s: %w", path, err)
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bfree * uint64(stat.Bsize)
	available := stat.Bavail * uint64(stat.Bsize)
	used := total - free

	return DiskUsage{
		TotalBytes:     total,
		UsedBytes:      used,
		AvailableBytes: available,
		Path:           path,
	}, nil
}

// RAMUsage holds RAM usage statistics in bytes (same physical memory, but
// broken down differently with buffer/cache detail).
type RAMUsage struct {
	TotalBytes   uint64
	UsedBytes    uint64
	FreeBytes    uint64
	BuffersBytes uint64
	CachedBytes  uint64
}

// String returns a human-readable representation like "3.2 GB / 16.0 GB (20.0%)".
func (r RAMUsage) String() string {
	pct := 0.0
	if r.TotalBytes > 0 {
		pct = float64(r.UsedBytes) / float64(r.TotalBytes) * 100
	}
	return fmt.Sprintf("%s / %s (%.1f%%)", formatBytes(r.UsedBytes), formatBytes(r.TotalBytes), pct)
}

// RAM returns the current RAM usage including buffer/cache breakdown.
func RAM() (RAMUsage, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return RAMUsage{}, fmt.Errorf("opening /proc/meminfo: %w", err)
	}
	defer file.Close()

	info := make(map[string]uint64)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		valStr := strings.TrimSpace(parts[1])
		valStr = strings.TrimSuffix(valStr, " kB")

		val, err := strconv.ParseUint(strings.TrimSpace(valStr), 10, 64)
		if err != nil {
			continue
		}

		info[key] = val * 1024 // convert kB to bytes
	}

	if err := scanner.Err(); err != nil {
		return RAMUsage{}, fmt.Errorf("scanning /proc/meminfo: %w", err)
	}

	total := info["MemTotal"]
	free := info["MemFree"]
	buffers := info["Buffers"]
	cached := info["Cached"]
	used := total - free - buffers - cached

	return RAMUsage{
		TotalBytes:   total,
		UsedBytes:    used,
		FreeBytes:    free,
		BuffersBytes: buffers,
		CachedBytes:  cached,
	}, nil
}

// formatBytes converts bytes to a human-readable string (e.g. "3.2 GB").
func formatBytes(b uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case b >= TB:
		return fmt.Sprintf("%.1f TB", float64(b)/float64(TB))
	case b >= GB:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
