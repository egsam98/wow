package api

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"net"

	"github.com/egsam98/errors"
	"github.com/rs/zerolog/log"

	"github.com/egsam98/wow/internal/pow"
)

const maxMessageLen = 1024 // 1KB

// write to connection
func write(conn net.Conn, msg message) error {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "marshal %+v", msg)
	}
	body, err := json.Marshal(operation{
		Code:    msg.opCode(),
		Message: msgBytes,
	})
	if err != nil {
		return errors.Wrap(err, "marshal packet")
	}
	if err := binary.Write(conn, binary.LittleEndian, uint32(len(body))); err != nil {
		return errors.Wrap(err, "write size")
	}
	if _, err := conn.Write(body); err != nil {
		return errors.Wrap(err, "write")
	}
	log.Debug().IPAddr("ip", ip(conn)).Msgf("Write %#v", msg)
	return nil
}

// read from connection
func read(conn net.Conn) (message, error) {
	var size uint32
	if err := binary.Read(conn, binary.LittleEndian, &size); err != nil {
		return nil, errors.Wrap(err, "read size")
	}
	if size > maxMessageLen {
		return nil, errors.New("too large message")
	}
	buf := make([]byte, size)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return nil, errors.Wrap(err, "read")
	}
	var op operation
	if err := json.Unmarshal(buf, &op); err != nil {
		return nil, errors.Wrap(err, "unmarshal %s into %T", buf, op)
	}

	var msg message
	switch op.Code {
	case powNonceReq:
		msg = new(powNonceRequest)
	case powChallengeResp:
		msg = new(powChallengeResponse)
	case errorResp:
		msg = new(ErrorResponse)
	case streamTombstoneResp:
		msg = new(streamTombstoneResponse)
	case phraseReq:
		msg = new(PhraseRequest)
	case phraseResp:
		msg = new(PhraseResponse)
	case allPhrasesReq:
		msg = new(AllPhrasesRequest)
	default:
		return nil, errors.Errorf("unexpected command: %s", op.Code)
	}
	if err := json.Unmarshal(op.Message, msg); err != nil {
		return nil, errors.Wrap(err, "unmarshal packet message %s into %T", op.Message, msg)
	}
	log.Debug().IPAddr("ip", ip(conn)).Msgf("Read %#v", msg)
	return msg, nil
}

// opCode is a code of `operation` corresponding to every `message`
type opCode string

const (
	powNonceReq         opCode = "pow_nonce_req"
	powChallengeResp    opCode = "pow_challenge_resp"
	streamTombstoneResp opCode = "stream_tombstone_resp"
	errorResp           opCode = "error_resp"
	phraseReq           opCode = "phrase_req"
	phraseResp          opCode = "phrase_resp"
	allPhrasesReq       opCode = "all_phrases_req"
)

// operation is primary DTO that is transferred in TCP connection
type operation struct {
	Code    opCode          `json:"code"`
	Message json.RawMessage `json:"message"`
}

type message interface {
	opCode() opCode
}

type powNonceRequest struct {
	Nonce [8]byte `json:"nonce"`
}

func (*powNonceRequest) opCode() opCode { return powNonceReq }

type powChallengeResponse struct {
	Challenge [pow.ChalLen]byte `json:"challenge"`
	Zeros     uint              `json:"zeros"`
}

func (*powChallengeResponse) opCode() opCode { return powChallengeResp }

type streamTombstoneResponse struct{}

func (*streamTombstoneResponse) opCode() opCode { return streamTombstoneResp }

type ErrorResponse struct {
	Message string `json:"message"`
}

func (e *ErrorResponse) Error() string { return e.Message }
func (*ErrorResponse) opCode() opCode  { return errorResp }

type PhraseRequest struct{}

func (*PhraseRequest) opCode() opCode { return phraseReq }

type PhraseResponse struct {
	Quote  string `json:"quote"`
	Author string `json:"author"`
}

func (*PhraseResponse) opCode() opCode { return phraseResp }

type AllPhrasesRequest struct{}

func (*AllPhrasesRequest) opCode() opCode { return allPhrasesReq }
