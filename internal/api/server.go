package api

import (
	"context"
	"fmt"
	"io"
	"iter"
	"net"
	"os"
	"sync/atomic"
	"time"

	"github.com/egsam98/errors"
	"github.com/rs/zerolog/log"

	"github.com/egsam98/wow/internal/pow"
)

// Server serves TCP connection
type Server struct {
	addr        string
	tcpDeadline time.Duration
	handler     ServerHandler
	puzzle      *pow.Puzzle
	conns       atomic.Int32
}

type ServerHandler interface {
	Phrase(context.Context, *PhraseRequest) (*PhraseResponse, error)
	AllPhrases(context.Context, *AllPhrasesRequest) iter.Seq2[*PhraseResponse, error]
}

func NewServer(addr string, tcpDeadline time.Duration, handler ServerHandler, puzzle *pow.Puzzle) Server {
	return Server{
		handler:     handler,
		addr:        addr,
		tcpDeadline: tcpDeadline,
		puzzle:      puzzle,
	}
}

// Listen accepts incoming TCP connections handling them in `Server.handle` method.
// The method blocks until the context is canceled
func (s *Server) Listen(ctx context.Context) error {
	lis, err := new(net.ListenConfig).Listen(ctx, "tcp", s.addr)
	if err != nil {
		return errors.Wrap(err, "listen server")
	}

	go func() {
		<-ctx.Done()
		if err := lis.Close(); err != nil {
			log.Err(err).Msg("Close listener")
		}
	}()

	for {
		conn, err := lis.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			log.Err(err).Msg("Accept TCP connection")
			continue
		}

		go s.handle(ctx, conn)
	}
}

func (s *Server) Close() {
	for s.conns.Load() > 0 {
		continue
	}
}

// handle connection in separate loop
func (s *Server) handle(ctx context.Context, conn net.Conn) {
	s.conns.Add(1)
	defer s.conns.Add(-1)
	defer conn.Close()

	handle := func() error {
		_ = conn.SetDeadline(time.Now().Add(s.tcpDeadline))

		msg, err := read(conn)
		if err != nil {
			return err
		}

		if err := s.requestPoW(conn); err != nil {
			return err
		}

		var res message
		switch msg := msg.(type) {
		case *PhraseRequest:
			res, err = s.handler.Phrase(ctx, msg)
			return respond(conn, res, err)
		case *AllPhrasesRequest:
			it := s.handler.AllPhrases(ctx, msg)
			return respondStream(conn, it)
		default:
			return write(conn, &ErrorResponse{Message: fmt.Sprintf("unexpected message %v (%T)", msg, msg)})
		}
	}

	for {
		switch err := handle(); {
		case err == nil:
			continue
		case errors.Is(err, io.EOF):
		case errors.Is(err, os.ErrDeadlineExceeded):
			log.Debug().Err(err).IPAddr("from", ip(conn)).Msg("Deadline timeout")
		default:
			if err := write(conn, &ErrorResponse{Message: "internal error"}); err != nil {
				log.Err(err).IPAddr("to", ip(conn)).Msg("Write")
			}
			log.Err(err).IPAddr("from", ip(conn)).Msg("Handle")
		}
		return
	}
}

// requestPoW requests Proof of Work from connection before granting access to resource
func (s *Server) requestPoW(conn net.Conn) error {
	challenge, zeros, err := s.puzzle.Challenge(uint(s.conns.Load()))
	if err != nil {
		return err
	}
	if err := write(conn, &powChallengeResponse{Challenge: challenge, Zeros: zeros}); err != nil {
		return err
	}

	msg, err := read(conn)
	if err != nil {
		return err
	}
	req, ok := msg.(*powNonceRequest)
	if !ok {
		return write(conn, &ErrorResponse{Message: "powNonceRequest is expected"})
	}

	if err := pow.Verify(challenge, zeros, req.Nonce); err != nil {
		return write(conn, &ErrorResponse{Message: err.Error()})
	}
	return nil
}

func respond[T message](conn net.Conn, res T, err error) error {
	if err != nil {
		return write(conn, &ErrorResponse{Message: err.Error()})
	}
	return write(conn, res)
}

func respondStream[T message](conn net.Conn, it iter.Seq2[T, error]) error {
	for res, err := range it {
		if err != nil {
			return write(conn, &ErrorResponse{Message: err.Error()})
		}
		if err := write(conn, res); err != nil {
			return err
		}
	}
	return write(conn, new(streamTombstoneResponse))
}

// ip extracts IP address from net.Conn
// Panics if address is not *net.TCPAddr
func ip(conn net.Conn) net.IP {
	return conn.RemoteAddr().(*net.TCPAddr).IP
}
