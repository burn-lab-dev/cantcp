# API статистики

[English](STATS.md) | **Русский**

Разрабатывается **[BURN-LAB](https://burn-lab.ru)** — встраиваемый Linux:
драйверы, CAN и промышленная телеметрия.

Демон отдаёт статистику отдельным HTTP-листенером (`127.0.0.1:29537` по
умолчанию, `stats.listen`; выключается `--no-stats` или
`stats.enable: false`). Листенер работает по обычному HTTP: держите его на
loopback или внутренней сети, аутентификации в нём нет.

## Эндпоинты

| Метод и путь | Содержимое |
|---|---|
| `GET /healthz` | `200` и `{"status":"ok"}`, пока процесс жив |
| `GET /api/v1/stats` | JSON-снимок (ниже) |
| `GET /metrics` | формат текстовой экспозиции Prometheus 0.0.4 |

`cantcp-cli stats` оборачивает `/api/v1/stats` и печатает таблицу или
сырой JSON.

## JSON-снимок

```json
{
  "version": "v0.1.0",
  "uptime_seconds": 3600,
  "can": {
    "interface": "can0",
    "frames_read": 120345,
    "frames_written": 42,
    "read_errors": 0,
    "write_errors": 1
  },
  "tcp": {
    "connections_current": 2,
    "connections_total": 17,
    "frames_in": 42,
    "frames_out": 120000,
    "bytes_in": 2100,
    "bytes_out": 6100000,
    "dropped": 345
  },
  "clients": [
    {
      "socket": "10.0.0.5:51234",
      "since": "2026-09-28T11:00:00Z",
      "frames_in": 39,
      "frames_out": 120000,
      "bytes_in": 1950,
      "bytes_out": 6100000,
      "dropped": 345
    }
  ]
}
```

- `can.frames_read` / `frames_written` — кадры, обменянные с шиной;
- `can.read_errors` / `write_errors` — неудачные операции с шиной;
- `tcp.frames_in` / `frames_out` — кадры, принятые от клиентов и отправленные
  им (то есть записанные в шину и прочитанные из неё);
- `tcp.bytes_*` — байты TCP, включая накладные расходы TLS;
- `tcp.dropped` — кадры, отброшенные из-за переполнения очереди клиента или
  лимита скорости; счётчики по клиентам — в `clients[]`;
- `clients[]` отсортирован по адресу сокета; при отключении клиент исчезает
  из списка, агрегатные счётчики сохраняют его итоги.

Типы: все счётчики — беззнаковые 64-битные; `since` в RFC 3339;
`uptime_seconds` монотонен и не отрицателен.

## Метрики Prometheus

```sh
curl -s http://127.0.0.1:29537/metrics
```

```
# HELP cantcp_build_info Build information; the version label carries the release.
# TYPE cantcp_build_info gauge
cantcp_build_info{version="v0.1.0"} 1
# HELP cantcp_uptime_seconds Seconds since the daemon started.
# TYPE cantcp_uptime_seconds gauge
cantcp_uptime_seconds 3600
# HELP cantcp_can_frames_total CAN frames exchanged with the bus; direction="in" is read from the bus, direction="out" is written to it.
# TYPE cantcp_can_frames_total counter
cantcp_can_frames_total{direction="in"} 120345
cantcp_can_frames_total{direction="out"} 42
# HELP cantcp_can_errors_total Failed CAN operations; direction is read or write.
# TYPE cantcp_can_errors_total counter
cantcp_can_errors_total{direction="read"} 0
cantcp_can_errors_total{direction="write"} 1
# HELP cantcp_tcp_connections_current Clients connected right now.
# TYPE cantcp_tcp_connections_current gauge
cantcp_tcp_connections_current 2
# HELP cantcp_tcp_connections_total Connections accepted since the daemon started.
# TYPE cantcp_tcp_connections_total counter
cantcp_tcp_connections_total 17
# HELP cantcp_tcp_frames_total Frames exchanged with the clients; ...
# TYPE cantcp_tcp_frames_total counter
cantcp_tcp_frames_total{direction="in"} 42
cantcp_tcp_frames_total{direction="out"} 120000
# HELP cantcp_tcp_bytes_total TCP bytes exchanged with the clients; ...
# TYPE cantcp_tcp_bytes_total counter
cantcp_tcp_bytes_total{direction="in"} 2100
cantcp_tcp_bytes_total{direction="out"} 6100000
# HELP cantcp_dropped_frames_total Frames dropped because a client queue was full or the client exceeded the rate limit.
# TYPE cantcp_dropped_frames_total counter
cantcp_dropped_frames_total 345
```

## Замечания по эксплуатации

- У листенера статистики свои таймауты чтения, записи и заголовков: зависший
  сборщик не израсходует демон.
- Счётчики живут только в памяти: рестарт их обнуляет. Для истории
  используйте Prometheus или другой сборщик с `rate()`/`increase()`.
- Следите за `cantcp_dropped_frames_total` и `cantcp_can_errors_total`: это
  два сигнала, что демон под нагрузкой или шина нездорова.
- Если список клиентов растёт неожиданными адресами, проверьте `listen`,
  `allow_plain` и файрвол — подключиться может любой, кто дотянется до
  порта.
