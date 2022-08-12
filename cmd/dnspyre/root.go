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
	"github.com/tantalor93/dnspyre/v2/internal/sysutil"
)

var (
	// Version is set during release of project during build process.
	Version = "development"

	author = "Ondrej Benkovsky <obenky@gmail.com>"
)

var (
	pApp = kingpin.New("dnspyre", "A high QPS DNS benchmark.").Author(author)

	pServer = pApp.Flag("server", "DNS server IP:port to test. IPv6 is also supported, for example '[fddd:dddd::]:53'. "+
		"Also DoH servers are supported such as `https://1.1.1.1/dns-query`, when such server is provided, the benchmark automatically switches to the use of DoH. "+
		"Note that path on which DoH server handles requests (like `/dns-query`) has to be provided as well.").Short('s').Default("127.0.0.1").String()

	pTypes       = pApp.Flag("type", "Query type. Repeatable flag. If multiple query types are specified then each query will be duplicated for each type.").Short('t').Default("A").Enums(getSupportedDNSTypes()...)
	pCount       = pApp.Flag("number", "How many times the provided queries are repeated. Note that the total number of queries issued = types*number*concurrency*len(queries).").Short('n').Int64()
	pConcurrency = pApp.Flag("concurrency", "Number of concurrent queries to issue.").Short('c').Default("1").Uint32()

	pRate     = pApp.Flag("rate-limit", "Apply a global questions / second rate limit.").Short('l').Default("0").Int()
	pQperConn = pApp.Flag("query-per-conn", "Queries on a connection before creating a new one. 0: unlimited").Default("0").Int64()

	pRecurse = pApp.Flag("recurse", "Allow DNS recursion.").Short('r').Default("false").Bool()

	pProbability = pApp.Flag("probability", "Each hostname from file will be used with provided probability. Value 1 and above means that each hostname from file will be used by each concurrent benchmark goroutine. Useful for randomizing queries across benchmark goroutines.").Default("1").Float64()

	pUDPSize = pApp.Flag("edns0", "Enable EDNS0 with specified size.").Default("0").Uint16()
	pEdnsOpt = pApp.Flag("ednsopt", "code[:value], Specify EDNS option with code point code and optionally payload of value as a hexadecimal string. code must be arbitrary numeric value.").Default("").String()

	pTCP = pApp.Flag("tcp", "Use TCP fot DNS requests.").Default("false").Bool()
	pDOT = pApp.Flag("dot", "Use DoT for DNS requests.").Default("false").Bool()

	pWriteTimeout = pApp.Flag("write", "DNS write timeout.").Default("1s").Duration()
	pReadTimeout  = pApp.Flag("read", "DNS read timeout.").Default(dnsTimeout.String()).Duration()

	pRCodes = pApp.Flag("codes", "Enable counting DNS return codes.").Default("true").Bool()

	pHistMin     = pApp.Flag("min", "Minimum value for timing histogram.").Default((time.Microsecond * 400).String()).Duration()
	pHistMax     = pApp.Flag("max", "Maximum value for histogram.").Default(dnsTimeout.String()).Duration()
	pHistPre     = pApp.Flag("precision", "Significant figure for histogram precision.").Default("1").PlaceHolder("[1-5]").Int()
	pHistDisplay = pApp.Flag("distribution", "Display distribution histogram of timings to stdout.").Default("true").Bool()

	pCsv = pApp.Flag("csv", "Export distribution to CSV.").Default("").PlaceHolder("/path/to/file.csv").String()

	pSilent = pApp.Flag("silent", "Disable stdout.").Default("false").Bool()
	pColor  = pApp.Flag("color", "ANSI Color output.").Default("true").Bool()

	pPlotDir    = pApp.Flag("plot", "Plot benchmark results and export them to directory.").Default("").PlaceHolder("/path/to/folder").String()
	pPlotFormat = pApp.Flag("plotf", "Format of graphs. Supported formats png, svg, pdf.").Default("png").Enum("png", "svg", "pdf")

	pDoHmethod   = pApp.Flag("doh-method", "HTTP method to use for DoH requests. Supported values: get, post.").Default("post").Enum("get", "post")
	pDoHProtocol = pApp.Flag("doh-protocol", "HTTP protocol to use for DoH requests. Supported values: 1.1, 2.").Default("1.1").Enum("1.1", "2")

	pDuration = pApp.Flag("duration", "Specifies for how long the benchmark should be executing, the benchmark will run for the specified time "+
		"while sending DNS requests in infinite loop based on data source. After running for specified duration, the benchmark is cancelled. "+
		"This option is exclusive with --number option. The duration is specified in GO duration format e.g. 10s, 15m, 1h.").
		PlaceHolder("1m").Short('d').Duration()

	pQueries = pApp.Arg("queries", "Queries to issue. Can be local file referenced using @<file-path>, for example @data/2-domains."+
		"Can also be resource accessible using HTTP, like https://raw.githubusercontent.com/Tantalor93/dnspyre/master/data/1000-domains, in that "+
		"case the file will be downloaded and saved inmemory.").Required().Strings()
)

const (
	fileNoBuffer = 9 // app itself needs about 9 for libs
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
		Duration:     *pDuration,
		Queries:      *pQueries,
	}

	lim, err := sysutil.RlimitStack()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Cannot check limit of number of files. Skipping check. Please make sure it is sufficient manually.", err)
	} else {
		needed := uint64(bench.Concurrency) + uint64(fileNoBuffer)
		if lim < needed {
			fmt.Fprintf(os.Stderr, "Current process limit for number of files is %d and insufficient for level of requested concurrency.", lim)
			os.Exit(1)
		}
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
	res := bench.Run(ctx)
	end := time.Now()

	bench.PrintReport(res, end.Sub(start))
}

func getSupportedDNSTypes() []string {
	keys := make([]string, 0, len(dns.StringToType))
	for k := range dns.StringToType {
		keys = append(keys, k)
	}
	return keys
}
