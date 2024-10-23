package api

import (
	"context"
	"encoding/binary"
	"math"
	"net"
	"time"

	"github.com/egsam98/errors"

	"github.com/egsam98/wow/internal/pow"
)

type Client struct {
	conn net.Conn
}

func Dial(addr string) (*Client, error) {
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return nil, errors.Wrap(err, "connect to WordsOfWisdom server")
	}
	return &Client{conn: conn}, nil
}

func (c *Client) Phrase(ctx context.Context) (*PhraseResponse, error) {
	return clientDo[*PhraseRequest, *PhraseResponse](c, ctx, new(PhraseRequest))
}

func (c *Client) Close() error { return c.conn.Close() }

func clientDo[In, Out message](c *Client, ctx context.Context, req In) (Out, error) {
	if err := write(c.conn, req); err != nil {
		return *new(Out), errors.Wrap(err, "%T: write request", req)
	}
	for {
		msg, err := read(c.conn)
		if err != nil {
			return *new(Out), errors.Wrap(err, "%T: read response", req)
		}
		switch msg := msg.(type) {
		case *PowChallengeResponse:
			nonce, err := computePoW(ctx, msg.Challenge, msg.Zeroes)
			if err != nil {
				return *new(Out), err
			}
			if err := write(c.conn, &PowNonceRequest{Challenge: msg.Challenge, Nonce: nonce}); err != nil {
				return *new(Out), errors.Wrap(err, "PowNonceRequest: write request")
			}
		case Out:
			return msg, nil
		case *ErrorResponse:
			return *new(Out), msg
		default:
			return *new(Out), errors.Errorf("unexpected response message %#v", msg)
		}
	}
}

func computePoW(ctx context.Context, challenge [pow.ChalLen]byte, zeroes uint) ([8]byte, error) {
	puzzle, err := pow.NewPuzzle(zeroes)
	if err != nil {
		return [8]byte{}, err
	}

	var nonce [8]byte
	for i := uint64(0); i <= math.MaxUint64; i++ {
		select {
		case <-ctx.Done():
			return nonce, ctx.Err()
		default:
		}

		binary.LittleEndian.PutUint64(nonce[:], i)
		err = puzzle.Verify(challenge, nonce)
		if !errors.Is(err, pow.ErrVerify) {
			break
		}
	}
	return nonce, err
}
