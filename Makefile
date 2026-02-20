.PHONY: build run lint clean

build:
	go build -o cloudbeats-backup-generator ./cmd

run: build
	./cloudbeats-backup-generator $(ARGS)

lint:
	go vet ./...

clean:
	rm -f cloudbeats-backup-generator
