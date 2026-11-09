package domain

import (
	"errors"
	"testing"
)

func TestQuoteCalendarDaysAndTiers(t *testing.T) {
	rate := Rate{Category: "Standard", DailyCents: 2500, Tiers: []Tier{{MinDays: 3, DailyCents: 2200}, {MinDays: 7, DailyCents: 1800}}}
	for _, tc := range []struct {
		start, end  string
		days, price int64
	}{
		{"2026-11-01", "2026-11-01", 1, 2500},
		{"2026-11-01", "2026-11-02", 2, 2500},
		{"2026-11-01", "2026-11-03", 3, 2200},
		{"2026-11-01", "2026-11-07", 7, 1800},
		{"2028-02-28", "2028-03-01", 3, 2200},
	} {
		q, err := CalculateQuote(rate, tc.start, tc.end)
		if err != nil || q.Days != tc.days || q.DailyCents != tc.price || q.TotalCents != tc.days*tc.price {
			t.Fatalf("%+v: quote=%+v err=%v", tc, q, err)
		}
	}
}
func TestRejectInvalidDatesAndRates(t *testing.T) {
	good := Rate{Category: "Standard", DailyCents: 2500}
	for _, dates := range [][2]string{{"2026-02-30", "2026-03-01"}, {"2026-11-02", "2026-11-01"}, {"bad", "2026-11-01"}, {"2026-1-1", "2026-11-01"}} {
		if _, err := CalculateQuote(good, dates[0], dates[1]); !errors.Is(err, ErrInvalid) {
			t.Fatalf("accepted %v: %v", dates, err)
		}
	}
	for _, r := range []Rate{{Category: "", DailyCents: 10}, {Category: "S", DailyCents: 0}, {Category: "S", DailyCents: -1}, {Category: "S", DailyCents: 100000001}, {Category: "S", DailyCents: 100, Tiers: []Tier{{MinDays: 1, DailyCents: 90}}}, {Category: "S", DailyCents: 100, Tiers: []Tier{{MinDays: 3, DailyCents: 90}, {MinDays: 3, DailyCents: 80}}}} {
		if err := r.Validate(); !errors.Is(err, ErrInvalid) {
			t.Fatalf("accepted %+v", r)
		}
	}
}
