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
	categories := []string{"City", "Trekking", "E-Bike"}
	s := store.New(categories, clock)
	for n, c := range categories {
		daily := []int64{1800, 2400, 3500}[n]
		if err := s.SaveRate(domain.Rate{Category: c, DailyCents: daily, Tiers: []domain.Tier{{MinDays: 3, DailyCents: daily - 300}, {MinDays: 7, DailyCents: daily - 500}}}); err != nil {
			return nil, fmt.Errorf("seed rate: %w", err)
		}
	}
	items := []domain.Item{
		{Name: "City Cruiser M", Category: "City", Size: "M · 50 cm", Details: "Stadtrad · 7 Gänge"},
		{Name: "City Cruiser L", Category: "City", Size: "L · 55 cm", Details: "Stadtrad · 7 Gänge"},
		{Name: "Trail Tour M", Category: "Trekking", Size: "M · 51 cm", Details: "Trekkingrad · 21 Gänge"},
		{Name: "Trail Tour L", Category: "Trekking", Size: "L · 56 cm", Details: "Trekkingrad · 21 Gänge"},
		{Name: "Volt City M", Category: "E-Bike", Size: "M · 48 cm", Details: "E-Bike · 500 Wh Akku"},
		{Name: "Volt City L", Category: "E-Bike", Size: "L · 53 cm", Details: "E-Bike · 500 Wh Akku"},
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
	return handler.New(s, handler.Config{Title: "Fahrrad-Verleih", ItemLabel: "Fahrräder", SizeLabel: "Rahmengröße"}, web.Assets), nil
}
func main() {
	addr := flag.String("addr", "127.0.0.1:8081", "HTTP listen address")
	flag.Parse()
	app, err := newApplication(time.Now)
	if err != nil {
		log.Fatal(err)
	}
	server := &http.Server{Addr: *addr, Handler: app, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 * 1024}
	log.Printf("Fahrrad-Verleih: http://%s — Daten nur im Arbeitsspeicher, Strg+C beendet den Server", *addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
