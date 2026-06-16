package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cascade-analytics/cascade/services/ingest/internal/queue"
	"github.com/cascade-analytics/cascade/services/ingest/internal/schema"
	"github.com/cascade-analytics/cascade/services/ingest/internal/tenant"
	"github.com/google/uuid"
)

type CaptureHandler struct {
	q *queue.Enqueuer
}

func NewCaptureHandler(q *queue.Enqueuer) *CaptureHandler {
	return &CaptureHandler{q: q}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// Capture handles POST /v1/capture — single event.
func (h *CaptureHandler) Capture(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := tenant.FromContext(r.Context())

	var ev schema.CaptureEvent
	if err := json.NewDecoder(r.Body).Decode(&ev); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := validateEvent(ev); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	stored := toStored(ev, tenantID)
	id, err := h.q.Enqueue(r.Context(), stored)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "enqueue failed")
		return
	}
	_ = id
	writeJSON(w, http.StatusAccepted, map[string]string{"event_id": stored.EventID})
}

// Batch handles POST /v1/batch — up to 1000 events.
func (h *CaptureHandler) Batch(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := tenant.FromContext(r.Context())

	var req struct {
		Events []schema.CaptureEvent `json:"events"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if len(req.Events) == 0 {
		writeError(w, http.StatusBadRequest, "events array is empty")
		return
	}
	if len(req.Events) > 1000 {
		writeError(w, http.StatusBadRequest, "maximum 1000 events per batch")
		return
	}

	ids := make([]string, 0, len(req.Events))
	for _, ev := range req.Events {
		if err := validateEvent(ev); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("event validation: %s", err))
			return
		}
		stored := toStored(ev, tenantID)
		if _, err := h.q.Enqueue(r.Context(), stored); err != nil {
			writeError(w, http.StatusInternalServerError, "enqueue failed")
			return
		}
		ids = append(ids, stored.EventID)
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"accepted":  len(ids),
		"event_ids": ids,
	})
}

func validateEvent(ev schema.CaptureEvent) error {
	if ev.Event == "" {
		return fmt.Errorf("event field is required")
	}
	if ev.DistinctID == "" {
		return fmt.Errorf("distinct_id field is required")
	}
	if ev.Timestamp == "" {
		return fmt.Errorf("timestamp field is required")
	}
	return nil
}

func toStored(ev schema.CaptureEvent, tenantID string) schema.StoredEvent {
	ts, err := time.Parse(time.RFC3339Nano, ev.Timestamp)
	if err != nil {
		ts = time.Now().UTC()
	}

	props := make(map[string]string, len(ev.Properties))
	for k, v := range ev.Properties {
		props[k] = fmt.Sprintf("%v", v)
	}

	return schema.StoredEvent{
		EventID:    uuid.Must(uuid.NewV7()).String(),
		TenantID:   tenantID,
		Event:      ev.Event,
		DistinctID: ev.DistinctID,
		Timestamp:  ts,
		ReceivedAt: time.Now().UTC(),
		Properties: props,
	}
}
