package tests

import (
	"math"
	"reflect"
	"testing"

	"github.com/realjonathanhester/ride-hailing-simulator/matching"
	"github.com/realjonathanhester/ride-hailing-simulator/model"
	"github.com/realjonathanhester/ride-hailing-simulator/polyline"
	"github.com/realjonathanhester/ride-hailing-simulator/simulator"
)

const minimumOutputSpacingMeters = 20

// denseRouter is the manhattan router with a geometry point every 5 m, like
// a real road router's detailed polylines.
type denseRouter struct{}

func (denseRouter) Route(start model.Location, waypoints []model.Location, departAt int64) ([]model.Leg, error) {
	legs, err := manhattanRouter{bent: true}.Route(start, waypoints, departAt)
	for i := range legs {
		for j, segment := range legs[i].Segments {
			dense := []model.Location{segment.Polyline[0]}
			for k := 1; k < len(segment.Polyline); k++ {
				a, b := segment.Polyline[k-1], segment.Polyline[k]
				steps := int(math.Ceil(polyline.HaversineMeters(a, b) / 5))
				for s := 1; s <= steps; s++ {
					f := float64(s) / float64(steps)
					dense = append(dense, model.Location{Lat: a.Lat + (b.Lat-a.Lat)*f, Lng: a.Lng + (b.Lng-a.Lng)*f})
				}
			}
			legs[i].Segments[j].Polyline = dense
		}
	}
	return legs, err
}

func TestThinKeepsEndpointsStopsAndSpacing(t *testing.T) {
	// A 3 m-step arc of about 1 km, with a stop part-way along.
	var points []model.Location
	for i := 0; i <= 330; i++ {
		angle := float64(i) / 330 * math.Pi / 2
		points = append(points, model.Location{Lat: 30.27 + 0.006*math.Sin(angle), Lng: -97.74 + 0.006*(1-math.Cos(angle))})
	}
	const stop = 101
	thinned := polyline.Thin(points, minimumOutputSpacingMeters, func(i int) bool { return i == stop })
	if thinned[0] != points[0] || thinned[len(thinned)-1] != points[len(points)-1] {
		t.Fatalf("endpoints not kept")
	}
	foundStop := false
	for i, p := range thinned {
		foundStop = foundStop || p == points[stop]
		if i > 0 && i < len(thinned)-1 && p != points[stop] {
			if d := polyline.HaversineMeters(thinned[i-1], p); d < minimumOutputSpacingMeters {
				t.Errorf("points %d and %d are %.1f m apart", i-1, i, d)
			}
		}
	}
	if !foundStop {
		t.Errorf("stop point dropped")
	}
	if len(thinned) >= len(points)/5 {
		t.Errorf("thinned %d of %d points, want far fewer", len(thinned), len(points))
	}
	if got := polyline.Thin(points[:2], minimumOutputSpacingMeters, nil); len(got) != 2 {
		t.Errorf("two-point polyline thinned to %d points", len(got))
	}
}

// Output polylines from detailed road geometry have no two consecutive points
// closer than 20 m except between endpoints and stops, and thinning the
// output changes nothing else.
func TestOutputPolylinesAreThinned(t *testing.T) {
	settings := model.Settings{PoolingEnabled: true, ReassignmentEnabled: true, DefaultNumRequestedSeats: 1, DefaultNumAvailableSeats: 4,
		AverageSpeedMetersPerSecond: 10, MatchingHook: "greedy_nearest_eta", RoutingHook: "manhattan"}
	requests := []model.Request{
		{RequestID: "a_kezar_to_coit", RequestedAt: T + 60, Origin: kezarStadium, Destination: coitTower},
		{RequestID: "b_painted_ladies_to_union_square", RequestedAt: T + 560, Origin: paintedLadies, Destination: unionSquare},
	}
	supply := []model.Supply{{SupplyID: "car", OnlineAt: T, OnlineLocation: unionSquare}}
	dense, err := simulator.New(settings, denseRouter{}, matching.NewGreedy()).Run(requests, supply)
	if err != nil {
		t.Fatal(err)
	}
	sparse := runManhattan(t, true, true)
	if !reflect.DeepEqual(dense.Trips, sparse.Trips) {
		t.Errorf("detailed geometry changed trips:\n%v\n%v", dense.Trips, sparse.Trips)
	}
	for i := range dense.SupplySummary {
		if a, b := dense.SupplySummary[i][:12], sparse.SupplySummary[i][:12]; !reflect.DeepEqual(a, b) {
			t.Errorf("detailed geometry changed the supply summary: %v vs %v", a, b)
		}
	}
	stopList := []model.Location{kezarStadium, coitTower, paintedLadies, unionSquare}
	// isStop allows for the 1e-5 degree rounding of encoded polylines.
	isStop := func(p model.Location) bool {
		for _, stop := range stopList {
			if polyline.HaversineMeters(p, stop) < 1.5 {
				return true
			}
		}
		return false
	}
	check := func(what, encoded string) {
		points := polyline.Decode(encoded)
		for i := 1; i < len(points); i++ {
			if d := polyline.HaversineMeters(points[i-1], points[i]); d < minimumOutputSpacingMeters {
				required := func(k int) bool { return k == 0 || k == len(points)-1 || isStop(points[k]) }
				if !required(i-1) || !required(i) {
					t.Errorf("%s: points %d and %d are %.1f m apart", what, i-1, i, d)
				}
			}
		}
	}
	for _, row := range dense.SupplyEvents {
		check(row[2]+" driven_polyline at "+row[1], row[10])
		check(row[2]+" planned_polyline at "+row[1], row[11])
	}
	for _, summary := range dense.SupplySummary {
		points := polyline.Decode(summary[12])
		for _, stop := range stopList {
			found := false
			for _, p := range points {
				found = found || polyline.HaversineMeters(p, stop) < 1.5
			}
			if !found {
				t.Errorf("full_polyline lost stop %v", stop)
			}
		}
		if len(points) > 1000 {
			t.Errorf("full_polyline has %d points, want it thinned", len(points))
		}
		check("full_polyline", summary[12])
	}
}
