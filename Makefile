run:
	go run ./cmd/server

migrate-up:
	# Requires golang-migrate installed (https://github.com/golang-migrate/migrate)
	migrate -database "$(POSTGRES_URL)" -path migrations up

migrate-down:
	migrate -database "$(POSTGRES_URL)" -path migrations down

tidy:
	go mod tidy

test:
	go test ./...
