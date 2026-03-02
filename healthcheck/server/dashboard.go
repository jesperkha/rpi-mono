package server

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/jesperkha/healthcheck/health"
)

type dashboardData struct {
	Uptime     string
	CPU        cpuData
	Disk       diskData
	RAM        ramData
	Containers []health.DockerContainer
	UpdatedAt  string
}

type cpuData struct {
	Load1  string
	Load5  string
	Load15 string
	Cores  int
}

type diskData struct {
	Used    string
	Total   string
	Percent string
}

type ramData struct {
	Used    string
	Total   string
	Buffers string
	Cached  string
	Percent string
}

func dashboardHandler() http.HandlerFunc {
	tmpl := template.Must(template.ParseFiles("web/dashboard.html"))

	return func(w http.ResponseWriter, r *http.Request) {
		data := dashboardData{
			UpdatedAt: time.Now().Format("15:04:05"),
		}

		// Uptime
		if uptime, err := health.Uptime(); err != nil {
			log.Printf("uptime error: %v", err)
			data.Uptime = "N/A"
		} else {
			data.Uptime = formatDuration(uptime)
		}

		// CPU Load
		if cpu, err := health.AverageCPULoad(); err != nil {
			log.Printf("cpu load error: %v", err)
			data.CPU = cpuData{Load1: "N/A", Load5: "N/A", Load15: "N/A"}
		} else {
			data.CPU = cpuData{
				Load1:  fmt.Sprintf("%.2f", cpu.Load1),
				Load5:  fmt.Sprintf("%.2f", cpu.Load5),
				Load15: fmt.Sprintf("%.2f", cpu.Load15),
				Cores:  cpu.Cores,
			}
		}

		// Disk
		if disk, err := health.Disk("/"); err != nil {
			log.Printf("disk error: %v", err)
			data.Disk = diskData{Used: "N/A", Total: "N/A", Percent: "0"}
		} else {
			pct := 0.0
			if disk.TotalBytes > 0 {
				pct = float64(disk.UsedBytes) / float64(disk.TotalBytes) * 100
			}
			data.Disk = diskData{
				Used:    formatBytes(disk.UsedBytes),
				Total:   formatBytes(disk.TotalBytes),
				Percent: fmt.Sprintf("%.1f", pct),
			}
		}

		// RAM
		if ram, err := health.RAM(); err != nil {
			log.Printf("ram error: %v", err)
			data.RAM = ramData{Used: "N/A", Total: "N/A", Buffers: "N/A", Cached: "N/A", Percent: "0"}
		} else {
			pct := 0.0
			if ram.TotalBytes > 0 {
				pct = float64(ram.UsedBytes) / float64(ram.TotalBytes) * 100
			}
			data.RAM = ramData{
				Used:    formatBytes(ram.UsedBytes),
				Total:   formatBytes(ram.TotalBytes),
				Buffers: formatBytes(ram.BuffersBytes),
				Cached:  formatBytes(ram.CachedBytes),
				Percent: fmt.Sprintf("%.1f", pct),
			}
		}

		// Docker containers
		if containers, err := health.DockerContainers(r.Context()); err != nil {
			log.Printf("docker error: %v", err)
			data.Containers = nil
		} else {
			data.Containers = containers
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.Execute(w, data); err != nil {
			log.Printf("template error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
}

// formatDuration returns a human-readable duration string like "3d 5h 22m".
func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

// formatBytes converts bytes to a human-readable string.
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
