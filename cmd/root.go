package cmd

import (
	"context"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/miekg/dns"
)

var (
	// Version is set during release of project during build process.
	Version string

	author = "Ondrej Benkovsky <obenky@gmail.com>"
)

var (
	pApp = kingpin.New("dnspyre", "A high QPS DNS benchmark.").Author(author)

	benchmark Benchmark
)

func init() {
	pApp.Flag("server", "Server represents (plain DNS, DoT, DoH or DoQ) server, which will be benchmarked. "+
		"Format depends on the DNS protocol, that should be used for DNS benchmark. "+
		"For plain DNS (either over UDP or TCP) the format is <IP/host>[:port], if port is not provided then port 53 is used. "+
		"For DoT the format is <IP/host>[:port], if port is not provided then port 853 is used. "+
		"For DoH the format is https://<IP/host>[:port][/path] or http://<IP/host>[:port][/path], if port is not provided then either 443 or 80 port is used. If no path is provided, then /dns-query is used. "+
		"For DoQ the format is quic://<IP/host>[:port], if port is not provided then port 853 is used.").Short('s').Default("127.0.0.1").StringVar(&benchmark.Server)

	pApp.Flag("type", "Query type. Repeatable flag. If multiple query types are specified then each query will be duplicated for each type.").
		Short('t').Default("A").EnumsVar(&benchmark.Types, getSupportedDNSTypes()...)

	pApp.Flag("number", "How many times the provided queries are repeated. Note that the total number of queries issued = types*number*concurrency*len(queries).").
		Short('n').Int64Var(&benchmark.Count)

	pApp.Flag("concurrency", "Number of concurrent queries to issue.").
		Short('c').Default("1").Uint32Var(&benchmark.Concurrency)

	pApp.Flag("rate-limit", "Apply a global questions / second rate limit.").
		Short('l').Default("0").IntVar(&benchmark.Rate)

	pApp.Flag("rate-limit-worker", "Apply a questions / second rate limit for each concurrent worker specified by --concurrency option.").
		Default("0").IntVar(&benchmark.RateLimitWorker)

	pApp.Flag("query-per-conn", "Queries on a connection before creating a new one. 0: unlimited. Applicable for plain DNS and DoT, this option is not considered for DoH or DoQ.").
		Default("0").Int64Var(&benchmark.QperConn)

	pApp.Flag("recurse", "Allow DNS recursion. Enabled by default.").
		Short('r').Default("true").BoolVar(&benchmark.Recurse)

	pApp.Flag("probability", "Each provided hostname will be used with provided probability. Value 1 and above means that each hostname will be used by each concurrent benchmark goroutine. Useful for randomizing queries across benchmark goroutines.").
		Default("1").Float64Var(&benchmark.Probability)

	pApp.Flag("ednsopt", "code[:value], Specify EDNS option with code point code and optionally payload of value as a hexadecimal string. code must be an arbitrary numeric value.").
		Default("").StringVar(&benchmark.EdnsOpt)

	pApp.Flag("dnssec", "Allow DNSSEC (sets DO bit for all DNS requests to 1)").
		Default("false").BoolVar(&benchmark.DNSSEC)

	pApp.Flag("edns0", "Configures EDNS0 usage in DNS requests send by benchmark and configures EDNS0 buffer size to the specified value. When 0 is configured, then EDNS0 is not used.").
		Default("0").Uint16Var(&benchmark.Edns0)

	pApp.Flag("tcp", "Use TCP for DNS requests.").Default("false").BoolVar(&benchmark.TCP)

	pApp.Flag("dot", "Use DoT (DNS over TLS) for DNS requests.").Default("false").BoolVar(&benchmark.DOT)

	pApp.Flag("write", "write timeout.").Default("1s").DurationVar(&benchmark.WriteTimeout)

	pApp.Flag("read", "read timeout.").Default("3s").DurationVar(&benchmark.ReadTimeout)

	pApp.Flag("connect", "connect timeout.").Default("1s").DurationVar(&benchmark.ConnectTimeout)

	pApp.Flag("request", "request timeout.").Default("5s").DurationVar(&benchmark.RequestTimeout)

	pApp.Flag("codes", "Enable counting DNS return codes. Enabled by default.").
		Default("true").BoolVar(&benchmark.Rcodes)

	pApp.Flag("min", "Minimum value for timing histogram.").
		Default((time.Microsecond * 400).String()).DurationVar(&benchmark.HistMin)

	pApp.Flag("max", "Maximum value for timing histogram.").DurationVar(&benchmark.HistMax)

	pApp.Flag("precision", "Significant figure for histogram precision.").
		Default("1").PlaceHolder("[1-5]").IntVar(&benchmark.HistPre)

	pApp.Flag("distribution", "Display distribution histogram of timings to stdout. Enabled by default.").
		Default("true").BoolVar(&benchmark.HistDisplay)

	pApp.Flag("csv", "Export distribution to CSV.").
		Default("").PlaceHolder("/path/to/file.csv").StringVar(&benchmark.Csv)

	pApp.Flag("json", "Report benchmark results as JSON.").BoolVar(&benchmark.JSON)

	pApp.Flag("silent", "Disable stdout.").Default("false").BoolVar(&benchmark.Silent)

	pApp.Flag("color", "ANSI Color output. Enabled by default.").
		Default("true").BoolVar(&benchmark.Color)

	pApp.Flag("plot", "Plot benchmark results and export them to the directory.").
		Default("").PlaceHolder("/path/to/folder").StringVar(&benchmark.PlotDir)

	pApp.Flag("plotf", "Format of graphs. Supported formats: png, jpg.").
		Default("png").EnumVar(&benchmark.PlotFormat, "png", "jpg")

	pApp.Flag("doh-method", "HTTP method to use for DoH requests. Supported values: get, post.").
		Default("post").EnumVar(&benchmark.DohMethod, getMethod, postMethod)

	pApp.Flag("doh-protocol", "HTTP protocol to use for DoH requests. Supported values: 1.1, 2 and 3.").
		Default(http1Proto).EnumVar(&benchmark.DohProtocol, http1Proto, http2Proto, http3Proto)

	pApp.Flag("insecure", "Disables server TLS certificate validation. Applicable for DoT, DoH and DoQ.").
		Default("false").BoolVar(&benchmark.Insecure)

	pApp.Flag("duration", "Specifies for how long the benchmark should be executing, the benchmark will run for the specified time "+
		"while sending DNS requests in an infinite loop based on the data source. After running for the specified duration, the benchmark is canceled. "+
		"This option is exclusive with --number option. The duration is specified in GO duration format e.g. 10s, 15m, 1h.").
		PlaceHolder("1m").Short('d').DurationVar(&benchmark.Duration)

	pApp.Flag("progress", "Controls whether the progress bar is shown. Enabled by default.").
		Default("true").BoolVar(&benchmark.ProgressBar)

	pApp.Arg("queries", "Queries to issue. It can be a local file referenced using @<file-path>, for example @data/2-domains. "+
		"It can also be resource accessible using HTTP, like https://raw.githubusercontent.com/Tantalor93/dnspyre/master/data/1000-domains, in that "+
		"case, the file will be downloaded and saved in-memory. "+
		"These data sources can be combined, for example \"google.com @data/2-domains https://raw.githubusercontent.com/Tantalor93/dnspyre/master/data/2-domains\"").
		Required().StringsVar(&benchmark.Queries)

	info, ok := debug.ReadBuildInfo()
	if ok && len(Version) == 0 {
		Version = info.Main.Version
	}
}

// Execute starts main logic of command.
func Execute() {
	pApp.Version(Version)
	kingpin.MustParse(pApp.Parse(os.Args[1:]))

	sigsInt := make(chan os.Signal, 8)
	signal.Notify(sigsInt, syscall.SIGINT)

	defer close(sigsInt)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		_, ok := <-sigsInt
		if !ok {
			// standard exit based on channel close
			return
		}
		cancel()
		<-sigsInt
		os.Exit(1)
	}()

	start := time.Now()
	res, err := benchmark.Run(ctx)
	end := time.Now()

	if err != nil {
		errPrint(os.Stderr, "There was an error while starting benchmark: %s\n", err.Error())
	} else {
		if err := benchmark.PrintReport(os.Stdout, res, end.Sub(start)); err != nil {
			errPrint(os.Stderr, "There was an error while printing report: %s\n", err.Error())
		}
	}
}

func getSupportedDNSTypes() []string {
	keys := make([]string, 0, len(dns.StringToType))
	for k := range dns.StringToType {
		keys = append(keys, k)
	}
	return keys
}
