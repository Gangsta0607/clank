BIN     := bin/clank
# Версия из даты сборки: VCS тут fossil и завязываться на него незачем.
VERSION ?= $(shell date +%Y.%m.%d)
PREFIX  ?= /usr/local

GOFLAGS := -trimpath
LDFLAGS := -s -w -X main.version=$(VERSION)

# Список платформ для релиза
PLATFORMS := \
	darwin/amd64 \
	darwin/arm64 \
	linux/amd64 \
	linux/arm64 \
	windows/amd64 \
	windows/arm64 \
	freebsd/amd64

.PHONY: all build install clean check fmt vet release publish

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
	rm -rf bin build

# Сборка релизных архивов
release: clean
	@mkdir -p build/release
	@for plt in $(PLATFORMS); do \
		os=$${plt%/*}; \
		arch=$${plt#*/}; \
		ext=""; [ "$$os" = "windows" ] && ext=".exe"; \
		name="clank-$(VERSION)-$$os-$$arch"; \
		out="build/release/$$name"; \
		echo "Building $$os/$$arch..."; \
		GOOS=$$os GOARCH=$$arch go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o "$$out$$ext" . ; \
		if [ "$$os" = "windows" ]; then \
			zip -j "$$out.zip" "$$out$$ext" README.md LICENSE; \
		else \
			cp README.md LICENSE build/release/ ; \
			tar -czf "$$out.tar.gz" -C build/release "$$name" README.md LICENSE; \
			rm build/release/README.md build/release/LICENSE ; \
		fi; \
		rm "$$out$$ext"; \
	done
	@echo "Генерация контрольных сумм..."
	@cd build/release && (shasum -a 256 clank-* 2>/dev/null || sha256sum clank-* 2>/dev/null) > checksums.txt
	@echo "Релизы собраны в build/release/"

publish: release
