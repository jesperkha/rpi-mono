-- +goose Up
CREATE TABLE images (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    filename TEXT NOT NULL,
    upload_date DATE NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_images_upload_date ON images(upload_date);
CREATE INDEX idx_images_user_id ON images(user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_images_user_id;
DROP INDEX IF EXISTS idx_images_upload_date;
DROP TABLE images;
