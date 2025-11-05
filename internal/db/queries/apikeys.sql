-- name: CheckIfAPIKeyExists :one
select count(*) from api_keys where prefix = $1;

-- name: CreateApiKey :one
insert into api_keys (
    user_id,
    name,
    key_hash,
    prefix,
    scope,
    expires_at
)
values (
    $1, $2, $3, $4, $5, $6
)
returning *;

-- name: ListApiKeysByUser :many
select
    api_key_id,
    name,
    prefix,
    scope,
    is_revoked,
    expires_at
from api_keys
    where user_id = $1
order by created_at desc;

-- name: GetApiKeyByPrefix :one
select
    ak.api_key_id,
    ak.key_hash,
    ak.expires_at,
    u.user_id,
    u.first_name,
    u.last_name,
    u.role,
    u.email
from api_keys ak
    join users u using(user_id)
where prefix = $1;

-- name: RevokeApiKey :exec
update api_keys
set is_revoked = true,
    revoked_at = now()
where api_key_id = $1;


-- name: UpdateApiKeyLastUsed :exec
update api_keys
    set last_used_at = $2
where api_key_id = $1;

-- name: DeleteApiKey :exec
delete from api_keys
    where api_key_id = $1;

