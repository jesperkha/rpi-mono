package database_test

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jesperkha/dagensbilde/database"
)

// testDB creates a fresh in-memory database with all migrations applied.
// It returns the database and a cleanup function.
func testDB(t *testing.T) *database.DB {
	t.Helper()

	// Use a temp file so that SQLite foreign keys / WAL work properly.
	tmp := filepath.Join(t.TempDir(), "test.db")

	db, err := database.New(tmp)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}

	// Find the sql migrations directory relative to this test file.
	migrationsDir := findMigrationsDir(t)

	if err := db.Migrate(migrationsDir); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	t.Cleanup(func() { db.Close() })
	return db
}

func findMigrationsDir(t *testing.T) string {
	t.Helper()
	// Walk upward from the working directory to find the sql/ folder.
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		candidate := filepath.Join(dir, "sql")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find sql/ migrations directory")
		}
		dir = parent
	}
}

func today() string {
	return time.Now().UTC().Format("2006-01-02")
}

// ===================== User Tests =====================

func TestCreateUser(t *testing.T) {
	db := testDB(t)

	id, err := db.CreateUser("Alice")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected positive id, got %d", id)
	}
}

func TestGetUserByID(t *testing.T) {
	db := testDB(t)

	id, _ := db.CreateUser("Bob")
	u, err := db.GetUserByID(id)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if u.Name != "Bob" {
		t.Fatalf("expected name Bob, got %s", u.Name)
	}
	if u.ID != id {
		t.Fatalf("expected id %d, got %d", id, u.ID)
	}
}

func TestGetUserByID_NotFound(t *testing.T) {
	db := testDB(t)

	_, err := db.GetUserByID(9999)
	if err == nil {
		t.Fatal("expected error for missing user")
	}
}

func TestGetUserByName(t *testing.T) {
	db := testDB(t)

	db.CreateUser("Charlie")
	u, err := db.GetUserByName("Charlie")
	if err != nil {
		t.Fatalf("GetUserByName: %v", err)
	}
	if u.Name != "Charlie" {
		t.Fatalf("expected Charlie, got %s", u.Name)
	}
}

func TestGetUserByName_NotFound(t *testing.T) {
	db := testDB(t)

	_, err := db.GetUserByName("NoSuchUser")
	if err == nil {
		t.Fatal("expected error for missing user")
	}
}

// ===================== Image Tests =====================

func TestCreateImage(t *testing.T) {
	db := testDB(t)
	uid, _ := db.CreateUser("Alice")

	imgID, err := db.CreateImage(uid, "photo.jpg", today())
	if err != nil {
		t.Fatalf("CreateImage: %v", err)
	}
	if imgID <= 0 {
		t.Fatalf("expected positive image id, got %d", imgID)
	}
}

func TestGetImageByID(t *testing.T) {
	db := testDB(t)
	uid, _ := db.CreateUser("Alice")
	imgID, _ := db.CreateImage(uid, "photo.jpg", today())

	img, err := db.GetImageByID(imgID)
	if err != nil {
		t.Fatalf("GetImageByID: %v", err)
	}
	if img.Filename != "photo.jpg" {
		t.Fatalf("expected photo.jpg, got %s", img.Filename)
	}
	if img.UserID != uid {
		t.Fatalf("expected user_id %d, got %d", uid, img.UserID)
	}
}

func TestGetImageByID_NotFound(t *testing.T) {
	db := testDB(t)

	_, err := db.GetImageByID(9999)
	if err == nil {
		t.Fatal("expected error for missing image")
	}
}

func TestHasUploadedToday(t *testing.T) {
	db := testDB(t)
	uid, _ := db.CreateUser("Alice")
	date := today()

	has, err := db.HasUploadedToday(uid, date)
	if err != nil {
		t.Fatal(err)
	}
	if has {
		t.Fatal("should not have uploaded yet")
	}

	db.CreateImage(uid, "photo.jpg", date)

	has, err = db.HasUploadedToday(uid, date)
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Fatal("should have uploaded")
	}
}

func TestHasUploadedToday_DifferentDate(t *testing.T) {
	db := testDB(t)
	uid, _ := db.CreateUser("Alice")

	db.CreateImage(uid, "photo.jpg", "2025-01-01")

	has, err := db.HasUploadedToday(uid, "2025-01-02")
	if err != nil {
		t.Fatal(err)
	}
	if has {
		t.Fatal("should not count upload from different date")
	}
}

func TestHasUploadedToday_DifferentUser(t *testing.T) {
	db := testDB(t)
	uid1, _ := db.CreateUser("Alice")
	uid2, _ := db.CreateUser("Bob")
	date := today()

	db.CreateImage(uid1, "photo.jpg", date)

	has, err := db.HasUploadedToday(uid2, date)
	if err != nil {
		t.Fatal(err)
	}
	if has {
		t.Fatal("different user should not be affected")
	}
}

func TestGetTodayImages_Empty(t *testing.T) {
	db := testDB(t)

	images, err := db.GetTodayImages(today())
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 0 {
		t.Fatalf("expected 0 images, got %d", len(images))
	}
}

func TestGetTodayImages_WithLikes(t *testing.T) {
	db := testDB(t)
	date := today()

	uid1, _ := db.CreateUser("Alice")
	uid2, _ := db.CreateUser("Bob")
	uid3, _ := db.CreateUser("Charlie")

	img1, _ := db.CreateImage(uid1, "a.jpg", date)
	img2, _ := db.CreateImage(uid2, "b.jpg", date)

	// img2 gets 2 likes, img1 gets 1 like
	db.LikeImage(img2, uid1)
	db.LikeImage(img2, uid3)
	db.LikeImage(img1, uid2)

	images, err := db.GetTodayImages(date)
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 2 {
		t.Fatalf("expected 2 images, got %d", len(images))
	}

	// Ordered by like count descending
	if images[0].LikeCount != 2 {
		t.Fatalf("first image should have 2 likes, got %d", images[0].LikeCount)
	}
	if images[0].UserName != "Bob" {
		t.Fatalf("first image uploader should be Bob, got %s", images[0].UserName)
	}
	if images[1].LikeCount != 1 {
		t.Fatalf("second image should have 1 like, got %d", images[1].LikeCount)
	}
}

func TestGetTodayImages_OnlyReturnsRequestedDate(t *testing.T) {
	db := testDB(t)
	uid, _ := db.CreateUser("Alice")

	db.CreateImage(uid, "old.jpg", "2025-06-01")
	db.CreateImage(uid, "today.jpg", "2025-06-02")

	images, err := db.GetTodayImages("2025-06-02")
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(images))
	}
	if images[0].Filename != "today.jpg" {
		t.Fatalf("expected today.jpg, got %s", images[0].Filename)
	}
}

// ===================== Like Tests =====================

func TestLikeImage(t *testing.T) {
	db := testDB(t)
	uid1, _ := db.CreateUser("Alice")
	uid2, _ := db.CreateUser("Bob")
	imgID, _ := db.CreateImage(uid1, "photo.jpg", today())

	if err := db.LikeImage(imgID, uid2); err != nil {
		t.Fatalf("LikeImage: %v", err)
	}

	count, _ := db.GetLikeCount(imgID)
	if count != 1 {
		t.Fatalf("expected 1 like, got %d", count)
	}
}

func TestLikeImage_DuplicateFails(t *testing.T) {
	db := testDB(t)
	uid1, _ := db.CreateUser("Alice")
	uid2, _ := db.CreateUser("Bob")
	imgID, _ := db.CreateImage(uid1, "photo.jpg", today())

	db.LikeImage(imgID, uid2)
	err := db.LikeImage(imgID, uid2)
	if !errors.Is(err, database.ErrAlreadyLiked) {
		t.Fatalf("expected ErrAlreadyLiked, got %v", err)
	}
}

func TestLikeImage_MultipleUsersCanLike(t *testing.T) {
	db := testDB(t)
	uid1, _ := db.CreateUser("Alice")
	uid2, _ := db.CreateUser("Bob")
	uid3, _ := db.CreateUser("Charlie")
	imgID, _ := db.CreateImage(uid1, "photo.jpg", today())

	db.LikeImage(imgID, uid2)
	db.LikeImage(imgID, uid3)

	count, _ := db.GetLikeCount(imgID)
	if count != 2 {
		t.Fatalf("expected 2 likes, got %d", count)
	}
}

func TestLikeImage_UserCanLikeMultipleImages(t *testing.T) {
	db := testDB(t)
	uid1, _ := db.CreateUser("Alice")
	uid2, _ := db.CreateUser("Bob")
	img1, _ := db.CreateImage(uid1, "a.jpg", today())
	img2, _ := db.CreateImage(uid1, "b.jpg", "2025-01-01")

	if err := db.LikeImage(img1, uid2); err != nil {
		t.Fatal(err)
	}
	if err := db.LikeImage(img2, uid2); err != nil {
		t.Fatal(err)
	}

	c1, _ := db.GetLikeCount(img1)
	c2, _ := db.GetLikeCount(img2)
	if c1 != 1 || c2 != 1 {
		t.Fatalf("expected 1 like each, got %d and %d", c1, c2)
	}
}

func TestUnlikeImage(t *testing.T) {
	db := testDB(t)
	uid1, _ := db.CreateUser("Alice")
	uid2, _ := db.CreateUser("Bob")
	imgID, _ := db.CreateImage(uid1, "photo.jpg", today())

	db.LikeImage(imgID, uid2)
	if err := db.UnlikeImage(imgID, uid2); err != nil {
		t.Fatalf("UnlikeImage: %v", err)
	}

	count, _ := db.GetLikeCount(imgID)
	if count != 0 {
		t.Fatalf("expected 0 likes after unlike, got %d", count)
	}
}

func TestUnlikeImage_NotLiked(t *testing.T) {
	db := testDB(t)
	uid1, _ := db.CreateUser("Alice")
	uid2, _ := db.CreateUser("Bob")
	imgID, _ := db.CreateImage(uid1, "photo.jpg", today())

	err := db.UnlikeImage(imgID, uid2)
	if err == nil {
		t.Fatal("expected error when unliking non-liked image")
	}
}

func TestHasLiked(t *testing.T) {
	db := testDB(t)
	uid1, _ := db.CreateUser("Alice")
	uid2, _ := db.CreateUser("Bob")
	imgID, _ := db.CreateImage(uid1, "photo.jpg", today())

	liked, _ := db.HasLiked(imgID, uid2)
	if liked {
		t.Fatal("should not have liked yet")
	}

	db.LikeImage(imgID, uid2)
	liked, _ = db.HasLiked(imgID, uid2)
	if !liked {
		t.Fatal("should have liked")
	}
}

func TestGetLikeCount_NoLikes(t *testing.T) {
	db := testDB(t)
	uid, _ := db.CreateUser("Alice")
	imgID, _ := db.CreateImage(uid, "photo.jpg", today())

	count, err := db.GetLikeCount(imgID)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected 0 likes, got %d", count)
	}
}

// ===================== Daily Winner Tests =====================

func TestGetDailyWinner(t *testing.T) {
	db := testDB(t)
	date := "2025-07-04"

	uid1, _ := db.CreateUser("Alice")
	uid2, _ := db.CreateUser("Bob")
	uid3, _ := db.CreateUser("Charlie")

	img1, _ := db.CreateImage(uid1, "a.jpg", date)
	db.CreateImage(uid2, "b.jpg", date)

	db.LikeImage(img1, uid2)
	db.LikeImage(img1, uid3)

	w, err := db.GetDailyWinner(date)
	if err != nil {
		t.Fatalf("GetDailyWinner: %v", err)
	}
	if w.ImageID != img1 {
		t.Fatalf("expected winner image id %d, got %d", img1, w.ImageID)
	}
	if w.LikeCount != 2 {
		t.Fatalf("expected 2 likes, got %d", w.LikeCount)
	}
	if w.UserName != "Alice" {
		t.Fatalf("expected Alice, got %s", w.UserName)
	}
}

func TestGetDailyWinner_Tiebreaker(t *testing.T) {
	db := testDB(t)
	date := "2025-08-01"

	uid1, _ := db.CreateUser("Alice")
	uid2, _ := db.CreateUser("Bob")
	uid3, _ := db.CreateUser("Charlie")

	// img1 uploaded first
	img1, _ := db.CreateImage(uid1, "first.jpg", date)
	img2, _ := db.CreateImage(uid2, "second.jpg", date)

	// Same number of likes
	db.LikeImage(img1, uid3)
	db.LikeImage(img2, uid3)

	w, err := db.GetDailyWinner(date)
	if err != nil {
		t.Fatal(err)
	}
	// Tiebreaker = earliest upload (img1)
	if w.ImageID != img1 {
		t.Fatalf("tiebreaker should pick earliest upload: expected %d, got %d", img1, w.ImageID)
	}
}

func TestGetDailyWinner_NoImages(t *testing.T) {
	db := testDB(t)

	_, err := db.GetDailyWinner("2099-12-31")
	if err == nil {
		t.Fatal("expected error for date with no images")
	}
	if !errors.Is(err, sql.ErrNoRows) {
		// The error should wrap sql.ErrNoRows
		if !containsString(err.Error(), "no rows") {
			t.Fatalf("expected no rows error, got: %v", err)
		}
	}
}

func TestGetAllDailyWinners(t *testing.T) {
	db := testDB(t)

	uid1, _ := db.CreateUser("Alice")
	uid2, _ := db.CreateUser("Bob")
	uid3, _ := db.CreateUser("Charlie")

	// Day 1
	img1, _ := db.CreateImage(uid1, "day1.jpg", "2025-09-01")
	db.CreateImage(uid2, "day1b.jpg", "2025-09-01")
	db.LikeImage(img1, uid2)
	db.LikeImage(img1, uid3)

	// Day 2
	img3, _ := db.CreateImage(uid2, "day2.jpg", "2025-09-02")
	db.LikeImage(img3, uid1)

	winners, err := db.GetAllDailyWinners()
	if err != nil {
		t.Fatalf("GetAllDailyWinners: %v", err)
	}
	if len(winners) != 2 {
		t.Fatalf("expected 2 winners, got %d", len(winners))
	}

	// Newest first
	if winners[0].UploadDate != "2025-09-02" {
		t.Fatalf("expected first winner date 2025-09-02, got %s", winners[0].UploadDate)
	}
	if winners[1].UploadDate != "2025-09-01" {
		t.Fatalf("expected second winner date 2025-09-01, got %s", winners[1].UploadDate)
	}
	if winners[1].LikeCount != 2 {
		t.Fatalf("day1 winner should have 2 likes, got %d", winners[1].LikeCount)
	}
}

func TestGetAllDailyWinners_Empty(t *testing.T) {
	db := testDB(t)

	winners, err := db.GetAllDailyWinners()
	if err != nil {
		t.Fatal(err)
	}
	if len(winners) != 0 {
		t.Fatalf("expected 0 winners, got %d", len(winners))
	}
}

func TestGetAllDailyWinners_SingleImagePerDay(t *testing.T) {
	db := testDB(t)
	uid, _ := db.CreateUser("Alice")

	db.CreateImage(uid, "solo.jpg", "2025-10-01")

	winners, err := db.GetAllDailyWinners()
	if err != nil {
		t.Fatal(err)
	}
	if len(winners) != 1 {
		t.Fatalf("expected 1 winner, got %d", len(winners))
	}
	if winners[0].LikeCount != 0 {
		t.Fatalf("expected 0 likes, got %d", winners[0].LikeCount)
	}
}

// ===================== Migration Test =====================

func TestMigrate_BadDir(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "test.db")
	db, err := database.New(tmp)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Migrate("/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for bad migrations dir")
	}
}

// helper
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
