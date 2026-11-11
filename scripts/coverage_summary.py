#!/usr/bin/env python3
"""Общий отчет о покрытии прикладного кода (backend + frontend одним числом).

Берет backend/coverage/coverage.out (Go) и frontend/coverage/lcov.info (Vitest), считает
покрытые строки кода вместе, а не среднее двух процентов. Имитации корпоративных систем
(mockservices, src/mocks) в расчет не входят. Пишет coverage/summary.md и печатает итог.

Запуск после тестов:  python scripts/coverage_summary.py
"""
import os
import sys
from collections import defaultdict

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
GO = os.path.join(ROOT, "backend", "coverage", "coverage.out")
LCOV = os.path.join(ROOT, "frontend", "coverage", "lcov.info")
EXCLUDE = ("/mockservices/", "/cmd/", "src/mocks/")


def go_lines(path):
    """Строки Go: блок покрыт, если count > 0 хотя бы в одном прогоне (профили пакетов сливаются)."""
    blocks = {}
    with open(path, encoding="utf-8") as f:
        for line in f:
            if line.startswith("mode:") or not line.strip():
                continue
            loc, stmts, count = line.rsplit(" ", 2)
            if any(x in loc for x in EXCLUDE):
                continue
            blocks[loc] = (int(stmts), blocks.get(loc, (0, 0))[1] or int(count) > 0)
    per_file = defaultdict(lambda: [0, 0])
    for loc, (stmts, hit) in blocks.items():
        name = loc.split(":")[0]
        per_file[name][1] += stmts
        if hit:
            per_file[name][0] += stmts
    return per_file


def lcov_lines(path):
    per_file, name = {}, None
    with open(path, encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if line.startswith("SF:"):
                name = line[3:]
            elif line.startswith("LH:") and name:
                per_file.setdefault(name, [0, 0])[0] = int(line[3:])
            elif line.startswith("LF:") and name:
                per_file.setdefault(name, [0, 0])[1] = int(line[3:])
    return {k: v for k, v in per_file.items() if not any(x in k.replace("\\", "/") for x in EXCLUDE)}


def total(per_file):
    hit = sum(v[0] for v in per_file.values())
    all_ = sum(v[1] for v in per_file.values())
    return hit, all_


def pct(hit, all_):
    return 100.0 * hit / all_ if all_ else 0.0


def main():
    parts = []
    for title, path, reader in (("backend (Go, операторы)", GO, go_lines), ("frontend (TS, строки)", LCOV, lcov_lines)):
        if not os.path.exists(path):
            print(f"нет отчета {path}: сначала запустите тесты", file=sys.stderr)
            return 1
        parts.append((title, *total(reader(path))))
    hit = sum(p[1] for p in parts)
    all_ = sum(p[2] for p in parts)
    lines = ["# Покрытие прикладного кода тестами", "", "| Часть | Покрыто | Всего | % |", "|---|---|---|---|"]
    for title, h, a in parts:
        lines.append(f"| {title} | {h} | {a} | {pct(h, a):.1f} |")
    lines.append(f"| итого | {hit} | {all_} | {pct(hit, all_):.1f} |")
    lines += ["", "Имитации корпоративных систем (backend/internal/mockservices, frontend/src/mocks) не учитываются."]
    os.makedirs(os.path.join(ROOT, "coverage"), exist_ok=True)
    with open(os.path.join(ROOT, "coverage", "summary.md"), "w", encoding="utf-8") as f:
        f.write("\n".join(lines) + "\n")
    print("\n".join(lines))
    return 0 if pct(hit, all_) > 80 else 2


if __name__ == "__main__":
    sys.exit(main())
