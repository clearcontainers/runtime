//
// Copyright (c) 2017 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package main

import (
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/01org/ciao/templateutils"
)

type stock struct {
	Ticker    string
	Name      string
	LastTrade time.Time
	Current   float64
	High      float64
	Low       float64
	Volume    int
}

// Cols that creates a new type that execludes certain fields from a struct
// Table that generates a new Table
// Percentage that generates a Percentage
// head, tail and rows that act on slices and return an empty slice if no data is available
// ascending which returns a slice of numbers that can be used for iteration purposes
// sorting

var fictionalStocks = []stock{
	{"BCOM.L", "Big Company", time.Date(2017, time.March, 17, 11, 01, 00, 00, time.UTC), 120.23, 150.00, 119.00, 7500000},
	{"SMAL.L", "Small Company", time.Date(2017, time.March, 17, 10, 59, 00, 00, time.UTC), 1.06, 1.06, 1.10, 750},
	{"MEDI.L", "Medium Company", time.Date(2017, time.March, 17, 12, 23, 00, 00, time.UTC), 77.00, 75.11, 81.12, 300122},
	{"PICO.L", "Tiny Corp", time.Date(2017, time.March, 16, 16, 01, 00, 00, time.UTC), 0.59, 0.57, 0.63, 155},
	{"HENT.L", "Happy Enterprises", time.Date(2017, time.March, 17, 9, 45, 00, 00, time.UTC), 756.11, 600.00, 10000, 6395624278},
	{"LONL.L", "Lonely Systems", time.Date(2017, time.March, 17, 13, 45, 00, 00, time.UTC), 1245.00, 1200.00, 1245.00, 19003},
}

var code string

var cfg *templateutils.Config

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s -f [stocks]\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, templateutils.GenerateUsageDecorated("f", fictionalStocks, cfg))
	}
	flag.StringVar(&code, "f", "", "string containing the template code to execute")
	cfg = templateutils.NewConfig(templateutils.OptAllFns)

	if err := cfg.AddCustomFn(sumVolume, "sumVolume", sumVolumeHelp); err != nil {
		panic(err)
	}
}

func sumVolume(stocks []stock) int {
	total := 0
	for _, s := range stocks {
		total += s.Volume
	}
	return total
}

const sumVolumeHelp = `- sumVolume computes the total volume all stocks in a []stock slice`

func stocks() error {
	if code != "" {
		err := templateutils.OutputToTemplate(os.Stdout, "stocks", code, fictionalStocks, cfg)
		if err != nil {
			return fmt.Errorf("Unable to execute template : %v", err)
		}

		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 12, 8, 1, ' ', 0)
	fmt.Fprintln(w, "Ticker\tName\tLast Trade\tCurrent\tDay High\tDay Low\tVolume\t")
	for _, s := range fictionalStocks {
		fmt.Fprintf(w, "%s\t%s\t%s\t%.2f\t%.2f\t%.2f\t%d\t\n",
			s.Ticker, s.Name, s.LastTrade.Format(time.Kitchen), s.Current, s.High, s.Low, s.Volume)
	}
	_ = w.Flush()

	return nil
}

var commands = map[string]func() error{
	"stocks": stocks,
}

func main() {
	flag.Parse()

	if len(flag.Args()) != 1 {
		flag.Usage()
		os.Exit(1)
	}

	fn := commands[flag.Args()[0]]
	if fn == nil {
		flag.Usage()
		os.Exit(1)
	}

	if err := fn(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
