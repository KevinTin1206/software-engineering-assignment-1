package store

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"rental/internal/domain"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func fixture(t *testing.T) (*Store, domain.Item, domain.Person) {
	t.Helper()
	s := New([]string{"Standard", "Premium"}, func() time.Time { return time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC) })
	item, err := s.SaveItem(0, domain.Item{Name: "Test ski", Category: "Standard", Size: "170 cm"})
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.SavePerson(0, domain.Person{FirstName: "Ada", LastName: "Test", DocumentType: "Demo", DocumentNumber: "FAKE-1"})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SaveRate(domain.Rate{Category: "Standard", DailyCents: 2500, Tiers: []domain.Tier{{MinDays: 3, DailyCents: 2000}}}); err != nil {
		t.Fatal(err)
	}
	return s, item, p
}
func reserve(t *testing.T, s *Store, item, person int, start, end string) domain.Rental {
	t.Helper()
	r, err := s.Reserve(domain.ReservationInput{ItemID: item, PersonID: person, StartDate: start, EndDate: end})
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func TestRentalLifecycleAndQuoteSnapshot(t *testing.T) {
	s, item, p := fixture(t)
	r := reserve(t, s, item.ID, p.ID, "2026-11-01", "2026-11-03")
	if r.Quote.TotalCents != 6000 {
		t.Fatalf("quote %+v", r.Quote)
	}
	if err := s.SaveRate(domain.Rate{Category: "Standard", DailyCents: 9900}); err != nil {
		t.Fatal(err)
	}
	if snap := s.Snapshot(); snap.Rentals[0].Quote.TotalCents != 6000 || snap.Dashboard.Available != 0 || snap.Dashboard.ReservedToday != 1 {
		t.Fatalf("snapshot %+v", snap)
	}
	if _, err := s.Transition(r.ID, "checkout"); err != nil {
		t.Fatal(err)
	}
	if snap := s.Snapshot(); snap.Dashboard.Rented != 1 || snap.Dashboard.Available != 0 {
		t.Fatal(snap.Dashboard)
	}
	if _, err := s.Transition(r.ID, "cancel"); !errors.Is(err, domain.ErrConflict) {
		t.Fatal(err)
	}
	if _, err := s.Transition(r.ID, "return"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Transition(r.ID, "return"); !errors.Is(err, domain.ErrConflict) {
		t.Fatal(err)
	}
	if s.Snapshot().Dashboard.Available != 1 {
		t.Fatal("return did not release equipment")
	}
	reserve(t, s, item.ID, p.ID, "2026-11-01", "2026-11-01")
}
func TestOverlapInclusiveAndCancellation(t *testing.T) {
	s, i, p := fixture(t)
	r := reserve(t, s, i.ID, p.ID, "2026-11-03", "2026-11-05")
	for _, d := range [][2]string{{"2026-11-01", "2026-11-03"}, {"2026-11-05", "2026-11-06"}, {"2026-11-04", "2026-11-04"}, {"2026-11-01", "2026-11-08"}} {
		if _, err := s.Reserve(domain.ReservationInput{ItemID: i.ID, PersonID: p.ID, StartDate: d[0], EndDate: d[1]}); !errors.Is(err, domain.ErrConflict) {
			t.Fatalf("overlap %v: %v", d, err)
		}
	}
	reserve(t, s, i.ID, p.ID, "2026-11-06", "2026-11-06")
	if _, err := s.Transition(r.ID, "checkout"); !errors.Is(err, domain.ErrConflict) {
		t.Fatal("early checkout", err)
	}
	if _, err := s.Transition(r.ID, "cancel"); err != nil {
		t.Fatal(err)
	}
	reserve(t, s, i.ID, p.ID, "2026-11-03", "2026-11-05")
}
func TestConcurrentReservationsAndCheckout(t *testing.T) {
	s, i, p := fixture(t)
	var wins atomic.Int32
	var wg sync.WaitGroup
	for n := 0; n < 32; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.Reserve(domain.ReservationInput{ItemID: i.ID, PersonID: p.ID, StartDate: "2026-11-01", EndDate: "2026-11-02"}); err == nil {
				wins.Add(1)
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatalf("successful competing reservations=%d", wins.Load())
	}
	r := s.Snapshot().Rentals[0]
	wins.Store(0)
	for n := 0; n < 32; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.Transition(r.ID, "checkout"); err == nil {
				wins.Add(1)
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatal(wins.Load())
	}
}
func TestCatalogValidationAndHistoryProtection(t *testing.T) {
	s, i, p := fixture(t)
	if _, err := s.SaveItem(0, domain.Item{Name: "", Category: "Standard"}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatal(err)
	}
	if _, err := s.SaveItem(0, domain.Item{Name: "X", Category: "Unknown"}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatal(err)
	}
	if _, err := s.SavePerson(0, domain.Person{FirstName: "A"}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatal(err)
	}
	if _, err := s.SavePerson(0, domain.Person{FirstName: "A", LastName: "B", Email: "invalid"}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatal(err)
	}
	if _, err := s.Reserve(domain.ReservationInput{ItemID: i.ID, PersonID: 999, StartDate: "2026-11-01", EndDate: "2026-11-02"}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatal(err)
	}
	if _, err := s.Reserve(domain.ReservationInput{ItemID: i.ID, PersonID: p.ID, StartDate: "2026-10-31", EndDate: "2026-11-02"}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatal(err)
	}
	reserve(t, s, i.ID, p.ID, "2026-11-02", "2026-11-03")
	if err := s.DeleteItem(i.ID); !errors.Is(err, domain.ErrConflict) {
		t.Fatal(err)
	}
	if err := s.DeletePerson(p.ID); !errors.Is(err, domain.ErrConflict) {
		t.Fatal(err)
	}
	i.Category = "Premium"
	if _, err := s.SaveItem(i.ID, i); !errors.Is(err, domain.ErrConflict) {
		t.Fatal(err)
	}
	snap := s.Snapshot()
	snap.Items[0].Name = "mutation"
	snap.Rates[0].Tiers[0].DailyCents = 1
	if fresh := s.Snapshot(); fresh.Items[0].Name == "mutation" || fresh.Rates[0].Tiers[0].DailyCents == 1 {
		t.Fatal("snapshot aliases store")
	}
}
func TestOverduePhysicalLoanPreventsCheckout(t *testing.T) {
	s, i, p := fixture(t)
	r := reserve(t, s, i.ID, p.ID, "2026-11-01", "2026-11-01")
	if _, err := s.Transition(r.ID, "checkout"); err != nil {
		t.Fatal(err)
	}
	next := reserve(t, s, i.ID, p.ID, "2026-11-02", "2026-11-02")
	s.now = func() time.Time { return time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC) }
	if _, err := s.Transition(next.ID, "checkout"); !errors.Is(err, domain.ErrConflict) {
		t.Fatal(err)
	}
	if snap := s.Snapshot(); snap.Dashboard.Overdue != 1 || snap.Dashboard.Available != 0 {
		t.Fatal(snap.Dashboard)
	}
	if _, err := s.Transition(r.ID, "return"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Transition(next.ID, "checkout"); err != nil {
		t.Fatal(err)
	}
}
func TestIdentityDocumentMemoryLifecycle(t *testing.T) {
	s, _, p := fixture(t)
	var b bytes.Buffer
	png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	if err := s.PutDocument(p.ID, b.Bytes()); err != nil {
		t.Fatal(err)
	}
	doc, err := s.GetDocument(p.ID)
	if err != nil || doc.ContentType != "image/png" || !bytes.Equal(doc.Data, b.Bytes()) {
		t.Fatalf("document %+v %v", doc, err)
	}
	doc.Data[0] = 0
	again, _ := s.GetDocument(p.ID)
	if again.Data[0] == 0 {
		t.Fatal("document aliases store")
	}
	if !s.Snapshot().People[0].HasDocument {
		t.Fatal("missing document indicator")
	}
	if err := s.PutDocument(p.ID, []byte("not an image")); !errors.Is(err, domain.ErrInvalid) {
		t.Fatal(err)
	}
	if err := s.PutDocument(999, b.Bytes()); !errors.Is(err, domain.ErrNotFound) {
		t.Fatal(err)
	}
	if err := s.PutDocument(p.ID, make([]byte, 5*1024*1024+1)); !errors.Is(err, domain.ErrInvalid) {
		t.Fatal(err)
	}
	if err := s.DeleteDocument(p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetDocument(p.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatal(err)
	}
}

func TestStaleQuoteRejectedWithoutCreatingRental(t *testing.T) {
	s, i, p := fixture(t)
	q, err := s.Quote(i.ID, "2026-11-01", "2026-11-03")
	if err != nil {
		t.Fatal(err)
	}
	expected := q.TotalCents
	if err := s.SaveRate(domain.Rate{Category: "Standard", DailyCents: 3000}); err != nil {
		t.Fatal(err)
	}
	in := domain.ReservationInput{ItemID: i.ID, PersonID: p.ID, StartDate: "2026-11-01", EndDate: "2026-11-03", ExpectedTotalCents: &expected}
	if _, err := s.Reserve(in); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("stale quote accepted: %v", err)
	}
	if len(s.Snapshot().Rentals) != 0 {
		t.Fatal("stale quote created a rental")
	}
	expected = 9000
	r, err := s.Reserve(in)
	if err != nil {
		t.Fatal(err)
	}
	if r.ID != 1 || r.Quote.TotalCents != 9000 {
		t.Fatalf("fresh quote %+v", r)
	}
}
