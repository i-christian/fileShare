-- +goose Up
create type cleanup_counts as (
    refresh_tokens_deleted int,
    action_tokens_deleted int,
    api_keys_deleted int
);

-- +goose StatementBegin
create or replace function run_all_cleanups()
returns cleanup_counts as $$
declare
    v_refresh_tokens_deleted int;
    v_action_tokens_deleted int;
    v_api_keys_deleted int;
begin
    -- name: DeleteExpiredRefreshTokens :exec
    delete from refresh_tokens where expires_at < now();
    get diagnostics v_refresh_tokens_deleted = row_count;

    -- name: DeleteExpiredActionTokens :exec
    delete from action_tokens where expires_at < now();
    get diagnostics v_action_tokens_deleted = row_count;

    -- name: DeleteExpiredAPIKeys :exec
    delete from api_keys where expires_at < now();
    get diagnostics v_api_keys_deleted = row_count;

    return row(
        v_refresh_tokens_deleted,
        v_action_tokens_deleted,
        v_api_keys_deleted
    )::cleanup_counts;
end;
$$ language plpgsql;
-- +goose StatementEnd

-- +goose Down
drop function if exists run_all_cleanups();
drop type if exists cleanup_counts;
