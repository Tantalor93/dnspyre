package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/miekg/dns"
)

var (
	// Version is set during release of project during build process.
	Version = "development"

	author = "Ondrej Benkovsky <obenky@gmail.com>"
)

var (
	pApp = kingpin.New("dnspyre", "A high QPS DNS benchmark.").Author(author)

	benchmark Benchmark
)

func init() {
	pApp.Flag("server", "DNS server IP:port to test. IPv6 is also supported, for example '[fddd:dddd::]:53'. "+
		"DoH (DNS over HTTPS) servers are supported such as `https://1.1.1.1/dns-query`, when such server is provided, the benchmark automatically switches to the use of DoH. "+
		"Note that path on which the DoH server handles requests (like `/dns-query`) has to be provided as well. DoQ (DNS over QUIC) servers are also supported, such as `quic://dns.adguard-dns.com`, "+
		"when such server is provided the benchmark switches to the use of DoQ.").Short('s').Default("127.0.0.1").StringVar(&benchmark.Server)

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

	pApp.Flag("edns0", "Enable EDNS0 with specified size.").Default("0").Uint16Var(&benchmark.UDPSize)

	pApp.Flag("ednsopt", "code[:value], Specify EDNS option with code point code and optionally payload of value as a hexadecimal string. code must be an arbitrary numeric value.").
		Default("").StringVar(&benchmark.EdnsOpt)

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
		Default("post").EnumVar(&benchmark.DohMethod, "get", "post")

	pApp.Flag("doh-protocol", "HTTP protocol to use for DoH requests. Supported values: 1.1, 2 and 3.").
		Default("1.1").EnumVar(&benchmark.DohProtocol, "1.1", "2", "3")

	pApp.Flag("insecure", "Disables server TLS certificate validation. Applicable for DoT, DoH and DoQ.").
		Default("false").BoolVar(&benchmark.Insecure)

	pApp.Flag("duration", "Specifies for how long the benchmark should be executing, the benchmark will run for the specified time "+
		"while sending DNS requests in an infinite loop based on the data source. After running for the specified duration, the benchmark is canceled. "+
		"This option is exclusive with --number option. The duration is specified in GO duration format e.g. 10s, 15m, 1h.").
		PlaceHolder("1m").Short('d').DurationVar(&benchmark.Duration)

	pApp.Arg("queries", "Queries to issue. It can be a local file referenced using @<file-path>, for example @data/2-domains. "+
		"It can also be resource accessible using HTTP, like https://raw.githubusercontent.com/Tantalor93/dnspyre/master/data/1000-domains, in that "+
		"case, the file will be downloaded and saved in-memory.").Required().StringsVar(&benchmark.Queries)
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
		fmt.Fprintf(os.Stderr, "\nCancelling benchmark ^C, again to terminate now.\n")
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
