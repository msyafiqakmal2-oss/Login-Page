.PHONY: dev rust go up test
rust:   ; cd hasher && cargo run --release
go:     ; cd gateway && HASHER_URL=http://localhost:8081 go run .
dev:    ; cd gateway && go run .      # tanpa Rust (hash fallback dev)
up:     ; docker compose up --build
test:   ; cd gateway && go vet ./... && go test ./...
