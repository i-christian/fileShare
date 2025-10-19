-- +goose Up

-- +goose StatementBegin
-- Function to auto-update "updated_at" timestamps
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose StatementBegin
-- Function to prevent demotion/deletion of the last admin
CREATE OR REPLACE FUNCTION protect_last_admin()
RETURNS TRIGGER AS $$
DECLARE
    admin_count INT;
BEGIN
    IF OLD.role = 'admin' THEN
        SELECT count(*) INTO admin_count FROM users WHERE role = 'admin' AND id != OLD.id;

        IF admin_count = 0 AND (TG_OP = 'UPDATE' AND NEW.role != 'admin') THEN
            RAISE EXCEPTION 'Cannot demote the last admin user.';
        END IF;

        IF admin_count = 0 AND TG_OP = 'DELETE' THEN
             RAISE EXCEPTION 'Cannot delete the last admin user.';
        END IF;
    END IF;

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    ELSE
        RETURN NEW;
    END IF;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- Apply triggers to relevant tables
CREATE TRIGGER trigger_set_updated_at_users
    BEFORE UPDATE ON users
        FOR EACH ROW
            EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trigger_set_updated_at_files
    BEFORE UPDATE ON files
        FOR EACH ROW
            EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trigger_set_updated_at_upload_sessions
    BEFORE UPDATE ON upload_sessions
        FOR EACH ROW
            EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trigger_protect_last_admin
    BEFORE UPDATE OR DELETE ON users
        FOR EACH ROW
            EXECUTE FUNCTION protect_last_admin();


-- +goose Down
-- Drop triggers first
DROP TRIGGER IF EXISTS trigger_protect_last_admin ON users;
DROP TRIGGER IF EXISTS trigger_set_updated_at_upload_sessions ON upload_sessions;
DROP TRIGGER IF EXISTS trigger_set_updated_at_files ON files;
DROP TRIGGER IF EXISTS trigger_set_updated_at_users ON users;

-- Drop functions
DROP FUNCTION IF EXISTS protect_last_admin();
DROP FUNCTION IF EXISTS set_updated_at();

