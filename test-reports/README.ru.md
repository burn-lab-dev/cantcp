# Отчёты о тестировании

[English](README.md) | **Русский**

Разрабатывается **[BURN-LAB](https://burn-lab.ru)** — встраиваемый Linux:
драйверы, CAN и промышленная телеметрия.

Отчёты прогонов против канона протокола
([cantcp-spec](https://github.com/burn-lab-dev/cantcp-spec)) и пары
демон/клиент:

- автоматические прогоны: `scripts/vcan-smoke.sh` в CI выкладывает
  `report.json` и `report.md` артефактами и в summary джобы;
- коммитимые отчёты: один Markdown-файл на прогон, имя
  `YYYY-MM-DD-<среда>.md` (и `.ru.md`), создаётся после ручного прогона
  (`scripts/vcan-smoke.sh --report DIR`, `scripts/hw-smoke.sh --report DIR`);
- workflow релиза зеркалит этот каталог на GitHub Pages (`/test-reports/`).

Отчёт фиксирует среду (машина, ядро, Go, версии библиотек), интерфейс и
MTU или битрейт, таблицу проверок с результатами и найденные отклонения.
Рядом с Markdown при необходимости кладётся сырой `report.json`.

| Отчёт | Среда |
|---|---|
| [2026-09-28-vcan-smoke.ru.md](2026-09-28-vcan-smoke.ru.md) | виртуальная шина (vcan0, MTU 72), первый полный прогон |
| [2026-09-29-hardware.ru.md](2026-09-29-hardware.ru.md) | два адаптера PCAN-USB, одна шина, 125k–1000k, пакетный сервис и APT-бинарники |
| [2026-09-29-arm.ru.md](2026-09-29-arm.ru.md) | плата aarch64 (Ubuntu 22.04), arm64-пакет из репозитория, vcan |
| [2026-09-29-soak.ru.md](2026-09-29-soak.ru.md) | 60-минутный soak на vcan0, контроль ресурсов и стабильности |
