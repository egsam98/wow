package pow

import (
	"crypto/rand"
	"crypto/sha256"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

const ChalLen = 8

var ErrVerify = errors.New("pow: verification failed")

type Puzzle struct {
	zeroes uint
}

func NewPuzzle(zeroes uint) (*Puzzle, error) {
	if zeroes == 0 || zeroes > sha256.Size {
		return nil, errors.Errorf("zeroes must be in range [1, %d]", sha256.Size)
	}
	return &Puzzle{zeroes: zeroes}, nil
}

func (p *Puzzle) Challenge() ([ChalLen]byte, uint) {
	var buf [ChalLen]byte
	_, _ = rand.Read(buf[:])
	return buf, p.zeroes
}

func (p *Puzzle) Verify(challenge [ChalLen]byte, nonce [8]byte) error {
	h := sha256.New()
	h.Write(challenge[:])
	h.Write(nonce[:])
	for _, b := range h.Sum(nil)[:p.zeroes] {
		if b != 0 {
			return ErrVerify
		}
	}
	return nil
}

func (p *Puzzle) AdjustZeroes(conns uint) {
	var zeroes uint = 4
	if conns < 100 {
		zeroes = 3
	}

	if zeroes == p.zeroes {
		return
	}
	p.zeroes = zeroes
	log.Info().
		Uint("active_connections", conns).
		Uint("zeroes", p.zeroes).Msg("Puzzle: zeroes parameter has been adjusted")
}