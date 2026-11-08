// Command nearestidle runs the simulator with a custom matcher: each waiting
// request gets the closest idle car by straight-line distance.
//
//	go run ./examples/nearestidle -settings examples/nearestidle/settings.toml \
//	  -supply benchmarks/austin_weekday_100/supply.csv \
//	  -requests benchmarks/austin_weekday_100/requests.csv -output out/nearestidle
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/realjonathanhester/ride-hailing-simulator/model"
	"github.com/realjonathanhester/ride-hailing-simulator/routing"
	"github.com/realjonathanhester/ride-hailing-simulator/runner"
)

// NearestIdle implements model.Matcher.
type NearestIdle struct{}

// Match is called on every trigger (request, supply online/offline, batch
// tick). It returns the plans to replace; cars not mentioned keep theirs.
func (NearestIdle) Match(in model.MatchInput) []model.PlanChange {
	taken := map[string]bool{}
	var changes []model.PlanChange
	for _, r := range in.Requests { // sorted by request_id
		if r.Status != model.StatusRequested {
			continue
		}
		best, bestMeters := "", int64(-1)
		for _, s := range in.Supplies { // sorted by supply_id
			if taken[s.SupplyID] || len(s.Plan) > 0 || s.State == model.StateGoingOffline {
				continue
			}
			if d := routing.HaversineMeters(s.Location, r.Origin); bestMeters < 0 || d < bestMeters {
				best, bestMeters = s.SupplyID, d
			}
		}
		if best == "" {
			continue
		}
		taken[best] = true
		changes = append(changes, model.PlanChange{SupplyID: best, Plan: []model.Stop{
			{RequestID: r.RequestID, Type: model.Pickup},
			{RequestID: r.RequestID, Type: model.Dropoff},
		}})
	}
	return changes
}

func main() {
	if err := runner.RegisterMatcher("nearest_idle", func() (model.Matcher, error) { return NearestIdle{}, nil }); err != nil {
		panic(err)
	}
	settings := flag.String("settings", "", "settings TOML with [hooks] matching = \"nearest_idle\"")
	supply := flag.String("supply", "", "supply CSV")
	requests := flag.String("requests", "", "requests CSV")
	output := flag.String("output", "", "output directory")
	flag.Parse()
	if _, err := runner.RunFiles(*settings, runner.Paths{Supply: *supply, Requests: *requests, OutputDirectory: *output}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
