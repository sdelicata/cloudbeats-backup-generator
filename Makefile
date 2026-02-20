.PHONY: build run lint test clean

build:
	go build -o cloudbeats-backup-generator ./cmd

run: build
	./cloudbeats-backup-generator $(ARGS)

lint:
	go vet ./...

test:
	go test ./...

clean:
	rm -f cloudbeats-backup-generator
