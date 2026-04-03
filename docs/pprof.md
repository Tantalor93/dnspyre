---
title: Profiling (pprof)
layout: default
parent: Examples
---

# Profiling (pprof)
*dnspyre* can be configured to expose Go profiling information via the standard `net/http/pprof` HTTP endpoint. This is useful for diagnosing
performance issues, memory usage, goroutine leaks, and other runtime behavior of *dnspyre* during benchmark execution.

This is controlled by the `--pprof` flag. The flag accepts a `host:port` address, where the pprof HTTP server will be started. The profiling
information is available under the `/debug/pprof/` path.

## Available profiles
The following profiles are available at `/debug/pprof/`:
* **allocs** – A sampling of all past memory allocations
* **block** – Stack traces that led to blocking on synchronization primitives
* **goroutine** – Stack traces of all current goroutines
* **heap** – A sampling of memory allocations of live objects
* **mutex** – Stack traces of holders of contended mutexes
* **threadcreate** – Stack traces that led to the creation of new OS threads
* **profile** – CPU profile (use `go tool pprof` to retrieve)

## Example
To expose pprof profiling on `:6060` you would specify `--pprof ':6060'`
```
dnspyre --server 8.8.8.8 -c 10 -t A --duration 5m --pprof ':6060' google.com
```

While the benchmark is running, you can use `go tool pprof` to collect and analyze profiles:
```shell
# Collect a 30-second CPU profile
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Analyze heap memory usage
go tool pprof http://localhost:6060/debug/pprof/heap

# View goroutine stacks
curl http://localhost:6060/debug/pprof/goroutine?debug=1
```
