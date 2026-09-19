BIN     := bin/clank
# Версия из даты сборки: VCS тут fossil и завязываться на него незачем.
VERSION ?= $(shell date +%Y.%m.%d)
PREFIX  ?= /usr/local

GOFLAGS := -trimpath
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: all build install clean check fmt vet

all: build

build:
	@mkdir -p bin
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN) .
	@echo "собрано: $(BIN) ($(VERSION))"

install: build
	@mkdir -p $(PREFIX)/bin
	install -m 0755 $(BIN) $(PREFIX)/bin/clank
	@echo "установлено: $(PREFIX)/bin/clank"

check: fmt vet
	go test ./...

fmt:
	gofmt -l -w .

vet:
	go vet ./...

clean:
	rm -rf bin
