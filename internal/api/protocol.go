package api

import (
	"bufio"
	"encoding/json"
	"io"
	"net"

	"github.com/pkg/errors"

	"github.com/egsam98/wow/internal/pow"
)

const maxMessageLen = 1024 // 1KB
const terminal = '\n'

func write(conn net.Conn, msg message) error {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return errors.Wrapf(err, "marshal %+v", msg)
	}
	body, err := json.Marshal(operation{
		Code:    msg.opCode(),
		Message: msgBytes,
	})
	if err != nil {
		return errors.Wrap(err, "marshal packet")
	}
	_, err = conn.Write(append(body, terminal))
	return errors.Wrap(err, "write")
}

func read(conn net.Conn) (message, error) {
	reader := bufio.NewReader(io.LimitReader(conn, maxMessageLen))
	buf, err := reader.ReadBytes(terminal)
	if err != nil {
		return nil, errors.Wrap(err, "read")
	}
	var op operation
	if err := json.Unmarshal(buf, &op); err != nil {
		return nil, errors.Wrapf(err, "unmarshal %s into %T", buf, op)
	}

	var msg message
	switch op.Code {
	case phraseReq:
		msg = new(PhraseRequest)
	case phraseResp:
		msg = new(PhraseResponse)
	case powNonceReq:
		msg = new(PowNonceRequest)
	case powChallengeResp:
		msg = new(PowChallengeResponse)
	case errorResp:
		msg = new(ErrorResponse)
	default:
		return nil, errors.Errorf("unexpected command: %s", op.Code)
	}
	if err := json.Unmarshal(op.Message, msg); err != nil {
		return nil, errors.Wrapf(err, "unmarshal packet message %s into %T", op.Message, msg)
	}
	return msg, nil
}

type opCode string

const (
	phraseReq        opCode = "phrase_req"
	phraseResp       opCode = "phrase_resp"
	powNonceReq      opCode = "pow_nonce_req"
	powChallengeResp opCode = "pow_challenge_resp"
	errorResp        opCode = "error_resp"
)

type operation struct {
	Code    opCode          `json:"code"`
	Message json.RawMessage `json:"message"`
}

type message interface {
	opCode() opCode
}

type PowNonceRequest struct {
	Challenge [pow.ChalLen]byte `json:"challenge"`
	Nonce     [8]byte           `json:"nonce"`
}

func (*PowNonceRequest) opCode() opCode { return powNonceReq }

type PowChallengeResponse struct {
	Challenge [pow.ChalLen]byte `json:"challenge"`
	Zeroes    uint              `json:"zeroes"`
}

func (*PowChallengeResponse) opCode() opCode { return powChallengeResp }

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
