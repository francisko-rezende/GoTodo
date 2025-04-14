# GoTodo

## Setup

- create project using `git mod init`
- setup basic folder structure (check let's go further book)
- setup db
  - create postgres container using docker compose
  - enter the container using `docker exec -it <container-name> bash`
  - start postgres using `psql -U postgres`
  - create db for project using `CREATE DATABASE <db-name>;`
  - connect to the database using `\c <db-name>;`
  - create role for project using `create role <role-name> with login password '<password>'` (for dev, I'm using gotodo and Batata-d0ce)
  - create case insensitive text extension using `CREATE EXTENSION IF NOT EXISTS citext`
  - set db owner to new user `alter database <db-name> owner to <user-name>`
- install db driver using `go get github.com/jackc/pgx/v5/pgxpool`
- install dot env lib using `go get github.com/joho/godotenv`
- consume db dsn from .env like so

```golang
package main

import (
 "fmt"
 "log"
 "os"

 "github.com/joho/godotenv"
)

func main() {
 err := godotenv.Load()
 if err != nil {
  log.Fatal("error loading .env file")
  os.Exit(1)
 }

 dsn := os.Getenv("DB_DSN")

 fmt.Println("Hello, GoTodo")
}
```

- setup configuration struct, app struct with logger and command line flags

```golang

package main

import (
 "context"
 "flag"
 "fmt"
 "log"
 "log/slog"
 "os"
 "time"

 "github.com/jackc/pgx/v5/pgxpool"
 "github.com/joho/godotenv"
)

const version = "1.0.0"


// here's the config
type config struct {
 port int
 env  string
 db   struct {
  dsn             string
  maxOpenConns    int
  minConns        int
  maxConnIdleTime time.Duration
 }
}

// and the app with the logger
type application struct {
 config config
 logger *slog.Logger
}

func main() {
  logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
 err := godotenv.Load()
 if err != nil {
  logger.Error("error loading .env file")
  os.Exit(1)
 }

 dsn := os.Getenv("DB_DSN")

 if dsn == "" {
  logger.Error("required DB_DSN env var missing")
  os.Exit(1)
 }

  // here the command vars parsing starts happening
 var cfg config

 flag.StringVar(&cfg.db.dsn, "db-dsn", dsn, "PostgreSQL DSN")
 flag.IntVar(&cfg.port, "port", 4000, "API server port")
 flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")
 flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
 flag.IntVar(&cfg.db.minConns, "db-max-min-conns", 6, "PostgreSQL min connections")
 flag.DurationVar(&cfg.db.maxConnIdleTime, "db-max-idle-time", 15*time.Minute, "PostgreSQL max connection idle time")

 ...
}

```

- setup connection pool like so

```golang
// create this function
func openDB(cfg config) (*pgxpool.Pool, error) {
 ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
 defer cancel()

 poolConfig, err := pgxpool.ParseConfig(cfg.db.dsn)
 if err != nil {
  return nil, err
 }

 poolConfig.MaxConns = int32(cfg.db.maxOpenConns)
 poolConfig.MinConns = int32(cfg.db.minConns) // use ~25% of MaxConns
 poolConfig.MaxConnIdleTime = cfg.db.maxConnIdleTime

 connectionPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
 if err != nil {
  return nil, err
 }

 err = connectionPool.Ping(ctx)
 if err != nil {
  connectionPool.Close()
  return nil, err
 }

 return connectionPool, nil
}

// and use like so

func main() {
  ...
 db, err := openDB(cfg)
 if err != nil {
  logger.Error("failed to open db connection pool")
  os.Exit(1)
 }

 defer db.Close()

 logger.Info("db connection established")
  ...
}
```

- create makefile with this content

```bash
include ./.env

# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
 @echo 'Usage:'
 @sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

.PHONY: confirm
confirm:
 @echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## run/api: run the cmd/api application
.PHONY: run/api
run/api:
 go run ./cmd/api

## db/psql: connect to the database using psql
.PHONY: db/psql
db/psql:
 psql $(DB_DSN)

## db/migrations/new name=$1: create a new database migration
.PHONY: db/migration/new
db/migration/new:
 @echo 'Creating migration files for ${name}'
 migrate create -seq -ext=.sql -dir=./migrations ${name}

## db/migrations/up: apply all up database migrations
.PHONY: db/migration/up
db/migration/up: confirm

 @echo 'Running up migrations...'
 migrate -path ./migrations -database $(DB_DSN) up

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## tidy: format all .go files and tidy module dependencies

.PHONY: tidy
tidy:
 @echo 'Formatting .go files...'
 -go fmt ./...
 @echo 'Tidying module dependencies...'
 go mod tidy
 @echo 'Verifying and vendoring module dependencies...'
 go mod verify
 go mod vendor

## audit: run quality control checks
.PHONY: audit
audit:
 @echo 'Checking module dependencies'
 go mod tidy -diff
 go mod verify
 @echo 'Vetting code...'
 go vet ./...
 staticcheck ./...
 @echo 'Running tests...'
 go test -race -vet=off ./...

# ==================================================================================== #
# BUILD
# ==================================================================================== #

## build/api: build the cmd/api application
.PHONY: build/api
build/api:
 @echo 'Building cmd/api...'
 go build -ldflags='-s' -o=./bin/api ./cmd/api
 GOOS=linux GOARCH=amd64 go build -ldflags='-s' -o=./bin/linux_amd64/api ./cmd/api
 GOOS=linux GOARCH=arm64 go build -ldflags='-s' -o=./bin/linux_arm64/api ./cmd/api

```

## Basic http server

- create a routes.go file using a router (I used httprouter) like so

```golang
package main

import (
 "net/http"

 "github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
 router := httprouter.New()

 router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)

 return router
}
```

- create a server that takes the routes we just created in its Handler

```golang
package main

import (
 "fmt"
 "net/http"
)

func (app *application) serve() error {
 srv := &http.Server{
  Addr:    fmt.Sprintf(":%d", app.config.port),
  Handler: app.routes(),
 }

 app.logger.Info("starting server", "addr", srv.Addr, "env", app.config.env)

 err := srv.ListenAndServe()
 if err != nil {
  return err
 }

 return nil
}
```

- on api's main file, use the serve function

```golang
 err = app.serve()
 if err != nil {
  logger.Error(err.Error())
  os.Exit(1)
 }
```

## Todo management

### writeJSON

- create a helper to write json to responses like so

```golang
package main

import (
 "encoding/json"
 "net/http"
)

func (app *application) writeJSON(w http.ResponseWriter, status int, data any, headers http.Header) error {
 // convert/marshal the data into json
  // we can also use MarshalIndent(data, "", "\t") to indent json and make them more readable
  js, err := json.Marshal(data)
 if err != nil {
  return err
 }

  // add a new line at the end to make the xp in commandline better
 js = append(js, '\n')

  // add headers received by writeJSON to response
 for key, value := range headers {
  w.Header()[key] = value
 }

  // set application/json header, status and add json to response
 w.Header().Set("Content-Type", "application/json")
 w.WriteHeader(status)
 w.Write(js)

 return nil
}
```

- `writeJSON` is used like so

```golang
package main

import (
 "net/http"
)

func (app *application) healthcheckHandler(w http.ResponseWriter, r *http.Request) {
 // create a map that will be send as json
  data := map[string]string{
  "status":      "available",
  "environment": app.config.env,
  "version":     version,
 }

  // pass the map along with the other args to writeJSON
 err := app.writeJSON(w, http.StatusOK, data, nil)
 if err != nil {
  app.logger.Error(err.Error())
  http.Error(w, "The server encountered a problem and could not process your request", http.StatusInternalServerError)
 }
}
```

- we can also envelope the responses using this type `type envelop = map[string]any`. Then the data would have this type instead of any in writeJSON

### Error responses

- centralize errors like so

```golang
package main

import (
 "fmt"
 "net/http"
)

func (app *application) logError(r *http.Request, err error) {
 var (
  method = r.Method
  uri    = r.URL.RequestURI()
 )

 app.logger.Error(err.Error(), "method", method, "uri", uri)
}

func (app *application) errorResponse(w http.ResponseWriter, r *http.Request, status int, message any) {
 env := envelope{"error": message}

 err := app.writeJSON(w, status, env, nil)
 if err != nil {
  app.logError(r, err)
  w.WriteHeader(500)
 }
}

func (app *application) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
 app.logError(r, err)

 message := "the server encountered a problem and could not process your request"

 app.errorResponse(w, r, http.StatusInternalServerError, message)
}

func (app *application) notFoundResponse(w http.ResponseWriter, r *http.Request) {
 message := "the requested resource could not be found"
 app.errorResponse(w, r, http.StatusNotFound, message)
}

func (app *application) methodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
 message := fmt.Sprintf("the %s method is not supported for this resource", r.Method)
 app.errorResponse(w, r, http.StatusMethodNotAllowed, message)
}
```

- start using error responses instead of the router's default not found and method not allowed responses

```golang
package main

import (
 "net/http"

 "github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
 router := httprouter.New()
  // these two lines
 router.NotFound = http.HandlerFunc(app.notFoundResponse)
 router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

 router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)

 return router
}

```

### readJSON

- create a helper method to read json value sent in request bodies

```golang
func (app *application) readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
 const ONE_MEGABYTE = 1_048_576

  // limite the max size of payloas to 1mb
 r.Body = http.MaxBytesReader(w, r.Body, ONE_MEGABYTE)
 dec := json.NewDecoder(r.Body)
  // stop json with unknown fields from being processed
 dec.DisallowUnknownFields()

 err := dec.Decode(dst)
 if err != nil {
  var syntaxError *json.SyntaxError
  var unmarshalTypeError *json.UnmarshalTypeError
  var invalidUnmarshalError *json.InvalidUnmarshalError
  var maxBytesError *http.MaxBytesError

  switch {
    // catch json syntax issues
  case errors.As(err, &syntaxError):
   return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)

    // catch json syntax issues
  case errors.Is(err, io.ErrUnexpectedEOF):
   return errors.New("body contains badly-formed JSON")

    // catch fields that have incorrect type
  case errors.As(err, &unmarshalTypeError):
   if unmarshalTypeError.Field != "" {
    return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
   }
   return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)

    // catch empty body error
  case errors.Is(err, io.EOF):
   return errors.New("body must not be empty")

    // catch unknown fields
  case strings.HasPrefix(err.Error(), "json: unknown field"):
   fieldName := strings.TrimPrefix(err.Error(), "json: unkown field")
   return fmt.Errorf("body has unkown key %s", fieldName)

    // catch fields that are too big
  case errors.As(err, &maxBytesError):
   return fmt.Errorf("body must not be larger than %d bytes", maxBytesError.Limit)

    // happens when something that is not a non-nil pointer ends up here
  case errors.As(err, &invalidUnmarshalError):
   panic(err)
  default:
   return err
  }
 }

  // check if there are more than one json values being passed
 err = dec.Decode(&struct{}{})

 if !errors.Is(err, io.EOF) {
  return errors.New("body must only contain a single JSON value")
 }

 return nil
}
```

### validator

- create a validator package like so

```golang
package validator

import (
 "regexp"
)

// email regex, not used right now
var EmailRX = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

// validator type
type Validator struct {
 Errors map[string]string
}

// helper for creating a new validator
func New() *Validator {
 return &Validator{Errors: make(map[string]string)}
}

// method for checking if there are any errors
func (v *Validator) Valid() bool {
 return len(v.Errors) == 0
}

// method for adding an error if it doesnt exist at this point
func (v *Validator) AddError(key, message string) {
 if _, exists := v.Errors[key]; !exists {
  v.Errors[key] = message
 }
}

// method for actually conducting the checks using the validator
func (v *Validator) Check(ok bool, key, message string) {
 if !ok {
  v.AddError(key, message)
 }
}

```

- also create a helper for invalid payload responses

```golang
func (app *application) failedValidationResponse(w http.ResponseWriter, r *http.Request, errors map[string]string) {
 app.errorResponse(w, r, http.StatusUnprocessableEntity, errors)
}
```

### create todo

- create a todo model with the method insert for creating a new todo

```golang
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

```

- create a models file with a helper for initializing the models

```golang

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

```

- hook up the models in main.go

```golang
type application struct {
 config config
 models data.Models //here
 logger *slog.Logger
}

...

app := &application{
  config: cfg,
  models: data.NewModels(db), //and here
  logger: logger,
 }

```

- create a todo model like so with a method to create todos

```golang

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

```

- create a todo handler with a

```golang
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

```

- and use it in the routes

```golang
package main

import (
 "net/http"

 "github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
 router := httprouter.New()

 router.NotFound = http.HandlerFunc(app.notFoundResponse)
 router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

 router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)
 router.HandlerFunc(http.MethodPost, "/v1/todos", app.createTodo) //here

 return router
}

```

### list todos

[] implement filters

- the query params will be processed using a filters helper like so

```golang
package data

type Metadata struct {
 CurrentPage  int `json:"current_page,omitempty"`
 PageSize     int `json:"page_size,omitempty"`
 FirstPage    int `json:"first_page,omitempty"`
 LastPage     int `json:"last_page"` //no omitempty so values return when list is empty
 TotalRecords int `json:"total_records"` //no omitempty so values return when list is empty
}

type Filters struct {
 Page         int
 PageSize     int
 Sort         string
 Order        string
 SortSafeList []string
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
```

[] create methods for reading ints and string from query params

```golang
// helpers.go
func (app *application) readString(qs url.Values, key string, defaultValue string) string {
	s := qs.Get(key)

	if s == "" {
		return defaultValue
	}

	return s
}

func (app *application) readInt(qs url.Values, key string, defaultValue int, v *validator.Validator) int {
	s := qs.Get(key)

	if s == "" {
		return defaultValue
	}

	i, err := strconv.Atoi(s)
	if err != nil {
		v.AddError(key, "must be an integer")
	}

	return i
}
```

[] implement query param reading

```golang
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

// method continues...
}
```

[] validate filters

```golang
	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

//

func ValidateFilters(v *validator.Validator, f Filters) {
	v.Check(f.Page > 0, "page", "must be greater than 0")
	v.Check(f.Page <= 10_000_000, "page", "must be less than ten million")
	v.Check(f.PageSize > 0, "page_size", "must be greater than 0")
	v.Check(f.PageSize <= 100, "page_size", "must be less than a hundred")

	v.Check(validator.PermittedValue(f.Sort, f.SortSafeList...), "sort", fmt.Sprintf(`"%v" is an invalid sort value, use one of the following: %v`, f.Sort, f.SortSafeList))
	v.Check(validator.PermittedValue(f.Order, f.OrderSafeList...), "order", fmt.Sprintf(`"%v" is an invalid order value, use one of the following: %v`, f.Order, f.OrderSafeList))
}
```
[] implement todos model

```golang
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
```

[] use the get all todos method in the handler

```golang

	todos, metadata, err := app.models.Todos.GetAll(input.Search, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"todos": todos, "metada": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
```

- the full handler looks like this

```golang
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
```

### delete todo

- create helper for reading id path params

```golang
func (app *application) readIDParam(r *http.Request) (int64, error) {
	params := httprouter.ParamsFromContext(r.Context())

	id, err := strconv.ParseInt(params.ByName("id"), 10, 64)

	if err != nil || id < 1 {
		return 0, errors.New("invalid id parameter")
	}

	return id, nil
}
```
- create delete model for deleting todos

```golang
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
```

- create a handler for deleting todos

```golang
func (app *application) deleteTodoHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	err = app.models.Todos.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}

		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"message": "todo deleted successfuly"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
```
- add new route for delenting todos

```golang
func (app *application) routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)
	router.HandlerFunc(http.MethodPost, "/v1/todos", app.createTodoHandler)
	router.HandlerFunc(http.MethodGet, "/v1/todos", app.listTodosHandler)
	router.HandlerFunc(http.MethodDelete, "/v1/todos/:id", app.deleteTodoHandler) //here

	return router
}

```