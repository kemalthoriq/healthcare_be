package therapist

import "time"

type Therapist struct {
	ID        int64
	Name      string
	Slug      string
	Title     string
	Bio       string
	PhotoURL  string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Input struct {
	Name     string
	Title    string
	Bio      string
	PhotoURL string
	IsActive bool
}
