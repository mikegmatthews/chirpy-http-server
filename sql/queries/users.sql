-- name: CreatUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (
       gen_random_uuid(),
       now(),
       now(),
       $1,
       $2
)
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;

-- name: DeleteAllUsers :exec
DELETE FROM users;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1;

-- name: UpdateUserEmailPass :one
UPDATE users
SET email = $2, hashed_password = $3, updated_at = now()
WHERE id = $1
RETURNING *;
