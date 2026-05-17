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
