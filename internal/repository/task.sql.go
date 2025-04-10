// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: task.sql

package repository

import (
	"context"
	sql "database/sql"
)

const createTask = `-- name: CreateTask :execresult
INSERT INTO tasks (
    task_id, status
) VALUES (
 ?, ?
)
`

type CreateTaskParams struct {
	TaskID string
	Status string
}

func (q *Queries) CreateTask(ctx context.Context, arg CreateTaskParams) (sql.Result, error) {
	return q.db.ExecContext(ctx, createTask, arg.TaskID, arg.Status)
}

const getTask = `-- name: GetTask :one
SELECT id, task_id, status, result, created_at, updated_at
FROM tasks
WHERE task_id = ?
`

func (q *Queries) GetTask(ctx context.Context, taskID string) (Task, error) {
	row := q.db.QueryRowContext(ctx, getTask, taskID)
	var i Task
	err := row.Scan(
		&i.ID,
		&i.TaskID,
		&i.Status,
		&i.Result,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const updateTaskResult = `-- name: UpdateTaskResult :exec
UPDATE tasks 
SET status = ?, result = ? WHERE task_id = ?
`

type UpdateTaskResultParams struct {
	Status string
	Result sql.NullString
	TaskID string
}

func (q *Queries) UpdateTaskResult(ctx context.Context, arg UpdateTaskResultParams) error {
	_, err := q.db.ExecContext(ctx, updateTaskResult, arg.Status, arg.Result, arg.TaskID)
	return err
}

const updateTaskStatus = `-- name: UpdateTaskStatus :execresult
UPDATE tasks 
SET status = ? WHERE task_id = ?
`

type UpdateTaskStatusParams struct {
	Status string
	TaskID string
}

func (q *Queries) UpdateTaskStatus(ctx context.Context, arg UpdateTaskStatusParams) (sql.Result, error) {
	return q.db.ExecContext(ctx, updateTaskStatus, arg.Status, arg.TaskID)
}
