package server

import (
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jesperkha/admin/docker"
)

type DashboardData struct {
	Containers []docker.Container
	Error      string
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
		containers, err := dockerClient.ListContainers(r.Context())

		data := DashboardData{
			Containers: containers,
		}

		if err != nil {
			data.Error = err.Error()
			log.Printf("Error listing containers: %v", err)
		}

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
