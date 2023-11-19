---
title: Basics
layout: default
parent: Examples
---

# Basics
*dnspyre* is a tool for benchmarking DNS servers, it works by spawning configured number of concurrent worker thread, where each worker thread
is sending DNS queries for a domains provided to the *dnspyre* tool. The *dnspyre* runs until one of the conditions is met:
* configured number of repetitions of domain queries is sent (if `--number` flag is specified)
* the required duration of benchmark run elapses (if `--duration` flag is specified)
* benchmark is interrupted with the SIGINT signal

## Run benchmark with the configured number of repetitions
This example runs the benchmark in 10 parallel threads, where each thread will send 2 `example.com.` DNS queries
of type `A` one after another to the `8.8.8.8` server

```
dnspyre -n 2 -c 10 --server 8.8.8.8 example.com
```

## Run benchmark over specified time
This example runs the benchmark in 10 parallel threads for a duration of 30 seconds while sending `example.com` DNS queries of type `A`
to the `8.8.8.8` server

```
dnspyre --duration 30s -c 10 --server 8.8.8.8 google.com
```
