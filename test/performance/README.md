# Performance Tests

## Running tests

1. `go test ./test/performance -v`
2. `go test ./test/performance -run TestMessageBusThroughput -v`
3. `go test ./test/performance -run TestRunCreationThroughput -v`
4. `go test ./test/performance -run TestConcurrentAgents -v`
5. `go test ./test/performance -run TestSSELatency -v`

## Running benchmarks

1. `go test ./test/performance -run ^$ -bench . -benchmem`
2. `go test ./test/performance -run ^$ -bench BenchmarkMessageBusWrite -benchmem`
3. `go test ./test/performance -run ^$ -bench BenchmarkRunCreation -benchmem`

## Profiling

1. `go test -memprofile=mem.prof ./...`
2. `go tool pprof mem.prof`
3. `go test -cpuprofile=cpu.prof ./...`
4. `go tool pprof cpu.prof`

## Threshold overrides

1. `PERF_MESSAGEBUS_TARGET` sets the minimum message bus throughput (messages/sec)
2. `PERF_RUN_CREATION_TARGET` sets the minimum run creation throughput (runs/sec)
3. `PERF_SSE_LATENCY_MS` sets the maximum SSE latency (ms)

## Load test script

1. `scripts/load-test.sh`
