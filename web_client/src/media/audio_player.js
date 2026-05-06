export class AudioPlayer {
  playUrl(url) {
    const audio = new Audio(url);
    return audio.play();
  }
}
