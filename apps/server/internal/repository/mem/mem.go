package mem

import (
	_ "embed"
	"encoding/json"
	"math/rand/v2"

	"github.com/pkg/errors"

	"github.com/egsam98/wow/apps/server/internal/repository"
)

//go:embed phrases.json
var phrases json.RawMessage

type Repository struct {
	// sync.RWMutex not necessary for read-only storage
	phrases []repository.Phrase
}

func NewRepository() (*Repository, error) {
	var self Repository
	if err := json.Unmarshal(phrases, &self.phrases); err != nil {
		return nil, errors.Wrapf(err, "unmarshal %s into %T", phrases, self.phrases)
	}
	return &self, nil
}

func (r *Repository) Phrase() (*repository.Phrase, error) {
	i := rand.IntN(len(r.phrases))
	return &r.phrases[i], nil
}
