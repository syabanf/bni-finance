.PHONY: run build test test-integration vet tidy fmt check

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
