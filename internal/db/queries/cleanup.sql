-- name: GetExpiredDeletedFiles :many
-- Fetch files that have been soft-deleted for more than specific duration (e.g., 30 days)
select file_id, storage_key, thumbnail_key
from files
    where is_deleted = true 
        and deleted_at < $1;

-- name: HardDeleteFiles :exec
-- Permanently remove file records
delete from files where file_id = any($1::uuid[]);

-- name: DeleteExpiredRefreshTokens :exec
delete from refresh_tokens where expires_at < now();

-- name: DeleteExpiredActionTokens :exec
delete from action_tokens where expires_at < now();

-- name: DeleteExpiredAPIKeys :exec
delete from api_keys where expires_at < now();
