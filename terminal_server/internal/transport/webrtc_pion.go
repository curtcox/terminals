package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

// PionWebRTCSignalEngine provides server-managed signaling and forwarding
// hooks for stream routes marked with metadata webrtc_mode=server_managed.
type PionWebRTCSignalEngine struct {
	mu      sync.Mutex
	api     *webrtc.API
	streams map[string]*pionStream
}

type pionStream struct {
	sessions   map[string]*pionSession
	publishers map[string][]*webrtc.TrackRemote
}

type pionSession struct {
	streamID    string
	deviceID    string
	pc          *webrtc.PeerConnection
	pending     []WebRTCSignalEngineResponse
	localTracks map[string]*webrtc.TrackLocalStaticRTP
}

// NewPionWebRTCSignalEngine creates a Pion-backed signaling engine.
func NewPionWebRTCSignalEngine() (*PionWebRTCSignalEngine, error) {
	me := &webrtc.MediaEngine{}
	if err := me.RegisterDefaultCodecs(); err != nil {
		return nil, fmt.Errorf("register default codecs: %w", err)
	}
	api := webrtc.NewAPI(webrtc.WithMediaEngine(me))
	return &PionWebRTCSignalEngine{
		api:     api,
		streams: map[string]*pionStream{},
	}, nil
}

// HandleSignal processes one incoming signal and returns server-generated
// outbound signals (typically answer/candidates) for target devices.
func (m *PionWebRTCSignalEngine) HandleSignal(_ context.Context, req WebRTCSignalEngineRequest) ([]WebRTCSignalEngineResponse, error) {
	streamID := strings.TrimSpace(req.StreamID)
	deviceID := strings.TrimSpace(req.DeviceID)
	signalType := strings.ToLower(strings.TrimSpace(req.Signal.SignalType))
	if streamID == "" || deviceID == "" || signalType == "" {
		return nil, fmt.Errorf("invalid webrtc signal request")
	}

	session, err := m.getOrCreateSession(streamID, deviceID)
	if err != nil {
		return nil, err
	}

	switch signalType {
	case "offer":
		sdp, err := parseSDPPayload(req.Signal.Payload)
		if err != nil {
			return nil, err
		}
		if err := session.pc.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: sdp}); err != nil {
			return nil, fmt.Errorf("set remote offer: %w", err)
		}
		answer, err := session.pc.CreateAnswer(nil)
		if err != nil {
			return nil, fmt.Errorf("create answer: %w", err)
		}
		if err := session.pc.SetLocalDescription(answer); err != nil {
			return nil, fmt.Errorf("set local answer: %w", err)
		}
		local := session.pc.LocalDescription()
		if local == nil || strings.TrimSpace(local.SDP) == "" {
			return nil, fmt.Errorf("local answer unavailable")
		}
		payload, err := encodeSDPPayload(local.SDP)
		if err != nil {
			return nil, err
		}
		m.enqueue(session, WebRTCSignalEngineResponse{
			TargetDeviceID: deviceID,
			Signal: WebRTCSignalResponse{
				StreamID:   streamID,
				SignalType: "answer",
				Payload:    payload,
			},
		})
	case "answer":
		sdp, err := parseSDPPayload(req.Signal.Payload)
		if err != nil {
			return nil, err
		}
		if err := session.pc.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeAnswer, SDP: sdp}); err != nil {
			return nil, fmt.Errorf("set remote answer: %w", err)
		}
	case "candidate":
		candidate, err := parseCandidatePayload(req.Signal.Payload)
		if err != nil {
			return nil, err
		}
		if err := session.pc.AddICECandidate(candidate); err != nil {
			return nil, fmt.Errorf("add ice candidate: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported webrtc signal type: %s", signalType)
	}

	return m.drainPending(session), nil
}

// RemoveStream closes all server-side peer connections for the stream.
func (m *PionWebRTCSignalEngine) RemoveStream(streamID string) {
	streamID = strings.TrimSpace(streamID)
	if streamID == "" {
		return
	}
	m.mu.Lock()
	stream := m.streams[streamID]
	delete(m.streams, streamID)
	m.mu.Unlock()
	if stream == nil {
		return
	}
	for _, session := range stream.sessions {
		_ = session.pc.Close()
	}
}

func (m *PionWebRTCSignalEngine) getOrCreateSession(streamID, deviceID string) (*pionSession, error) {
	m.mu.Lock()
	stream := m.streams[streamID]
	if stream == nil {
		stream = &pionStream{
			sessions:   map[string]*pionSession{},
			publishers: map[string][]*webrtc.TrackRemote{},
		}
		m.streams[streamID] = stream
	}
	if session := stream.sessions[deviceID]; session != nil {
		m.mu.Unlock()
		return session, nil
	}
	m.mu.Unlock()

	pc, err := m.api.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		return nil, fmt.Errorf("create peer connection: %w", err)
	}
	if _, err := pc.AddTransceiverFromKind(
		webrtc.RTPCodecTypeAudio,
		webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionSendrecv},
	); err != nil {
		_ = pc.Close()
		return nil, fmt.Errorf("add audio transceiver: %w", err)
	}
	if _, err := pc.AddTransceiverFromKind(
		webrtc.RTPCodecTypeVideo,
		webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionSendrecv},
	); err != nil {
		_ = pc.Close()
		return nil, fmt.Errorf("add video transceiver: %w", err)
	}

	session := &pionSession{
		streamID:    streamID,
		deviceID:    deviceID,
		pc:          pc,
		pending:     []WebRTCSignalEngineResponse{},
		localTracks: map[string]*webrtc.TrackLocalStaticRTP{},
	}

	pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}
		payloadBytes, err := json.Marshal(candidate.ToJSON())
		if err != nil {
			return
		}
		m.enqueue(session, WebRTCSignalEngineResponse{
			TargetDeviceID: deviceID,
			Signal: WebRTCSignalResponse{
				StreamID:   streamID,
				SignalType: "candidate",
				Payload:    string(payloadBytes),
			},
		})
	})

	pc.OnTrack(func(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		m.registerPublisherTrack(streamID, deviceID, track)
		go m.forwardTrack(streamID, deviceID, track)
	})

	m.mu.Lock()
	stream = m.streams[streamID]
	if stream == nil {
		stream = &pionStream{
			sessions:   map[string]*pionSession{},
			publishers: map[string][]*webrtc.TrackRemote{},
		}
		m.streams[streamID] = stream
	}
	stream.sessions[deviceID] = session
	m.mu.Unlock()

	m.attachExistingPublishers(streamID, deviceID)
	return session, nil
}

func (m *PionWebRTCSignalEngine) enqueue(session *pionSession, response WebRTCSignalEngineResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()
	session.pending = append(session.pending, response)
}

func (m *PionWebRTCSignalEngine) drainPending(session *pionSession) []WebRTCSignalEngineResponse {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(session.pending) == 0 {
		return nil
	}
	out := append([]WebRTCSignalEngineResponse(nil), session.pending...)
	session.pending = session.pending[:0]
	return out
}

func (m *PionWebRTCSignalEngine) registerPublisherTrack(streamID, publisherDeviceID string, track *webrtc.TrackRemote) {
	if track == nil {
		return
	}
	m.mu.Lock()
	stream := m.streams[streamID]
	if stream == nil {
		stream = &pionStream{
			sessions:   map[string]*pionSession{},
			publishers: map[string][]*webrtc.TrackRemote{},
		}
		m.streams[streamID] = stream
	}
	stream.publishers[publisherDeviceID] = append(stream.publishers[publisherDeviceID], track)
	m.mu.Unlock()

	m.attachPublisherToSubscribers(streamID, publisherDeviceID, track)
}

func (m *PionWebRTCSignalEngine) attachExistingPublishers(streamID, subscriberDeviceID string) {
	m.mu.Lock()
	stream := m.streams[streamID]
	m.mu.Unlock()
	if stream == nil {
		return
	}
	for publisherDeviceID, tracks := range stream.publishers {
		if publisherDeviceID == subscriberDeviceID {
			continue
		}
		for _, track := range tracks {
			m.ensureLocalForwardTrack(streamID, subscriberDeviceID, publisherDeviceID, track)
		}
	}
}

func (m *PionWebRTCSignalEngine) attachPublisherToSubscribers(streamID, publisherDeviceID string, track *webrtc.TrackRemote) {
	m.mu.Lock()
	stream := m.streams[streamID]
	m.mu.Unlock()
	if stream == nil {
		return
	}
	for subscriberDeviceID := range stream.sessions {
		if subscriberDeviceID == publisherDeviceID {
			continue
		}
		m.ensureLocalForwardTrack(streamID, subscriberDeviceID, publisherDeviceID, track)
	}
}

func (m *PionWebRTCSignalEngine) ensureLocalForwardTrack(
	streamID, subscriberDeviceID, publisherDeviceID string,
	track *webrtc.TrackRemote,
) {
	if track == nil {
		return
	}
	localKey := publisherDeviceID + "|" + track.ID()

	m.mu.Lock()
	stream := m.streams[streamID]
	if stream == nil {
		m.mu.Unlock()
		return
	}
	session := stream.sessions[subscriberDeviceID]
	if session == nil {
		m.mu.Unlock()
		return
	}
	if session.localTracks[localKey] != nil {
		m.mu.Unlock()
		return
	}
	m.mu.Unlock()

	local, err := webrtc.NewTrackLocalStaticRTP(track.Codec().RTPCodecCapability, "fwd-"+track.ID(), "stream-"+streamID)
	if err != nil {
		return
	}
	if _, err := session.pc.AddTrack(local); err != nil {
		return
	}

	m.mu.Lock()
	if currentStream := m.streams[streamID]; currentStream != nil {
		if currentSession := currentStream.sessions[subscriberDeviceID]; currentSession != nil {
			currentSession.localTracks[localKey] = local
		}
	}
	m.mu.Unlock()
}

func (m *PionWebRTCSignalEngine) forwardTrack(streamID, publisherDeviceID string, track *webrtc.TrackRemote) {
	for {
		packet, _, err := track.ReadRTP()
		if err != nil {
			return
		}
		m.forwardPacket(streamID, publisherDeviceID, track.ID(), packet)
	}
}

func (m *PionWebRTCSignalEngine) forwardPacket(streamID, publisherDeviceID, publisherTrackID string, packet *rtp.Packet) {
	if packet == nil {
		return
	}
	key := publisherDeviceID + "|" + publisherTrackID

	m.mu.Lock()
	stream := m.streams[streamID]
	if stream == nil {
		m.mu.Unlock()
		return
	}
	locals := make([]*webrtc.TrackLocalStaticRTP, 0, len(stream.sessions))
	for subscriberDeviceID, session := range stream.sessions {
		if subscriberDeviceID == publisherDeviceID {
			continue
		}
		if local := session.localTracks[key]; local != nil {
			locals = append(locals, local)
		}
	}
	m.mu.Unlock()

	for _, local := range locals {
		_ = local.WriteRTP(packet)
	}
}

type sdpPayload struct {
	SDP string `json:"sdp"`
}

func parseSDPPayload(payload string) (string, error) {
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return "", fmt.Errorf("empty sdp payload")
	}
	if strings.HasPrefix(payload, "{") {
		var parsed sdpPayload
		if err := json.Unmarshal([]byte(payload), &parsed); err != nil {
			return "", fmt.Errorf("decode sdp payload: %w", err)
		}
		sdp := strings.TrimSpace(parsed.SDP)
		if sdp == "" {
			return "", fmt.Errorf("sdp payload missing sdp")
		}
		return sdp, nil
	}
	return payload, nil
}

func encodeSDPPayload(sdp string) (string, error) {
	bytes, err := json.Marshal(sdpPayload{SDP: sdp})
	if err != nil {
		return "", fmt.Errorf("encode sdp payload: %w", err)
	}
	return string(bytes), nil
}

func parseCandidatePayload(payload string) (webrtc.ICECandidateInit, error) {
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return webrtc.ICECandidateInit{}, fmt.Errorf("empty candidate payload")
	}
	if strings.HasPrefix(payload, "{") {
		var parsed webrtc.ICECandidateInit
		if err := json.Unmarshal([]byte(payload), &parsed); err != nil {
			return webrtc.ICECandidateInit{}, fmt.Errorf("decode candidate payload: %w", err)
		}
		if strings.TrimSpace(parsed.Candidate) == "" {
			return webrtc.ICECandidateInit{}, fmt.Errorf("candidate payload missing candidate")
		}
		return parsed, nil
	}
	return webrtc.ICECandidateInit{Candidate: payload}, nil
}
