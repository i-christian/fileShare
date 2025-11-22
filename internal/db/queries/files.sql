-- name: GetFileByChecksum :one
-- GetFileByChecksum function returns an existing file storage key to avoid file duplications
select
    count(checksum),
    storage_key
from files
    where checksum = $1
        and user_id = $2
        and is_deleted = false
    group by storage_key;

-- name: CreateFile :one
insert into files (user_id, filename, storage_key, mime_type, size_bytes, checksum)
    values($1, $2, $3, $4, $5, $6)
returning file_id, filename, mime_type, size_bytes, created_at, visibility, checksum, version;

-- name: GetFileInfo :one
-- Retrieve metadata of a file from the database.
select
    file_id,
    user_id as owner_id,
    filename,
    mime_type,
    storage_key,
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

-- name: ListPublicFiles :many
select
    u.user_id as owner_id,
    u.last_name,
    u.first_name,
    f.file_id,
    f.filename,
    f.mime_type,
    f.size_bytes,
    f.thumbnail_key,
    f.checksum,
    f.tags,
    f.version
from files f
    join users u
        on f.user_id = u.user_id
    where f.visibility = 'public'
        and f.is_deleted = false
    order by f.created_at desc
    limit $1 offset $2;

-- name: CountPublicFiles :one
select
    count(*)
from files
    where visibility = 'public'
        and is_deleted = false;

-- name: ListUserFiles :many
select 
    f.file_id, f.filename, f.mime_type, f.size_bytes, f.visibility, f.created_at, f.tags
from files f
    where f.user_id = $1
        and f.is_deleted = false
    order by f.created_at desc
    limit $2 offset $3;

-- name: CountUserFiles :one
select count(*) from files
    where user_id = $1 and is_deleted = false;

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
