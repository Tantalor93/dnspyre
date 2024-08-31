---
title: Delaying requests
layout: default
parent: Examples
---

# Delaying requests
v3.4.0
{: .label .label-yellow }
*dnspyre* by default generates queries one after another as soon as the previous query is finished. In some cases you might want to delay
the queries. This is possible using `--request-delay` flag. This option allows user to specify either constant or randomized delay to be added
before sending query.

## Constant delay
To specify constant delay, you can specify arbitrary GO duration as parameter to the `--request-delay` flag. Each parallel worker will 
always wait for the specified duration before sending another query

```
dnspyre --duration 10s  --server '1.1.1.1' google.com --request-delay 2s
```

## Randomized delay
To specify randomized delay, you can specify interval of GO durations `<GO duration>-<GO duration>` as parameter to the `--request-delay` flag.
Each parallel worker will always wait for the random duration specified by the interval. If you specify interval `1s-2s`, workers will wait
between 1 second and 2 seconds before sending another query

```
dnspyre --duration 10s  --server '1.1.1.1' google.com --request-delay 1s-2s
```
