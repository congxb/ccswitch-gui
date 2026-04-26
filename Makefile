.PHONY: build clean run tidy

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS = -s -w -H windowsgui -X main.version=$(VERSION)

build:
	CGO_ENABLED=1 go build -ldflags "$(LDFLAGS)" -o ccswitch-gui.exe .

build-debug:
	CGO_ENABLED=1 go build -ldflags "-X main.version=$(VERSION)" -o ccswitch-gui.exe .

clean:
	rm -f ccswitch-gui.exe

run: build
	./ccswitch-gui.exe

tidy:
	go mod tidy

fmt:
	gofmt -w .

vet:
	go vet ./...
