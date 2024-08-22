package dnsbench_test

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tantalor93/dnspyre/v3/pkg/dnsbench"
)

func assertResult(t *testing.T, rs []*dnsbench.ResultStats) {
	t.Helper()
	if assert.Len(t, rs, 2, "Run(ctx) rstats") {
		rs0 := rs[0]
		rs1 := rs[1]
		assertResultStats(t, rs0)
		assertResultStats(t, rs1)
		assertTimings(t, rs0)
		assertTimings(t, rs1)
	}
}

func assertResultStats(t *testing.T, rs *dnsbench.ResultStats) {
	t.Helper()
	assert.NotNil(t, rs.Hist, "Run(ctx) rstats histogram")

	if assert.NotNil(t, rs.Codes, "Run(ctx) rstats codes") {
		assert.EqualValues(t, 2, rs.Codes[0], "Run(ctx) rstats codes NOERROR, state:"+fmt.Sprint(rs.Codes))
	}

	if assert.NotNil(t, rs.Qtypes, "Run(ctx) rstats qtypes") {
		assert.EqualValues(t, 1, rs.Qtypes[dns.TypeToString[dns.TypeA]], "Run(ctx) rstats qtypes A, state:"+fmt.Sprint(rs.Codes))
		assert.EqualValues(t, 1, rs.Qtypes[dns.TypeToString[dns.TypeAAAA]], "Run(ctx) rstats qtypes AAAA, state:"+fmt.Sprint(rs.Codes))
	}

	assert.EqualValues(t, 2, rs.Counters.Total, "Run(ctx) total counter")
	assert.Zero(t, rs.Counters.IOError, "error counter")
	assert.EqualValues(t, 2, rs.Counters.Success, "Run(ctx) success counter")
	assert.Zero(t, rs.Counters.IDmismatch, "Run(ctx) mismatch counter")
	assert.Zero(t, rs.Counters.Truncated, "Run(ctx) truncated counter")
}

func assertTimings(t *testing.T, rs *dnsbench.ResultStats) {
	t.Helper()
	if assert.Len(t, rs.Timings, 2, "Run(ctx) rstats timings") {
		t0 := rs.Timings[0]
		t1 := rs.Timings[1]
		assert.NotZero(t, t0.Duration, "Run(ctx) rstats timings duration")
		assert.NotZero(t, t0.Start, "Run(ctx) rstats timings start")
		assert.NotZero(t, t1.Duration, "Run(ctx) rstats timings duration")
		assert.NotZero(t, t1.Start, "Run(ctx) rstats timings start")
	}
}

type requestLog struct {
	worker    int
	requestid int
	qname     string
	qtype     string
	respid    int
	rcode     string
	respflags string
	err       string
	duration  time.Duration
}

func assertRequestLogStructure(t *testing.T, reader io.Reader) {
	t.Helper()
	pattern := `.*worker:\[(.*)\] reqid:\[(.*)\] qname:\[(.*)\] qtype:\[(.*)\] respid:\[(.*)\] rcode:\[(.*)\] respflags:\[(.*)\] err:\[(.*)\] duration:\[(.*)\]$`
	regex := regexp.MustCompile(pattern)
	scanner := bufio.NewScanner(reader)
	var requestLogs []requestLog
	for scanner.Scan() {
		line := scanner.Text()

		matches := regex.FindStringSubmatch(line)
		require.Len(t, matches, 10, "request log does not have expected structure")

		workerID, err := strconv.Atoi(matches[1])
		require.NoError(t, err, "worker ID is not number")

		requestID, err := strconv.Atoi(matches[2])
		require.NoError(t, err, "request ID is not number")

		qname := matches[3]
		qtype := matches[4]

		respID, err := strconv.Atoi(matches[5])
		require.NoError(t, err, "response ID is not number")

		rcode := matches[6]
		respflags := matches[7]
		errstr := matches[8]

		dur, err := time.ParseDuration(matches[9])
		require.NoError(t, err, "duration is not correct time duration")

		requestLogs = append(requestLogs, requestLog{
			worker:    workerID,
			requestid: requestID,
			qname:     qname,
			qtype:     qtype,
			respid:    respID,
			rcode:     rcode,
			respflags: respflags,
			err:       errstr,
			duration:  dur,
		})
	}

	workerIDs := map[int]int{}
	qtypes := map[string]int{}

	for _, v := range requestLogs {
		workerIDs[v.worker]++
		qtypes[v.qtype]++

		assert.Equal(t, "example.org.", v.qname)
		assert.NotZero(t, v.requestid)
		assert.NotZero(t, v.respid)
		assert.Equal(t, "NOERROR", v.rcode)
		assert.Equal(t, "qr rd", v.respflags)
		assert.Equal(t, "<nil>", v.err)
		assert.NotZero(t, v.duration)
	}
	assert.Equal(t, map[int]int{0: 2, 1: 2}, workerIDs)
	assert.Equal(t, map[string]int{"AAAA": 2, "A": 2}, qtypes)
}

// A returns an A record from rr. It panics on errors.
func A(rr string) *dns.A { r, _ := dns.NewRR(rr); return r.(*dns.A) }
