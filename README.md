[![Release](https://img.shields.io/github/release/Tantalor93/dnspyre/all.svg)](https://github.com/tantalor93/dnspyre/releases)
[![Docker Pulls](https://img.shields.io/docker/pulls/tantalor93/dnspyre.svg)](https://hub.docker.com/r/tantalor93/dnspyre)
[![Go Report Card](https://goreportcard.com/badge/github.com/tantalor93/dnspyre/v2)](https://goreportcard.com/report/github.com/tantalor93/dnspyre/v2)
[![Tantalor93](https://circleci.com/gh/Tantalor93/dnspyre/tree/master.svg?style=svg)](https://circleci.com/gh/Tantalor93/dnspyre?branch=master)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/tantalor93/dnspyre/v2/blob/master/LICENSE)
[![codecov](https://codecov.io/gh/Tantalor93/dnspyre/branch/master/graph/badge.svg?token=MC6PK2OLMK)](https://codecov.io/gh/Tantalor93/dnspyre)
[![lint](https://github.com/Tantalor93/dnspyre/actions/workflows/lint.yml/badge.svg?branch=master)](https://github.com/Tantalor93/dnspyre/actions/workflows/lint.yml)

# dnspyre

Command-line DNS benchmark tool built to stress test and measure the performance of DNS servers. You can easily run benchmark from MacOS, Linux or Windows systems.

This tool is based and originally forked from [dnstrace](https://github.com/redsift/dnstrace), but was largely rewritten and enhanced with additional functionality.

This tool supports wide variety of options to customize DNS benchmark and benchmark output. For example, you can:
* benchmark DNS servers using DNS queries over UDP or TCP
* benchmark DNS servers with all kinds of query types like A, AAAA, CNAME, HTTPS, ... (`--type` option)
* benchmark DNS servers with a lot of parallel queries and connections (`--number`, `--concurrency` options)
* benchmark DNS servers for a specified duration (`--duration` option)
* benchmark DNS servers with DoT
* benchmark DNS servers using DoH
* benchmark DNS servers using DoQ
* benchmark DNS servers with uneven random load from provided high volume resources (see `--probability` option)
* plot benchmark results via CLI histogram or plot the benchmark results as boxplot, histogram, line graphs and export them via all kind of image formats like png, svg and pdf. (see `--plot` and `--plotf` options)

## Documentation 
For installation guide, examples and more, see the [documentation page](https://tantalor93.github.io/dnspyre/) 
