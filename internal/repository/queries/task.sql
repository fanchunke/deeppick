-- name: CreateTask :execresult
INSERT INTO tasks (
    task_id, status
) VALUES (
 ?, ?
);

-- name: GetTask :one
SELECT *
FROM tasks
WHERE task_id = ?;

-- name: UpdateTaskStatus :execresult
UPDATE tasks 
SET status = ? WHERE task_id = ?;

-- name: UpdateTaskResult :execresult
UPDATE tasks 
SET status = ? AND result = ? WHERE task_id = ?;