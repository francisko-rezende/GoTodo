package data

import (
	"GoTodo/internal/data/validator"
	"context"
	"database/sql"
	"errors"
	"fmt"
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

func (t *TodosModel) Insert(todo *Todo) error {
	query := `
	INSERT INTO todos (title, description, due_date, is_completed)
	VALUES ($1, $2, $3, $4)
	RETURNING id, created_at
	`

	args := []any{todo.Title, todo.Description, todo.DueDate, todo.IsCompleted}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return t.DB.QueryRow(ctx, query, args...).Scan(&todo.ID, &todo.CreatedAt)
}

func (t *TodosModel) Get(id int64) (*Todo, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	query := `
	SELECT id, created_at, title, description, due_date, is_completed
	FROM todos
	where id = $1`

	var todo Todo

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := t.DB.QueryRow(ctx, query, id).Scan(&todo.ID, &todo.CreatedAt, &todo.Title, &todo.Description, &todo.DueDate, &todo.IsCompleted)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &todo, nil
}

func (t *TodosModel) GetAll(search string, filters Filters) ([]*Todo, Metadata, error) {
	countQuery := `
        SELECT count(*)
        FROM todos
        WHERE (
            to_tsvector('simple', title) @@ plainto_tsquery('simple', $1) OR
            to_tsvector('simple', description) @@ plainto_tsquery('simple', $1) OR
            $1 = ''
        )`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var totalRecords int
	err := t.DB.QueryRow(ctx, countQuery, search).Scan(&totalRecords)
	if err != nil {
		return nil, Metadata{}, err
	}

	todosQuery := fmt.Sprintf(`
        SELECT id, created_at, title, description, due_date, is_completed
        FROM todos
        WHERE (
            to_tsvector('simple', title) @@ plainto_tsquery('simple', $1) OR
            to_tsvector('simple', description) @@ plainto_tsquery('simple', $1) OR
            $1 = ''
        )
        ORDER BY %s %s, id ASC
        LIMIT $2 OFFSET $3
    `, filters.sortColumn(), filters.sortDirection())

	rows, err := t.DB.Query(ctx, todosQuery, search, filters.limit(), filters.offset())
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	todos := []*Todo{}
	for rows.Next() {
		var todo Todo
		err := rows.Scan(
			&todo.ID,
			&todo.CreatedAt,
			&todo.Title,
			&todo.Description,
			&todo.DueDate,
			&todo.IsCompleted,
		)
		if err != nil {
			return nil, Metadata{}, err
		}
		todos = append(todos, &todo)
	}

	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)
	return todos, metadata, nil
}

func (t *TodosModel) Delete(id int64) error {
	if id < 1 {
		return ErrRecordNotFound
	}

	query := `
	DELETE FROM todos
	WHERE id = $1
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := t.DB.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected := result.RowsAffected()

	if rowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}

func (t *TodosModel) Update(todo *Todo) error {
	query := `
	UPDATE todos
	SET title = $1, description = $2, due_date = $3, is_completed = $4
	WHERE id = $5
	RETURNING id
	`

	args := []any{
		todo.Title,
		todo.Description,
		todo.DueDate,
		todo.IsCompleted,
		todo.ID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := t.DB.QueryRow(ctx, query, args...).Scan(&todo.ID)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		}
		return err
	}

	return nil
}

func ValidateTodo(v *validator.Validator, todo *Todo) {
	v.Check(todo.Title != "", "title", "must be provided")
	v.Check(len(todo.Title) <= 500, "title", "must not have more than 500 characters long")
}
