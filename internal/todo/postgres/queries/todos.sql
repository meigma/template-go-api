-- name: UpsertTodo :exec
-- Insert-or-replace by primary key.
INSERT INTO todos (id, title, status, created_at, completed_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (id) DO UPDATE
  SET title = EXCLUDED.title,
      status = EXCLUDED.status,
      created_at = EXCLUDED.created_at,
      completed_at = EXCLUDED.completed_at;

-- name: GetTodo :one
SELECT id, title, status, created_at, completed_at FROM todos WHERE id = $1;

-- name: ListTodos :many
SELECT id, title, status, created_at, completed_at FROM todos ORDER BY created_at;

-- Optional status filter via sqlc.narg: a NULL argument disables the filter, a
-- non-NULL value restricts to that status. Uncomment and regenerate to use.
--
-- -- name: ListTodosByStatus :many
-- SELECT id, title, status, created_at, completed_at FROM todos
-- WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))
-- ORDER BY created_at;
