# clank

Минималистичный CLI-интерфейс к любому OpenAI-совместимому API.

Основной кейс: отправить вывод терминала через пайп, задать вопрос и получить ответ прямым текстом в терминале без необходимости открывать браузер.

Поддерживает интерактивный запуск shell-команд (`run_shell_command`), ответы на вопросы (`question`) и просмотр изображений (`view_image` или `-i / --image`).

---

## Особенности

- **Нулевые внешние зависимости:** только стандартная библиотека Go (Go 1.22+), единственный бинарный файл.
- **Поддержка любых OpenAI-совместимых провайдеров:** OpenAI, DeepSeek, LocalAI, Ollama, vLLM и др.
- **Цепочки моделей (fallback):** при сбоях сети, 5xx или 429 переключается на следующую модель в списке.
- **Интерактивные shell-команды:** модель может запрашивать выполнение команд для сбора контекста и решения задач.
- **Белый список команд:** безопасные команды (например, `ls`, `git`, `cat`) можно разрешить исполнять без запроса.
- **Привязка сессий к терминалу (`-r`):** история сохраняется отдельно для каждой вкладки/панели (по TTY и PPID).
- **Поддержка reasoning-моделей:** корректная обработка DeepSeek-R1 (`reasoning_content` и `<think>`-блоков).

---

## Установка и сборка

Требуется установленный **Go 1.22+**.

```bash
make          # Сборка в bin/clank
make install  # Установка в /usr/local/bin/clank (PREFIX переопределяем)
```

Запуск проверок (форматирование, vet, тесты):

```bash
make check
```

---

## Быстрый старт

### 1. Первоначальная настройка

Запустите интерактивный мастер настройки:

```bash
clank config init
```

Или создайте именованный профиль вручную:

```bash
clank config add openrouter
clank config set-url https://openrouter.ai/api/v1
clank config set-key your-api-key
clank config set-model deepseek/deepseek-chat
clank config use openrouter
```

Проверить поддержку tool calls моделью:

```bash
clank config test-tools
```

---

## Использование

```bash
# Простой вопрос
clank "почему не стартует nginx"

# Передача контекста через пайп
dmesg | tail -n 50 | clank "разбери ошибку"

# Продолжение диалога в текущей вкладке (-r / --resume)
clank -r "попробуй другой вариант"

# Явное чтение stdin и продолжение контекста
git diff | clank -c -r "напиши commit message"

# Подробный лог работы (-v) или режим без лишнего вывода (-q)
clank -v "найди большие файлы в /var/log"
```

---

## Конфигурация и управление сессиями

```bash
# Управление профилями
clank config list
clank config show
clank config use <profile_name>

# Управление списком авторазрешённых команд
clank config allow-rm <command>
clank config allow-clear

# Сессии
clank session list
clank session show
clank session clear
```

---

## Лицензия

MIT / Public Domain.
