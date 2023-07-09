---
title: Plotting graphs
layout: default
parent: Examples
---

# Plotting graphs
dnspyre is able to also visualize the benchmark results as graphs, plotting is enabled by using `--plot` option and providing valid path where to export graphs.
Graphs are exported into the new subdirectory `graphs-<RFC3339 timestamp>` on provided path

For example, this command
```
dnspyre -d 30s -c 2 --server 8.8.8.8 --plot . https://raw.githubusercontent.com/Tantalor93/dnspyre/master/data/1000-domains
```

generates these graphs:
* response latency histogram, see [Latency histogram](#latency-histogram) section
* response latency boxplot, see [Latency boxplot](#latency-boxplot) section
* barchart of response codes, see [Response codes barchart](#response-codes-barchart) section
* throughput of DNS server during the benchmark, see [Throughput line graph](#throughput-line-graph) section
* linegraphs of observed latencies of responses of DNS server, see [Latency line plot](#latency-line-plot) section

## Latency histogram
Shows the distribution of response latencies 

![latency histogram](graphs/latency-histogram.png)

## Latency boxplot
Shows the distribution of response latencies

![latency boxplot](graphs/latency-boxplot.png)

## Response codes barchart
Shows the distribution of DNS server response codes

![responses bar](graphs/responses-barchart.png)

## Throughput line graph
Shows the throughput of DNS requests during benchmark execution

![throughput line](graphs/throughput-lineplot.png)

## Latency line plot
Shows the latencies of DNS responses during benchmark execution

![latency line](graphs/latency-lineplot.png)
