.PHONY: build run lint test fmt clean

build:
	go build -o cloudbeats-backup-generator ./cmd

run: build
	./cloudbeats-backup-generator $(ARGS)

lint:
	golangci-lint run

fmt:
	gofumpt -w .
	gci write --section standard --section default --section "prefix(github.com/simon/cloudbeats-backup-generator)" .

test:
	go test -race ./...

clean:
	rm -f cloudbeats-backup-generator
