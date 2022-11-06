-- +goose Up
-- +goose StatementBegin
CREATE TABLE messages (
   id BIGSERIAL NOT NULL,
   room_id BIGINT NOT NULL,
   sender CHAR(32) NOT NULL,
   timestamp TIMESTAMP NOT NULL,
   user_type SMALLINT NOT NULL,
   message TEXT NOT NULL,

   UNIQUE (id),
   CONSTRAINT fk_rooms_room_id FOREIGN KEY (room_id) REFERENCES rooms (id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE messages;
-- +goose StatementEnd
