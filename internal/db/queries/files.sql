-- name: CreateFile :one
insert into files (user_id, filename, storage_key, mime_type, size_bytes, checksum)
    values($1, $2, $3, $4, $5, $6)
returning *;

-- name: GetFileInfo :one
-- Retrieve metadata of a file from the database.
select
    file_id,
    user_id as owner_id,
    filename,
    storage_key,
    mime_type,
    size_bytes,
    visibility,
    thumbnail_key,
    checksum,
    tags,
    version
from files
    where is_deleted = false
        and file_id = $1;

-- name: GetFileOwner :one
select
    u.user_id,
    u.last_name,
    u.first_name,
    u.email
from users u
    join files f
        on u.user_id = f.user_id
        and f.file_id = $1;

-- name: ListFiles :many
select
    u.user_id,
    u.last_name,
    u.first_name,
    u.email,
    f.file_id,
    f.filename,
    f.storage_key,
    f.mime_type,
    f.size_bytes,
    f.visibility,
    f.thumbnail_key,
    f.checksum,
    f.tags,
    f.version
from files f
    join users u
        on f.user_id = u.user_id;

-- name: UpdateFileName :exec
update files
    set
        filename = $1,
        version = version + 1,
        updated_at = now()
where file_id = $2
    and version = $3;

-- name: SetFileVisibility :one
update files
    set
        visibility = $1,
        version = version + 1
where file_id = $2
    and version = $3
    returning visibility;

-- name: DeleteFile :exec
-- Sets file is deleted tag to true and adds a the specified date for a background task to delete it.
update files
    set
        is_deleted = true,
        deleted_at = $1,
        version = version + 1
where file_id = $2
    and version = $3;
