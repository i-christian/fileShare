-- name: CreateRefreshToken :one
insert into refresh_tokens(user_id, token, expires_at, created_at, revoked)
    values ($1, $2, $3, $4, $5)
returning refresh_token_id;

-- name: GetRefreshToken :one
select
    refresh_token_id,
    user_id,
    token,
    expires_at,
    created_at,
    revoked
from refresh_tokens
    where token = $1;

-- name: RevokeRefreshToken :exec
update refresh_tokens
    set revoked = true
        where token = $1;

-- name: DeleteRefreshToken :exec
delete from refresh_tokens where token = $1;
