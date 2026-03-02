package database

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// --- Users ---

// CreateUser inserts a new user and returns its ID.
func (db *DB) CreateUser(name string) (int64, error) {
	result, err := db.Exec(
		`INSERT INTO users (name, created_at) VALUES (?, ?)`,
		name, time.Now().UTC(),
	)
	if err != nil {
		return 0, fmt.Errorf("create user: %w", err)
	}
	return result.LastInsertId()
}

// GetUserByID returns a single user by ID.
func (db *DB) GetUserByID(id int64) (*User, error) {
	var u User
	if err := db.Get(&u, `SELECT * FROM users WHERE id = ?`, id); err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &u, nil
}

// GetUserByName returns a single user by name.
func (db *DB) GetUserByName(name string) (*User, error) {
	var u User
	if err := db.Get(&u, `SELECT * FROM users WHERE name = ?`, name); err != nil {
		return nil, fmt.Errorf("get user by name: %w", err)
	}
	return &u, nil
}

// --- Images ---

// CreateImage inserts a new image record.  uploadDate should be YYYY-MM-DD.
func (db *DB) CreateImage(userID int64, filename, uploadDate string) (int64, error) {
	result, err := db.Exec(
		`INSERT INTO images (user_id, filename, upload_date, created_at) VALUES (?, ?, ?, ?)`,
		userID, filename, uploadDate, time.Now().UTC(),
	)
	if err != nil {
		return 0, fmt.Errorf("create image: %w", err)
	}
	return result.LastInsertId()
}

// HasUploadedToday checks whether the user already uploaded an image for the
// given date (YYYY-MM-DD).
func (db *DB) HasUploadedToday(userID int64, date string) (bool, error) {
	var count int
	err := db.Get(&count,
		`SELECT COUNT(*) FROM images WHERE user_id = ? AND upload_date = ?`,
		userID, date,
	)
	if err != nil {
		return false, fmt.Errorf("has uploaded today: %w", err)
	}
	return count > 0, nil
}

// GetImageByID returns a single image by ID.
func (db *DB) GetImageByID(id int64) (*Image, error) {
	var img Image
	if err := db.Get(&img, `SELECT * FROM images WHERE id = ?`, id); err != nil {
		return nil, fmt.Errorf("get image by id: %w", err)
	}
	return &img, nil
}

// DeleteImage removes an image and all its associated likes.
func (db *DB) DeleteImage(id int64) error {
	tx, err := db.Beginx()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM likes WHERE image_id = ?`, id); err != nil {
		return fmt.Errorf("delete likes: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM images WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete image: %w", err)
	}

	return tx.Commit()
}

// GetTodayImages returns all images for the given date (YYYY-MM-DD) with their
// like counts and uploader names, ordered by like count descending.
func (db *DB) GetTodayImages(date string) ([]ImageWithLikes, error) {
	var images []ImageWithLikes
	err := db.Select(&images, `
		SELECT
			i.id,
			i.user_id,
			i.filename,
			DATE(i.upload_date) AS upload_date,
			i.created_at,
			u.name AS user_name,
			COALESCE(COUNT(l.id), 0) AS like_count
		FROM images i
		JOIN users u ON u.id = i.user_id
		LEFT JOIN likes l ON l.image_id = i.id
		WHERE DATE(i.upload_date) = ?
		GROUP BY i.id
		ORDER BY like_count DESC, i.created_at ASC
	`, date)
	if err != nil {
		return nil, fmt.Errorf("get today images: %w", err)
	}
	return images, nil
}

// --- Likes ---

var ErrAlreadyLiked = errors.New("user already liked this image")

// LikeImage adds a like from a user to an image.  Returns ErrAlreadyLiked if
// the user has already liked this image.
func (db *DB) LikeImage(imageID, userID int64) error {
	_, err := db.Exec(
		`INSERT INTO likes (image_id, user_id, created_at) VALUES (?, ?, ?)`,
		imageID, userID, time.Now().UTC(),
	)
	if err != nil {
		// SQLite UNIQUE constraint error contains "UNIQUE constraint failed"
		if isUniqueViolation(err) {
			return ErrAlreadyLiked
		}
		return fmt.Errorf("like image: %w", err)
	}
	return nil
}

// UnlikeImage removes a like.  Returns sql.ErrNoRows (wrapped) if nothing was
// deleted.
func (db *DB) UnlikeImage(imageID, userID int64) error {
	result, err := db.Exec(
		`DELETE FROM likes WHERE image_id = ? AND user_id = ?`,
		imageID, userID,
	)
	if err != nil {
		return fmt.Errorf("unlike image: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("unlike image: %w", sql.ErrNoRows)
	}
	return nil
}

// GetLikeCount returns the number of likes for an image.
func (db *DB) GetLikeCount(imageID int64) (int, error) {
	var count int
	err := db.Get(&count, `SELECT COUNT(*) FROM likes WHERE image_id = ?`, imageID)
	if err != nil {
		return 0, fmt.Errorf("get like count: %w", err)
	}
	return count, nil
}

// HasLiked checks whether a user has liked a specific image.
func (db *DB) HasLiked(imageID, userID int64) (bool, error) {
	var count int
	err := db.Get(&count,
		`SELECT COUNT(*) FROM likes WHERE image_id = ? AND user_id = ?`,
		imageID, userID,
	)
	if err != nil {
		return false, fmt.Errorf("has liked: %w", err)
	}
	return count > 0, nil
}

// --- Results / Daily Winners ---

// GetDailyWinner returns the winning image for a specific date (YYYY-MM-DD).
// Tie-breaker: earliest upload time.  Returns sql.ErrNoRows (wrapped) if no
// images exist for the date.
func (db *DB) GetDailyWinner(date string) (*DailyWinner, error) {
	var w DailyWinner
	err := db.Get(&w, `
		SELECT
			i.id AS image_id,
			i.filename,
			DATE(i.upload_date) AS upload_date,
			u.name AS user_name,
			COALESCE(COUNT(l.id), 0) AS like_count
		FROM images i
		JOIN users u ON u.id = i.user_id
		LEFT JOIN likes l ON l.image_id = i.id
		WHERE DATE(i.upload_date) = ?
		GROUP BY i.id
		ORDER BY like_count DESC, i.created_at ASC
		LIMIT 1
	`, date)
	if err != nil {
		return nil, fmt.Errorf("get daily winner: %w", err)
	}
	return &w, nil
}

// GetAllDailyWinners returns the winning image for each day that has images,
// ordered by date descending (newest first).
func (db *DB) GetAllDailyWinners() ([]DailyWinner, error) {
	// Use a window function to rank images within each day, then pick rank 1.
	var winners []DailyWinner
	err := db.Select(&winners, `
		SELECT image_id, filename, upload_date, user_name, like_count
		FROM (
			SELECT
				i.id AS image_id,
				i.filename,
				DATE(i.upload_date) AS upload_date,
				u.name AS user_name,
				COALESCE(COUNT(l.id), 0) AS like_count,
				ROW_NUMBER() OVER (
					PARTITION BY DATE(i.upload_date)
					ORDER BY COUNT(l.id) DESC, i.created_at ASC
				) AS rn
			FROM images i
			JOIN users u ON u.id = i.user_id
			LEFT JOIN likes l ON l.image_id = i.id
			GROUP BY i.id
		) ranked
		WHERE rn = 1
		ORDER BY upload_date DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("get all daily winners: %w", err)
	}
	return winners, nil
}

// isUniqueViolation detects a SQLite UNIQUE constraint violation.
func isUniqueViolation(err error) bool {
	return err != nil && contains(err.Error(), "UNIQUE constraint failed")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
