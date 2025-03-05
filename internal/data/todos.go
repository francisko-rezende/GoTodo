package data

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Todo struct {
	ID          int64     `json:"id"`
	CreatedAt   time.Time `json:"-"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	DueDate     time.Time `json:"due_date"`
	IsCompleted bool      `json:"is_completed"`
}

type TodosModel struct {
	DB *pgxpool.Pool
}

func (td *TodosModel) Insert(todo *Todo) error {
	query := `
	INSERT INTO todos (title, description, due_date, is_completed)
	VALUES ($1, $2, $3, $4)
	RETURNING id, created_at
	`

	args := []any{todo.Title, todo.Description, todo.DueDate, todo.IsCompleted}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return td.DB.QueryRow(ctx, query, args...).Scan(&todo.ID, &todo.CreatedAt)
}
