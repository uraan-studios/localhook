build:
	@go build -o bin/sshtunnel

run: build
	@./bin/sshtunnel

deploy: build
	@./bin/sshtunnel -domain live.localhook.online -httpPort 5000 -sshPort 2222

test: 
	@go test -v ./...