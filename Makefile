server:
	go run *.go
	
client:
	@read -p  "insert your playername... " NAME; \
	go run *.go -p $$NAME

lint:
	gofumpt -l -w .
	
.PHONY: server client lint