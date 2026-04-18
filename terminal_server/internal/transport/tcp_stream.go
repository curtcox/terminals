package transport

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"

	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	"google.golang.org/protobuf/proto"
)

// TCPEnvelopeStream adapts a length-framed TCP socket to WireEnvelope streaming.
type TCPEnvelopeStream struct {
	conn   net.Conn
	ctx    context.Context
	reader *bufio.Reader
	mu     sync.Mutex
}

// NewTCPEnvelopeStream creates a TCP envelope stream adapter.
func NewTCPEnvelopeStream(ctx context.Context, conn net.Conn) *TCPEnvelopeStream {
	if ctx == nil {
		ctx = context.Background()
	}
	return &TCPEnvelopeStream{
		conn:   conn,
		ctx:    ctx,
		reader: bufio.NewReader(conn),
	}
}

// ReadEnvelope reads one frame and decodes a WireEnvelope.
func (s *TCPEnvelopeStream) ReadEnvelope() (*controlv1.WireEnvelope, error) {
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(s.reader, lenBuf); err != nil {
		return nil, err
	}
	size := binary.BigEndian.Uint32(lenBuf)
	if size == 0 {
		return nil, fmt.Errorf("invalid tcp envelope size 0")
	}
	payload := make([]byte, size)
	if _, err := io.ReadFull(s.reader, payload); err != nil {
		return nil, err
	}
	envelope := &controlv1.WireEnvelope{}
	if err := proto.Unmarshal(payload, envelope); err != nil {
		return nil, fmt.Errorf("decode tcp envelope: %w", err)
	}
	return envelope, nil
}

// WriteEnvelope encodes and writes one length-framed WireEnvelope.
func (s *TCPEnvelopeStream) WriteEnvelope(envelope *controlv1.WireEnvelope) error {
	payload, err := proto.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("encode tcp envelope: %w", err)
	}
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(payload)))

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.conn.Write(lenBuf); err != nil {
		return err
	}
	if _, err := s.conn.Write(payload); err != nil {
		return err
	}
	return nil
}

// Context returns the stream context.
func (s *TCPEnvelopeStream) Context() context.Context {
	return s.ctx
}

// Carrier returns TCP carrier metadata for hello negotiation.
func (s *TCPEnvelopeStream) Carrier() controlv1.CarrierKind {
	return controlv1.CarrierKind_CARRIER_KIND_TCP
}
