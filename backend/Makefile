.PHONY: run build test test-integration vet tidy fmt check db-create db-schema db-seed db-reset db-drop

# --- aplikasi ---------------------------------------------------------------

run:      ## Jalankan server (baca .env)
	go run ./cmd/api

build:    ## Build binary ke bin/api
	go build -o bin/api ./cmd/api

test:     ## Jalankan unit test (integration test dilewati)
	go test ./... -count=1

## Jalankan integration test terhadap Postgres sungguhan. WAJIB database sekali
## pakai — test ini me-TRUNCATE tabel, dan menolak nama yang tidak mengandung
## 'test'/'dev'.
##   make test-integration TEST_DATABASE_URL=postgres://…/bni_finance_dev
test-integration:
	@test -n "$(TEST_DATABASE_URL)" || (echo "set TEST_DATABASE_URL dulu" && exit 1)
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test ./internal/api/ -run TestLive -v -count=1

vet:
	go vet ./...

fmt:
	gofmt -w .

tidy:
	go mod tidy

check: vet test  ## Gate sebelum commit

stress:      ## Stress test lapisan API (throughput + latensi)
	go test ./internal/api/ -run TestStress -v -count=1

stress-race: ## Stress test dengan deteksi data race
	go test ./internal/api/ -run 'TestStress|TestConcurrentSettle' -race -count=1

# --- database ---------------------------------------------------------------
#
# Postgres lokal, tanpa Supabase. PSQL bisa ditimpa bila binary-nya tidak ada di
# PATH, misalnya pada instalasi Homebrew yang tidak di-link:
#   make db-reset PSQL=/opt/homebrew/opt/postgresql@17/bin/psql

DB_NAME ?= bni_finance_dev
DB_HOST ?= localhost
DB_PORT ?= 5432
DB_USER ?= postgres
PSQL    ?= psql
DB_URL  ?= postgresql://$(DB_USER)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)

db-create: ## Buat database bila belum ada
	@$(PSQL) "postgresql://$(DB_USER)@$(DB_HOST):$(DB_PORT)/postgres" -tAc \
		"SELECT 1 FROM pg_database WHERE datname='$(DB_NAME)'" | grep -q 1 \
		|| $(PSQL) "postgresql://$(DB_USER)@$(DB_HOST):$(DB_PORT)/postgres" -c "CREATE DATABASE $(DB_NAME)"
	@echo "database $(DB_NAME) siap"

db-schema: ## Terapkan skema (idempoten)
	$(PSQL) "$(DB_URL)" -v ON_ERROR_STOP=1 -q -f ../db/schema.sql
	@echo "skema diterapkan"

db-seed:  ## Isi data contoh untuk pengembangan
	$(PSQL) "$(DB_URL)" -v ON_ERROR_STOP=1 -q -f ../db/seed.sql
	@echo "data contoh dimuat"

db-reset: db-create db-schema db-seed ## Siapkan database lokal dari nol

db-drop:  ## Hapus database — menghapus SEMUA data di dalamnya
	@printf 'Hapus database %s beserta seluruh isinya? [y/N] ' "$(DB_NAME)"; \
	read answer; [ "$$answer" = "y" ] || { echo "dibatalkan"; exit 1; }
	$(PSQL) "postgresql://$(DB_USER)@$(DB_HOST):$(DB_PORT)/postgres" -c "DROP DATABASE IF EXISTS $(DB_NAME)"
