.PHONY: run build test clean migrate docker

run:
	go run cmd/api/main.go

build:
	go build -o bin/api cmd/api/main.go

test:
	go test -v ./...

clean:
	rm -rf bin/
	rm -rf tmp/

docker:
	docker-compose up --build

migrate-up:
	migrate -path migrations -database "postgres://admin:secret@localhost:5432/promt_market?sslmode=disable" up

migrate-down:
	migrate -path migrations -database "postgres://admin:secret@localhost:5432/promt_market?sslmode=disable" down
