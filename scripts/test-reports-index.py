#!/usr/bin/env python3
"""Render the test-report directory as a small HTML page for GitHub Pages.

Usage: scripts/test-reports-index.py <dir>

The directory may contain report.md/report.json files produced by
scripts/vcan-smoke.sh and scripts/hw-smoke.sh, plus committed Markdown
reports. The script writes <dir>/index.html with links and inline contents.
"""

from __future__ import annotations

import html
import sys
from pathlib import Path


def render(directory: Path) -> str:
    """Build the HTML page for the report directory."""
    entries = sorted(
        (path for path in directory.iterdir() if path.suffix == ".md"),
        key=lambda path: path.name,
        reverse=True,
    )
    parts = [
        "<!DOCTYPE html>",
        "<html lang='en'><head><meta charset='utf-8'>",
        "<title>cantcp test reports</title>",
        "<style>",
        "body{font-family:sans-serif;max-width:64rem;margin:2rem auto;padding:0 1rem}",
        "pre{background:#f4f4f4;padding:1rem;overflow-x:auto}",
        "a{color:#0366d6}",
        "</style></head><body>",
        "<h1>cantcp test reports</h1>",
        "<p>Generated from the release workflow; the canonical files live in "
        "<a href='https://github.com/burn-lab-dev/cantcp/tree/main/docs/test-reports'>"
        "docs/test-reports</a>.</p>",
        "<ul>",
    ]
    for entry in entries:
        parts.append(f"<li><a href='#{html.escape(entry.name)}'>{html.escape(entry.name)}</a></li>")
    parts.append("</ul>")
    for entry in entries:
        parts.append(f"<h2 id='{html.escape(entry.name)}'>{html.escape(entry.name)}</h2>")
        parts.append(f"<pre>{html.escape(entry.read_text(encoding='utf-8'))}</pre>")
    parts.append("</body></html>")
    return "\n".join(parts) + "\n"


def main(argv: list[str]) -> int:
    """Write index.html into the given directory."""
    if len(argv) != 2:
        print("usage: scripts/test-reports-index.py <dir>", file=sys.stderr)
        return 2
    directory = Path(argv[1])
    if not directory.is_dir():
        print(f"test-reports-index: {directory}: no such directory", file=sys.stderr)
        return 2
    sys.stdout.write(render(directory))
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
