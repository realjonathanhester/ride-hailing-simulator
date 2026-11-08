package progress

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestFormatShowsPercentBarClockAndCounts(t *testing.T) {
	location := time.FixedZone("CST", -6*3600)
	snap := Snapshot{
		Now: 1767765600 + 3600, StartAt: 1767765600, HorizonAt: 1767765600 + 7200,
		DriversOnline: 10, DriversIdle: 4, DriversEnRoute: 2, DriversOnTrip: 3, DriversRepositioning: 1,
		RequestsWaiting: 5, RequestsAssigned: 8, RequestsCompleted: 20, RequestsCanceled: 1, RequestsTotal: 34,
	}
	line := Format(snap, 30*time.Second, location)
	assertContains(t, line, "[==========----------]")
	assertContains(t, line, "50.0%")
	assertContains(t, line, "T+3600 ·")
	assertContains(t, line, "elapsed 30s")
	assertContains(t, line, "remaining 30s")
	assertContains(t, line, "drivers 10 online (4 idle, 2 en route, 3 on trip, 1 repositioning)")
	assertContains(t, line, "requests 5 waiting, 8 assigned, 20 completed, 1 canceled / 34")
	if strings.Contains(line, "osrm") {
		t.Fatalf("osrm section shown without OSRM: %s", line)
	}
}

func TestFormatIncludesOsrmCountsWhenPresent(t *testing.T) {
	snap := Snapshot{
		Now: 100, StartAt: 0, HorizonAt: 100,
		OSRM: true, OSRMTable: 12, OSRMRoute: 3, OSRMCacheHits: 40,
	}
	line := Format(snap, time.Second, nil)
	assertContains(t, line, "osrm 12 table, 3 route, 40 cache hits")
	assertContains(t, line, "T+100")
}

func TestFormatShowsOsrmCacheHitRate(t *testing.T) {
	snap := Snapshot{OSRM: true, OSRMCacheHits: 30, OSRMCacheMisses: 10}
	line := Format(snap, time.Second, nil)
	assertContains(t, line, "30 cache hits (75.0% hit rate)")
}

func TestNonTerminalReporterWritesPlainLines(t *testing.T) {
	var buf bytes.Buffer
	reporter := &Reporter{Writer: &buf, Terminal: false, Interval: time.Nanosecond}
	snap := Snapshot{Now: 50, StartAt: 0, HorizonAt: 100, DriversOnline: 1, DriversIdle: 1, RequestsTotal: 1}
	reporter.Update(snap)
	snap.Now = 100
	snap.RequestsCompleted = 1
	reporter.Force(snap)
	reporter.Finish()
	text := buf.String()
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) < 2 {
		t.Fatalf("want at least 2 lines, got %q", text)
	}
	assertContains(t, lines[0], "50.0%")
	assertContains(t, lines[len(lines)-1], "100.0%")
	assertContains(t, lines[len(lines)-1], "1 completed")
	if strings.Contains(text, "\r") {
		t.Fatalf("non-terminal writer used carriage returns: %q", text)
	}
}

func TestTerminalReporterRedrawsInPlace(t *testing.T) {
	var buf bytes.Buffer
	reporter := &Reporter{Writer: &buf, Terminal: true, Interval: time.Nanosecond}
	snap := Snapshot{Now: 25, StartAt: 0, HorizonAt: 100}
	reporter.Update(snap)
	snap.Now = 50
	reporter.Force(snap)
	reporter.Finish()
	text := buf.String()
	if !strings.Contains(text, "\r") {
		t.Fatalf("terminal writer did not redraw in place: %q", text)
	}
	if !strings.HasSuffix(text, "\n") {
		t.Fatalf("Finish did not end with a newline: %q", text)
	}
	assertContains(t, text, "50.0%")
}

func assertContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("got %q, want it to contain %q", got, want)
	}
}

func TestFormatDoneIsComplete(t *testing.T) {
	// An early stop ends before the horizon; the final line still reads 100%.
	snap := Snapshot{Now: 3000, StartAt: 2000, HorizonAt: 5000, Done: true}
	line := Format(snap, time.Second, nil)
	assertContains(t, line, "100.0%")
	assertContains(t, line, "[====================]")
}

func TestFormatSummary(t *testing.T) {
	line := FormatSummary(Summary{Completed: 99, Canceled: 1, Total: 100, MedianWait: "416", P90Wait: "781", FleetUtilization: "0.295777", OutputDirectory: "out/a"})
	assertContains(t, line, "99 completed, 1 canceled / 100 requests")
	assertContains(t, line, "wait median 416s, p90 781s")
	assertContains(t, line, "fleet utilization 29.6%")
	assertContains(t, line, "outputs in out/a")
}

func TestFormatBeforeFirstRequest(t *testing.T) {
	location := time.FixedZone("CST", -6*3600)
	// Supply comes online before the first request.
	snap := Snapshot{Now: 1000, StartAt: 2000, HorizonAt: 5000}
	line := Format(snap, time.Second, location)
	assertContains(t, line, "before first request ·")
	if strings.Contains(line, "T-") {
		t.Fatalf("negative offset shown: %q", line)
	}
	assertContains(t, line, "0.0%")
}
