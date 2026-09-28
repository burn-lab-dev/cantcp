# Развёртывание

[English](DEPLOY.md) | **Русский**

Разрабатывается **[BURN-LAB](https://burn-lab.ru)** — встраиваемый Linux:
драйверы, CAN и промышленная телеметрия.

## Пакеты

Релизы публикуют по два `.deb`-пакета на архитектуру:

| Пакет | Содержимое |
|---|---|
| `cantcpd` | `/usr/bin/cantcpd`, `/etc/cantcp/cantcpd.json` (conffile), systemd-юнит, man-страница, примеры конфигурации |
| `cantcp-cli` | `/usr/bin/cantcp-cli`, man-страница, примеры конфигурации |

Архитектуры: `amd64`, `arm64`, `armhf` (ARMv7). Установка из файла:

```sh
sudo apt install ./cantcpd_0.1.0_amd64.deb ./cantcp-cli_0.1.0_amd64.deb
```

или из APT-репозитория:

```sh
curl -fsSL https://burn-lab-dev.github.io/cantcp/cantcp.gpg \
  | sudo gpg --dearmor -o /usr/share/keyrings/cantcp.gpg
echo "deb [signed-by=/usr/share/keyrings/cantcp.gpg] https://burn-lab-dev.github.io/cantcp stable main" \
  | sudo tee /etc/apt/sources.list.d/cantcp.list
sudo apt update
sudo apt install cantcpd cantcp-cli
```

Репозиторий пересобирается из `.deb`-файлов релиза workflow'ом релиза и
подписывается ключом проекта; публичный ключ — `cantcp.gpg`.

## systemd

Пакет `cantcpd` ставит `/lib/systemd/system/cantcpd.service`. Юнит:

- работает от системного пользователя `cantcp` с единственной capability
  `CAP_NET_RAW`;
- изолирован: `ProtectSystem=strict`, `ProtectHome=yes`, `PrivateTmp=yes`,
  `RestrictAddressFamilies=AF_INET AF_INET6 AF_UNIX AF_CAN`,
  `NoNewPrivileges=yes`, `SystemCallFilter=@system-service @network-io`;
- перезапускается при сбое (`Restart=always`, `RestartSec=2s`);
- перечитывается по `systemctl reload cantcpd` (`ExecReload` посылает
  `SIGHUP`).

```sh
sudo systemctl status cantcpd
sudo journalctl -u cantcpd -f
sudo systemctl reload cantcpd      # после ротации TLS-материалов
```

Пакет включает и запускает юнит после установки. Если `can0` ещё не
существует, демон завершится с понятной ошибкой, а systemd будет
перезапускать его до появления интерфейса — поднять реальный интерфейс CAN
обязан администратор (`ip link set can0 up type can bitrate 500000`).

### Одна шина на экземпляр

Один экземпляр демона обслуживает один интерфейс CAN. Для нескольких шин
запускаются несколько экземпляров со своими конфигурациями и портами:

```ini
# /etc/systemd/system/cantcpd@.service
[Unit]
Description=cantcp gateway on %i
After=network.target

[Service]
ExecStart=/usr/bin/cantcpd --config /etc/cantcp/%i.json
# ... параметры изоляции из пакетного юнита ...
User=cantcp
Group=cantcp
AmbientCapabilities=CAP_NET_RAW
CapabilityBoundingSet=CAP_NET_RAW

[Install]
WantedBy=multi-user.target
```

```ini
# /etc/cantcp/can1.json
{"listen": "127.0.0.1:29546", "can": {"interface": "can1"}}
```

```sh
sudo systemctl enable --now cantcpd@can1
```

У каждого экземпляра свой порт статистики (`stats.listen`), лимиты и поток
логов: сбой на одной шине остаётся на одной шине.

## Кросс-компиляция

Бинари — на чистом Go, без cgo и системных библиотек. Сборка под любую цель:

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath ./cmd/cantcpd
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath ./cmd/cantcpd
CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -trimpath ./cmd/cantcpd
```

Цели релизов и их аналоги в Debian:

| Цель | GOOS | GOARCH | GOARM | Debian |
|---|---|---|---|---|
| x86-64 | linux | amd64 | — | `amd64` |
| ARM64 | linux | arm64 | — | `arm64` |
| ARMv7 | linux | arm | 7 | `armhf` |

`make cross` собирает три цели в `bin/`, `make deb` упаковывает их в
`.deb`-пакеты в `dist/`. Версия берётся из git-тега (`git describe`),
поэтому перед релизной сборкой поставьте тег.

## Сборка пакетов самостоятельно

```sh
git tag v0.1.0            # если тега ещё нет
make cross                # бинари
make deb VERSION=v0.1.0   # dist/*.deb
```

`scripts/build-deb.sh` — простой POSIX-скрипт вокруг `go build` и
`dpkg-deb`; `scripts/apt-repo.sh` превращает каталог `.deb` в подписанный
APT-репозиторий (`APT_GPG_KEY` — приватный ключ в ASCII armor,
`dpkg-scanpackages` и `apt-ftparchive` делают раскладку).

## USB-адаптеры CAN и виртуальные шины

```sh
# Реальное железо (драйверы SocketCAN в ядре).
sudo ip link set can0 up type can bitrate 500000

# Виртуальная шина: тесты без устройства.
sudo modprobe vcan
sudo ip link add dev vcan0 type vcan
sudo ip link set up vcan0
```

Быстрая проверка после установки:

```sh
cantcpd --can vcan0 &                 # или: sudo systemctl start cantcpd
cantcp-cli send --id 123 --data 11223344
candump vcan0                          # can-utils; кадр на шине
cantcp-cli stats
kill %1
```

## Мониторинг

- `/healthz` — liveness для systemd или балансировщика;
- `/api/v1/stats` — полный JSON-снимок для скриптов;
- `/metrics` — Prometheus: снимайте метрики `cantcp_*` и настраивайте
  алерты на `rate(cantcp_dropped_frames_total[5m]) > 0` и
  `increase(cantcp_can_errors_total[1h]) > 0`.
