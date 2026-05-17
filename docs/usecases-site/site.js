document.querySelectorAll("table").forEach((table) => {
  const tbody = table.querySelector("tbody");
  table.querySelectorAll("th button").forEach((button, column) => {
    button.addEventListener("click", () => {
      const rows = Array.from(tbody.querySelectorAll("tr"));
      const direction = button.dataset.direction === "asc" ? -1 : 1;
      table.querySelectorAll("th button").forEach((other) => {
        other.dataset.direction = "";
      });
      button.dataset.direction = direction === 1 ? "asc" : "desc";
      rows.sort((a, b) => {
        const left = a.children[column].innerText.trim();
        const right = b.children[column].innerText.trim();
        return left.localeCompare(right, undefined, { numeric: true }) * direction;
      });
      rows.forEach((row) => tbody.appendChild(row));
    });
  });
});

document.querySelectorAll(".frame-scrubber").forEach((scrubber) => {
  const frames = JSON.parse(scrubber.dataset.frames || "[]");
  const image = scrubber.querySelector(".frame-preview-image");
  const label = scrubber.querySelector(".frame-preview-label");
  const range = scrubber.querySelector(".frame-range");
  if (!frames.length || !image || !label || !range) return;
  range.addEventListener("input", () => {
    const frame = frames[Number(range.value)];
    if (!frame) return;
    image.src = frame.src;
    image.alt = frame.label;
    label.textContent = frame.label;
  });
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
