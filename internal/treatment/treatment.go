package treatment

import "time"

type Treatment struct {
	ID              int64
	Name            string
	Slug            string
	Description     string
	ImageURL        string
	Price           int64
	DurationMinutes int32
	IsActive        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Input struct {
	Name            string
	Description     string
	ImageURL        string
	Price           int64
	DurationMinutes int32
	IsActive        bool
}
