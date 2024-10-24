package pow

import (
	"crypto/rand"
	"crypto/sha256"

	"github.com/egsam98/errors"
)

const ChalLen = 8

var ErrVerify = errors.New("pow: verification failed")
var ErrInvalidZeros = errors.Errorf("pow: zeros must be in range [1, %d]", sha256.Size)

// Puzzle is Proof of work algorithm inspired by Hashcash.
// It issues randomized challenge bytearray and required zeroes amount.
// The task is to select a nonce such that SHA-256(challenge + nonce) produces hash sequence starting with N zeros.
// Example: challenge = yg65xf, zeroes = 3, nonce = 5agt, SHA-256(yg65xf5agt) = 000gtgtth5dg, i.e. generated starts with 3 zeros.
// Zeros amount is determined in Complexity function that depends on active connections number.
type Puzzle struct {
	complex Complexity
}

type Complexity func(openConns uint) uint

func NewPuzzle(cmplx Complexity) (*Puzzle, error) {
	if cmplx == nil {
		return nil, errors.New("complexity func is required")
	}
	return &Puzzle{complex: cmplx}, nil
}

// Challenge issues new challenge sequence, required zeros amount.
// Errors:
// - ErrInvalidZeros see `validateZeros` func
func (p *Puzzle) Challenge(conns uint) ([ChalLen]byte, uint, error) {
	var buf [ChalLen]byte
	zeros := p.complex(conns)
	if err := validateZeros(zeros); err != nil {
		return buf, 0, err
	}
	_, err := rand.Read(buf[:])
	return buf, zeros, err
}

// Verify received nonce.
// Errors:
// - ErrInvalidZeros see `validateZeros` func
// - ErrVerify if verification is failed
func Verify(challenge [ChalLen]byte, zeros uint, nonce [8]byte) error {
	if err := validateZeros(zeros); err != nil {
		return err
	}
	h := sha256.New()
	h.Write(challenge[:])
	h.Write(nonce[:])
	for _, b := range h.Sum(nil)[:zeros] {
		if b != 0 {
			return ErrVerify
		}
	}
	return nil
}

func validateZeros(val uint) error {
	if val == 0 || val > sha256.Size {
		return ErrInvalidZeros
	}
	return nil
}
