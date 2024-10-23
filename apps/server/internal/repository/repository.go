package repository

type Repository interface {
	Phrase() (*Phrase, error)
}

// DTOs

type Phrase struct {
	Quote  string `json:"quote"`
	Author string `json:"author"`
}
