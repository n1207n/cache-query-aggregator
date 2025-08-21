-- name: CreatePost :one
INSERT INTO posts (
    user_id,
    content
) VALUES (
    $1, $2
) RETURNING *;

-- name: GetPost :one
SELECT * FROM posts
WHERE id = $1 LIMIT 1;

-- name: ListPostsByUser :many
SELECT * FROM posts
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2
OFFSET $3;

-- name: CreatePostsInBatch :copyfrom
INSERT INTO posts (
    user_id,
    content
) VALUES (
    $1, $2
);