-- name: CreateUser :one
insert into users (first_name, last_name, email,  password_hash)
    values($1, $2, $3, $4)
returning *;
