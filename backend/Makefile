.PHONY: run build test vet tidy fmt check

run:      ## Jalankan server (baca .env)
	go run ./cmd/api

build:    ## Build binary ke bin/api
	go build -o bin/api ./cmd/api

test:     ## Jalankan unit test
	go test ./... -count=1

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
