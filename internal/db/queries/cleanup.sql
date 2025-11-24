-- name: GetExpiredDeletedFiles :many
-- Fetch files that have been soft-deleted for more than specific duration (e.g., 7 days)
select file_id, storage_key, thumbnail_key
from files
    where is_deleted = true 
        and deleted_at < now()
    limit $1;

-- name: HardDeleteFiles :exec
-- Permanently remove file records
delete from files where file_id = any(sqlc.arg(file_ids)::uuid[]);
