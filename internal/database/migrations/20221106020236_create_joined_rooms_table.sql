-- +goose Up
-- +goose StatementBegin
CREATE TABLE joined_rooms (
   id BIGSERIAL NOT NULL,
   user_id CHAR(32) NOT NULL,
   room_id BIGINT NOT NULL,

   UNIQUE (id),
   UNIQUE (user_id, room_id),
   CONSTRAINT fk_rooms_room_id FOREIGN KEY (room_id) REFERENCES rooms (id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE joined_rooms;
-- +goose StatementEnd
