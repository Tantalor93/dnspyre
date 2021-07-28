package dnstrace

import (
	"fmt"
	"os"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

func plotHistogram(file string, times []float64) {
	var values plotter.Values
	values = append(values, times...)
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

func plotBoxPlot(file, server string, times []float64) {
	var values plotter.Values
	values = append(values, times...)
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
