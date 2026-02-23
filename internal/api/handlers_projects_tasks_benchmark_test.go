package api

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

func makeBenchmarkProjectRunAt(
	tb testing.TB,
	root, projectID, taskID, runID, status, stdoutContent string,
	start time.Time,
) *storage.RunInfo {
	tb.Helper()

	runDir := filepath.Join(root, projectID, taskID, "runs", runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		tb.Fatalf("mkdir run: %v", err)
	}
	stdoutPath := filepath.Join(runDir, "agent-stdout.txt")
	if err := os.WriteFile(stdoutPath, []byte(stdoutContent), 0o644); err != nil {
		tb.Fatalf("write stdout: %v", err)
	}
	info := &storage.RunInfo{
		RunID:      runID,
		ProjectID:  projectID,
		TaskID:     taskID,
		Status:     status,
		StartTime:  start.UTC(),
		StdoutPath: stdoutPath,
	}
	if status != storage.StatusRunning {
		info.EndTime = start.Add(time.Second).UTC()
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		tb.Fatalf("write run-info: %v", err)
	}
	return info
}

func setupBenchmarkTaskRuns(
	b *testing.B,
	taskCount int,
	runsPerTask int,
) (string, []*storage.RunInfo) {
	b.Helper()

	root := b.TempDir()
	projectID := "proj-bench"
	runs := make([]*storage.RunInfo, 0, taskCount*runsPerTask)
	startBase := time.Date(2026, time.February, 23, 3, 0, 0, 0, time.UTC)

	for taskIndex := 0; taskIndex < taskCount; taskIndex++ {
		taskID := fmt.Sprintf("task-%04d", taskIndex)
		taskDir := filepath.Join(root, projectID, taskID)
		if err := os.MkdirAll(taskDir, 0o755); err != nil {
			b.Fatalf("mkdir task dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("benchmark task\n"), 0o644); err != nil {
			b.Fatalf("write TASK.md: %v", err)
		}

		for runIndex := 0; runIndex < runsPerTask; runIndex++ {
			runID := fmt.Sprintf("run-%04d-%04d", taskIndex, runIndex)
			runDir := filepath.Join(taskDir, "runs", runID)
			if err := os.MkdirAll(runDir, 0o755); err != nil {
				b.Fatalf("mkdir run dir: %v", err)
			}

			stdoutPath := filepath.Join(runDir, "agent-stdout.txt")
			if err := os.WriteFile(stdoutPath, []byte("stdout payload\n"), 0o644); err != nil {
				b.Fatalf("write stdout: %v", err)
			}
			outputPath := filepath.Join(runDir, "output.md")
			if err := os.WriteFile(outputPath, []byte("output payload\n"), 0o644); err != nil {
				b.Fatalf("write output: %v", err)
			}

			start := startBase.Add(time.Duration(taskIndex*runsPerTask+runIndex) * time.Second)
			status := storage.StatusCompleted
			if runIndex == runsPerTask-1 {
				status = storage.StatusRunning
			}
			info := &storage.RunInfo{
				RunID:      runID,
				ProjectID:  projectID,
				TaskID:     taskID,
				Status:     status,
				StartTime:  start,
				StdoutPath: stdoutPath,
				OutputPath: outputPath,
			}
			if status != storage.StatusRunning {
				info.EndTime = start.Add(15 * time.Second)
			}
			runs = append(runs, info)
		}
	}

	return root, runs
}

func benchmarkBuildTasksWithQueue(
	b *testing.B,
	includeRunFiles bool,
) {
	root, runs := setupBenchmarkTaskRuns(b, 120, 40)
	const projectID = "proj-bench"

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tasks := buildTasksWithQueue(root, projectID, runs, nil, includeRunFiles)
		if len(tasks) == 0 {
			b.Fatalf("expected tasks to be built")
		}
	}
}

func BenchmarkBuildTasksWithQueueSummaryOnly(b *testing.B) {
	benchmarkBuildTasksWithQueue(b, false)
}

func BenchmarkBuildTasksWithQueueWithRunFiles(b *testing.B) {
	benchmarkBuildTasksWithQueue(b, true)
}

func setupBenchmarkProjectTasksServer(
	b *testing.B,
	projectCount int,
	tasksPerProject int,
	runsPerTask int,
	cacheTTL time.Duration,
) (*Server, string) {
	b.Helper()

	root := b.TempDir()
	targetProjectID := "proj-000"
	baseTime := time.Date(2026, time.February, 23, 4, 0, 0, 0, time.UTC)

	for projectIndex := 0; projectIndex < projectCount; projectIndex++ {
		projectID := fmt.Sprintf("proj-%03d", projectIndex)
		for taskIndex := 0; taskIndex < tasksPerProject; taskIndex++ {
			taskID := fmt.Sprintf("task-%03d-%03d", projectIndex, taskIndex)
			for runIndex := 0; runIndex < runsPerTask; runIndex++ {
				runID := fmt.Sprintf("run-%03d-%03d-%03d", projectIndex, taskIndex, runIndex)
				status := storage.StatusCompleted
				if runIndex == runsPerTask-1 && projectIndex == 0 && taskIndex%3 == 0 {
					status = storage.StatusRunning
				}
				content := fmt.Sprintf(
					"benchmark stdout project=%s task=%s run=%s\n",
					projectID,
					taskID,
					runID,
				)
				runInfo := makeBenchmarkProjectRunAt(
					b,
					root,
					projectID,
					taskID,
					runID,
					status,
					content,
					baseTime.Add(time.Duration(projectIndex*tasksPerProject*runsPerTask+taskIndex*runsPerTask+runIndex)*time.Second),
				)
				runDir := filepath.Dir(runInfo.StdoutPath)
				outputPath := filepath.Join(runDir, "output.md")
				if err := os.WriteFile(outputPath, []byte("output payload\n"), 0o644); err != nil {
					b.Fatalf("write output: %v", err)
				}
				runInfo.OutputPath = outputPath
				if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), runInfo); err != nil {
					b.Fatalf("rewrite run-info with output path: %v", err)
				}
			}
		}
	}

	server, err := NewServer(Options{
		RootDir:                 root,
		DisableTaskStart:        true,
		ProjectRunsFlatCacheTTL: cacheTTL,
		Logger:                  log.New(io.Discard, "", 0),
	})
	if err != nil {
		b.Fatalf("NewServer: %v", err)
	}
	return server, targetProjectID
}

func BenchmarkProjectTasksPreparationLegacyAllRuns(b *testing.B) {
	server, projectID := setupBenchmarkProjectTasksServer(b, 8, 18, 14, 0)
	rootDir := server.rootDir

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runs, err := server.allRunInfos()
		if err != nil {
			b.Fatalf("allRunInfos: %v", err)
		}
		tasks := buildTasksWithQueue(rootDir, projectID, runs, server.taskQueueSnapshot(), true)
		if len(tasks) == 0 {
			b.Fatalf("expected tasks")
		}
	}
}

func BenchmarkProjectTasksPreparationOptimizedProjectScoped(b *testing.B) {
	server, projectID := setupBenchmarkProjectTasksServer(b, 8, 18, 14, time.Hour)
	rootDir := server.rootDir

	// Warm cache once.
	if _, err := server.projectRunInfos(projectID); err != nil {
		b.Fatalf("warm projectRunInfos: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runs, err := server.projectRunInfos(projectID)
		if err != nil {
			b.Fatalf("projectRunInfos: %v", err)
		}
		tasks := buildTasksWithQueue(rootDir, projectID, runs, server.taskQueueSnapshot(), false)
		if len(tasks) == 0 {
			b.Fatalf("expected tasks")
		}
	}
}

func BenchmarkProjectTasksEndpointCached(b *testing.B) {
	server, projectID := setupBenchmarkProjectTasksServer(b, 8, 18, 14, time.Hour)
	path := fmt.Sprintf("/api/projects/%s/tasks", projectID)
	req := httptest.NewRequest(http.MethodGet, path, nil)

	// Warm cache once.
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		b.Fatalf("warm request status=%d body=%s", rec.Code, rec.Body.String())
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("request status=%d body=%s", rec.Code, rec.Body.String())
		}
	}
}

func BenchmarkProjectTasksEndpointNoCache(b *testing.B) {
	server, projectID := setupBenchmarkProjectTasksServer(b, 8, 18, 14, 0)
	path := fmt.Sprintf("/api/projects/%s/tasks", projectID)
	req := httptest.NewRequest(http.MethodGet, path, nil)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("request status=%d body=%s", rec.Code, rec.Body.String())
		}
	}
}

func benchmarkProjectTaskDetailEndpoint(b *testing.B, includeRunFiles bool) {
	server, projectID := setupBenchmarkProjectTasksServer(b, 4, 20, 80, time.Hour)
	taskID := "task-000-007"

	path := fmt.Sprintf("/api/projects/%s/tasks/%s", projectID, taskID)
	if !includeRunFiles {
		path += "?include_files=0"
	}
	req := httptest.NewRequest(http.MethodGet, path, nil)

	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		b.Fatalf("warm request status=%d body=%s", rec.Code, rec.Body.String())
	}
	respBytes := float64(rec.Body.Len())

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("request status=%d body=%s", rec.Code, rec.Body.String())
		}
		respBytes = float64(rec.Body.Len())
	}
	b.ReportMetric(respBytes, "resp_bytes/op")
}

func BenchmarkProjectTaskDetailEndpointWithFiles(b *testing.B) {
	benchmarkProjectTaskDetailEndpoint(b, true)
}

func BenchmarkProjectTaskDetailEndpointWithoutFiles(b *testing.B) {
	benchmarkProjectTaskDetailEndpoint(b, false)
}

func benchmarkProjectTaskRunsEndpoint(b *testing.B, includeRunFiles bool) {
	server, projectID := setupBenchmarkProjectTasksServer(b, 4, 20, 80, time.Hour)
	taskID := "task-000-007"

	path := fmt.Sprintf("/api/projects/%s/tasks/%s/runs?limit=500", projectID, taskID)
	if !includeRunFiles {
		path += "&include_files=0"
	}
	req := httptest.NewRequest(http.MethodGet, path, nil)

	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		b.Fatalf("warm request status=%d body=%s", rec.Code, rec.Body.String())
	}
	respBytes := float64(rec.Body.Len())

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("request status=%d body=%s", rec.Code, rec.Body.String())
		}
		respBytes = float64(rec.Body.Len())
	}
	b.ReportMetric(respBytes, "resp_bytes/op")
}

func BenchmarkProjectTaskRunsEndpointWithFiles(b *testing.B) {
	benchmarkProjectTaskRunsEndpoint(b, true)
}

func BenchmarkProjectTaskRunsEndpointWithoutFiles(b *testing.B) {
	benchmarkProjectTaskRunsEndpoint(b, false)
}

func benchmarkProjectTaskDetailBuildPath(
	b *testing.B,
	includeRunFiles bool,
	useLegacyAllTasks bool,
) {
	root, runs := setupBenchmarkTaskRuns(b, 120, 40)
	const (
		projectID = "proj-bench"
		taskID    = "task-0071"
	)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if useLegacyAllTasks {
			tasks := buildTasksWithQueue(root, projectID, runs, nil, includeRunFiles)
			found := false
			for _, task := range tasks {
				if task.ID == taskID {
					runtime.KeepAlive(task)
					found = true
					break
				}
			}
			if !found {
				b.Fatalf("task %s not found", taskID)
			}
			continue
		}

		task, ok := buildTaskWithQueue(root, projectID, taskID, runs, nil, includeRunFiles)
		if !ok {
			b.Fatalf("task %s not found", taskID)
		}
		runtime.KeepAlive(task)
	}
}

func BenchmarkProjectTaskDetailBuildLegacyWithFiles(b *testing.B) {
	benchmarkProjectTaskDetailBuildPath(b, true, true)
}

func BenchmarkProjectTaskDetailBuildDirectWithFiles(b *testing.B) {
	benchmarkProjectTaskDetailBuildPath(b, true, false)
}

func BenchmarkProjectTaskDetailBuildLegacyWithoutFiles(b *testing.B) {
	benchmarkProjectTaskDetailBuildPath(b, false, true)
}

func BenchmarkProjectTaskDetailBuildDirectWithoutFiles(b *testing.B) {
	benchmarkProjectTaskDetailBuildPath(b, false, false)
}
