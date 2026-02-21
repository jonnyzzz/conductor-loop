package metrics

import (
	"strings"
	"testing"
	"time"
)

func TestNewRegistry(t *testing.T) {
	r := New()
	if r == nil {
		t.Fatal("New() returned nil")
	}
}

func TestNilRegistrySafe(t *testing.T) {
	var r *Registry
	r.IncActiveRuns()
	r.DecActiveRuns()
	r.IncCompletedRuns()
	r.IncFailedRuns()
	r.IncBusAppends()
	r.RecordRequest("GET", 200)
	out := r.Render()
	if out != "" {
		t.Fatalf("nil registry Render() should return empty string, got %q", out)
	}
}

func TestActiveRunsGauge(t *testing.T) {
	r := New()
	r.IncActiveRuns()
	r.IncActiveRuns()
	r.DecActiveRuns()

	out := r.Render()
	if !strings.Contains(out, "conductor_active_runs_total 1\n") {
		t.Fatalf("expected active_runs_total=1 in output:\n%s", out)
	}
}

func TestCompletedRunsCounter(t *testing.T) {
	r := New()
	r.IncCompletedRuns()
	r.IncCompletedRuns()
	r.IncCompletedRuns()

	out := r.Render()
	if !strings.Contains(out, "conductor_completed_runs_total 3\n") {
		t.Fatalf("expected completed_runs_total=3 in output:\n%s", out)
	}
}

func TestFailedRunsCounter(t *testing.T) {
	r := New()
	r.IncFailedRuns()

	out := r.Render()
	if !strings.Contains(out, "conductor_failed_runs_total 1\n") {
		t.Fatalf("expected failed_runs_total=1 in output:\n%s", out)
	}
}

func TestBusAppendsCounter(t *testing.T) {
	r := New()
	for i := 0; i < 5; i++ {
		r.IncBusAppends()
	}

	out := r.Render()
	if !strings.Contains(out, "conductor_messagebus_appends_total 5\n") {
		t.Fatalf("expected messagebus_appends_total=5 in output:\n%s", out)
	}
}

func TestRecordRequest(t *testing.T) {
	r := New()
	r.RecordRequest("GET", 200)
	r.RecordRequest("GET", 200)
	r.RecordRequest("POST", 201)

	out := r.Render()
	if !strings.Contains(out, `conductor_api_requests_total{method="GET",status="200"} 2`) {
		t.Fatalf("expected GET:200=2 in output:\n%s", out)
	}
	if !strings.Contains(out, `conductor_api_requests_total{method="POST",status="201"} 1`) {
		t.Fatalf("expected POST:201=1 in output:\n%s", out)
	}
}

func TestUptimeIncreases(t *testing.T) {
	r := New()
	time.Sleep(10 * time.Millisecond)
	out := r.Render()
	if !strings.Contains(out, "conductor_uptime_seconds") {
		t.Fatalf("expected uptime_seconds in output:\n%s", out)
	}
	// uptime should be > 0
	if strings.Contains(out, "conductor_uptime_seconds 0\n") {
		t.Fatalf("uptime should be > 0 after sleeping:\n%s", out)
	}
}

func TestRenderPrometheusFormat(t *testing.T) {
	r := New()
	out := r.Render()

	// Check HELP and TYPE annotations are present
	for _, metric := range []string{
		"conductor_uptime_seconds",
		"conductor_active_runs_total",
		"conductor_completed_runs_total",
		"conductor_failed_runs_total",
		"conductor_messagebus_appends_total",
		"conductor_api_requests_total",
	} {
		if !strings.Contains(out, "# HELP "+metric) {
			t.Errorf("missing HELP for %s", metric)
		}
		if !strings.Contains(out, "# TYPE "+metric) {
			t.Errorf("missing TYPE for %s", metric)
		}
	}
}

func TestSortStrings(t *testing.T) {
	ss := []string{"c", "a", "b"}
	sortStrings(ss)
	if ss[0] != "a" || ss[1] != "b" || ss[2] != "c" {
		t.Fatalf("sortStrings failed: %v", ss)
	}
}
