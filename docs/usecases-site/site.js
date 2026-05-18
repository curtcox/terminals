document.querySelectorAll("table").forEach((table) => {
  const tbody = table.querySelector("tbody");
  table.querySelectorAll("th button").forEach((button) => {
    button.addEventListener("click", () => {
      const rows = Array.from(tbody.querySelectorAll("tr"));
      const direction = button.dataset.direction === "asc" ? -1 : 1;
      table.querySelectorAll("th button").forEach((other) => {
        other.dataset.direction = "";
        other.closest("th").setAttribute("aria-sort", "none");
      });
      button.dataset.direction = direction === 1 ? "asc" : "desc";
      button.closest("th").setAttribute("aria-sort", direction === 1 ? "ascending" : "descending");
      const sortKey = button.dataset.sortKey;
      rows.sort((a, b) => {
        const left = a.dataset[`sort${sortKey[0].toUpperCase()}${sortKey.slice(1)}`] || "";
        const right = b.dataset[`sort${sortKey[0].toUpperCase()}${sortKey.slice(1)}`] || "";
        return left.localeCompare(right, undefined, { numeric: true }) * direction;
      });
      rows.forEach((row) => tbody.appendChild(row));
    });
  });
});

const textFilter = document.getElementById("usecase-filter");
const statusFilter = document.getElementById("status-filter");
if (textFilter && statusFilter) {
  const rows = Array.from(document.querySelectorAll("tbody tr"));
  const applyFilters = () => {
    const query = textFilter.value.trim().toLowerCase();
    const status = statusFilter.value;
    rows.forEach((row) => {
      const matchesText = !query || row.dataset.filter.includes(query);
      const matchesStatus = !status || row.dataset.status === status;
      row.hidden = !(matchesText && matchesStatus);
    });
    document.querySelectorAll(".family").forEach((section) => {
      section.hidden = !section.querySelector("tbody tr:not([hidden])");
    });
  };
  textFilter.addEventListener("input", applyFilters);
  statusFilter.addEventListener("change", applyFilters);
}

// Render a ui.Descriptor JSON tree into a container element,
// reusing the same CSS classes as the web client renderer.
function renderDescriptorNode(container, node) {
  if (!node || !node.type) return;
  const props = node.props || {};
  let el;
  switch (node.type) {
    case "text":
      el = document.createElement("p");
      el.className = "sd sd-text";
      el.textContent = props.value || "";
      if (props.style) el.dataset.style = props.style;
      if (props.color) el.style.color = props.color;
      break;
    case "button":
      el = document.createElement("div");
      el.className = "sd sd-button";
      el.textContent = props.label || "";
      break;
    case "image":
      el = document.createElement("div");
      el.className = "sd sd-image";
      el.textContent = props.url ? "[image: " + props.url.slice(0, 40) + "]" : "[image]";
      break;
    case "video_surface":
      el = document.createElement("div");
      el.className = "sd sd-video-surface";
      el.textContent = "[video: " + (props.track_id || "") + "]";
      break;
    case "audio_visualizer":
      el = document.createElement("div");
      el.className = "sd sd-audio-visualizer";
      el.textContent = "[audio: " + (props.stream_id || "") + "]";
      break;
    case "fullscreen":
    case "keep_awake":
    case "brightness":
      el = document.createElement("div");
      el.className = "sd sd-" + node.type.replace(/_/g, "-");
      break;
    default:
      el = document.createElement("div");
      el.className = "sd sd-" + node.type.replace(/_/g, "-");
  }
  if (props.background) el.style.background = props.background;
  if (node.type === "grid" && props.columns) {
    el.style.gridTemplateColumns = "repeat(" + props.columns + ", minmax(0, 1fr))";
  }
  for (const child of (node.children || [])) {
    renderDescriptorNode(el, child);
  }
  container.appendChild(el);
}

document.querySelectorAll(".frame-scrubber").forEach((scrubber) => {
  const frames = JSON.parse(scrubber.dataset.frames || "[]");
  const preview = scrubber.querySelector(".frame-live-preview");
  const label = scrubber.querySelector(".frame-preview-label");
  const range = scrubber.querySelector(".frame-range");
  if (!frames.length || !label || !range) return;

  function showFrame(frame) {
    if (!frame) return;
    label.textContent = frame.label;
    if (preview) {
      preview.replaceChildren();
      preview.setAttribute("aria-label", frame.label);
      if (frame.descriptor) {
        renderDescriptorNode(preview, frame.descriptor);
      }
    }
  }

  showFrame(frames[0]);
  range.addEventListener("input", () => showFrame(frames[Number(range.value)]));
});

(function () {
  const links = document.querySelectorAll(".frame-link");
  if (!links.length) return;
  const dialog = document.createElement("dialog");
  dialog.id = "frame-lightbox";
  dialog.innerHTML =
    '<figure><button class="lightbox-close" aria-label="Close">×</button>' +
    '<img id="lightbox-img" src="" alt=""><figcaption id="lightbox-caption"></figcaption></figure>';
  document.body.appendChild(dialog);
  dialog.querySelector(".lightbox-close").addEventListener("click", () => dialog.close());
  dialog.addEventListener("click", (e) => { if (e.target === dialog) dialog.close(); });
  links.forEach((link) => {
    link.addEventListener("click", (e) => {
      e.preventDefault();
      document.getElementById("lightbox-img").src = link.href;
      document.getElementById("lightbox-img").alt = link.querySelector("span").textContent;
      document.getElementById("lightbox-caption").textContent = link.querySelector("span").textContent;
      dialog.showModal();
    });
  });
}());
