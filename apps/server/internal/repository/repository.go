package repository

// Repository provides access to phrases in database
type Repository interface {
	Phrase() (*Phrase, error)
	AllPhrases() ([]Phrase, error)
}

// DTOs

type Phrase struct {
	Quote  string `json:"quote"`
	Author string `json:"author"`
}
