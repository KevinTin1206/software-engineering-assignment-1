package domain

import (
	"fmt"
	"strings"
	"time"
)

func ParseDate(value string) (time.Time, error) {
	d, err := time.Parse("2006-01-02", value)
	if err != nil || d.Format("2006-01-02") != value {
		return time.Time{}, fmt.Errorf("%w: Datum muss YYYY-MM-DD entsprechen", ErrInvalid)
	}
	return d, nil
}
func (r Rate) Validate() error {
	if strings.TrimSpace(r.Category) == "" || r.DailyCents < 1 || r.DailyCents > 100000000 {
		return fmt.Errorf("%w: Kategorie und positiver Tagespreis (max. 1.000.000 EUR) erforderlich", ErrInvalid)
	}
	seen := map[int64]bool{}
	for _, t := range r.Tiers {
		if t.MinDays < 2 || t.MinDays > 3660000 || t.DailyCents < 1 || t.DailyCents > 100000000 || seen[t.MinDays] {
			return fmt.Errorf("%w: Preisstaffeln benötigen eindeutige Mindesttage ab 2 und positive Preise", ErrInvalid)
		}
		seen[t.MinDays] = true
	}
	return nil
}
func CalculateQuote(rate Rate, start, end string) (Quote, error) {
	if err := rate.Validate(); err != nil {
		return Quote{}, err
	}
	a, err := ParseDate(start)
	if err != nil {
		return Quote{}, err
	}
	b, err := ParseDate(end)
	if err != nil {
		return Quote{}, err
	}
	if b.Before(a) {
		return Quote{}, fmt.Errorf("%w: Ende liegt vor Beginn", ErrInvalid)
	}
	days := (b.Unix()-a.Unix())/86400 + 1
	price := rate.DailyCents
	threshold := int64(1)
	rule := "Standard-Tagespreis"
	for _, t := range rate.Tiers {
		if t.MinDays <= days && t.MinDays > threshold {
			threshold = t.MinDays
			price = t.DailyCents
			rule = fmt.Sprintf("Ab %d Tagen", t.MinDays)
		}
	}
	return Quote{Days: days, DailyCents: price, TotalCents: days * price, Rule: rule}, nil
}
