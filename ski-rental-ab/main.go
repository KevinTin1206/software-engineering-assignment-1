package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"rental/internal/domain"
	"rental/internal/handler"
	"rental/internal/store"
	"rental/web"
	"time"
)

func newApplication(clock func() time.Time) (http.Handler, error) {
	categories := []string{"Standard", "Premium", "Kinder"}
	s := store.New(categories, clock)
	for n, c := range categories {
		daily := []int64{2500, 3900, 1500}[n]
		if err := s.SaveRate(domain.Rate{Category: c, DailyCents: daily, Tiers: []domain.Tier{{MinDays: 3, DailyCents: daily - 300}, {MinDays: 7, DailyCents: daily - 500}}}); err != nil {
			return nil, fmt.Errorf("seed rate: %w", err)
		}
	}
	items := []domain.Item{
		{Name: "Alpine Classic 170", Category: "Standard", Size: "170 cm", Details: "Allround-Ski · inkl. Bindung"},
		{Name: "Alpine Classic 160", Category: "Standard", Size: "160 cm", Details: "Allround-Ski · inkl. Bindung"},
		{Name: "Summit Pro 178", Category: "Premium", Size: "178 cm", Details: "Sportlicher Carving-Ski"},
		{Name: "Summit Pro 168", Category: "Premium", Size: "168 cm", Details: "Sportlicher Carving-Ski"},
		{Name: "Junior Snow 120", Category: "Kinder", Size: "120 cm", Details: "Kinder-Ski"},
		{Name: "Junior Snow 140", Category: "Kinder", Size: "140 cm", Details: "Kinder-Ski"},
	}
	for _, i := range items {
		if _, err := s.SaveItem(0, i); err != nil {
			return nil, fmt.Errorf("seed item: %w", err)
		}
	}
	for _, p := range []domain.Person{{FirstName: "Anna", LastName: "Muster", Email: "anna@example.invalid", DocumentType: "Demo-Ausweis", DocumentNumber: "FAKE-001"}, {FirstName: "Lukas", LastName: "Beispiel", Email: "lukas@example.invalid"}, {FirstName: "Mia", LastName: "Demo"}} {
		if _, err := s.SavePerson(0, p); err != nil {
			return nil, fmt.Errorf("seed person: %w", err)
		}
	}
	today := clock().Format("2006-01-02")
	tomorrow := clock().AddDate(0, 0, 1).Format("2006-01-02")
	r, err := s.Reserve(domain.ReservationInput{ItemID: 1, PersonID: 1, StartDate: today, EndDate: tomorrow})
	if err != nil {
		return nil, fmt.Errorf("seed rental: %w", err)
	}
	if _, err := s.Transition(r.ID, "checkout"); err != nil {
		return nil, fmt.Errorf("seed checkout: %w", err)
	}
	if _, err := s.Reserve(domain.ReservationInput{ItemID: 3, PersonID: 2, StartDate: today, EndDate: tomorrow}); err != nil {
		return nil, fmt.Errorf("seed reservation: %w", err)
	}
	return handler.New(s, handler.Config{Title: "Ski-Verleih", ItemLabel: "Ski", SizeLabel: "Länge / Größe"}, web.Assets), nil
}
func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "HTTP listen address")
	flag.Parse()
	app, err := newApplication(time.Now)
	if err != nil {
		log.Fatal(err)
	}
	server := &http.Server{Addr: *addr, Handler: app, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 * 1024}
	log.Printf("Ski-Verleih: http://%s — Daten nur im Arbeitsspeicher, Strg+C beendet den Server", *addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
