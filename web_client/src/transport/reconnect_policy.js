export class ReconnectPolicy {
  constructor({ baseMs = 250, maxMs = 5000 } = {}) {
    this.baseMs = baseMs;
    this.maxMs = maxMs;
  }

  delayForAttempt(attempt) {
    return Math.min(this.maxMs, this.baseMs * 2 ** Math.max(0, attempt - 1));
  }
}
