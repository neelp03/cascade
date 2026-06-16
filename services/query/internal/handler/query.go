package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cascade-analytics/cascade/services/query/internal/store"
	"github.com/cascade-analytics/cascade/services/query/internal/tenant"
)

type QueryHandler struct {
	store *store.ClickHouseStore
}

func NewQueryHandler(s *store.ClickHouseStore) *QueryHandler {
	return &QueryHandler{store: s}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// Count handles GET /v1/count
func (h *QueryHandler) Count(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := tenant.FromContext(r.Context())
	q := r.URL.Query()

	event := q.Get("event")
	if event == "" {
		writeError(w, http.StatusBadRequest, "event param required")
		return
	}
	from, to, err := parseTimeRange(q.Get("from"), q.Get("to"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	count, err := h.store.CountEvents(r.Context(), tenantID, event, from, to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"count": count})
}

// TimeSeries handles GET /v1/timeseries
func (h *QueryHandler) TimeSeries(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := tenant.FromContext(r.Context())
	q := r.URL.Query()

	event := q.Get("event")
	if event == "" {
		writeError(w, http.StatusBadRequest, "event param required")
		return
	}
	interval := q.Get("interval")
	if interval == "" {
		interval = "hour"
	}
	from, to, err := parseTimeRange(q.Get("from"), q.Get("to"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	pts, err := h.store.TimeSeries(r.Context(), tenantID, event, from, to, interval, "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}

	type point struct {
		Timestamp string `json:"timestamp"`
		Count     uint64 `json:"count"`
	}
	series := make([]point, len(pts))
	for i, p := range pts {
		series[i] = point{Timestamp: p.Timestamp.Format(time.RFC3339), Count: p.Count}
	}
	writeJSON(w, http.StatusOK, map[string]any{"series": series})
}

func parseTimeRange(fromStr, toStr string) (time.Time, time.Time, error) {
	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid from: %s", fromStr)
	}
	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid to: %s", toStr)
	}
	return from, to, nil
}
