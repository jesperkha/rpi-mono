package server

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/jesperkha/healthcheck/config"
	"github.com/jesperkha/notifier"
)

type Server struct {
	mux     *chi.Mux
	config  *config.Config
	cleanup func()
}

func New(config *config.Config) *Server {
	mux := chi.NewMux()

	mux.Use(middleware.Logger)
	mux.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	mux.Get("/", dashboardHandler())
	mux.Get("/ping", pingHandler())
	mux.Get("/manifest.json", manifestHandler())
	mux.Get("/assets/*", assetsHandler())

	cleanup := func() {
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
