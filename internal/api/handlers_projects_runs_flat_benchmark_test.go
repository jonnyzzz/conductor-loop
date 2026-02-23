package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/runstate"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

func benchmarkProjectRunsFlatScan(
	b *testing.B,
	scan func(*Server, string) ([]flatRunItem, error),
) {
	root := b.TempDir()
	const (
		projectCount    = 24
		tasksPerProject = 9
		runsPerTask     = 7
	)
	targetProjectID := "proj-07"
	seedRunsFlatBenchmarkData(b, root, projectCount, tasksPerProject, runsPerTask)

	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		b.Fatalf("NewServer: %v", err)
	}
	server.logger.SetOutput(io.Discard)

	// Warm-up to ensure disk caches are populated for steady-state comparison.
	if _, err := scan(server, targetProjectID); err != nil {
		b.Fatalf("warm-up scan failed: %v", err)
	}

	expectedRuns := tasksPerProject * runsPerTask
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		items, err := scan(server, targetProjectID)
		if err != nil {
			b.Fatalf("scan failed: %v", err)
		}
		if len(items) != expectedRuns {
			b.Fatalf("expected %d runs, got %d", expectedRuns, len(items))
		}
		runtime.KeepAlive(items)
	}
}

func BenchmarkProjectRunsFlatScanLegacyAllRoots(b *testing.B) {
	benchmarkProjectRunsFlatScan(b, legacyProjectRunsFlatScan)
}

func BenchmarkProjectRunsFlatScanScopedProject(b *testing.B) {
	benchmarkProjectRunsFlatScan(b, scopedProjectRunsFlatScan)
}

func BenchmarkProjectRunsFlatScanScopedProjectUncached(b *testing.B) {
	benchmarkProjectRunsFlatScan(b, scopedProjectRunsFlatScanUncached)
}

func benchmarkProjectRunsFlatEndpoint(
	b *testing.B,
	url string,
	expectedRuns int,
) {
	benchmarkProjectRunsFlatEndpointWithConfig(b, url, expectedRuns, 4)
}

func benchmarkProjectRunsFlatEndpointWithConfig(
	b *testing.B,
	url string,
	expectedRuns int,
	activeTaskCount int,
	runsPerTaskOverride ...int,
) {
	benchmarkProjectRunsFlatEndpointWithConfigCache(
		b,
		url,
		expectedRuns,
		activeTaskCount,
		true,
		runsPerTaskOverride...,
	)
}

func benchmarkProjectRunsFlatEndpointWithConfigCache(
	b *testing.B,
	url string,
	expectedRuns int,
	activeTaskCount int,
	cacheEnabled bool,
	runsPerTaskOverride ...int,
) {
	root := b.TempDir()
	const (
		projectCount       = 4
		tasksPerProject    = 20
		targetProjectIndex = 3
	)
	runsPerTask := 6
	if len(runsPerTaskOverride) > 0 && runsPerTaskOverride[0] > 0 {
		runsPerTask = runsPerTaskOverride[0]
	}
	seedRunsFlatBenchmarkDataSkewed(
		b,
		root,
		projectCount,
		tasksPerProject,
		runsPerTask,
		activeTaskCount,
		targetProjectIndex,
	)

	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		b.Fatalf("NewServer: %v", err)
	}
	server.logger.SetOutput(io.Discard)
	if !cacheEnabled {
		server.projectRunsCache = nil
	}

	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		b.Fatalf("warm-up status=%d", rec.Code)
	}
	var payload struct {
		Runs []flatRunItem `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		b.Fatalf("warm-up decode: %v", err)
	}
	if len(payload.Runs) != expectedRuns {
		b.Fatalf("expected %d runs, got %d", expectedRuns, len(payload.Runs))
	}
	respBytes := float64(rec.Body.Len())

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("status=%d", rec.Code)
		}
		respBytes = float64(rec.Body.Len())
		runtime.KeepAlive(rec.Body.Len())
	}
	b.ReportMetric(respBytes, "resp_bytes/op")
}

func BenchmarkProjectRunsFlatEndpointFullPayload(b *testing.B) {
	benchmarkProjectRunsFlatEndpoint(
		b,
		"/api/projects/proj-03/runs/flat",
		20*6,
	)
}

func BenchmarkProjectRunsFlatEndpointFullPayloadUncached(b *testing.B) {
	benchmarkProjectRunsFlatEndpointWithConfigCache(
		b,
		"/api/projects/proj-03/runs/flat",
		20*6,
		4,
		false,
	)
}

func BenchmarkProjectRunsFlatEndpointActiveOnly(b *testing.B) {
	benchmarkProjectRunsFlatEndpoint(
		b,
		"/api/projects/proj-03/runs/flat?active_only=1",
		4,
	)
}

func BenchmarkProjectRunsFlatEndpointActiveWithSelectedTask(b *testing.B) {
	benchmarkProjectRunsFlatEndpoint(
		b,
		"/api/projects/proj-03/runs/flat?active_only=1&selected_task_id=task-010",
		4+6,
	)
}

func BenchmarkProjectRunsFlatEndpointActiveWithSelectedTaskLimited(b *testing.B) {
	benchmarkProjectRunsFlatEndpoint(
		b,
		"/api/projects/proj-03/runs/flat?active_only=1&selected_task_id=task-010&selected_task_limit=1",
		4+1,
	)
}

func BenchmarkProjectRunsFlatEndpointActiveOnlyNoActive(b *testing.B) {
	benchmarkProjectRunsFlatEndpointWithConfig(
		b,
		"/api/projects/proj-03/runs/flat?active_only=1",
		20,
		0,
	)
}

func BenchmarkProjectRunsFlatEndpointActiveWithSelectedTaskHeavy(b *testing.B) {
	benchmarkProjectRunsFlatEndpointWithConfig(
		b,
		"/api/projects/proj-03/runs/flat?active_only=1&selected_task_id=task-010",
		2+120,
		2,
		120,
	)
}

func BenchmarkProjectRunsFlatEndpointActiveWithSelectedTaskHeavyLimited(b *testing.B) {
	benchmarkProjectRunsFlatEndpointWithConfig(
		b,
		"/api/projects/proj-03/runs/flat?active_only=1&selected_task_id=task-010&selected_task_limit=1",
		2+1,
		2,
		120,
	)
}

func benchmarkReadRunInfoFiles(
	b *testing.B,
	readFn func([]string) ([]*storage.RunInfo, error),
) {
	root := b.TempDir()
	const (
		projectCount = 1
		tasksPerProj = 40
		runsPerTask  = 80
	)
	seedRunsFlatBenchmarkData(b, root, projectCount, tasksPerProj, runsPerTask)

	projectDir := filepath.Join(root, "proj-00")
	paths, err := collectRunInfoPaths(projectDir)
	if err != nil {
		b.Fatalf("collect run-info paths: %v", err)
	}
	expectedRuns := tasksPerProj * runsPerTask

	infos, err := readFn(paths)
	if err != nil {
		b.Fatalf("warm-up read failed: %v", err)
	}
	if len(infos) != expectedRuns {
		b.Fatalf("warm-up expected %d runs, got %d", expectedRuns, len(infos))
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		infos, err := readFn(paths)
		if err != nil {
			b.Fatalf("read failed: %v", err)
		}
		if len(infos) != expectedRuns {
			b.Fatalf("expected %d runs, got %d", expectedRuns, len(infos))
		}
		runtime.KeepAlive(infos)
	}
}

func BenchmarkReadRunInfoFilesSequential(b *testing.B) {
	benchmarkReadRunInfoFiles(b, readRunInfoFilesSequential)
}

func BenchmarkReadRunInfoFilesParallel(b *testing.B) {
	benchmarkReadRunInfoFiles(b, readRunInfoFiles)
}

func collectRunInfoPaths(root string) ([]string, error) {
	paths := make([]string, 0, 1024)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "run-info.yaml" {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return paths, nil
}

func readRunInfoFilesSequential(paths []string) ([]*storage.RunInfo, error) {
	out := make([]*storage.RunInfo, 0, len(paths))
	for _, path := range paths {
		info, err := runstate.ReadRunInfo(path)
		if err != nil {
			return nil, err
		}
		out = append(out, info)
	}
	return out, nil
}

func legacyProjectRunsFlatScan(s *Server, projectID string) ([]flatRunItem, error) {
	allRuns, err := s.allRunInfos()
	if err != nil {
		return nil, err
	}
	items := make([]flatRunItem, 0, len(allRuns))
	for _, run := range allRuns {
		if run.ProjectID != projectID {
			continue
		}
		item := flatRunItem{
			ID:            run.RunID,
			TaskID:        run.TaskID,
			Agent:         run.AgentType,
			Status:        run.Status,
			ExitCode:      run.ExitCode,
			StartTime:     run.StartTime,
			ParentRunID:   run.ParentRunID,
			PreviousRunID: run.PreviousRunID,
		}
		if !run.EndTime.IsZero() {
			end := run.EndTime
			item.EndTime = &end
		}
		items = append(items, item)
	}
	return items, nil
}

func scopedProjectRunsFlatScan(s *Server, projectID string) ([]flatRunItem, error) {
	projectRuns, err := s.projectRunInfos(projectID)
	if err != nil {
		return nil, err
	}
	items := make([]flatRunItem, 0, len(projectRuns))
	for _, run := range projectRuns {
		item := flatRunItem{
			ID:            run.RunID,
			TaskID:        run.TaskID,
			Agent:         run.AgentType,
			Status:        run.Status,
			ExitCode:      run.ExitCode,
			StartTime:     run.StartTime,
			ParentRunID:   run.ParentRunID,
			PreviousRunID: run.PreviousRunID,
		}
		if !run.EndTime.IsZero() {
			end := run.EndTime
			item.EndTime = &end
		}
		items = append(items, item)
	}
	return items, nil
}

func scopedProjectRunsFlatScanUncached(s *Server, projectID string) ([]flatRunItem, error) {
	projectRuns, err := s.scanProjectRunInfos(projectID)
	if err != nil {
		return nil, err
	}
	items := make([]flatRunItem, 0, len(projectRuns))
	for _, run := range projectRuns {
		item := flatRunItem{
			ID:            run.RunID,
			TaskID:        run.TaskID,
			Agent:         run.AgentType,
			Status:        run.Status,
			ExitCode:      run.ExitCode,
			StartTime:     run.StartTime,
			ParentRunID:   run.ParentRunID,
			PreviousRunID: run.PreviousRunID,
		}
		if !run.EndTime.IsZero() {
			end := run.EndTime
			item.EndTime = &end
		}
		items = append(items, item)
	}
	return items, nil
}

func seedRunsFlatBenchmarkData(
	b *testing.B,
	root string,
	projectCount int,
	tasksPerProject int,
	runsPerTask int,
) {
	base := time.Date(2026, time.February, 22, 18, 0, 0, 0, time.UTC)
	for projectIndex := 0; projectIndex < projectCount; projectIndex++ {
		projectID := fmt.Sprintf("proj-%02d", projectIndex)
		for taskIndex := 0; taskIndex < tasksPerProject; taskIndex++ {
			taskID := fmt.Sprintf("task-%02d", taskIndex)
			for runIndex := 0; runIndex < runsPerTask; runIndex++ {
				runID := fmt.Sprintf("run-%02d-%02d-%02d", projectIndex, taskIndex, runIndex)
				runDir := filepath.Join(root, projectID, taskID, "runs", runID)
				if err := os.MkdirAll(runDir, 0o755); err != nil {
					b.Fatalf("mkdir run dir: %v", err)
				}

				start := base.Add(time.Duration(projectIndex*tasksPerProject*runsPerTask+taskIndex*runsPerTask+runIndex) * time.Second)
				info := &storage.RunInfo{
					Version:   1,
					RunID:     runID,
					ProjectID: projectID,
					TaskID:    taskID,
					AgentType: "codex",
					Status:    storage.StatusCompleted,
					ExitCode:  0,
					StartTime: start,
					EndTime:   start.Add(2 * time.Second),
				}
				if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
					b.Fatalf("write run-info: %v", err)
				}
			}
		}
	}
}

func seedRunsFlatBenchmarkDataSkewed(
	b *testing.B,
	root string,
	projectCount int,
	tasksPerProject int,
	runsPerTask int,
	activeTaskCount int,
	targetProjectIndex int,
) {
	base := time.Date(2026, time.February, 22, 19, 0, 0, 0, time.UTC)
	for projectIndex := 0; projectIndex < projectCount; projectIndex++ {
		projectID := fmt.Sprintf("proj-%02d", projectIndex)
		for taskIndex := 0; taskIndex < tasksPerProject; taskIndex++ {
			taskID := fmt.Sprintf("task-%03d", taskIndex)
			for runIndex := 0; runIndex < runsPerTask; runIndex++ {
				runID := fmt.Sprintf("run-%02d-%03d-%03d", projectIndex, taskIndex, runIndex)
				runDir := filepath.Join(root, projectID, taskID, "runs", runID)
				if err := os.MkdirAll(runDir, 0o755); err != nil {
					b.Fatalf("mkdir run dir: %v", err)
				}

				start := base.Add(time.Duration(projectIndex*tasksPerProject*runsPerTask+taskIndex*runsPerTask+runIndex) * time.Second)
				status := storage.StatusCompleted
				end := start.Add(2 * time.Second)
				if projectIndex == targetProjectIndex && taskIndex < activeTaskCount && runIndex == runsPerTask-1 {
					status = storage.StatusRunning
					end = time.Time{}
				}
				info := &storage.RunInfo{
					Version:   1,
					RunID:     runID,
					ProjectID: projectID,
					TaskID:    taskID,
					AgentType: "codex",
					Status:    status,
					ExitCode:  0,
					StartTime: start,
					EndTime:   end,
				}
				if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
					b.Fatalf("write run-info: %v", err)
				}
			}
		}
	}
}
