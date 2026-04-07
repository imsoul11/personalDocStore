SHELL := /bin/bash

.PHONY: help docker-up docker-down docker-logs docker-ps server worker

help:
	@echo "docStore shortcuts"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-up       Start postgres + rabbitmq"
	@echo "  make docker-down     Stop containers (keeps volume)"
	@echo "  make docker-ps       Show container status"
	@echo "  make docker-logs     Tail logs"
	@echo ""
	@echo "Go:"
	@echo "  make server          Run API server"
	@echo "  make worker          Run worker"

docker-up:
	docker compose -f docker-compose.yml up -d

docker-down:
	docker compose -f docker-compose.yml down

docker-ps:
	docker compose -f docker-compose.yml ps

docker-logs:
	docker compose -f docker-compose.yml logs -f --tail=200

server:
	HOST=127.0.0.1 PORT=8080 go run ./cmd/docstore-server/main.go

worker:
	go run ./cmd/docstore-worker/main.go

