build:
	@go build -o bin/sshtunnel

run: build
	@./bin/sshtunnel

test: 
	@go test -v ./...