package database

import "time"

type User struct {
	ID        int64     `db:"id"`
	Name      string    `db:"name"`
	CreatedAt time.Time `db:"created_at"`
}

type Image struct {
	ID         int64     `db:"id"`
	UserID     int64     `db:"user_id"`
	Filename   string    `db:"filename"`
	UploadDate string    `db:"upload_date"` // DATE stored as YYYY-MM-DD
	CreatedAt  time.Time `db:"created_at"`
}

// ImageWithLikes is returned when querying today's images or results.
type ImageWithLikes struct {
	ID         int64     `db:"id"`
	UserID     int64     `db:"user_id"`
	Filename   string    `db:"filename"`
	UploadDate string    `db:"upload_date"`
	CreatedAt  time.Time `db:"created_at"`
	UserName   string    `db:"user_name"`
	LikeCount  int       `db:"like_count"`
}

type Like struct {
	ID        int64     `db:"id"`
	ImageID   int64     `db:"image_id"`
	UserID    int64     `db:"user_id"`
	CreatedAt time.Time `db:"created_at"`
}

// DailyWinner is returned from the results/winners query.
type DailyWinner struct {
	ImageID    int64  `db:"image_id"`
	Filename   string `db:"filename"`
	UploadDate string `db:"upload_date"`
	UserName   string `db:"user_name"`
	LikeCount  int    `db:"like_count"`
}
