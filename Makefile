.PHONY: dev test migrate build

build:
	go build ./...

test:
	go test ./...

dev:
	docker compose -f deploy/docker-compose.yml up --build

migrate:
	goose -dir migrations postgres "$(CONVKIT_DSN)" up
