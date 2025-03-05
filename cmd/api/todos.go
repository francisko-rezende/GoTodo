package main

import (
	"GoTodo/internal/data"
	"GoTodo/internal/data/validator"
	"fmt"
	"net/http"
	"time"
)

func (app *application) createTodo(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title       string    `json:"title"`
		Description string    `json:"description"`
		DueDate     time.Time `json:"due_date"`
		IsCompleted bool      `json:"is_complete"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	todo := &data.Todo{
		Title:       input.Title,
		Description: input.Description,
		DueDate:     input.DueDate,
		IsCompleted: input.IsCompleted,
	}

	v := validator.New()

	v.Check(todo.Title != "", "title", "must be provided")
	v.Check(len([]rune(todo.Title)) <= 500, "title", "must not be more than 500 characters long")

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
	}

	err = app.models.Todos.Insert(todo)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/todos/%d", todo.ID))

	err = app.writeJSON(w, http.StatusCreated, envelope{"todo": todo}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
