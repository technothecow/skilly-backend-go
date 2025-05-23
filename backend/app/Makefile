.PHONY: gen run test

default: run

gen:
	@echo "==> Running go generate..."
	@go generate ./...
	@echo "==> Go generate finished."

run:
	docker-compose -f docker/docker-compose.dev.yml up --build

test:
	go test skilly/tests -v