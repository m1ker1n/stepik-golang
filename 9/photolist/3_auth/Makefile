COMMIT?=$(shell git rev-parse --short HEAD)
BUILD_TIME?=$(shell date -u '+%Y-%m-%d_%H:%M:%S')

all: run

run: 
	@echo "-- starting app"
	go run .

.PHONY: dc
dc: 
	@echo "-- starting docker compose"
	docker-compose up
