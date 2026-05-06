export class MediaSurfaceRegistry {
  constructor() {
    this.surfaces = new Map();
  }

  createSurface(kind, componentId, mediaId) {
    const element = document.createElement(kind === "video" ? "video" : "div");
    element.className = `sd sd-${kind}-surface`;
    element.setAttribute("data-component-id", componentId);
    element.setAttribute("data-media-id", mediaId);
    if (kind === "video") {
      element.autoplay = true;
      element.playsInline = true;
      element.muted = true;
    }
    this.surfaces.set(componentId, element);
    return element;
  }

  attachStream(componentId, stream) {
    const element = this.surfaces.get(componentId);
    if (element && "srcObject" in element) element.srcObject = stream;
  }
}
