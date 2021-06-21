# Makefile

build:
	mkdir -p functions
	GOOS=linux GOARCH=amd64 go build -o functions/fitness ./cmd/fitness/main.go

clean:
	rm -rf functions
