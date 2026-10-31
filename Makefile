test:
	go test -short ./...

coverage:
	go test -short -covermode=atomic -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

vet:
	go vet ./...

build:
	go build ./...

image:
	docker build -t urtube:local .

# Same linter and version as CI.
lint:
	docker run --rm -v "$(CURDIR)":/app -w /app -v urtube-golangci-cache:/root/.cache \
		golangci/golangci-lint:v2.15.0 golangci-lint run ./...
