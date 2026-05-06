export class FakeSocket {
  static OPEN = 1;

  constructor(url) {
    this.url = url;
    this.readyState = FakeSocket.OPEN;
    this.sent = [];
    FakeSocket.instances.push(this);
  }

  send(data) {
    this.sent.push(data);
  }

  close() {
    this.readyState = 3;
    this.onclose?.({ code: 1000, reason: "", wasClean: true });
  }

  receive(data) {
    this.onmessage?.({ data });
  }

  open() {
    this.onopen?.();
  }
}

FakeSocket.instances = [];
