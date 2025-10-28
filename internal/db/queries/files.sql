-- name: CreateFile :one
insert into files (user_id, filename, storage_key, mime_type, size_bytes, checksum)
    values($1, $2, $3, $4, $5, $6)
returning *;


