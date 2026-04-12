package server

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/jesperkha/admin/config"
	"github.com/jesperkha/admin/docker"
	"github.com/jesperkha/notifier"
)

type Server struct {
	mux     *chi.Mux
	config  *config.Config
	cleanup func()
}

func New(config *config.Config) *Server {
	mux := chi.NewMux()

	// Initialize Docker client
	dockerClient, err := docker.NewClient()
	if err != nil {
		log.Fatalf("Failed to create docker client: %v", err)
	}

	// Initialize auth middleware
	auth := NewAuthMiddleware(config.PasswordHash)

	mux.Use(middleware.Logger)
	mux.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	// Apply auth middleware to all routes
	mux.Use(auth.Middleware)

	// Public routes (but still go through auth middleware which allows /login)
	mux.Get("/login", loginPageHandler())
	mux.Post("/login", loginHandler(auth))

	// Protected routes
	mux.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	})
	mux.Get("/dashboard", dashboardHandler(dockerClient))
	mux.Get("/ping", pingHandler())
	mux.Get("/assets/*", assetsHandler())
	mux.Get("/manifest.json", manifestHandler())
	mux.Post("/containers/{id}/toggle", toggleContainerHandler(dockerClient))
	mux.Post("/actions/{name}", actionHandler())
	mux.Get("/logout", logoutHandler(auth))

	cleanup := func() {
		dockerClient.Close()
	}

	return &Server{
		mux:     mux,
		config:  config,
		cleanup: cleanup,
	}
}

func (s *Server) ListenAndServe(notif *notifier.Notifier) {
	done, finish := notif.Register()

	server := &http.Server{
		Addr:    s.config.Port,
		Handler: s.mux,
	}

	go func() {
		<-done
		if err := server.Shutdown(context.Background()); err != nil {
			log.Println(err)
		}

		log.Println("cleaning up...")
		s.cleanup()
		finish()
	}()

	log.Println("listening on port " + s.config.Port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Println(err)
	}
}
