-- name: UpsertTodo :exec
-- Insert-or-replace, honoring the todo.Repository.Save upsert contract.
INSERT INTO todos (id, title, status, created_at, completed_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (id) DO UPDATE
  SET title = EXCLUDED.title,
      status = EXCLUDED.status,
      completed_at = EXCLUDED.completed_at;

-- name: GetTodo :one
SELECT id, title, status, created_at, completed_at FROM todos WHERE id = $1;

-- name: ListTodos :many
SELECT id, title, status, created_at, completed_at FROM todos ORDER BY created_at;

-- Example (commented): optional status filter via sqlc.narg, illustrating the
-- dynamic-query pattern without shipping a query-builder dependency. A NULL
-- argument disables the filter; a non-NULL value restricts to that status.
-- Uncomment and regenerate to use it.
--
-- -- name: ListTodosByStatus :many
-- SELECT id, title, status, created_at, completed_at FROM todos
-- WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))
-- ORDER BY created_at;
