BIN     := bin/clank
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || date +%Y.%m.%d)
PREFIX  ?= /usr/local

export CGO_ENABLED := 0
GOFLAGS := -trimpath
LDFLAGS := -s -w -X main.version=$(VERSION)

# Список всех 11 целевых платформ для релиза
PLATFORMS := \
	darwin/amd64 \
	darwin/arm64 \
	linux/amd64 \
	linux/arm64 \
	linux/386 \
	linux/arm \
	windows/amd64 \
	windows/arm64 \
	windows/386 \
	freebsd/amd64 \
	freebsd/386

.PHONY: all build install clean check fmt vet release publish publish-patch publish-minor publish-major patch minor major

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

# Сборка релизных архивов (локально или в GitHub Actions)
release: clean
	@mkdir -p build/release
	@for plt in $(PLATFORMS); do \
		os=$${plt%/*}; \
		arch=$${plt#*/}; \
		ext=""; [ "$$os" = "windows" ] && ext=".exe"; \
		name="clank-$(VERSION)-$$os-$$arch"; \
		out="build/release/$$name"; \
		echo "Building $$os/$$arch (CGO_ENABLED=0)..."; \
		GOOS=$$os GOARCH=$$arch go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o "$$out$$ext" . ; \
		if [ "$$os" = "windows" ]; then \
			zip -q -j "$$out.zip" "$$out$$ext" README.md LICENSE; \
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

# Хелпер для создания и пуша тега (SemVer bump: patch, minor, major)
define do_publish
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "Ошибка: в рабочей директории есть незакоммиченные изменения! Закоммитьте их перед публикацией."; \
		exit 1; \
	fi
	@echo "Запуск проверок make check..."
	@$(MAKE) check
	@latest=$$(git describe --tags --abbrev=0 --match "v[0-9]*.[0-9]*.[0-9]*" 2>/dev/null || true); \
	if [ -n "$(VERSION)" ] && [ "$(VERSION)" != "$$latest" ]; then \
		target_ver="$(VERSION)"; \
		case "$$target_ver" in v*) ;; *) target_ver="v$$target_ver" ;; esac; \
	else \
		kind="$(1)"; \
		if [ -z "$$latest" ]; then \
			if [ "$$kind" = "major" ]; then target_ver="v1.0.0"; else target_ver="v0.1.0"; fi; \
		else \
			clean="$${latest#v}"; \
			major=$$(echo "$$clean" | cut -d. -f1); \
			minor=$$(echo "$$clean" | cut -d. -f2); \
			patch=$$(echo "$$clean" | cut -d. -f3); \
			case "$$kind" in \
				major) target_ver="v$$((major + 1)).0.0" ;; \
				minor) target_ver="v$${major}.$$((minor + 1)).0" ;; \
				*)     target_ver="v$${major}.$${minor}.$$((patch + 1))" ;; \
			esac; \
		fi; \
	fi; \
	echo "Предыдущая версия: $${latest:-<нет>}"; \
	echo "Новая версия:      $$target_ver"; \
	echo "Создаю тег $$target_ver..."; \
	git tag -a "$$target_ver" -m "Release $$target_ver"; \
	echo "Отправляю тег $$target_ver в origin..."; \
	git push origin "$$target_ver"; \
	echo "Готово! Тег $$target_ver отправлен на GitHub. Сборка релиза запущена в Actions."
endef

publish:
	$(call do_publish,patch)

publish-patch: publish
patch: publish

publish-minor:
	$(call do_publish,minor)

minor: publish-minor

publish-major:
	$(call do_publish,major)

major: publish-major
