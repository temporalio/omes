package metrics

import "fmt"

// DefaultDerivedQueries returns a curated list of PromQL queries that precompute
// worker/process metrics into Omni-friendly metric lines.
func DefaultDerivedQueries() []PromQuery {
	queries := []PromQuery{
		{
			Name:  "process_cpu_percent_raw",
			Query: fmt.Sprintf(`process_cpu_percent{job="%s"}`, JobWorkerProcess),
		},
		{
			Name:  "process_cpu_percent_smoothed_1m",
			Query: fmt.Sprintf(`avg_over_time(process_cpu_percent{job="%s"}[1m])`, JobWorkerProcess),
		},
		{
			Name:  "process_memory_bytes",
			Query: fmt.Sprintf(`process_resident_memory_bytes{job="%s"}`, JobWorkerProcess),
		},
		{
			Name:  "process_memory_percent",
			Query: fmt.Sprintf(`process_memory_percent{job="%s"}`, JobWorkerProcess),
		},
	}

	gaugeMetrics := []struct{ name, promName string }{
		{"num_pollers", "temporal_num_pollers"},
		{"worker_task_slots_available", "temporal_worker_task_slots_available"},
		{"worker_task_slots_used", "temporal_worker_task_slots_used"},
	}
	for _, m := range gaugeMetrics {
		queries = append(queries, derivedGaugeQuery(m.name, m.promName))
	}

	histogramMetrics := []struct{ name, promName string }{
		{"workflow_task_execution_latency_seconds", "temporal_workflow_task_execution_latency"},
		{"workflow_task_schedule_to_start_latency_seconds", "temporal_workflow_task_schedule_to_start_latency"},
		{"workflow_endtoend_latency_seconds", "temporal_workflow_endtoend_latency"},
		{"activity_execution_latency_seconds", "temporal_activity_execution_latency"},
		{"activity_schedule_to_start_latency_seconds", "temporal_activity_schedule_to_start_latency"},
	}
	for _, m := range histogramMetrics {
		queries = append(queries, derivedHistogramQuantileQuery(m.name, m.promName, 0.50, "p50"))
		queries = append(queries, derivedHistogramQuantileQuery(m.name, m.promName, 0.99, "p99"))
	}

	queries = append(queries, PromQuery{
		Name:  "workflow_completions_per_second",
		Query: fmt.Sprintf(`sum(rate(temporal_workflow_endtoend_latency_count{job="%s"}[1m]))`, JobWorkerApp),
	})

	return queries
}

func derivedGaugeQuery(name, promName string) PromQuery {
	return PromQuery{
		Name:  name,
		Query: fmt.Sprintf(`sum(%s{job="%s"})`, promName, JobWorkerApp),
	}
}

func derivedHistogramQuantileQuery(name, promName string, quantile float64, percentile string) PromQuery {
	return PromQuery{
		Name:  name + "_" + percentile,
		Query: fmt.Sprintf(`histogram_quantile(%.2f, sum(rate(%s_bucket{job="%s"}[1m])) by (le))`, quantile, promName, JobWorkerApp),
	}
}
