-- +goose Up
CREATE TABLE likes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    image_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (image_id) REFERENCES images(id),
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE(image_id, user_id)
);

CREATE INDEX idx_likes_image_id ON likes(image_id);
CREATE INDEX idx_likes_user_id ON likes(user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_likes_user_id;
DROP INDEX IF EXISTS idx_likes_image_id;
DROP TABLE likes;
