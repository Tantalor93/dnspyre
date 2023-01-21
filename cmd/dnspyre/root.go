package dnspyre

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/miekg/dns"
)

var (
	// Version is set during release of project during build process.
	Version = "development"

	author = "Ondrej Benkovsky <obenky@gmail.com>"
)

var (
	pApp = kingpin.New("dnspyre", "A high QPS DNS benchmark.").Author(author)

	pServer = pApp.Flag("server", "DNS server IP:port to test. IPv6 is also supported, for example '[fddd:dddd::]:53'. "+
		"Also, DoH (DNS over HTTPS) servers are supported such as `https://1.1.1.1/dns-query`, when such server is provided, the benchmark automatically switches to the use of DoH. "+
		"Note that path on which the DoH server handles requests (like `/dns-query`) has to be provided as well.").Short('s').Default("127.0.0.1").String()

	pTypes       = pApp.Flag("type", "Query type. Repeatable flag. If multiple query types are specified then each query will be duplicated for each type.").Short('t').Default("A").Enums(getSupportedDNSTypes()...)
	pCount       = pApp.Flag("number", "How many times the provided queries are repeated. Note that the total number of queries issued = types*number*concurrency*len(queries).").Short('n').Int64()
	pConcurrency = pApp.Flag("concurrency", "Number of concurrent queries to issue.").Short('c').Default("1").Uint32()

	pRate     = pApp.Flag("rate-limit", "Apply a global questions / second rate limit.").Short('l').Default("0").Int()
	pQperConn = pApp.Flag("query-per-conn", "Queries on a connection before creating a new one. 0: unlimited. Applicable for plain DNS and DoT, this option is not considered for DoH.").Default("0").Int64()

	pRecurse = pApp.Flag("recurse", "Allow DNS recursion.").Short('r').Default("false").Bool()

	pProbability = pApp.Flag("probability", "Each provided hostname will be used with provided probability. Value 1 and above means that each hostname will be used by each concurrent benchmark goroutine. Useful for randomizing queries across benchmark goroutines.").Default("1").Float64()

	pUDPSize = pApp.Flag("edns0", "Enable EDNS0 with specified size.").Default("0").Uint16()
	pEdnsOpt = pApp.Flag("ednsopt", "code[:value], Specify EDNS option with code point code and optionally payload of value as a hexadecimal string. code must be an arbitrary numeric value.").Default("").String()

	pTCP = pApp.Flag("tcp", "Use TCP for DNS requests.").Default("false").Bool()
	pDOT = pApp.Flag("dot", "Use DoT (DNS over TLS) for DNS requests.").Default("false").Bool()

	pWriteTimeout = pApp.Flag("write", "DNS write timeout.").Default("1s").Duration()
	pReadTimeout  = pApp.Flag("read", "DNS read timeout.").Default(dnsTimeout.String()).Duration()

	pRCodes = pApp.Flag("codes", "Enable counting DNS return codes. Enabled by default. Specifying --no-codes disables code counting.").Default("true").Bool()

	pHistMin     = pApp.Flag("min", "Minimum value for timing histogram.").Default((time.Microsecond * 400).String()).Duration()
	pHistMax     = pApp.Flag("max", "Maximum value for timing histogram.").Default(dnsTimeout.String()).Duration()
	pHistPre     = pApp.Flag("precision", "Significant figure for histogram precision.").Default("1").PlaceHolder("[1-5]").Int()
	pHistDisplay = pApp.Flag("distribution", "Display distribution histogram of timings to stdout. Enabled by default. Specifying --no-distribution disables histogram display.").Default("true").Bool()

	pCsv = pApp.Flag("csv", "Export distribution to CSV.").Default("").PlaceHolder("/path/to/file.csv").String()

	pSilent = pApp.Flag("silent", "Disable stdout.").Default("false").Bool()
	pColor  = pApp.Flag("color", "ANSI Color output. Enabled by default. By specifying --no-color disables coloring.").Default("true").Bool()

	pPlotDir    = pApp.Flag("plot", "Plot benchmark results and export them to the directory.").Default("").PlaceHolder("/path/to/folder").String()
	pPlotFormat = pApp.Flag("plotf", "Format of graphs. Supported formats: png, jpg.").Default("png").Enum("png", "jpg")

	pDoHmethod   = pApp.Flag("doh-method", "HTTP method to use for DoH requests. Supported values: get, post.").Default("post").Enum("get", "post")
	pDoHProtocol = pApp.Flag("doh-protocol", "HTTP protocol to use for DoH requests. Supported values: 1.1, 2.").Default("1.1").Enum("1.1", "2")

	pInsecure = pApp.Flag("insecure", "Disables server TLS certificate validation. Applicable both for DoT and DoH.").Default("false").Bool()

	pDuration = pApp.Flag("duration", "Specifies for how long the benchmark should be executing, the benchmark will run for the specified time "+
		"while sending DNS requests in an infinite loop based on the data source. After running for the specified duration, the benchmark is canceled. "+
		"This option is exclusive with --number option. The duration is specified in GO duration format e.g. 10s, 15m, 1h.").
		PlaceHolder("1m").Short('d').Duration()

	pQueries = pApp.Arg("queries", "Queries to issue. It can be a local file referenced using @<file-path>, for example @data/2-domains. "+
		"It can also be resource accessible using HTTP, like https://raw.githubusercontent.com/Tantalor93/dnspyre/master/data/1000-domains, in that "+
		"case, the file will be downloaded and saved in-memory.").Required().Strings()
)

// Execute starts main logic of command.
func Execute() {
	pApp.Version(Version)
	kingpin.MustParse(pApp.Parse(os.Args[1:]))

	bench := Benchmark{
		Server:       *pServer,
		Types:        *pTypes,
		Count:        *pCount,
		Concurrency:  *pConcurrency,
		Rate:         *pRate,
		QperConn:     *pQperConn,
		Recurse:      *pRecurse,
		Probability:  *pProbability,
		UDPSize:      *pUDPSize,
		EdnsOpt:      *pEdnsOpt,
		TCP:          *pTCP,
		DOT:          *pDOT,
		WriteTimeout: *pWriteTimeout,
		ReadTimeout:  *pReadTimeout,
		Rcodes:       *pRCodes,
		HistMin:      *pHistMin,
		HistMax:      *pHistMax,
		HistPre:      *pHistPre,
		HistDisplay:  *pHistDisplay,
		Csv:          *pCsv,
		Silent:       *pSilent,
		Color:        *pColor,
		PlotDir:      *pPlotDir,
		PlotFormat:   *pPlotFormat,
		DohMethod:    *pDoHmethod,
		DohProtocol:  *pDoHProtocol,
		Insecure:     *pInsecure,
		Duration:     *pDuration,
		Queries:      *pQueries,
	}

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

	// get going
	rand.Seed(time.Now().UnixNano())

	start := time.Now()
	res, err := bench.Run(ctx)
	end := time.Now()

	if err != nil {
		errPrint(os.Stderr, fmt.Sprintf("There was an error while starting benchmark: %s\n", err.Error()))
	} else {
		if err := bench.PrintReport(res, end.Sub(start)); err != nil {
			errPrint(os.Stderr, fmt.Sprintf("There was an error while printing report: %s\n", err.Error()))
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
