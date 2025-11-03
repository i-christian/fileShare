-- name: CreateActionToken :exec
insert into action_tokens(user_id, purpose, token_hash, expires_at)
    values($1, $2, $3, $4)
returning *; 

