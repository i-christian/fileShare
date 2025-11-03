-- name: CreateActionToken :exec
insert into action_tokens(user_id, purpose, token_hash, expires_at)
    values($1, $2, $3, $4)
returning *; 


-- name: GetActionTokenForUser :one
select
    at.token_hash,
    at.used,
    at.expires_at,
    at.purpose,
    u.is_verified,
    u.email,
    u.version,
    u.user_id
from action_tokens at
    join users u using (user_id)
where at.token_hash = $1
    and at.purpose = $2
    and u.user_id = $3
    and at.expires_at > $4;

-- name: DeleteActionToken :exec
delete from action_tokens
    where token_hash = $1
        and user_id = $2;
