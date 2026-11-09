package domain

import "errors"

var (
	ErrInvalid  = errors.New("Ungültige Eingabe")
	ErrNotFound = errors.New("Nicht gefunden")
	ErrConflict = errors.New("Konflikt")
)

const (
	Reserved         = "reserved"
	Active           = "active"
	Returned         = "returned"
	Cancelled        = "cancelled"
	MaxDocumentBytes = 5 * 1024 * 1024
)

type Item struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Size     string `json:"size"`
	Details  string `json:"details"`
}
type Person struct {
	ID             int    `json:"id"`
	FirstName      string `json:"firstName"`
	LastName       string `json:"lastName"`
	Email          string `json:"email"`
	Phone          string `json:"phone"`
	DocumentType   string `json:"documentType"`
	DocumentNumber string `json:"documentNumber"`
	HasDocument    bool   `json:"hasDocument"`
}
type Tier struct {
	MinDays    int64 `json:"minDays"`
	DailyCents int64 `json:"dailyCents"`
}
type Rate struct {
	Category   string `json:"category"`
	DailyCents int64  `json:"dailyCents"`
	Tiers      []Tier `json:"tiers"`
}
type Quote struct {
	Days       int64  `json:"days"`
	DailyCents int64  `json:"dailyCents"`
	TotalCents int64  `json:"totalCents"`
	Rule       string `json:"rule"`
}
type ReservationInput struct {
	ExpectedTotalCents *int64 `json:"expectedTotalCents,omitempty"`
	ItemID             int    `json:"itemId"`
	PersonID           int    `json:"personId"`
	StartDate          string `json:"startDate"`
	EndDate            string `json:"endDate"`
}
type Rental struct {
	ID         int    `json:"id"`
	ItemID     int    `json:"itemId"`
	PersonID   int    `json:"personId"`
	StartDate  string `json:"startDate"`
	EndDate    string `json:"endDate"`
	Status     string `json:"status"`
	Quote      Quote  `json:"quote"`
	CreatedAt  string `json:"createdAt"`
	ReturnedAt string `json:"returnedAt,omitempty"`
}

// Blocks compares inclusive calendar dates, extending overdue physical loans through today.
func (r Rental) Blocks(start, end, today string) bool {
	if r.Status != Reserved && r.Status != Active {
		return false
	}
	until := r.EndDate
	if r.Status == Active && until < today {
		until = today
	}
	return r.StartDate <= end && start <= until
}

type Document struct {
	Data        []byte
	ContentType string
}
type Dashboard struct {
	Available     int `json:"available"`
	Rented        int `json:"rented"`
	Total         int `json:"total"`
	People        int `json:"people"`
	ReservedToday int `json:"reservedToday"`
	Overdue       int `json:"overdue"`
}
type Snapshot struct {
	Items      []Item    `json:"items"`
	People     []Person  `json:"people"`
	Rentals    []Rental  `json:"rentals"`
	Rates      []Rate    `json:"rates"`
	Categories []string  `json:"categories"`
	Today      string    `json:"today"`
	Dashboard  Dashboard `json:"dashboard"`
}
