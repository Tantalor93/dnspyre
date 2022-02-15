package dnstrace

import (
	"fmt"
	"image/color"
	"os"
	"sort"

	"github.com/miekg/dns"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

func plotHistogramLatency(file string, times []Datapoint) {
	var values plotter.Values
	for _, v := range times {
		values = append(values, v.Duration)
	}
	p := plot.New()
	p.Title.Text = "Latencies distribution"

	hist, err := plotter.NewHist(values, 16)
	if err != nil {
		panic(err)
	}
	p.X.Label.Text = "latencies (ms)"
	p.Y.Label.Text = "count"
	p.Add(hist)

	if err := p.Save(6*vg.Inch, 6*vg.Inch, file); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to save plot.", err)
	}
}

func plotBoxPlotLatency(file, server string, times []Datapoint) {
	var values plotter.Values
	for _, v := range times {
		values = append(values, v.Duration)
	}
	p := plot.New()
	p.Title.Text = "Latencies distribution"
	p.Y.Label.Text = "latencies (ms)"
	p.NominalX(server)

	boxplot, err := plotter.NewBoxPlot(vg.Length(120), 0, values)
	if err != nil {
		panic(err)
	}
	p.Add(boxplot)

	if err := p.Save(6*vg.Inch, 6*vg.Inch, file); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to save plot.", err)
	}
}

func plotLineLatency(file string, times []Datapoint) {
	var values plotter.XYs
	for i, v := range times {
		values = append(values, plotter.XY{X: float64(i), Y: v.Duration})
	}
	p := plot.New()
	p.Title.Text = "Latencies progression during benchmark test"
	p.X.Label.Text = "number of requests since the start"
	p.Y.Label.Text = "latencies (ms)"

	l, err := plotter.NewLine(values)
	if err != nil {
		panic(err)
	}
	p.Add(l)

	if err := p.Save(6*vg.Inch, 6*vg.Inch, file); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to save plot.", err)
	}
}

func plotResponses(file string, rcodes map[int]int64) {
	var values plotter.Values
	var names []string
	for k, v := range rcodes {
		values = append(values, float64(v))
		names = append(names, dns.RcodeToString[k])
	}
	p := plot.New()
	p.Title.Text = "Responses distribution"

	barchart, err := plotter.NewBarChart(values, vg.Length(60))
	if err != nil {
		panic(err)
	}
	p.Add(barchart)
	p.NominalX(names...)
	p.Y.Label.Text = "count"
	barchart.Color = color.Gray{Y: 128}

	if err := p.Save(6*vg.Inch, 6*vg.Inch, file); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to save plot.", err)
	}
}

func plotLineThroughput(file string, times []Datapoint) {
	var values plotter.XYs
	m := make(map[int64]int64)
	for _, v := range times {
		unix := v.Start.Unix()
		if _, ok := m[unix]; !ok {
			m[unix] = 0
		}
		m[unix]++
	}

	for k, v := range m {
		values = append(values, plotter.XY{X: float64(k), Y: float64(v)})
	}

	sort.SliceStable(values, func(i, j int) bool {
		return values[i].X < values[j].X
	})

	p := plot.New()
	p.Title.Text = "Throughput per second"
	p.X.Label.Text = "time of test (s)"
	p.Y.Label.Text = "number of requests (per sec)"

	l, err := plotter.NewLine(values)
	if err != nil {
		panic(err)
	}
	p.Add(l)

	if err := p.Save(6*vg.Inch, 6*vg.Inch, file); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to save plot.", err)
	}
}
