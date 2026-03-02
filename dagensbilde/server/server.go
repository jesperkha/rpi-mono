package server

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/jesperkha/dagensbilde/config"
	"github.com/jesperkha/dagensbilde/database"
	"github.com/jesperkha/notifier"
)

type Server struct {
	mux     *chi.Mux
	config  *config.Config
	db      *database.DB
	tmpl    *templates
	cleanup func()
}

func New(cfg *config.Config, db *database.DB) *Server {
	mux := chi.NewMux()

	mux.Use(middleware.Logger)
	mux.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	s := &Server{
		mux:    mux,
		config: cfg,
		db:     db,
		tmpl:   loadTemplates(),
		cleanup: func() {
			db.Close()
		},
	}

	// Login page (unprotected)
	mux.Get("/login", s.loginPageHandler())
	mux.Get("/ping", pingHandler())
	mux.Get("/assets/*", assetsHandler())
	mux.Get("/manifest.json", manifestHandler())

	// Serve uploaded images (auth protected)
	mux.Group(func(r chi.Router) {
		r.Use(s.pageAuthMiddleware)
		r.Handle("/images/*", http.StripPrefix("/images/", http.FileServer(http.Dir(cfg.ImageDir))))
	})

	// Pages that require auth (redirect to /login if not authed)
	mux.Group(func(r chi.Router) {
		r.Use(s.pageAuthMiddleware)

		r.Get("/", s.homePageHandler())
		r.Get("/results", s.resultsPageHandler())
	})

	// API routes
	mux.Route("/api", func(r chi.Router) {
		r.Post("/login", s.loginHandler())

		// Protected API routes
		r.Group(func(r chi.Router) {
			r.Use(s.authMiddleware)

			r.Post("/upload", s.uploadHandler())
			r.Get("/images/today", s.getTodayImagesHandler())
			r.Get("/images/{id}/like", s.getImageLikeHandler())
			r.Post("/images/{id}/like", s.likeImageHandler())
			r.Delete("/images/{id}", s.deleteImageHandler())
			r.Get("/results", s.getResultsHandler())
			r.Get("/results/all", s.getAllResultsHandler())
		})
	})

	return s
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
