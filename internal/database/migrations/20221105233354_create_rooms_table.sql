-- +goose Up
-- +goose StatementBegin
CREATE TABLE rooms (
    id BIGSERIAL NOT NULL,
    owner CHAR(32) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    archived_at TIMESTAMPTZ,

    UNIQUE (id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE rooms;
-- +goose StatementEnd
