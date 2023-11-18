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
