package promo

import "time"

const DateLayout = "2006-01-02"

type Treatment struct {
	ID       int64
	Name     string
	Slug     string
	IsActive bool
}

type Promo struct {
	ID          int64
	Title       string
	Slug        string
	Description string
	ImageURL    string
	StartDate   time.Time
	EndDate     time.Time
	IsActive    bool
	Treatments  []Treatment
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Input struct {
	Title        string
	Description  string
	ImageURL     string
	StartDate    string
	EndDate      string
	IsActive     bool
	TreatmentIDs []int64
}

type Values struct {
	Title        string
	Description  string
	ImageURL     string
	StartDate    time.Time
	EndDate      time.Time
	IsActive     bool
	TreatmentIDs []int64
}
