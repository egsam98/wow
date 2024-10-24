package pow

import (
	"crypto/sha256"
	"encoding/binary"
	"math"
	"testing"
	"time"

	"github.com/egsam98/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPuzzle_Challenge(t *testing.T) {
	var expZeros uint = 2
	puzzle, err := NewPuzzle(func(uint) uint { return expZeros })
	require.NoError(t, err)

	challenge, zeros, err := puzzle.Challenge(0)
	assert.NoError(t, err)
	assert.Equal(t, expZeros, zeros)
	assert.Len(t, challenge, ChalLen)

	t.Run("zeros is out of [1, sha256.Size]", func(t *testing.T) {
		for _, cmplx := range []Complexity{
			func(uint) uint { return 0 },
			func(uint) uint { return sha256.Size + 1 },
		} {
			puzzle, err := NewPuzzle(cmplx)
			require.NoError(t, err)
			_, _, err = puzzle.Challenge(0)
			assert.ErrorIs(t, err, ErrInvalidZeros)
		}
	})
}

func TestPuzzle_Verify(t *testing.T) {
	puzzle, err := NewPuzzle(func(uint) uint { return 2 })
	require.NoError(t, err)
	challenge, zeros, err := puzzle.Challenge(0)
	require.NoError(t, err)

	assert.Eventually(t, func() bool {
		for i := uint64(0); i <= math.MaxUint64; i++ {
			var nonce [8]byte
			binary.LittleEndian.PutUint64(nonce[:], i)
			err = Verify(challenge, zeros, nonce)
			if err == nil {
				return true
			}
			assert.ErrorIs(t, err, ErrVerify)
		}
		return false
	}, 5*time.Second, 500*time.Millisecond)

	t.Run("invalid zeros", func(t *testing.T) {
		assert.Eventually(t, func() bool {
			err := Verify(challenge, 0, [8]byte{})
			return errors.Is(err, ErrInvalidZeros)
		}, time.Second, time.Millisecond)
	})
}
