export class WebRTCEngine {
  constructor({ peerConnectionFactory = () => new RTCPeerConnection(), mediaSurfaceRegistry } = {}) {
    this.peerConnectionFactory = peerConnectionFactory;
    this.mediaSurfaceRegistry = mediaSurfaceRegistry;
    this.peer = null;
  }

  ensurePeer() {
    if (!this.peer) this.peer = this.peerConnectionFactory();
    return this.peer;
  }

  async handleSignal(signal) {
    const peer = this.ensurePeer();
    if (signal.type === "offer") {
      await peer.setRemoteDescription(signal);
      return peer.createAnswer();
    }
    if (signal.type === "answer") return peer.setRemoteDescription(signal);
    if (signal.candidate) return peer.addIceCandidate(signal);
    return null;
  }
}
