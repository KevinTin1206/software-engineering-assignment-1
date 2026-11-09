package handler

import (
	"bytes"
	"encoding/json"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"rental/internal/domain"
	"rental/internal/store"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func testAPI(t *testing.T) http.Handler {
	t.Helper()
	s := store.New([]string{"Standard"}, func() time.Time { return time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC) })
	if err := s.SaveRate(domain.Rate{Category: "Standard", DailyCents: 2500}); err != nil {
		t.Fatal(err)
	}
	return New(s, Config{Title: "Ski-Verleih", ItemLabel: "Ski", SizeLabel: "Länge"}, fstest.MapFS{"index.html": {Data: []byte("<!doctype html><title>Rental</title>")}})
}
func call(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}
func expect(t *testing.T, w *httptest.ResponseRecorder, status int) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("HTTP %d want %d: %s", w.Code, status, w.Body.String())
	}
}
func setupPeopleAndItem(t *testing.T, h http.Handler) {
	t.Helper()
	expect(t, call(h, "POST", "/api/items", `{"name":"Demo ski","category":"Standard","size":"170 cm"}`), 201)
	expect(t, call(h, "POST", "/api/people", `{"firstName":"Ada","lastName":"Demo"}`), 201)
}
func TestHTTPRentalWorkflow(t *testing.T) {
	h := testAPI(t)
	setupPeopleAndItem(t, h)
	q := call(h, "GET", "/api/quote?itemId=1&startDate=2026-11-01&endDate=2026-11-02", "")
	expect(t, q, 200)
	var quote domain.Quote
	json.Unmarshal(q.Body.Bytes(), &quote)
	if quote.TotalCents != 5000 {
		t.Fatal(quote)
	}
	expect(t, call(h, "POST", "/api/rentals", `{"itemId":1,"personId":1,"startDate":"2026-11-01","endDate":"2026-11-02"}`), 201)
	expect(t, call(h, "POST", "/api/rentals", `{"itemId":1,"personId":1,"startDate":"2026-11-02","endDate":"2026-11-03"}`), 409)
	expect(t, call(h, "DELETE", "/api/people/1", ""), 409)
	expect(t, call(h, "POST", "/api/rentals/1/checkout", ""), 200)
	w := call(h, "GET", "/api/state", "")
	expect(t, w, 200)
	var snap domain.Snapshot
	if err := json.Unmarshal(w.Body.Bytes(), &snap); err != nil {
		t.Fatal(err)
	}
	if snap.Dashboard.Rented != 1 || snap.Dashboard.Available != 0 {
		t.Fatal(snap.Dashboard)
	}
	expect(t, call(h, "PUT", "/api/rates", `{"category":"Standard","dailyCents":3000,"tiers":[{"minDays":3,"dailyCents":2000}]}`), 200)
	expect(t, call(h, "POST", "/api/rentals/1/return", ""), 200)
	expect(t, call(h, "POST", "/api/rentals/1/return", ""), 409)
	expect(t, call(h, "POST", "/api/rentals", `{"itemId":1,"personId":1,"startDate":"2026-11-01","endDate":"2026-11-03"}`), 201)
	expect(t, call(h, "POST", "/api/rentals/2/cancel", ""), 200)
	expect(t, call(h, "PUT", "/api/items/1", `{"name":"Updated","category":"Standard"}`), 200)
	expect(t, call(h, "PUT", "/api/people/1", `{"firstName":"Ada","lastName":"Updated"}`), 200)
	state := call(h, "GET", "/api/state", "")
	json.Unmarshal(state.Body.Bytes(), &snap)
	if snap.Rentals[0].Quote.TotalCents != 5000 || snap.Rentals[1].Quote.TotalCents != 6000 {
		t.Fatal(snap.Rentals)
	}
}
func upload(h http.Handler, id string, data []byte) *httptest.ResponseRecorder {
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)
	part, _ := writer.CreateFormFile("document", "fake.png")
	part.Write(data)
	writer.Close()
	req := httptest.NewRequest("POST", "/api/people/"+id+"/document", &b)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}
func TestHTTPIdentityUploadReplaceDelete(t *testing.T) {
	h := testAPI(t)
	setupPeopleAndItem(t, h)
	var p, j bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	png.Encode(&p, img)
	jpeg.Encode(&j, img, nil)
	expect(t, upload(h, "1", p.Bytes()), 200)
	w := call(h, "GET", "/api/people/1/document", "")
	expect(t, w, 200)
	if !bytes.Equal(w.Body.Bytes(), p.Bytes()) || w.Header().Get("Content-Type") != "image/png" || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("incorrect image response")
	}
	expect(t, upload(h, "1", j.Bytes()), 200)
	w = call(h, "GET", "/api/people/1/document", "")
	if w.Header().Get("Content-Type") != "image/jpeg" {
		t.Fatal("replacement failed")
	}
	expect(t, upload(h, "1", []byte("<script>not an image</script>")), 400)
	expect(t, upload(h, "1", make([]byte, domain.MaxDocumentBytes+1)), 413)
	expect(t, upload(h, "999", p.Bytes()), 404)
	// Failed uploads must not overwrite the last valid document.
	if got := call(h, "GET", "/api/people/1/document", ""); !bytes.Equal(got.Body.Bytes(), j.Bytes()) {
		t.Fatal("failed upload mutated document")
	}
	expect(t, call(h, "DELETE", "/api/people/1/document", ""), 204)
	expect(t, call(h, "GET", "/api/people/1/document", ""), 404)
	expect(t, call(h, "DELETE", "/api/people/999/document", ""), 404)
}
func TestHTTPValidationAndLocalOrigin(t *testing.T) {
	h := testAPI(t)
	for _, tc := range []struct {
		method, path, body string
		status             int
	}{
		{"POST", "/api/items", `{`, 400},
		{"POST", "/api/items", `{"name":"X","category":"Standard","unknown":1}`, 400},
		{"POST", "/api/items", `{"name":"X","category":"Standard"} {}`, 400},
		{"POST", "/api/items", `{"name":"","category":"Standard"}`, 400},
		{"DELETE", "/api/items/bad", "", 400},
		{"DELETE", "/api/items/999", "", 404},
		{"GET", "/api/no-such-route", "", 404},
		{"GET", "/api/quote?itemId=no", "", 400},
		{"PUT", "/api/rates", `{"category":"Standard","dailyCents":-1}`, 400},
		{"POST", "/api/items", `{"name":"` + strings.Repeat("x", 70000) + `","category":"Standard"}`, 413},
	} {
		expect(t, call(h, tc.method, tc.path, tc.body), tc.status)
	}
	req := httptest.NewRequest("POST", "/api/people", strings.NewReader(`{"firstName":"A","lastName":"B"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://other.example")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	expect(t, w, 403)
	req = httptest.NewRequest("POST", "/api/items", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "text/plain")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	expect(t, w, 415)
	state := call(h, "GET", "/api/state", "")
	var snap domain.Snapshot
	json.Unmarshal(state.Body.Bytes(), &snap)
	if len(snap.Items) != 0 || len(snap.People) != 0 {
		t.Fatal("rejected request mutated state")
	}
}
func TestStaticAndConfiguration(t *testing.T) {
	h := testAPI(t)
	w := call(h, "GET", "/", "")
	expect(t, w, 200)
	if !strings.Contains(w.Body.String(), "<!doctype html>") {
		t.Fatal("missing frontend")
	}
	w = call(h, "GET", "/api/config", "")
	expect(t, w, 200)
	if !strings.Contains(w.Body.String(), "Ski-Verleih") {
		t.Fatal(w.Body.String())
	}
	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("missing content-type protection")
	}
}
func TestUploadRequiresSingleFile(t *testing.T) {
	h := testAPI(t)
	setupPeopleAndItem(t, h)
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)
	var pngData bytes.Buffer
	png.Encode(&pngData, image.NewRGBA(image.Rect(0, 0, 1, 1)))
	for i := 0; i < 2; i++ {
		p, _ := writer.CreateFormFile("document", "demo.png")
		io.Copy(p, bytes.NewReader(pngData.Bytes()))
	}
	writer.Close()
	req := httptest.NewRequest("POST", "/api/people/1/document", &b)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	expect(t, w, 400)
	expect(t, call(h, "GET", "/api/people/1/document", ""), 404)
}

func TestHTTPStaleQuoteRequiresNewConfirmation(t *testing.T) {
	h := testAPI(t)
	setupPeopleAndItem(t, h)
	expect(t, call(h, "GET", "/api/quote?itemId=1&startDate=2026-11-01&endDate=2026-11-03", ""), 200)
	expect(t, call(h, "PUT", "/api/rates", `{"category":"Standard","dailyCents":3000}`), 200)
	expect(t, call(h, "POST", "/api/rentals", `{"itemId":1,"personId":1,"startDate":"2026-11-01","endDate":"2026-11-03","expectedTotalCents":7500}`), 409)
	var snap domain.Snapshot
	w := call(h, "GET", "/api/state", "")
	if err := json.Unmarshal(w.Body.Bytes(), &snap); err != nil {
		t.Fatal(err)
	}
	if len(snap.Rentals) != 0 {
		t.Fatal("unexpected rental from stale quote")
	}
	expect(t, call(h, "POST", "/api/rentals", `{"itemId":1,"personId":1,"startDate":"2026-11-01","endDate":"2026-11-03","expectedTotalCents":9000}`), 201)
}
