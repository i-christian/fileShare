-- name: CreateUser :one
-- CreateUser adds a new user into the database returning user information.
insert into users (first_name, last_name, email,  password_hash)
    values($1, $2, $3, $4)
on conflict(email)
    do nothing
returning *;

-- name: GetUserByEmail :one
-- GetUserByEmail retrieves a user from the database by email.
select
    user_id,
    email,
    first_name,
    last_name,
    password_hash,
    is_verified,
    role,
    last_login
from users
    where email = $1;

-- name: ActivateUserEmail :one
update users
    set
        is_verified = true,
        version = version + 1    
where user_id = $1
    and version = $2
    returning is_verified;

-- name: ChangePassword :exec
update users
    set
        password_hash = $1,
        version = version + 1
where user_id = $2
    and version = $3;

-- name: CheckIfEmailExists :one
select
    count(email)
from users
    where email = $1;

-- name: GetUserByID :one
select
    user_id,
    email,
    first_name,
    last_name,
    is_verified,
    role
from users
    where user_id = $1;


