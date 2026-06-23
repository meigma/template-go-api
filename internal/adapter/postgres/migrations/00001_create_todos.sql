-- +goose Up
CREATE TABLE todos (
    id           uuid        PRIMARY KEY,
    title        text        NOT NULL,
    status       text        NOT NULL DEFAULT 'open'
                             CHECK (status IN ('open', 'completed')),
    created_at   timestamptz NOT NULL,
    completed_at timestamptz
);

-- +goose Down
DROP TABLE todos;
