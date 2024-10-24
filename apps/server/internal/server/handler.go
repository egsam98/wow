package server

import (
	"context"
	"iter"

	"github.com/egsam98/wow/apps/server/internal/repository"
	"github.com/egsam98/wow/internal/api"
)

// Handler impls api.ServerHandler handling TCP requests
type Handler struct {
	repo repository.Repository
}

func NewHandler(repo repository.Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) Phrase(context.Context, *api.PhraseRequest) (*api.PhraseResponse, error) {
	phrase, err := h.repo.Phrase()
	if err != nil {
		return nil, err
	}
	return (*api.PhraseResponse)(phrase), nil
}

func (h *Handler) AllPhrases(context.Context, *api.AllPhrasesRequest) iter.Seq2[*api.PhraseResponse, error] {
	return func(yield func(*api.PhraseResponse, error) bool) {
		phrases, err := h.repo.AllPhrases()
		if err != nil {
			yield(nil, err)
			return
		}
		for _, phrase := range phrases {
			if !yield((*api.PhraseResponse)(&phrase), nil) {
				return
			}
		}
	}
}
