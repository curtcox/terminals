package transport

import (
	"context"
	"fmt"

	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
)

type generatedControlService struct {
	controlv1.UnimplementedTerminalControlServiceServer
	server *Server
}

func newGeneratedControlService(server *Server) *generatedControlService {
	return &generatedControlService{server: server}
}

func (s *generatedControlService) Connect(stream controlv1.TerminalControlService_ConnectServer) error {
	return s.server.Connect(generatedProtoStream{stream: stream})
}

type generatedProtoStream struct {
	stream controlv1.TerminalControlService_ConnectServer
}

func (s generatedProtoStream) RecvProto() (ProtoClientEnvelope, error) {
	return s.stream.Recv()
}

func (s generatedProtoStream) SendProto(envelope ProtoServerEnvelope) error {
	response, ok := envelope.(*controlv1.ConnectResponse)
	if !ok {
		return fmt.Errorf("unexpected proto server envelope %T", envelope)
	}
	return s.stream.Send(response)
}

func (s generatedProtoStream) Context() context.Context {
	return s.stream.Context()
}
