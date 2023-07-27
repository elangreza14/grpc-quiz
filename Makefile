server:
	go run cmd/quiz/main.go
	
client:
	@read -p  "insert your name... " PLAYER; \
	go run cmd/quiz/main.go -p $$PLAYER

lint:
	gofumpt -l -w .
	
.PHONY: server client lint