package server

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"time"
)

// templates holds the parsed template sets for each page.
type templates struct {
	login   *template.Template
	home    *template.Template
	results *template.Template
}

func loadTemplates() *templates {
	base := filepath.Join("web", "templates", "base.html")
	return &templates{
		login:   template.Must(template.ParseFiles(base, filepath.Join("web", "templates", "login.html"))),
		home:    template.Must(template.ParseFiles(base, filepath.Join("web", "templates", "home.html"))),
		results: template.Must(template.ParseFiles(base, filepath.Join("web", "templates", "results.html"))),
	}
}

// --- page data types ---

type loginPageData struct {
	LoggedIn  bool
	ActiveTab string
	Error     string
}

type homeImageData struct {
	ID        int64
	URL       string
	UserID    int64
	UserName  string
	LikeCount int
	CreatedAt string
	Liked     bool
}

type homePageData struct {
	LoggedIn      bool
	ActiveTab     string
	Today         string
	HasUploaded   bool
	CurrentUserID int64
	Images        []homeImageData
}

type winnerData struct {
	ImageID    int64
	URL        string
	UploadDate string
	UserName   string
	LikeCount  int
}

type resultsPageData struct {
	LoggedIn  bool
	ActiveTab string
	Winners   []winnerData
}

// --- page handlers ---

func pingHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ping"))
	}
}

func assetsHandler() http.HandlerFunc {
	return http.StripPrefix("/assets/", http.FileServer(http.Dir("web/assets"))).ServeHTTP
}

func manifestHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/manifest+json")
		http.ServeFile(w, r, "web/manifest.json")
	}
}

func (s *Server) loginPageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		errMsg := r.URL.Query().Get("error")
		data := loginPageData{
			LoggedIn:  false,
			ActiveTab: "",
			Error:     errMsg,
		}
		if err := s.tmpl.login.ExecuteTemplate(w, "base", data); err != nil {
			log.Printf("render login: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
	}
}

func (s *Server) homePageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)
		today := time.Now().UTC().Format("2006-01-02")

		images, err := s.db.GetTodayImages(today)
		if err != nil {
			log.Printf("get today images: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		hasUploaded, err := s.db.HasUploadedToday(userID, today)
		if err != nil {
			log.Printf("has uploaded today: %v", err)
			hasUploaded = false
		}

		var imageData []homeImageData
		for _, img := range images {
			liked, _ := s.db.HasLiked(img.ID, userID)
			imageData = append(imageData, homeImageData{
				ID:        img.ID,
				URL:       "/images/" + img.Filename,
				UserID:    img.UserID,
				UserName:  img.UserName,
				LikeCount: img.LikeCount,
				CreatedAt: img.CreatedAt.Format(time.RFC3339),
				Liked:     liked,
			})
		}

		data := homePageData{
			LoggedIn:      true,
			ActiveTab:     "home",
			Today:         today,
			HasUploaded:   hasUploaded,
			CurrentUserID: userID,
			Images:        imageData,
		}

		if err := s.tmpl.home.ExecuteTemplate(w, "base", data); err != nil {
			log.Printf("render home: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
	}
}

func (s *Server) resultsPageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		winners, err := s.db.GetAllDailyWinners()
		if err != nil {
			log.Printf("get all daily winners: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		var winData []winnerData
		for _, w := range winners {
			winData = append(winData, winnerData{
				ImageID:    w.ImageID,
				URL:        "/images/" + w.Filename,
				UploadDate: w.UploadDate,
				UserName:   w.UserName,
				LikeCount:  w.LikeCount,
			})
		}

		data := resultsPageData{
			LoggedIn:  true,
			ActiveTab: "results",
			Winners:   winData,
		}

		if err := s.tmpl.results.ExecuteTemplate(w, "base", data); err != nil {
			log.Printf("render results: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
	}
}
