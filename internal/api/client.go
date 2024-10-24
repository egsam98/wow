package api

import (
	"context"
	"encoding/binary"
	"iter"
	"math"
	"net"
	"time"

	"github.com/egsam98/errors"

	"github.com/egsam98/wow/internal/pow"
)

// Client connects to Words of Wisdom server
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
	return clientSync[*PhraseRequest, *PhraseResponse](c, ctx, new(PhraseRequest))
}

func (c *Client) AllPhrases(ctx context.Context) iter.Seq2[*PhraseResponse, error] {
	return clientStream[*AllPhrasesRequest, *PhraseResponse](c, ctx, new(AllPhrasesRequest))
}

func (c *Client) Close() error { return c.conn.Close() }

func clientSync[In, Out message](c *Client, ctx context.Context, req In) (Out, error) {
	var zero Out
	if err := write(c.conn, req); err != nil {
		return zero, errors.Wrap(err, "%T: write request", req)
	}
	for {
		msg, err := read(c.conn)
		if err != nil {
			return zero, errors.Wrap(err, "%T: read response", req)
		}
		switch msg := msg.(type) {
		case *powChallengeResponse:
			nonce, err := computePoW(ctx, msg.Challenge, msg.Zeros)
			if err != nil {
				return zero, err
			}
			if err := write(c.conn, &powNonceRequest{Nonce: nonce}); err != nil {
				return zero, errors.Wrap(err, "powNonceRequest: write request")
			}
		case Out:
			return msg, nil
		case *ErrorResponse:
			return zero, msg
		default:
			return zero, errors.Errorf("unexpected response message %#v", msg)
		}
	}
}

func clientStream[In, Out message](c *Client, ctx context.Context, req In) iter.Seq2[Out, error] {
	var zero Out
	if err := write(c.conn, req); err != nil {
		return func(yield func(Out, error) bool) { yield(zero, errors.Wrap(err, "%T: write request", req)) }
	}

	return func(yield func(Out, error) bool) {
		for {
			msg, err := read(c.conn)
			if err != nil {
				yield(zero, errors.Wrap(err, "%T: read response", req))
				return
			}
			switch msg := msg.(type) {
			case *powChallengeResponse:
				nonce, err := computePoW(ctx, msg.Challenge, msg.Zeros)
				if err != nil {
					yield(zero, err)
					return
				}
				if err := write(c.conn, &powNonceRequest{Nonce: nonce}); err != nil {
					yield(zero, errors.Wrap(err, "PowNonceRequest: write request"))
					return
				}
			case Out:
				if !yield(msg, nil) {
					return
				}
			case *streamTombstoneResponse:
				return
			case *ErrorResponse:
				yield(zero, msg)
				return
			default:
				yield(zero, errors.Errorf("unexpected response message %#v", msg))
				return
			}
		}
	}
}

// computePoW solves Proof of work on every call
func computePoW(ctx context.Context, challenge [pow.ChalLen]byte, zeros uint) ([8]byte, error) {
	var nonce [8]byte
	var err error
	for i := uint64(0); i <= math.MaxUint64; i++ {
		select {
		case <-ctx.Done():
			return nonce, ctx.Err()
		default:
		}

		binary.LittleEndian.PutUint64(nonce[:], i)
		if err = pow.Verify(challenge, zeros, nonce); !errors.Is(err, pow.ErrVerify) {
			break
		}
	}
	return nonce, err
}
