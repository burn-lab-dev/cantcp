# cantcp-cli

[English](CLI.md) | **Русский**

Разрабатывается **[BURN-LAB](https://burn-lab.ru)** — встраиваемый Linux:
драйверы, CAN и промышленная телеметрия.

`cantcp-cli` подключается к серверу cantcp (см. [SERVER.ru.md](SERVER.ru.md)),
слушает кадры CAN, отправляет их и читает статистику сервера.

## Команды

```
cantcp-cli <команда> [флаги]

  listen    подключиться и печатать кадры, принятые от сервера
  send      отправка кадров из флагов или candump-файла
  stats     запрос статистики сервера по HTTP
  version   печать версии
  help      общая справка
```

Флаги команды: `cantcp-cli <команда> -h`.

## Флаги соединения (все команды)

| Флаг | По умолчанию | Значение |
|---|---|---|
| `--config` | `/etc/cantcp/cantcp-cli.json` | JSON-файл конфигурации |
| `--server` | `127.0.0.1:29536` | адрес сервера cantcp |
| `--stats-server` | `http://127.0.0.1:29537` | URL статистики (команда `stats`) |
| `--tls` | выкл | соединение по TLS 1.3 |
| `--tls-ca` | — | файл CA (по умолчанию системные корни) |
| `--tls-cert`, `--tls-key` | — | клиентский сертификат для mutual TLS |
| `--tls-server-name` | — | проверяемое имя (по умолчанию хост из `--server`) |
| `--tls-insecure` | выкл | отключить проверку сертификата, **только отладка** |
| `--log-level`, `--log-format` | `info`, `text` | логирование |

## listen

```sh
# Печатать все кадры в формате candump.
cantcp-cli listen

# Только 0x7A0..0x7AF, JSON-строки, остановиться после 100 кадров.
cantcp-cli listen --id 7A0 --mask 7F0 --json --count 100

# Остановиться через 30 секунд.
cantcp-cli listen --timeout 30s
```

| Флаг | Значение |
|---|---|
| `--id`, `--mask` | шестнадцатеричный фильтр: кадр проходит, если `frame.id & mask == id & mask`; нулевая маска пропускает всё |
| `--json` | по одному JSON-объекту на кадр |
| `--count` | остановиться после n кадров (0 — без ограничения) |
| `--timeout` | остановиться через указанное время (0 — без ограничения) |
| `--can-name` | имя интерфейса в выводе candump (по умолчанию `can0`) |

Формат вывода candump:

```
(1696000000.123456) can0 123#11223344
(1696000000.123456) can0 12345678#1122
(1696000000.123456) can0 123#R
(1696000000.123456) can0 123#E00000004
(1696000000.123456) can0 123##1DEADBEEF
```

Таймстамп — локальное время приёма; цифра флагов CAN FD: бит 0 — BRS,
бит 1 — ESI. JSON-форма:

```json
{"time":"2026-09-28T12:00:00.123456Z","id":291,"fd":false,"data":"11223344"}
```

Кадры сбрасываются в вывод сразу, по одному: пайпы и сборщики логов видят
их немедленно. Код возврата 0 — чистое завершение (поток закрыт, достигнут
счётчик или таймаут).

## send

```sh
# Один классический кадр.
cantcp-cli send --id 123 --data 11223344

# Расширенный идентификатор, 100 раз с паузой 10 мс.
cantcp-cli send --id 1ABCDE --data 01 --count 100 --interval 10ms

# Кадр CAN FD с переключением битрейта.
cantcp-cli send --id 123 --data 0102030405060708090A0B0C --fd --brs

# Все кадры из candump-файла или из stdin.
cantcp-cli send --input frames.txt
cantcp-cli send - < frames.txt
```

| Флаг | Значение |
|---|---|
| `--id`, `--data` | идентификатор и данные, шестнадцатерично |
| `--ext` | принудительно расширенный (29-битный) идентификатор |
| `--fd`, `--brs` | кадр CAN FD и переключение битрейта |
| `--count`, `--interval` | повтор одного кадра (только с `--id`) |
| `--input` | файл со строками candump, `-` читает stdin |

Синтаксис строк ввода (тот же, что печатает `listen`):

```
123#11223344          классический кадр
12345678#1122         расширенный идентификатор (8 hex-цифр)
123#R                 кадр remote
123##1DEADBEEF        CAN FD, цифра флагов: бит 0 — BRS, бит 1 — ESI
(1696000000.1) can0 123#1122     полная строка candump
```

Пустые строки и строки, начинающиеся с `#`, пропускаются; битая строка
останавливает команду с указанием файла и номера строки. Кадр проверяется
(диапазон идентификатора, длина данных CAN FD) до отправки.

## stats

```sh
cantcp-cli stats                 # таблица для человека
cantcp-cli stats --json          # сырой JSON-ответ
cantcp-cli stats --stats-server http://10.0.0.5:29537
```

Команда запрашивает `<stats-server>/api/v1/stats`; формат ответа описан в
[STATS.ru.md](STATS.ru.md).

## Примеры TLS

```sh
cantcp-cli listen --server can-gateway.example:29536 \
  --tls --tls-ca /etc/cantcp/tls/ca.pem

cantcp-cli send --id 123 --data 01 \
  --server can-gateway.example:29536 --tls \
  --tls-ca /etc/cantcp/tls/ca.pem \
  --tls-cert /etc/cantcp/tls/client.pem \
  --tls-key  /etc/cantcp/tls/client-key.pem
```

`--tls-insecure` отключает проверку сертификата: удобно на стенде с
самоподписанным сертификатом и опасно где-либо ещё. Конфигурационный файл
клиента описан в [CONFIG.ru.md](CONFIG.ru.md).

## Свой клиент

`cantcp-cli` — лишь один клиент: сервер говорит на открытом потоке cantcp
(см. [PROTOCOL.ru.md](PROTOCOL.ru.md)), поэтому с ним могут работать и ваши
инструменты. Используйте парные библиотеки:

- Go: [cantcp-lib-go](https://github.com/burn-lab-dev/cantcp-lib-go),
  `examples/client`;
- Python: [cantcp-lib-python](https://github.com/burn-lab-dev/cantcp-lib-python),
  `examples/client.py`;
- канон протокола: [cantcp-spec](https://github.com/burn-lab-dev/cantcp-spec).

Замечание про потоки в Python: оборачивайте сокет через
`socket.makefile("rb", buffering=0)` для чтения (буферизованный reader ждёт
заполнения запрошенного размера) и `socket.makefile("wb")` с явным flush
для записи.

## Окружение и файл конфигурации

У каждого общего флага есть переменная `CANTCP_*` (`CANTCP_SERVER`,
`CANTCP_STATS_SERVER`, `CANTCP_TLS_ENABLE`, ...) и ключ в JSON-файле; флаги
приоритетнее окружения, окружение — файла, файл — умолчаний. См.
[CONFIG.ru.md](CONFIG.ru.md).
