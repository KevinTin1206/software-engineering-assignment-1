package store

import (
	"fmt"
	"net/mail"
	"rental/internal/domain"
	"sort"
	"strings"
	"unicode/utf8"
)

func text(value string, max int, required bool) (string, error) {
	v := strings.TrimSpace(value)
	if !utf8.ValidString(v) || utf8.RuneCountInString(v) > max || (required && v == "") {
		return "", fmt.Errorf("%w: Pflichtfeld fehlt oder Text ist zu lang (max. %d Zeichen)", domain.ErrInvalid, max)
	}
	return v, nil
}
func (s *Store) SaveItem(id int, in domain.Item) (domain.Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var err error
	if in.Name, err = text(in.Name, 120, true); err != nil {
		return domain.Item{}, err
	}
	if in.Size, err = text(in.Size, 80, false); err != nil {
		return domain.Item{}, err
	}
	if in.Details, err = text(in.Details, 500, false); err != nil {
		return domain.Item{}, err
	}
	if !s.categoryExists(in.Category) {
		return domain.Item{}, fmt.Errorf("%w: Unbekannte Kategorie", domain.ErrInvalid)
	}
	if id != 0 {
		old, ok := s.items[id]
		if !ok {
			return domain.Item{}, domain.ErrNotFound
		}
		if old.Category != in.Category {
			for _, r := range s.rentals {
				if r.ItemID == id {
					return domain.Item{}, fmt.Errorf("%w: Kategorie bei vorhandener Verleihhistorie nicht änderbar", domain.ErrConflict)
				}
			}
		}
	} else {
		s.nextItem++
		id = s.nextItem
	}
	in.ID = id
	s.items[id] = in
	return in, nil
}
func (s *Store) DeleteItem(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; !ok {
		return domain.ErrNotFound
	}
	for _, r := range s.rentals {
		if r.ItemID == id {
			return fmt.Errorf("%w: Ausrüstung mit Verleihhistorie kann nicht gelöscht werden", domain.ErrConflict)
		}
	}
	delete(s.items, id)
	return nil
}
func (s *Store) SavePerson(id int, in domain.Person) (domain.Person, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var err error
	for _, f := range []struct {
		value    *string
		max      int
		required bool
	}{{&in.FirstName, 80, true}, {&in.LastName, 80, true}, {&in.Email, 200, false}, {&in.Phone, 60, false}, {&in.DocumentType, 80, false}, {&in.DocumentNumber, 100, false}} {
		if *f.value, err = text(*f.value, f.max, f.required); err != nil {
			return domain.Person{}, err
		}
	}
	if in.Email != "" {
		address, err := mail.ParseAddress(in.Email)
		if err != nil || address.Address != in.Email {
			return domain.Person{}, fmt.Errorf("%w: E-Mail-Adresse ungültig", domain.ErrInvalid)
		}
	}
	if id != 0 {
		if _, ok := s.people[id]; !ok {
			return domain.Person{}, domain.ErrNotFound
		}
	} else {
		s.nextPerson++
		id = s.nextPerson
	}
	in.ID = id
	in.HasDocument = s.documents[id].Data != nil
	s.people[id] = in
	return in, nil
}
func (s *Store) DeletePerson(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.people[id]; !ok {
		return domain.ErrNotFound
	}
	for _, r := range s.rentals {
		if r.PersonID == id {
			return fmt.Errorf("%w: Person mit Verleihhistorie kann nicht gelöscht werden", domain.ErrConflict)
		}
	}
	delete(s.people, id)
	delete(s.documents, id)
	return nil
}
func (s *Store) SaveRate(in domain.Rate) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.categoryExists(in.Category) {
		return fmt.Errorf("%w: Unbekannte Kategorie", domain.ErrInvalid)
	}
	if err := in.Validate(); err != nil {
		return err
	}
	in = cloneRate(in)
	sort.Slice(in.Tiers, func(i, j int) bool { return in.Tiers[i].MinDays < in.Tiers[j].MinDays })
	s.rates[in.Category] = in
	return nil
}
