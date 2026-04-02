APP_NAME := go-mail-form
PORT ?= 3000

.PHONY: build run clean docker test

build:
	CGO_ENABLED=0 go build -o $(APP_NAME) .

test:
	go test ./...

run: build
	PORT=$(PORT) ./$(APP_NAME)

clean:
	rm -f $(APP_NAME)

docker:
	docker build -t $(APP_NAME) .
