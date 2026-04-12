.PHONY: up down build logs ps shell-db shell-backend

## up: build images and start all services (dev mode with override)
up:
	docker compose up --build

## up-prod: start without the dev override (production-like)
up-prod:
	docker compose -f docker-compose.yml up --build -d

## down: stop and remove containers (keeps volumes)
down:
	docker compose down

## down-volumes: stop containers AND delete all data volumes
down-volumes:
	docker compose down -v

## build: build the backend image without starting
build:
	docker compose build backend

## logs: tail all service logs
logs:
	docker compose logs -f

## ps: show running containers
ps:
	docker compose ps

## shell-db: open a psql shell in the db container
shell-db:
	docker compose exec db psql -U sptraffic -d sptraffic

## shell-backend: open a shell in the backend container
shell-backend:
	docker compose exec backend sh
