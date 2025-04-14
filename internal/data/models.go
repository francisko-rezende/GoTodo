package data

import (
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Models struct {
	Todos TodosModel
}

var ErrRecordNotFound = errors.New("record not found")

func NewModels(db *pgxpool.Pool) Models {
	return Models{
		Todos: TodosModel{DB: db},
	}
}
