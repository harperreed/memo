.PHONY: build test install clean

build:
	go build -o bin/memo ./cmd/memo

test:
	go test -v ./...

install:
	go install ./cmd/memo

clean:
	rm -rf bin/
