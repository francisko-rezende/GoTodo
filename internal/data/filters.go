package data

import (
	"GoTodo/internal/data/validator"
	"fmt"
)

type Metadata struct {
	CurrentPage  int `json:"current_page,omitempty"`
	PageSize     int `json:"page_size,omitempty"`
	FirstPage    int `json:"first_page,omitempty"`
	LastPage     int `json:"last_page"`
	TotalRecords int `json:"total_records"`
}

type Filters struct {
	Page          int
	PageSize      int
	Sort          string
	Order         string
	SortSafeList  []string
	OrderSafeList []string
}

func calculateMetadata(totalRecords, page, pageSize int) Metadata {
	return Metadata{
		CurrentPage:  page,
		PageSize:     pageSize,
		FirstPage:    1,
		LastPage:     (totalRecords + pageSize - 1) / pageSize,
		TotalRecords: totalRecords,
	}
}

func (f *Filters) sortColumn() string {
	for _, safeValue := range f.SortSafeList {
		if f.Sort == safeValue {
			return f.Sort
		}
	}

	panic("unsafe sort parameter: " + f.Sort)
}

func (f *Filters) sortDirection() string {
	if f.Order == "asc" {
		return "ASC"
	}

	return "DESC"
}

func (f *Filters) limit() int {
	return f.PageSize
}

func (f *Filters) offset() int {
	return (f.Page - 1) * f.PageSize
}

func ValidateFilters(v *validator.Validator, f Filters) {
	v.Check(f.Page > 0, "page", "must be greater than 0")
	v.Check(f.Page <= 10_000_000, "page", "must be less than ten million")
	v.Check(f.PageSize > 0, "page_size", "must be greater than 0")
	v.Check(f.PageSize <= 100, "page_size", "must be less than a hundred")

	v.Check(validator.PermittedValue(f.Sort, f.SortSafeList...), "sort", fmt.Sprintf(`"%v" is an invalid sort value, use one of the following: %v`, f.Sort, f.SortSafeList))
	v.Check(validator.PermittedValue(f.Order, f.OrderSafeList...), "order", fmt.Sprintf(`"%v" is an invalid order value, use one of the following: %v`, f.Order, f.OrderSafeList))
}
