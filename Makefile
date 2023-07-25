server:
	go run *.go
	
client:
	@read -p  "insert your username... " NAME; \
	go run *.go -u $$NAME

lint:
	gofumpt -l -w .
	
.PHONY: server client lint