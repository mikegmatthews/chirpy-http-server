-- name: CreatUser :one
INSERT INTO users (id, created_at, updated_at, email)
VALUES (
       get_random_uuid(),
       now(),
       now(),
       $1
)
RETURNING *;
