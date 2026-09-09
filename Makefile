.PHONY: run build test docker-up docker-down clean

run:
	go run ./cmd/api/main.go

build:
	go build -o bin/api.exe ./cmd/api/main.go

test:
	go test -v -race ./...

docker-up:
	docker compose up -d

docker-down:
	docker compose down

clean:
	rm -rf bin/
