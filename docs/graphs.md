---
title: Plotting graphs
layout: default
parent: Examples
---

# Plotting graphs
*dnspyre* is able to also visualize the benchmark results as graphs, plotting is enabled by using `--plot` option and providing valid path where to export graphs.
Graphs are exported into the new subdirectory `graphs-<timestamp>` on provided path

For example, this command

```
dnspyre -d 30s -c 20 --server 8.8.8.8 --plot . https://raw.githubusercontent.com/Tantalor93/dnspyre/master/data/10000-domains
```

generates these graphs:
* response latency histogram, see [Latency histogram](#latency-histogram) section
* response latency boxplot, see [Latency boxplot](#latency-boxplot) section
* barchart of response codes, see [Response codes barchart](#response-codes-barchart) section
* throughput of DNS server during the benchmark, see [Throughput line graph](#throughput-line-graph) section
* line graphs of observed latencies of responses of DNS server, see [Latency line plot](#latency-line-plot) section
* error rate over time, see [Error rate over time plot](#error-rate-over-time-plot) section

## Latency histogram
Shows the distribution of response latencies 

![latency histogram](graphs/latency-histogram.svg)

## Latency boxplot
Shows the distribution of response latencies

![latency boxplot](graphs/latency-boxplot.svg)

## Response codes barchart
Shows the distribution of DNS server response codes

![responses bar](graphs/responses-barchart.svg)

## Throughput line graph
Shows the throughput of DNS requests during benchmark execution

![throughput line](graphs/throughput-lineplot.svg)

## Latency line plot
Shows the latencies of DNS responses during benchmark execution

![latency line](graphs/latency-lineplot.svg)

## Error rate over time plot
Shows the number of IO errors during benchmark execution

![error rate line](graphs/errorrate-lineplot.svg)
