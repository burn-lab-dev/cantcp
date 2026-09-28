# cantcp

[![CI](https://github.com/burn-lab-dev/cantcp/actions/workflows/ci.yml/badge.svg)](https://github.com/burn-lab-dev/cantcp/actions/workflows/ci.yml)
[![CodeQL](https://github.com/burn-lab-dev/cantcp/actions/workflows/codeql.yml/badge.svg)](https://github.com/burn-lab-dev/cantcp/actions/workflows/codeql.yml)
[![Go version](https://img.shields.io/badge/Go-1.24%2B-00ADD8?logo=go&logoColor=white)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

[English](README.md) | **Русский**

Пара для протокола **cantcp** (Linux SocketCAN поверх TCP):

- **`cantcpd`** — демон: связывает один интерфейс SocketCAN и TCP-клиентов,
  в незащищённом режиме (доверенный периметр) или поверх TLS 1.3;
- **`cantcp-cli`** — клиент: слушает кадры CAN, отправляет их, читает
  статистику сервера.

Разрабатывается **[BURN-LAB](https://burn-lab.ru)** — встраиваемый Linux:
драйверы, CAN и промышленная телеметрия.

> **Статус: v0.** Протокол (`v0`) и командная строка могут меняться до
> v1.0.0. Кадры переносят парные библиотеки:
> [cantcp-lib-go](https://github.com/burn-lab-dev/cantcp-lib-go) и
> [cantcp-lib-python](https://github.com/burn-lab-dev/cantcp-lib-python).

## Возможности

- по одному статическому бинарю на компонент, без cgo и зависимостей;
- незащищённый TCP или TLS 1.3 с опциональным mutual TLS; режимы
  взаимоисключающие;
- безопасные умолчания: plain слушает только loopback, для выхода в сеть
  нужен явный `--allow-plain`;
- настройки из JSON-файла, переменных `CANTCP_*` и флагов командной строки;
  каждое значение валидируется при старте;
- очереди на клиента со счётчиком отброшенных кадров: медленный клиент не
  блокирует шину;
- лимиты соединений, таймаутов и скорости кадров;
- статистика по HTTP (`/api/v1/stats`, `/metrics`, `/healthz`);
- перечитывание TLS-материалов, уровня логов и лимитов по SIGHUP;
- `.deb`-пакеты для amd64, arm64 и armhf с ужесточённым systemd-юнитом и
  APT-репозиторий на GitHub Pages.

## Установка

Из APT-репозитория (amd64, arm64, armhf):

```sh
curl -fsSL https://burn-lab-dev.github.io/cantcp/cantcp.gpg \
  | sudo gpg --dearmor -o /usr/share/keyrings/cantcp.gpg
echo "deb [signed-by=/usr/share/keyrings/cantcp.gpg] https://burn-lab-dev.github.io/cantcp stable main" \
  | sudo tee /etc/apt/sources.list.d/cantcp.list
sudo apt update
sudo apt install cantcpd cantcp-cli
```

Из `.deb`-файла (скачать со страницы релизов):

```sh
sudo apt install ./cantcpd_0.1.0_amd64.deb ./cantcp-cli_0.1.0_amd64.deb
```

С тулчейном Go (Go 1.24 или новее):

```sh
go install github.com/burn-lab-dev/cantcp/cmd/cantcp-cli@latest
```

## Быстрый старт (незащищённый режим)

```sh
# На машине-шлюзе: реальный can0 или виртуальная шина для проверки.
sudo modprobe vcan
sudo ip link add dev vcan0 type vcan && sudo ip link set up vcan0

# Демон на loopback.
cantcpd --can vcan0

# В другом терминале: смотреть кадры и отправить один.
cantcp-cli listen
cantcp-cli send --id 123 --data 11223344

# Статистика сервера.
cantcp-cli stats
```

Проверка без CAN-железа: CLI общается с демоном по TCP, демон — с `vcan0`;
кадры на шине видно через `candump vcan0`, `cansend` или второй `cantcp-cli`.

## Режим TLS

```sh
cantcpd --listen 0.0.0.0:29536 \
  --tls-cert /etc/cantcp/tls/server.pem \
  --tls-key  /etc/cantcp/tls/server-key.pem \
  --tls-ca   /etc/cantcp/tls/ca.pem \
  --tls-client-auth require_and_verify

cantcp-cli listen --server can-gateway.example:29536 \
  --tls --tls-ca /etc/cantcp/tls/ca.pem \
  --tls-cert /etc/cantcp/tls/client.pem \
  --tls-key  /etc/cantcp/tls/client-key.pem
```

Генерация ключей и сертификатов: [docs/TLS-KEYS.ru.md](docs/TLS-KEYS.ru.md).
Подробно про TLS: [docs/TLS.ru.md](docs/TLS.ru.md).

## Кросс-компиляция

Бинари — на чистом Go (без cgo), работает любая цель Go. Релизы покрывают:

| Цель | GOOS | GOARCH | GOARM | Архитектура Debian |
|---|---|---|---|---|
| x86-64 | linux | amd64 | — | `amd64` |
| ARM64 | linux | arm64 | — | `arm64` |
| ARMv7 | linux | arm | 7 | `armhf` |

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./cmd/cantcpd
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ./cmd/cantcpd
CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build ./cmd/cantcpd
```

Любая другая цель — так же (`mipsle`, `riscv64`, ...). См.
[docs/DEPLOY.ru.md](docs/DEPLOY.ru.md).

## Документация

| Документ | Содержание |
|---|---|
| [docs/PROTOCOL.ru.md](docs/PROTOCOL.ru.md) | Поток cantcp и библиотеки |
| [docs/SERVER.ru.md](docs/SERVER.ru.md) | Поведение демона, режимы, лимиты, SIGHUP |
| [docs/CLI.ru.md](docs/CLI.ru.md) | Команды, фильтры, синтаксис кадров |
| [docs/CONFIG.ru.md](docs/CONFIG.ru.md) | JSON-конфигурация, источники, валидация |
| [docs/STATS.ru.md](docs/STATS.ru.md) | API статистики и метрики Prometheus |
| [docs/TLS.ru.md](docs/TLS.ru.md) | TLS 1.3 и mutual TLS |
| [docs/TLS-KEYS.ru.md](docs/TLS-KEYS.ru.md) | Генерация ключей через OpenSSL |
| [docs/DEPLOY.ru.md](docs/DEPLOY.ru.md) | Пакеты, systemd, APT, кросс-сборка |
| [SECURITY.ru.md](SECURITY.ru.md) | Модель угроз и чек-лист деплоя |
| [CONTRIBUTING.ru.md](CONTRIBUTING.ru.md) | Разработка и pull request'ы |

## Сборка из исходников

```sh
make build    # bin/cantcpd, bin/cantcp-cli для текущей системы
make test     # go test -race ./...
make cross    # linux/amd64, linux/arm64, linux/armv7
make deb      # .deb-пакеты в dist/ (нужен dpkg-deb)
```

## Лицензия

MIT — см. [LICENSE](LICENSE).
