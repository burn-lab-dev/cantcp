# Конфигурация

[English](CONFIG.md) | **Русский**

Разрабатывается **[BURN-LAB](https://burn-lab.ru)** — встраиваемый Linux:
драйверы, CAN и промышленная телеметрия.

## Источники и приоритет

Оба бинаря собирают конфигурацию из четырёх источников; следующий
перекрывает предыдущий:

1. встроенные умолчания;
2. JSON-файл (`--config`, по умолчанию `/etc/cantcp/cantcpd.json` у демона и
   `/etc/cantcp/cantcp-cli.json` у клиента);
3. переменные окружения `CANTCP_*`;
4. флаги командной строки (учитываются только реально переданные флаги,
   поэтому `--listen 127.0.0.1:29536` перекрывает и файл).

Файл, переданный через `--config`, обязан существовать; файл по умолчанию
может отсутствовать. Парсер JSON строгий: неизвестные ключи, неверные типы
и данные после значения — ошибки; опечатка останавливает старт, а не
игнорируется молча.

Длительности — строки Go: `5s`, `250ms`, `1m30s`.

## Демон (`cantcpd`)

| Ключ JSON | Флаг | Переменная | Умолчание | Валидация |
|---|---|---|---|---|
| `listen` | `--listen` | `CANTCP_LISTEN` | `127.0.0.1:29536` | `host:port`, порт 1..65535 |
| `allow_plain` | `--allow-plain` | `CANTCP_ALLOW_PLAIN` | `false` | нужен для plain на не-loopback адресе |
| `can.interface` | `--can` | `CANTCP_CAN` | `can0` | непусто, до 15 символов |
| `can.error_frames` | `--error-frames` | `CANTCP_ERROR_FRAMES` | `false` | boolean |
| `tls.cert_file` | `--tls-cert` | `CANTCP_TLS_CERT` | — | задаётся вместе с ключом |
| `tls.key_file` | `--tls-key` | `CANTCP_TLS_KEY` | — | задаётся вместе с сертификатом |
| `tls.ca_file` | `--tls-ca` | `CANTCP_TLS_CA` | — | нужен для проверки клиентов |
| `tls.client_auth` | `--tls-client-auth` | `CANTCP_TLS_CLIENT_AUTH` | `none` | `none`, `verify_if_given`, `require_and_verify` |
| `limits.max_connections` | `--max-connections` | `CANTCP_MAX_CONNECTIONS` | `16` | 1..4096 |
| `limits.client_queue` | `--client-queue` | `CANTCP_CLIENT_QUEUE` | `1024` | 1..1048576 |
| `limits.read_timeout` | `--read-timeout` | `CANTCP_READ_TIMEOUT` | `5s` | > 0 |
| `limits.write_timeout` | `--write-timeout` | `CANTCP_WRITE_TIMEOUT` | `5s` | > 0 |
| `limits.idle_timeout` | `--idle-timeout` | `CANTCP_IDLE_TIMEOUT` | `0s` | ≥ 0 |
| `limits.max_frames_per_second` | `--max-frames-per-second` | `CANTCP_MAX_FRAMES_PER_SECOND` | `0` | ≥ 0 |
| `stats.enable` | `--no-stats` | `CANTCP_STATS_ENABLE` | `true` | boolean |
| `stats.listen` | `--stats-listen` | `CANTCP_STATS_LISTEN` | `127.0.0.1:29537` | `host:port`, когда включено |
| `log.level` | `--log-level` | `CANTCP_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `log.format` | `--log-format` | `CANTCP_LOG_FORMAT` | `text` | `text`, `json` |

Примеры: [../examples/cantcpd-plain.json](../examples/cantcpd-plain.json) и
[../examples/cantcpd-tls.json](../examples/cantcpd-tls.json).

## Клиент (`cantcp-cli`)

| Ключ JSON | Флаг | Переменная | Умолчание | Валидация |
|---|---|---|---|---|
| `server` | `--server` | `CANTCP_SERVER` | `127.0.0.1:29536` | `host:port`, порт 1..65535 |
| `stats_server` | `--stats-server` | `CANTCP_STATS_SERVER` | `http://127.0.0.1:29537` | URL `http://` или `https://` |
| `tls.enable` | `--tls` | `CANTCP_TLS_ENABLE` | `false` | boolean; остальные ключи TLS требуют его |
| `tls.ca_file` | `--tls-ca` | `CANTCP_TLS_CA` | — | файл читается при подключении |
| `tls.cert_file` | `--tls-cert` | `CANTCP_TLS_CERT` | — | задаётся вместе с ключом |
| `tls.key_file` | `--tls-key` | `CANTCP_TLS_KEY` | — | задаётся вместе с сертификатом |
| `tls.server_name` | `--tls-server-name` | `CANTCP_TLS_SERVER_NAME` | — | любое DNS-имя |
| `tls.insecure` | `--tls-insecure` | `CANTCP_TLS_INSECURE` | `false` | только отладка |
| `log.level` | `--log-level` | `CANTCP_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `log.format` | `--log-format` | `CANTCP_LOG_FORMAT` | `text` | `text`, `json` |

Пример: [../examples/cantcp-cli.json](../examples/cantcp-cli.json).

## Правила валидации

Каждое значение проверяется до старта процесса или применения перечитывания:

- **адреса**: `host:port`, порт 1..65535; пустой хост — все интерфейсы;
- **незащищённый режим**: внешний адрес прослушивания без TLS требует
  `allow_plain`, иначе конфигурация отвергается. Loopback — `localhost`,
  `127.0.0.0/8` и `::1`;
- **TLS**: сертификат и ключ задаются вместе; `client_auth` отличный от
  `none` требует пары сертификатов и файла CA; CA без TLS отвергается;
  поддерживается только TLS 1.3;
- **интерфейс CAN**: непустой, до 15 символов, без пробелов, `/` и `:`;
  существование и тип интерфейса проверяются при старте демона;
- **лимиты**: соединения 1..4096, очередь 1..1048576, read/write-таймауты
  > 0, idle-таймаут и лимит скорости ≥ 0;
- **логи**: уровень и формат — из известных наборов.

Тексты ошибок называют ключ JSON (или поле из флага) и допустимые значения,
например:

```
can.interface: must not be empty
listen: plain mode on "0.0.0.0:29536" is not loopback; set allow_plain or configure TLS
limits.max_connections: 0 is out of range [1, 4096]
```

## Перечитывание

`SIGHUP` заново читает те же аргументы командной строки, окружение и файл,
валидирует результат и применяет TLS-материалы, уровень логов и лимиты.
Адрес прослушивания, интерфейс CAN, переключатель TLS и адрес статистики
фиксированы на время жизни процесса: меняйте их рестартом.
