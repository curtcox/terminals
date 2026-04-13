package transport

import (
	"context"
	"io"
)

type asyncFakeProtoStream struct {
	ctx    context.Context
	recvCh chan ProtoClientEnvelope
	sentCh chan ProtoServerEnvelope
}

func (a *asyncFakeProtoStream) RecvProto() (ProtoClientEnvelope, error) {
	env, ok := <-a.recvCh
	if !ok {
		return nil, io.EOF
	}
	return env, nil
}

func (a *asyncFakeProtoStream) SendProto(env ProtoServerEnvelope) error {
	a.sentCh <- env
	return nil
}

func (a *asyncFakeProtoStream) Context() context.Context {
	return a.ctx
}
