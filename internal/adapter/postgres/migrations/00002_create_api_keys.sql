-- +goose Up
CREATE TABLE api_keys (
    key        text   PRIMARY KEY,
    subject    text   NOT NULL,
    roles      text[] NOT NULL DEFAULT '{}'
);

-- +goose Down
DROP TABLE api_keys;
