.PHONY: dev db db-stop seed api web test

# Start everything: MongoDB (Docker), Go API with hot reload, Next.js.
dev: db
	$(MAKE) -j2 api web

# Local MongoDB in Docker (data persists in the mongo-data volume).
db:
	docker compose up -d mongo

db-stop:
	docker compose down

# Reset the local database with sample data. Refuses non-local hosts.
seed: db
	cd api && go run ./cmd/seed

api:
	cd api && go tool air

web:
	cd web && npm run dev

test:
	cd api && go test ./...
