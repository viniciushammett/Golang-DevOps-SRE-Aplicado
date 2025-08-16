#!/usr/bin/env python3
import argparse, csv, json
from datetime import datetime

def parse_args():
    ap = argparse.ArgumentParser()
    ap.add_argument("--input", required=True, help="CSV com colunas ts,source,msg (ts RFC3339)")
    ap.add_argument("--output", default="model.stats.json")
    ap.add_argument("--bucket", default="1m", help="granularidade (apenas 1m suportado)")
    return ap.parse_args()

def minute_bucket(ts: datetime):
    return ts.replace(second=0, microsecond=0).isoformat()

def main():
    args = parse_args()
    counts = {}  # key -> list of per-minute counts
    permin = {}  # (key, minute) -> count

    with open(args.input, newline='') as f:
        r = csv.DictReader(f)
        for row in r:
            ts = datetime.fromisoformat(row["ts"].replace("Z","+00:00"))
            source = row.get("source","unknown") or "unknown"
            key = f"src={source}"
            minute = minute_bucket(ts)
            permin[(key, minute)] = permin.get((key, minute), 0) + 1

    bykey = {}
    for (key, minute), c in permin.items():
        bykey.setdefault(key, []).append(c)

    def mean(xs): return sum(xs)/len(xs) if xs else 0.0
    def std(xs):
        if not xs or len(xs)<2: return 0.0
        m = mean(xs)
        return (sum((x-m)**2 for x in xs)/(len(xs)-1))**0.5

    out = {"keys": {}}
    for key, arr in bykey.items():
        out["keys"][key] = {"mean": mean(arr), "std": std(arr)}

    with open(args.output, "w") as f:
        json.dump(out, f, indent=2)
    print(f"Wrote {args.output} with {len(out['keys'])} keys")

if __name__ == "__main__":
    main()