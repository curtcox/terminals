package transport

import (
	"fmt"
	"strings"

	"github.com/pion/webrtc/v4"
)

func (m *PionWebRTCSignalEngine) handleSignalType(session *pionSession, streamID, deviceID, signalType, payload string) error {
	switch signalType {
	case "offer":
		return m.handleOfferSignal(session, streamID, deviceID, payload)
	case "answer":
		return m.handleAnswerSignal(session, payload)
	case "candidate":
		return m.handleCandidateSignal(session, payload)
	default:
		return fmt.Errorf("unsupported webrtc signal type: %s", signalType)
	}
}

func (m *PionWebRTCSignalEngine) handleOfferSignal(session *pionSession, streamID, deviceID, payload string) error {
	sdp, err := parseSDPPayload(payload)
	if err != nil {
		return err
	}
	if err := session.pc.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: sdp}); err != nil {
		return fmt.Errorf("set remote offer: %w", err)
	}
	answer, err := session.pc.CreateAnswer(nil)
	if err != nil {
		return fmt.Errorf("create answer: %w", err)
	}
	if err := session.pc.SetLocalDescription(answer); err != nil {
		return fmt.Errorf("set local answer: %w", err)
	}
	local := session.pc.LocalDescription()
	if local == nil || strings.TrimSpace(local.SDP) == "" {
		return fmt.Errorf("local answer unavailable")
	}
	encoded, err := encodeSDPPayload(local.SDP)
	if err != nil {
		return err
	}
	m.enqueue(session, WebRTCSignalEngineResponse{
		TargetDeviceID: deviceID,
		Signal: WebRTCSignalResponse{
			StreamID:   streamID,
			SignalType: "answer",
			Payload:    encoded,
		},
	})
	return nil
}

func (m *PionWebRTCSignalEngine) handleAnswerSignal(session *pionSession, payload string) error {
	sdp, err := parseSDPPayload(payload)
	if err != nil {
		return err
	}
	if err := session.pc.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeAnswer, SDP: sdp}); err != nil {
		return fmt.Errorf("set remote answer: %w", err)
	}
	return nil
}

func (m *PionWebRTCSignalEngine) handleCandidateSignal(session *pionSession, payload string) error {
	candidate, err := parseCandidatePayload(payload)
	if err != nil {
		return err
	}
	if err := session.pc.AddICECandidate(candidate); err != nil {
		return fmt.Errorf("add ice candidate: %w", err)
	}
	return nil
}
