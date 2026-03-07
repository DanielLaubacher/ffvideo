.PHONY: build install clean

build:
	go build -o bin/ffvideo ./cmd/ffvideo

install:
	go install ./cmd/ffvideo

clean:
	rm -rf bin/
