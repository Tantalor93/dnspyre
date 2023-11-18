---
title: JSON output
layout: default
parent: Examples
---

# JSON output
By specifying `--json` flag, *dnspyre* can output benchmark results in a JSON format, which is better for further automatic processing

```
dnspyre --duration 5s --server 8.8.8.8 google.com  --json
```

example of chaining of *dnspyre* with [jq](https://stedolan.github.io/jq/) for getting pretty JSON

```
dnspyre --duration 5s --server 8.8.8.8 google.com --no-distribution --json | jq '.'
```

like this

```
{
  "totalRequests": 276,
  "totalSuccessCodes": 276,
  "totalErrors": 0,
  "TotalIDmismatch": 0,
  "totalTruncatedResponses": 0,
  "responseRcodes": {
    "NOERROR": 276
  },
  "questionTypes": {
    "A": 276
  },
  "queriesPerSecond": 55.18,
  "benchmarkDurationSeconds": 5,
  "latencyStats": {
    "minMs": 12,
    "meanMs": 18,
    "stdMs": 13,
    "maxMs": 176,
    "p99Ms": 71,
    "p95Ms": 33,
    "p90Ms": 24,
    "p75Ms": 15,
    "p50Ms": 14
  }
}
```
