package store

import (
	"fmt"
	"rental/internal/domain"
	"time"
)

// quoteLocked is called only with the store mutex held.
func (s *Store) quoteLocked(itemID int, start, end string) (domain.Quote, error) {
	item, ok := s.items[itemID]
	if !ok {
		return domain.Quote{}, domain.ErrNotFound
	}
	rate, ok := s.rates[item.Category]
	if !ok {
		return domain.Quote{}, fmt.Errorf("%w: Kein Tarif für diese Kategorie", domain.ErrConflict)
	}
	q, err := domain.CalculateQuote(rate, start, end)
	if err != nil {
		return domain.Quote{}, err
	}
	if start < s.today() {
		return domain.Quote{}, fmt.Errorf("%w: Beginn darf nicht in der Vergangenheit liegen", domain.ErrInvalid)
	}
	return q, nil
}
func (s *Store) Quote(itemID int, start, end string) (domain.Quote, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.quoteLocked(itemID, start, end)
}
func (s *Store) Reserve(in domain.ReservationInput) (domain.Rental, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.people[in.PersonID]; !ok {
		return domain.Rental{}, fmt.Errorf("%w: Person", domain.ErrNotFound)
	}
	q, err := s.quoteLocked(in.ItemID, in.StartDate, in.EndDate)
	if err != nil {
		return domain.Rental{}, err
	}
	for _, r := range s.rentals {
		if r.ItemID == in.ItemID && r.Blocks(in.StartDate, in.EndDate, s.today()) {
			return domain.Rental{}, fmt.Errorf("%w: Ausrüstung im gewählten Zeitraum belegt", domain.ErrConflict)
		}
	}
	if in.ExpectedTotalCents != nil && *in.ExpectedTotalCents != q.TotalCents {
		return domain.Rental{}, fmt.Errorf("%w: Preis wurde geändert. Bitte neuen Preis prüfen und erneut bestätigen", domain.ErrConflict)
	}
	s.nextRental++
	r := domain.Rental{ID: s.nextRental, ItemID: in.ItemID, PersonID: in.PersonID, StartDate: in.StartDate, EndDate: in.EndDate, Status: domain.Reserved, Quote: q, CreatedAt: s.now().Format(time.RFC3339)}
	s.rentals[r.ID] = r
	return r, nil
}
func (s *Store) Transition(id int, action string) (domain.Rental, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.rentals[id]
	if !ok {
		return domain.Rental{}, domain.ErrNotFound
	}
	switch action {
	case "checkout":
		if r.Status != domain.Reserved || s.today() < r.StartDate || s.today() > r.EndDate {
			return domain.Rental{}, fmt.Errorf("%w: Ausgabe nur für Reservierungen innerhalb des Mietzeitraums", domain.ErrConflict)
		}
		for _, other := range s.rentals {
			if other.ItemID == r.ItemID && other.Status == domain.Active {
				return domain.Rental{}, fmt.Errorf("%w: Ausrüstung wurde noch nicht zurückgegeben", domain.ErrConflict)
			}
		}
		r.Status = domain.Active
	case "return":
		if r.Status != domain.Active {
			return domain.Rental{}, fmt.Errorf("%w: Nur ausgeliehene Ausrüstung kann zurückgegeben werden", domain.ErrConflict)
		}
		r.Status = domain.Returned
		r.ReturnedAt = s.now().Format(time.RFC3339)
	case "cancel":
		if r.Status != domain.Reserved {
			return domain.Rental{}, fmt.Errorf("%w: Nur Reservierungen können storniert werden", domain.ErrConflict)
		}
		r.Status = domain.Cancelled
	default:
		return domain.Rental{}, fmt.Errorf("%w: Unbekannte Aktion", domain.ErrInvalid)
	}
	s.rentals[id] = r
	return r, nil
}
