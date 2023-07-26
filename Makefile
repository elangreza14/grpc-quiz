server:
	go run cmd/quiz/main.go
	
client:
	@read -p  "insert your playername... " NAME; \
	go run cmd/quiz/main.go -p $$NAME

lint:
	gofumpt -l -w .
	
.PHONY: server client lint