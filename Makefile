.PHONY: build test run lint docker-build docker-run

build:
	go build -ldflags="-s -w" -o bin/disha-server ./cmd/server

test:
	go test ./... -v -count=1

run:
	go run ./cmd/server

lint:
	go vet ./...

docker-build:
	docker build -t disha-backend .

docker-run:
	docker run -p 8080:8080 --env-file .env disha-backend
