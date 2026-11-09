package main

import (
	"encoding/json"
	"net/http/httptest"
	"rental/internal/domain"
	"testing"
	"time"
)

func TestDemoApplicationStartsWithConsistentInventory(t *testing.T) {
	h, err := newApplication(func() time.Time { return time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/state", nil))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var snap domain.Snapshot
	if err := json.Unmarshal(w.Body.Bytes(), &snap); err != nil {
		t.Fatal(err)
	}
	if snap.Dashboard.Total != 6 || snap.Dashboard.Rented != 1 || snap.Dashboard.Available != 4 || snap.Dashboard.People != 3 || snap.Dashboard.ReservedToday != 1 {
		t.Fatal(snap.Dashboard)
	}
	for _, item := range snap.Items {
		if _, err := domain.CalculateQuote(snap.Rates[0], snap.Today, snap.Today); err != nil || item.Category == "" {
			t.Fatal("invalid seed", err)
		}
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/app.js", nil))
	if w.Code != 200 || w.Body.Len() == 0 {
		t.Fatal("embedded JS missing")
	}
}
func TestDomainSpecificConfiguration(t *testing.T) {
	h, err := newApplication(time.Now)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/config", nil))
	var config map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &config); err != nil {
		t.Fatal(err)
	}
	if config["title"] != "Ski-Verleih" || config["sizeLabel"] != "Länge / Größe" {
		t.Fatal(config)
	}
}
