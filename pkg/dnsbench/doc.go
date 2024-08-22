/*
Package dnsbench contains functionality for executing various plain DNS, DoH and DoQ Benchmarks.
Each DNS benchmark is represented by Benchmark struct that is used to set up benchmark as desired
and then execute the benchmark using Benchmark.Run. Each execution of Benchmark.Run returns slice
of ResultStats, where each element of the slice represents results of a single benchmark worker.
*/
package dnsbench
