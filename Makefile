VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build web go test dev-api dev-web clean

build: web go

web:
	cd web && npm ci && npm run build

go:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/tai ./cmd/tai

test:
	go vet ./...
	go test ./...
	cd web && npm run typecheck

# Development: run both in separate terminals, then open http://localhost:5173
dev-api:
	TAI_PASSWORD=$${TAI_PASSWORD:-devpassword} TAI_DB_PATH=$${TAI_DB_PATH:-tai.db} go run ./cmd/tai serve

dev-web:
	cd web && npm run dev

clean:
	rm -rf bin web/dist/assets web/dist/index.html
