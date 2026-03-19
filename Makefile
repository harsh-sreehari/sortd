# sortd Makefile

.PHONY: install build test clean

build:
	go build -o sortd ./cmd/sortd/main.go

install: build
	go install ./cmd/sortd/...
	systemctl --user daemon-reload
	systemctl --user restart sortd

test:
	go test ./...

clean:
	rm -f sortd
