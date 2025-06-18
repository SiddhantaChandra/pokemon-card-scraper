.PHONY: dev build test clean

dev:
	docker-compose -f docker-compose.dev.yml up

build:
	docker-compose build

test:
	go test ./backend/...
	cd frontend && npm test

clean:
	docker-compose down -v
	rm -rf data/ 