package server

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jesperkha/dagensbilde/database"
)

// maxUploadSize is the maximum file size for image uploads (5 MB).
const maxUploadSize = 5 << 20

// allowedTypes maps MIME types to file extensions for accepted image formats.
var allowedTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// --- JSON helpers ---

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// --- Auth middleware ---

// authMiddleware checks for a valid user_id cookie (set at login).
// Returns JSON 401 for API routes.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("user_id")
		if err != nil || cookie.Value == "" {
			jsonError(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		userID, err := strconv.ParseInt(cookie.Value, 10, 64)
		if err != nil {
			jsonError(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// Verify user exists
		_, err = s.db.GetUserByID(userID)
		if err != nil {
			jsonError(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// pageAuthMiddleware redirects to /login for page routes when not authenticated.
func (s *Server) pageAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("user_id")
		if err != nil || cookie.Value == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		userID, err := strconv.ParseInt(cookie.Value, 10, 64)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		if _, err = s.db.GetUserByID(userID); err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// getUserID extracts the authenticated user ID from the cookie.
func getUserID(r *http.Request) int64 {
	cookie, _ := r.Cookie("user_id")
	id, _ := strconv.ParseInt(cookie.Value, 10, 64)
	return id
}

// --- POST /api/login ---

type loginRequest struct {
	Password string `json:"password"`
	Name     string `json:"name"`
}

type loginResponse struct {
	UserID int64  `json:"user_id"`
	Name   string `json:"name"`
}

func (s *Server) loginHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		isForm := strings.HasPrefix(contentType, "application/x-www-form-urlencoded") ||
			strings.HasPrefix(contentType, "multipart/form-data")

		var name, password string

		if isForm {
			if err := r.ParseForm(); err != nil {
				http.Redirect(w, r, "/login?error=Ugyldig+forespørsel", http.StatusSeeOther)
				return
			}
			name = strings.TrimSpace(r.FormValue("name"))
			password = r.FormValue("password")
		} else {
			var req loginRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				jsonError(w, "invalid request body", http.StatusBadRequest)
				return
			}
			name = strings.TrimSpace(req.Name)
			password = req.Password
		}

		if name == "" {
			if isForm {
				http.Redirect(w, r, "/login?error=Navn+er+påkrevd", http.StatusSeeOther)
				return
			}
			jsonError(w, "name is required", http.StatusBadRequest)
			return
		}

		if password == "" {
			if isForm {
				http.Redirect(w, r, "/login?error=Passord+er+påkrevd", http.StatusSeeOther)
				return
			}
			jsonError(w, "password is required", http.StatusBadRequest)
			return
		}

		// Verify password against stored hash
		hash := sha256.Sum256([]byte(password))
		if fmt.Sprintf("%x", hash) != s.config.PasswordHash {
			if isForm {
				http.Redirect(w, r, "/login?error=Feil+passord", http.StatusSeeOther)
				return
			}
			jsonError(w, "wrong password", http.StatusUnauthorized)
			return
		}

		// Find or create user by name
		user, err := s.db.GetUserByName(name)
		if err != nil {
			// User doesn't exist – create
			id, createErr := s.db.CreateUser(name)
			if createErr != nil {
				log.Printf("create user: %v", createErr)
				if isForm {
					http.Redirect(w, r, "/login?error=Intern+feil", http.StatusSeeOther)
					return
				}
				jsonError(w, "internal error", http.StatusInternalServerError)
				return
			}
			user = &database.User{ID: id, Name: name}
		}

		// Set a cookie with the user ID
		http.SetCookie(w, &http.Cookie{
			Name:     "user_id",
			Value:    strconv.FormatInt(user.ID, 10),
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   60 * 60 * 24 * 365, // 1 year
		})

		if isForm {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		jsonOK(w, loginResponse{
			UserID: user.ID,
			Name:   user.Name,
		})
	}
}

// --- POST /api/upload ---

type uploadResponse struct {
	ImageID int64  `json:"image_id"`
	URL     string `json:"url"`
}

func (s *Server) uploadHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)
		today := time.Now().UTC().Format("2006-01-02")

		// Business rule: one upload per user per day
		has, err := s.db.HasUploadedToday(userID, today)
		if err != nil {
			log.Printf("has uploaded today: %v", err)
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}
		if has {
			jsonError(w, "you have already uploaded an image today", http.StatusConflict)
			return
		}

		// Limit request body size
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

		file, header, err := r.FormFile("image")
		if err != nil {
			if err.Error() == "http: request body too large" {
				jsonError(w, "file too large (max 5MB)", http.StatusRequestEntityTooLarge)
				return
			}
			jsonError(w, "missing image field", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Validate file type by reading the first 512 bytes
		buf := make([]byte, 512)
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			jsonError(w, "failed to read file", http.StatusBadRequest)
			return
		}
		contentType := http.DetectContentType(buf[:n])
		ext, ok := allowedTypes[contentType]
		if !ok {
			jsonError(w, fmt.Sprintf("unsupported file type: %s (allowed: jpg, png, webp)", contentType), http.StatusBadRequest)
			return
		}
		// Seek back to start for full copy
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			jsonError(w, "failed to process file", http.StatusInternalServerError)
			return
		}

		// Generate unique filename
		randBytes := make([]byte, 16)
		if _, err := rand.Read(randBytes); err != nil {
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}
		filename := hex.EncodeToString(randBytes) + ext

		// Ensure date directory exists: <imageDir>/YYYY-MM-DD/
		dateDir := filepath.Join(s.config.ImageDir, today)
		if err := os.MkdirAll(dateDir, 0o755); err != nil {
			log.Printf("mkdir %s: %v", dateDir, err)
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}

		// Write file to disk
		destPath := filepath.Join(dateDir, filename)
		dst, err := os.Create(destPath)
		if err != nil {
			log.Printf("create file %s: %v", destPath, err)
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, file); err != nil {
			log.Printf("write file: %v", err)
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}

		// Store filename relative to imageDir (YYYY-MM-DD/name.ext)
		relFilename := filepath.Join(today, filename)

		imgID, err := s.db.CreateImage(userID, relFilename, today)
		if err != nil {
			log.Printf("create image: %v", err)
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}

		_ = header // used only for FormFile

		jsonOK(w, uploadResponse{
			ImageID: imgID,
			URL:     "/images/" + relFilename,
		})
	}
}

// --- GET /api/images/today ---

type todayImageResponse struct {
	ID        int64  `json:"id"`
	URL       string `json:"url"`
	UserName  string `json:"user_name"`
	LikeCount int    `json:"like_count"`
	CreatedAt string `json:"created_at"`
}

func (s *Server) getTodayImagesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		today := time.Now().UTC().Format("2006-01-02")

		images, err := s.db.GetTodayImages(today)
		if err != nil {
			log.Printf("get today images: %v", err)
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}

		resp := make([]todayImageResponse, 0, len(images))
		for _, img := range images {
			resp = append(resp, todayImageResponse{
				ID:        img.ID,
				URL:       "/images/" + img.Filename,
				UserName:  img.UserName,
				LikeCount: img.LikeCount,
				CreatedAt: img.CreatedAt.Format(time.RFC3339),
			})
		}

		jsonOK(w, resp)
	}
}

// --- GET /api/images/:id/like ---

func (s *Server) getImageLikeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)

		imageIDStr := chi.URLParam(r, "id")
		imageID, err := strconv.ParseInt(imageIDStr, 10, 64)
		if err != nil {
			jsonError(w, "invalid image id", http.StatusBadRequest)
			return
		}

		liked, err := s.db.HasLiked(imageID, userID)
		if err != nil {
			log.Printf("has liked: %v", err)
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}

		count, err := s.db.GetLikeCount(imageID)
		if err != nil {
			log.Printf("get like count: %v", err)
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}

		jsonOK(w, map[string]any{
			"liked":      liked,
			"like_count": count,
		})
	}
}

// --- POST /api/images/:id/like ---

func (s *Server) likeImageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)

		imageIDStr := chi.URLParam(r, "id")
		imageID, err := strconv.ParseInt(imageIDStr, 10, 64)
		if err != nil {
			jsonError(w, "invalid image id", http.StatusBadRequest)
			return
		}

		// Verify image exists
		_, err = s.db.GetImageByID(imageID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				jsonError(w, "image not found", http.StatusNotFound)
				return
			}
			log.Printf("get image: %v", err)
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}

		// Toggle: if already liked, unlike; otherwise like
		liked, err := s.db.HasLiked(imageID, userID)
		if err != nil {
			log.Printf("has liked: %v", err)
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}

		if liked {
			if err := s.db.UnlikeImage(imageID, userID); err != nil {
				log.Printf("unlike image: %v", err)
				jsonError(w, "internal error", http.StatusInternalServerError)
				return
			}
		} else {
			if err := s.db.LikeImage(imageID, userID); err != nil {
				log.Printf("like image: %v", err)
				jsonError(w, "internal error", http.StatusInternalServerError)
				return
			}
		}

		count, err := s.db.GetLikeCount(imageID)
		if err != nil {
			log.Printf("get like count: %v", err)
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}

		jsonOK(w, map[string]any{
			"liked":      !liked,
			"like_count": count,
		})
	}
}

// --- DELETE /api/images/:id ---

func (s *Server) deleteImageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)

		imageIDStr := chi.URLParam(r, "id")
		imageID, err := strconv.ParseInt(imageIDStr, 10, 64)
		if err != nil {
			jsonError(w, "invalid image id", http.StatusBadRequest)
			return
		}

		img, err := s.db.GetImageByID(imageID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				jsonError(w, "image not found", http.StatusNotFound)
				return
			}
			log.Printf("get image: %v", err)
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}

		// Only the uploader can delete their image
		if img.UserID != userID {
			jsonError(w, "forbidden", http.StatusForbidden)
			return
		}

		// Delete file from disk
		filePath := filepath.Join(s.config.ImageDir, img.Filename)
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			log.Printf("remove file %s: %v", filePath, err)
		}

		// Delete from database (likes + image)
		if err := s.db.DeleteImage(imageID); err != nil {
			log.Printf("delete image: %v", err)
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}

		jsonOK(w, map[string]string{"status": "deleted"})
	}
}

// --- GET /api/results?date=YYYY-MM-DD ---

type winnerResponse struct {
	ImageID    int64  `json:"image_id"`
	URL        string `json:"url"`
	UploadDate string `json:"upload_date"`
	UserName   string `json:"user_name"`
	LikeCount  int    `json:"like_count"`
}

func (s *Server) getResultsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		date := r.URL.Query().Get("date")
		if date == "" {
			jsonError(w, "date query parameter is required (YYYY-MM-DD)", http.StatusBadRequest)
			return
		}

		// Validate date format
		if _, err := time.Parse("2006-01-02", date); err != nil {
			jsonError(w, "invalid date format, use YYYY-MM-DD", http.StatusBadRequest)
			return
		}

		winner, err := s.db.GetDailyWinner(date)
		if err != nil {
			if strings.Contains(err.Error(), "no rows") {
				jsonError(w, "no images for this date", http.StatusNotFound)
				return
			}
			log.Printf("get daily winner: %v", err)
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}

		jsonOK(w, winnerResponse{
			ImageID:    winner.ImageID,
			URL:        "/images/" + winner.Filename,
			UploadDate: winner.UploadDate,
			UserName:   winner.UserName,
			LikeCount:  winner.LikeCount,
		})
	}
}

// --- GET /api/results/all ---

func (s *Server) getAllResultsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		winners, err := s.db.GetAllDailyWinners()
		if err != nil {
			log.Printf("get all daily winners: %v", err)
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}

		resp := make([]winnerResponse, 0, len(winners))
		for _, w := range winners {
			resp = append(resp, winnerResponse{
				ImageID:    w.ImageID,
				URL:        "/images/" + w.Filename,
				UploadDate: w.UploadDate,
				UserName:   w.UserName,
				LikeCount:  w.LikeCount,
			})
		}

		jsonOK(w, resp)
	}
}
