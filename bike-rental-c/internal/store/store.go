package store

import (
	"rental/internal/domain"
	"sort"
	"sync"
	"time"
)

// Store owns all mutable state. Every exported operation takes the same mutex,
// making availability checks and reservations one atomic transaction.
type Store struct {
	mu                               sync.RWMutex
	items                            map[int]domain.Item
	people                           map[int]domain.Person
	rentals                          map[int]domain.Rental
	rates                            map[string]domain.Rate
	documents                        map[int]domain.Document
	categories                       []string
	nextItem, nextPerson, nextRental int
	now                              func() time.Time
}

func New(categories []string, clock func() time.Time) *Store {
	return &Store{items: map[int]domain.Item{}, people: map[int]domain.Person{}, rentals: map[int]domain.Rental{}, rates: map[string]domain.Rate{}, documents: map[int]domain.Document{}, categories: append([]string{}, categories...), now: clock}
}
func (s *Store) today() string            { return s.now().Format("2006-01-02") }
func cloneRate(r domain.Rate) domain.Rate { r.Tiers = append([]domain.Tier{}, r.Tiers...); return r }
func (s *Store) categoryExists(category string) bool {
	for _, c := range s.categories {
		if c == category {
			return true
		}
	}
	return false
}
func (s *Store) Snapshot() domain.Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snap := domain.Snapshot{Items: []domain.Item{}, People: []domain.Person{}, Rentals: []domain.Rental{}, Rates: []domain.Rate{}, Categories: append([]string{}, s.categories...), Today: s.today()}
	for _, i := range s.items {
		snap.Items = append(snap.Items, i)
	}
	for _, p := range s.people {
		p.HasDocument = s.documents[p.ID].Data != nil
		snap.People = append(snap.People, p)
	}
	occupied := map[int]bool{}
	reserved := map[int]bool{}
	for _, r := range s.rentals {
		snap.Rentals = append(snap.Rentals, r)
		if r.Blocks(snap.Today, snap.Today, snap.Today) {
			occupied[r.ItemID] = true
		}
		if r.Status == domain.Active {
			snap.Dashboard.Rented++
			if r.EndDate < snap.Today {
				snap.Dashboard.Overdue++
			}
		}
		if r.Status == domain.Reserved && r.Blocks(snap.Today, snap.Today, snap.Today) {
			reserved[r.ItemID] = true
		}
	}
	for _, category := range s.categories {
		if r, ok := s.rates[category]; ok {
			snap.Rates = append(snap.Rates, cloneRate(r))
		}
	}
	snap.Dashboard.Total = len(s.items)
	snap.Dashboard.People = len(s.people)
	snap.Dashboard.Available = len(s.items) - len(occupied)
	snap.Dashboard.ReservedToday = len(reserved)
	sort.Slice(snap.Items, func(i, j int) bool { return snap.Items[i].ID < snap.Items[j].ID })
	sort.Slice(snap.People, func(i, j int) bool { return snap.People[i].ID < snap.People[j].ID })
	sort.Slice(snap.Rentals, func(i, j int) bool { return snap.Rentals[i].ID < snap.Rentals[j].ID })
	return snap
}
