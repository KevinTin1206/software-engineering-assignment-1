package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"net/url"
	"rental/internal/domain"
	"rental/internal/store"
	"strconv"
	"strings"
)

type Config struct {
	Title     string `json:"title"`
	ItemLabel string `json:"itemLabel"`
	SizeLabel string `json:"sizeLabel"`
}
type server struct {
	store  *store.Store
	config Config
}
type statusError struct {
	code    int
	message string
}

func (e statusError) Error() string { return e.message }

type endpoint func(http.ResponseWriter, *http.Request) error

func (s *server) wrap(fn endpoint) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			code := http.StatusInternalServerError
			var se statusError
			var tooLarge *http.MaxBytesError
			switch {
			case errors.As(err, &se):
				code = se.code
			case errors.As(err, &tooLarge):
				code = 413
			case errors.Is(err, domain.ErrInvalid):
				code = 400
			case errors.Is(err, domain.ErrNotFound):
				code = 404
			case errors.Is(err, domain.ErrConflict):
				code = 409
			}
			message := err.Error()
			if code == 500 {
				log.Printf("API error: %v", err)
				message = "Interner Serverfehler"
			}
			writeJSON(w, code, map[string]string{"error": message})
		}
	}
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("Write JSON: %v", err)
	}
}
func decodeJSON(w http.ResponseWriter, r *http.Request, dest any) error {
	media, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if media != "application/json" {
		return statusError{415, "Content-Type application/json erforderlich"}
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dest); err != nil {
		var large *http.MaxBytesError
		if errors.As(err, &large) {
			return err
		}
		return fmt.Errorf("%w: Fehlerhaftes JSON: %v", domain.ErrInvalid, err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		var large *http.MaxBytesError
		if errors.As(err, &large) {
			return err
		}
		return fmt.Errorf("%w: Genau ein JSON-Objekt erforderlich", domain.ErrInvalid)
	}
	return nil
}
func parseID(value string) (int, error) {
	id, err := strconv.Atoi(value)
	if err != nil || id < 1 {
		return 0, fmt.Errorf("%w: ID muss eine positive Ganzzahl sein", domain.ErrInvalid)
	}
	return id, nil
}
func New(st *store.Store, config Config, assets fs.FS) http.Handler {
	s := &server{store: st, config: config}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/config", s.wrap(func(w http.ResponseWriter, r *http.Request) error { writeJSON(w, 200, s.config); return nil }))
	mux.HandleFunc("GET /api/state", s.wrap(func(w http.ResponseWriter, r *http.Request) error { writeJSON(w, 200, s.store.Snapshot()); return nil }))
	mux.HandleFunc("GET /api/quote", s.wrap(s.quote))
	mux.HandleFunc("POST /api/items", s.wrap(s.saveItem))
	mux.HandleFunc("PUT /api/items/{id}", s.wrap(s.saveItem))
	mux.HandleFunc("DELETE /api/items/{id}", s.wrap(s.deleteItem))
	mux.HandleFunc("POST /api/people", s.wrap(s.savePerson))
	mux.HandleFunc("PUT /api/people/{id}", s.wrap(s.savePerson))
	mux.HandleFunc("DELETE /api/people/{id}", s.wrap(s.deletePerson))
	mux.HandleFunc("PUT /api/rates", s.wrap(s.saveRate))
	mux.HandleFunc("POST /api/rentals", s.wrap(s.reserve))
	mux.HandleFunc("POST /api/rentals/{id}/{action}", s.wrap(s.transition))
	mux.HandleFunc("POST /api/people/{id}/document", s.wrap(s.uploadDocument))
	mux.HandleFunc("GET /api/people/{id}/document", s.wrap(s.getDocument))
	mux.HandleFunc("DELETE /api/people/{id}/document", s.wrap(s.deleteDocument))
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 404, map[string]string{"error": "API-Endpunkt nicht gefunden"})
	})
	files := http.FileServer(http.FS(assets))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		files.ServeHTTP(w, r)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' blob:; script-src 'self'; style-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'")
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
			if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
				writeJSON(w, 403, map[string]string{"error": "Zugriff nur vom lokalen Frontend erlaubt"})
				return
			}
			if origin := r.Header.Get("Origin"); origin != "" {
				u, err := url.Parse(origin)
				if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host != r.Host {
					writeJSON(w, 403, map[string]string{"error": "Fremder Origin nicht erlaubt"})
					return
				}
			}
		}
		mux.ServeHTTP(w, r)
	})
}
func (s *server) saveItem(w http.ResponseWriter, r *http.Request) error {
	var in domain.Item
	if err := decodeJSON(w, r, &in); err != nil {
		return err
	}
	id := 0
	var err error
	if r.Method == "PUT" {
		id, err = parseID(r.PathValue("id"))
		if err != nil {
			return err
		}
	}
	item, err := s.store.SaveItem(id, in)
	if err != nil {
		return err
	}
	status := 200
	if r.Method == "POST" {
		status = 201
	}
	writeJSON(w, status, item)
	return nil
}
func (s *server) deleteItem(w http.ResponseWriter, r *http.Request) error {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		return err
	}
	if err = s.store.DeleteItem(id); err != nil {
		return err
	}
	w.WriteHeader(204)
	return nil
}
func (s *server) savePerson(w http.ResponseWriter, r *http.Request) error {
	var in domain.Person
	if err := decodeJSON(w, r, &in); err != nil {
		return err
	}
	id := 0
	var err error
	if r.Method == "PUT" {
		id, err = parseID(r.PathValue("id"))
		if err != nil {
			return err
		}
	}
	person, err := s.store.SavePerson(id, in)
	if err != nil {
		return err
	}
	status := 200
	if r.Method == "POST" {
		status = 201
	}
	writeJSON(w, status, person)
	return nil
}
func (s *server) deletePerson(w http.ResponseWriter, r *http.Request) error {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		return err
	}
	if err = s.store.DeletePerson(id); err != nil {
		return err
	}
	w.WriteHeader(204)
	return nil
}
func (s *server) saveRate(w http.ResponseWriter, r *http.Request) error {
	var in domain.Rate
	if err := decodeJSON(w, r, &in); err != nil {
		return err
	}
	if err := s.store.SaveRate(in); err != nil {
		return err
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
	return nil
}
func (s *server) quote(w http.ResponseWriter, r *http.Request) error {
	id, err := parseID(r.URL.Query().Get("itemId"))
	if err != nil {
		return err
	}
	q, err := s.store.Quote(id, r.URL.Query().Get("startDate"), r.URL.Query().Get("endDate"))
	if err != nil {
		return err
	}
	writeJSON(w, 200, q)
	return nil
}
func (s *server) reserve(w http.ResponseWriter, r *http.Request) error {
	var in domain.ReservationInput
	if err := decodeJSON(w, r, &in); err != nil {
		return err
	}
	rental, err := s.store.Reserve(in)
	if err != nil {
		return err
	}
	writeJSON(w, 201, rental)
	return nil
}
func (s *server) transition(w http.ResponseWriter, r *http.Request) error {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		return err
	}
	rental, err := s.store.Transition(id, r.PathValue("action"))
	if err != nil {
		return err
	}
	writeJSON(w, 200, rental)
	return nil
}
func (s *server) uploadDocument(w http.ResponseWriter, r *http.Request) error {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		return err
	}
	r.Body = http.MaxBytesReader(w, r.Body, domain.MaxDocumentBytes+64*1024)
	reader, err := r.MultipartReader()
	if err != nil {
		return fmt.Errorf("%w: multipart/form-data erforderlich", domain.ErrInvalid)
	}
	part, err := reader.NextPart()
	if err != nil {
		return fmt.Errorf("%w: Genau eine Bilddatei erforderlich", domain.ErrInvalid)
	}
	defer part.Close()
	if part.FormName() != "document" || part.FileName() == "" {
		return fmt.Errorf("%w: Dateifeld muss document heißen", domain.ErrInvalid)
	}
	data, err := io.ReadAll(io.LimitReader(part, domain.MaxDocumentBytes+1))
	if err != nil {
		return err
	}
	if len(data) > domain.MaxDocumentBytes {
		return statusError{413, "Bild überschreitet 5 MiB"}
	}
	if _, err = reader.NextPart(); err != io.EOF {
		return fmt.Errorf("%w: Genau eine Bilddatei erforderlich", domain.ErrInvalid)
	}
	if err = s.store.PutDocument(id, data); err != nil {
		return err
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
	return nil
}
func (s *server) getDocument(w http.ResponseWriter, r *http.Request) error {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		return err
	}
	doc, err := s.store.GetDocument(id)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", doc.ContentType)
	w.Header().Set("Content-Disposition", "inline; filename=\"document\"")
	w.Header().Set("Content-Length", strconv.Itoa(len(doc.Data)))
	if _, err := w.Write(doc.Data); err != nil {
		log.Printf("Write document: %v", err)
	}
	return nil
}
func (s *server) deleteDocument(w http.ResponseWriter, r *http.Request) error {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		return err
	}
	if err = s.store.DeleteDocument(id); err != nil {
		return err
	}
	w.WriteHeader(204)
	return nil
}
