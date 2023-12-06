package cmd

import (
	"fmt"
	"image/color"
	"os"
	"sort"

	"github.com/miekg/dns"
	"github.com/montanaflynn/stats"
	"go-hep.org/x/hep/hplot"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

func plotHistogramLatency(file string, times []Datapoint) {
	if len(times) == 0 {
		// nothing to plot
		return
	}
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
	p.X.Label.Text = "Latencies (ms)"
	p.X.Tick.Marker = hplot.Ticks{N: 3, Format: "%.0f"}
	p.Y.Label.Text = "Number of requests"
	p.Y.Tick.Marker = hplot.Ticks{N: 3, Format: "%.0f"}
	hist.FillColor = color.RGBA{R: 175, G: 238, B: 238}
	p.Add(hist)

	if err := p.Save(6*vg.Inch, 6*vg.Inch, file); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to save plot.", err)
	}
}

func plotBoxPlotLatency(file, server string, times []Datapoint) {
	if len(times) == 0 {
		// nothing to plot
		return
	}
	var values plotter.Values
	for _, v := range times {
		values = append(values, v.Duration)
	}
	p := plot.New()
	p.Title.Text = "Latencies distribution"
	p.Y.Label.Text = "Latencies (ms)"
	p.Y.Tick.Marker = hplot.Ticks{N: 3, Format: "%.0f"}
	p.NominalX(server)

	boxplot, err := plotter.NewBoxPlot(vg.Length(120), 0, values)
	if err != nil {
		panic(err)
	}
	boxplot.FillColor = color.RGBA{R: 127, G: 188, B: 165, A: 1}
	p.Add(boxplot)

	if err := p.Save(6*vg.Inch, 6*vg.Inch, file); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to save plot.", err)
	}
}

func plotResponses(file string, rcodes map[int]int64) {
	if len(rcodes) == 0 {
		// nothing to plot
		return
	}
	sortedKeys := make([]int, 0)
	for k := range rcodes {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Ints(sortedKeys)

	colors := []color.Color{
		color.RGBA{R: 122, G: 195, B: 106},
		color.RGBA{R: 241, G: 90, B: 96},
		color.RGBA{R: 90, G: 155, B: 212},
		color.RGBA{R: 250, G: 167, B: 91},
		color.RGBA{R: 158, G: 103, B: 171},
		color.RGBA{R: 206, G: 112, B: 88},
		color.RGBA{R: 215, G: 127, B: 180},
	}
	colors = append(colors, plotutil.DarkColors...)

	p := plot.New()
	p.Title.Text = "Response code distribution"
	p.NominalX("Response codes")

	width := vg.Points(40)

	c := 0
	off := -vg.Length(len(rcodes)/2) * width
	for _, v := range sortedKeys {
		bar, err := plotter.NewBarChart(plotter.Values{float64(rcodes[v])}, width)
		if err != nil {
			panic(err)
		}
		p.Legend.Add(dns.RcodeToString[v], bar)
		bar.Color = colors[c%len(colors)]
		bar.Offset = off
		p.Add(bar)
		c++
		off += width
	}

	p.Y.Label.Text = "Number of requests"
	p.Y.Tick.Marker = hplot.Ticks{N: 3, Format: "%.0f"}
	p.Legend.Top = true

	if err := p.Save(6*vg.Inch, 6*vg.Inch, file); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to save plot.", err)
	}
}

func plotLineThroughput(file string, times []Datapoint) {
	if len(times) == 0 {
		// nothing to plot
		return
	}
	var values plotter.XYs
	m := make(map[int64]int64)

	if len(times) != 0 {
		first := times[0].Start.Unix()

		for _, v := range times {
			unix := v.Start.Unix() - first
			if _, ok := m[unix]; !ok {
				m[unix] = 0
			}
			m[unix]++
		}
	}

	for k, v := range m {
		values = append(values, plotter.XY{X: float64(k), Y: float64(v)})
	}

	sort.SliceStable(values, func(i, j int) bool {
		return values[i].X < values[j].X
	})

	p := plot.New()
	p.Title.Text = "Throughput per second"
	p.X.Label.Text = "Time of test (s)"
	p.X.Tick.Marker = hplot.Ticks{N: 3, Format: "%.0f"}
	p.Y.Label.Text = "Number of requests (per sec)"
	p.Y.Tick.Marker = hplot.Ticks{N: 3, Format: "%.0f"}

	l, err := plotter.NewLine(values)
	l.FillColor = color.RGBA{R: 175, G: 238, B: 238}
	if err != nil {
		panic(err)
	}
	p.Add(l)

	scatter, err := plotter.NewScatter(values)
	if err != nil {
		panic(err)
	}
	p.Add(scatter)

	if err := p.Save(6*vg.Inch, 6*vg.Inch, file); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to save plot.", err)
	}
}

type latencyMeasurements struct {
	p99 float64
	p95 float64
	p90 float64
	p50 float64
}

func plotLineLatencies(file string, times []Datapoint) {
	if len(times) == 0 {
		// nothing to plot
		return
	}
	m := make(map[int64]latencyMeasurements)

	timings := make([]float64, 0)
	if len(times) != 0 {
		first := times[0].Start.Unix()
		last := int64(0)

		for _, v := range times {
			unix := v.Start.Unix() - first
			if _, ok := m[unix]; !ok {
				m[unix] = latencyMeasurements{}
			}
			if unix != last {
				p99, err := stats.Percentile(timings, 99)
				if err != nil {
					panic(err)
				}
				p95, err := stats.Percentile(timings, 95)
				if err != nil {
					panic(err)
				}
				p90, err := stats.Percentile(timings, 90)
				if err != nil {
					panic(err)
				}
				p50, err := stats.Percentile(timings, 50)
				if err != nil {
					panic(err)
				}
				measure := m[unix]
				measure.p99 = p99
				measure.p95 = p95
				measure.p90 = p90
				measure.p50 = p50
				m[unix] = measure
				last = unix
			}
			timings = append(timings, v.Duration)
		}
	}

	var p99values plotter.XYs
	var p95values plotter.XYs
	var p90values plotter.XYs
	var p50values plotter.XYs

	for k, v := range m {
		p99values = append(p99values, plotter.XY{X: float64(k), Y: v.p99})
		p95values = append(p95values, plotter.XY{X: float64(k), Y: v.p95})
		p90values = append(p90values, plotter.XY{X: float64(k), Y: v.p90})
		p50values = append(p50values, plotter.XY{X: float64(k), Y: v.p50})
	}

	less := func(xys plotter.XYs) func(i, j int) bool {
		return func(i, j int) bool {
			return xys[i].X < xys[j].X
		}
	}

	sort.SliceStable(p99values, less(p99values))
	sort.SliceStable(p95values, less(p95values))
	sort.SliceStable(p90values, less(p90values))
	sort.SliceStable(p50values, less(p50values))

	p := plot.New()
	p.Title.Text = "Response latencies"
	p.X.Label.Text = "Time of test (s)"
	p.Y.Label.Text = "Latency (ms)"

	plotLine(p, p99values, plotutil.DarkColors[0], plotutil.SoftColors[0], "p99")
	plotLine(p, p95values, plotutil.DarkColors[1], plotutil.SoftColors[1], "p95")
	plotLine(p, p90values, plotutil.DarkColors[2], plotutil.SoftColors[2], "p90")
	plotLine(p, p50values, plotutil.DarkColors[3], plotutil.SoftColors[3], "p50")

	p.Legend.Top = true

	if err := p.Save(6*vg.Inch, 6*vg.Inch, file); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to save plot.", err)
	}
}

func plotErrorRate(file string, times []ErrorDatapoint) {
	if len(times) == 0 {
		// nothing to plot
		return
	}
	var values plotter.XYs
	m := make(map[int64]int64)

	if len(times) != 0 {
		first := times[0].Start.Unix()

		for _, v := range times {
			unix := v.Start.Unix() - first
			if _, ok := m[unix]; !ok {
				m[unix] = 0
			}
			m[unix]++
		}
	}

	for k, v := range m {
		values = append(values, plotter.XY{X: float64(k), Y: float64(v)})
	}

	sort.SliceStable(values, func(i, j int) bool {
		return values[i].X < values[j].X
	})

	p := plot.New()
	p.Title.Text = "Error rate over time"
	p.X.Label.Text = "Time of test (s)"
	p.X.Tick.Marker = hplot.Ticks{N: 3, Format: "%.0f"}
	p.Y.Label.Text = "Number of errors (per sec)"
	p.Y.Tick.Marker = hplot.Ticks{N: 3, Format: "%.0f"}

	l, err := plotter.NewLine(values)
	l.FillColor = plotutil.SoftColors[0]
	if err != nil {
		panic(err)
	}
	p.Add(l)

	scatter, err := plotter.NewScatter(values)
	if err != nil {
		panic(err)
	}
	p.Add(scatter)

	if err := p.Save(6*vg.Inch, 6*vg.Inch, file); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to save plot.", err)
	}
}

func plotLine(p *plot.Plot, values plotter.XYs, color color.Color, fill color.Color, name string) {
	l, err := plotter.NewLine(values)
	l.Color = color
	if err != nil {
		panic(err)
	}
	l.FillColor = fill
	p.Add(l)
	p.Legend.Add(name, l)
	scatter, err := plotter.NewScatter(values)
	if err != nil {
		panic(err)
	}
	scatter.Color = color
	p.Add(scatter)
}
