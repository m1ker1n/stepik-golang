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

.PHONY: cover
cover: 
	@echo "-- не забудьте поставить mockgen (см https://github.com/golang/mock)"
	@echo "-- находять в папке с файлом handlers_test.go"
	@echo "-- запускаем генерацию мока"
	mockgen -source=handlers.go -destination=handlers_mock.go -package=main Storage
	@echo "-- считаем тестовое покрытие для хендлера и строим отчет"
	go test -v -run Handler -coverprofile=handler.out
	go tool cover -html=handler.out -o coverage_handler.html
	rm handler.out
	@echo "-- считаем тестовое покрытие для стораджа и строим отчет"
	go test -v -run Storage -coverprofile=db_st.out
	go tool cover -html=db_st.out -o coverage_storage.html
	rm db_st.out