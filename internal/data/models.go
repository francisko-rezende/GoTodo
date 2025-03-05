package data

import "github.com/jackc/pgx/v5/pgxpool"

type Models struct {
	Todos TodosModel
}

func NewModels(db *pgxpool.Pool) Models {
	return Models{
		Todos: TodosModel{DB: db},
	}
}
