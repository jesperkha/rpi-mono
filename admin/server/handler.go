package server

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jesperkha/admin/actions"
	"github.com/jesperkha/admin/docker"
	"github.com/jesperkha/admin/health"
)

type DashboardData struct {
	Uptime     string
	CPU        cpuData
	Disk       diskData
	RAM        ramData
	Containers []docker.Container
	Services   []string
	UpdatedAt  string
	Error      string
}

// discoverServices scans the parent directory for subdirectories that contain
// a docker-compose.yaml file.
func discoverServices() []string {
	entries, err := os.ReadDir("../")
	if err != nil {
		return nil
	}
	var services []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := os.Stat("../" + e.Name() + "/docker-compose.yaml"); err == nil {
			services = append(services, e.Name())
		}
	}
	return services
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

type LoginData struct {
	Error string
}

func pingHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ping"))
	}
}

func indexHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/index.html")
	}
}

func assetsHandler() http.HandlerFunc {
	return http.StripPrefix("/assets/", http.FileServer(http.Dir("web/assets"))).ServeHTTP
}

func manifestHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/manifest.json")
	}
}

func dashboardHandler(dockerClient *docker.Client) http.HandlerFunc {
	tmpl := template.Must(template.ParseFiles("web/templates/dashboard.html"))

	return func(w http.ResponseWriter, r *http.Request) {
		data := DashboardData{
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

		data.Services = discoverServices()

		// Containers
		containers, err := dockerClient.ListContainers(r.Context())
		data.Containers = containers
		if err != nil {
			data.Error = err.Error()
			log.Printf("Error listing containers: %v", err)
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.Execute(w, data); err != nil {
			log.Printf("Error rendering template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
}

func toggleContainerHandler(dockerClient *docker.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		containerID := chi.URLParam(r, "id")
		if containerID == "" {
			http.Error(w, "Container ID required", http.StatusBadRequest)
			return
		}

		containers, err := dockerClient.ListContainers(r.Context())
		if err != nil {
			log.Printf("Error listing containers: %v", err)
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
			return
		}

		var targetContainer *docker.Container
		for _, c := range containers {
			if c.ID == containerID {
				targetContainer = &c
				break
			}
		}

		if targetContainer == nil {
			log.Printf("Container not found: %s", containerID)
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
			return
		}

		if targetContainer.State == "running" {
			if err := dockerClient.StopContainer(r.Context(), containerID); err != nil {
				log.Printf("Error stopping container: %v", err)
			}
		} else {
			if err := dockerClient.StartContainer(r.Context(), containerID); err != nil {
				log.Printf("Error starting container: %v", err)
			}
		}

		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	}
}

func loginPageHandler() http.HandlerFunc {
	tmpl := template.Must(template.ParseFiles("web/templates/login.html"))

	return func(w http.ResponseWriter, r *http.Request) {
		if err := tmpl.Execute(w, LoginData{}); err != nil {
			log.Printf("Error rendering template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
}

// SECURITY: No rate limiting — allows unlimited brute-force password attempts.
func loginHandler(auth *AuthMiddleware) http.HandlerFunc {
	tmpl := template.Must(template.ParseFiles("web/templates/login.html"))

	return func(w http.ResponseWriter, r *http.Request) {
		password := r.FormValue("password")

		if !auth.ValidatePassword(password) {
			data := LoginData{Error: "Invalid password"}
			tmpl.Execute(w, data)
			return
		}

		token := auth.CreateSession()
		// SECURITY: Secure flag is false — the session cookie is sent over
		// plain HTTP and can be intercepted on the network.
		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			Secure:   false, // Set to true in production with HTTPS
			SameSite: http.SameSiteStrictMode,
			MaxAge:   int(sessionDuration.Seconds()),
		})

		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	}
}

func logoutHandler(auth *AuthMiddleware) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err == nil {
			auth.InvalidateSession(cookie.Value)
		}

		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			MaxAge:   -1,
			Expires:  time.Unix(0, 0),
		})

		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

func actionHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		var err error
		switch name {
		case "pull-latest":
			err = actions.PullLatest()
		case "rebuild":
			name := r.FormValue("container")
			if name == "" {
				http.Error(w, "Container name required", http.StatusBadRequest)
				return
			}
			err = actions.Rebuild(name)
		default:
			http.Error(w, "Unknown action", http.StatusNotFound)
			return
		}
		if err != nil {
			log.Printf("Error running action %s: %v", name, err)
		}
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
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
