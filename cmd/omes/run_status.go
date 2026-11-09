package main

import (
	"encoding/json"
	"net/http"
)

func (r *scenarioRunner) statusHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		result := r.finalResult.Load()
		if result == nil {
			running := newRunResult(runEnd{}, r.snapshotCounts(), r.currentState())
			result = &running
		}
		data, err := json.Marshal(result)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(append(data, '\n')); err != nil {
			r.logger.Errorf("Writing run status: %v", err)
		}
	})
}
