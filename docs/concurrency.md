---
title: Concurrency
layout: default
parent: Examples
---

# Concurrency
*dnspyre* by default benchmarks a DNS server using a single thread (worker). This can be adjusted using `-c` (`--concurrency`) flag.

In this example, the benchmark runs in 10 parallel threads (option `-c`) , where each thread will send 2 (option `-n`) `example.com.` DNS queries 
of type `A` one after another to the `8.8.8.8` server

```
dnspyre -n 2 -c 10 --server 8.8.8.8 example.com
```

## CPU Limit

By default, *dnspyre* uses all available CPU cores for the benchmark. You can limit the number of CPU cores used by the benchmark using the `--cpu` flag.

This can be useful when:
- Running benchmarks on shared systems where you don't want to consume all CPU resources
- Testing performance under constrained CPU conditions
- Running multiple benchmark instances in parallel

Example limiting the benchmark to use only 2 CPU cores:

```
dnspyre -n 100 -c 10 --cpu 2 --server 8.8.8.8 example.com
```

When the `--cpu` flag is specified, *dnspyre* will display the CPU limit information:

```
Using 1 hostnames
Benchmarking 8.8.8.8:53 via udp with 10 concurrent requests 
Using 2 out of 4 available CPUs
```

{: .note }
The CPU limit is applied using Go's `runtime.GOMAXPROCS()` function and is restored to the original value after the benchmark completes.
