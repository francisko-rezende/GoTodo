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

## db/migrations/goto migration=$1: go to a specific migration
.PHONY: db/migration/goto
db/migration/goto: confirm

	@echo 'Going to migration version...'
	migrate -path ./migrations -database $(DB_DSN) goto ${migration} 

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

