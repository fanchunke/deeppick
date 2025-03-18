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

-- name: UpdateTaskResult :exec
UPDATE tasks 
SET status = ?, result = ? WHERE task_id = ?;