package main

import (
	"GoTodo/internal/data"
	"GoTodo/internal/data/validator"
	"fmt"
	"net/http"
	"time"
)

func (app *application) createTodoHandler(w http.ResponseWriter, r *http.Request) {
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

func (app *application) listTodosHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Search string
		data.Filters
	}

	qs := r.URL.Query()

	input.Search = app.readString(qs, "search", "")

	v := validator.New()

	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 10, v)
	input.Filters.Sort = app.readString(qs, "sort", "created_at")
	input.Filters.Order = app.readString(qs, "order", "desc")
	input.Filters.SortSafeList = []string{"is_complete", "due_date", "created_at"}
	input.Filters.OrderSafeList = []string{"asc", "desc"}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	todos, metadata, err := app.models.Todos.GetAll(input.Search, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"todos": todos, "metada": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
