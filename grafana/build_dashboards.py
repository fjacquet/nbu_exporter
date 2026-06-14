#!/usr/bin/env python3
"""Generate all nbu_exporter Grafana dashboards from grafana/gen/.

Run from the repo root:  python3 grafana/build_dashboards.py
Each dashboard is validated against the known nbu_* metric set before writing.
"""
import json
import sys

from grafana.gen import overview, jobs, storage, dataprotection
from grafana.gen.validate import check_dashboard

OUTPUTS = [
    ("grafana/nbu-overview.json", overview.build),
    ("grafana/nbu-jobs.json", jobs.build),
    ("grafana/nbu-storage.json", storage.build),
    ("grafana/nbu-dataprotection.json", dataprotection.build),
]


def main():
    failures = []
    for path, build in OUTPUTS:
        dash = build()
        unknown = check_dashboard(dash)
        if unknown:
            failures.append(f"{path}: unknown metrics {unknown}")
            continue
        with open(path, "w") as f:
            json.dump(dash, f, indent=2)
            f.write("\n")
        print(f"wrote {path} ({len(dash['panels'])} panels)")
    if failures:
        for msg in failures:
            print(f"ERROR: {msg}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
