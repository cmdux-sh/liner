/** One truthful clipboard interaction for landing, docs, and changelog commands. */
export function bindCommandCopy() {
  document.querySelectorAll<HTMLButtonElement>("[data-copy-button]").forEach((button) => {
    if (button.dataset.bound) return;
    button.dataset.bound = "true";
    const label = button.querySelector<HTMLElement>("[data-copy-icon]");
    const done = button.querySelector<HTMLElement>("[data-copy-done]");
    const status = button.parentElement?.querySelector<HTMLElement>("[role=status]");
    const originalLabel = label?.innerHTML ?? "Copy";
    const originalName = button.getAttribute("aria-label") ?? "Copy command";
    let timer: ReturnType<typeof setTimeout>;
    button.addEventListener("click", async () => {
      clearTimeout(timer);
      if (status) status.textContent = "";
      button.disabled = true;
      button.dataset.copyState = "pending";
      button.setAttribute("aria-label", "Copying command");
      if (done) done.style.display = "none";
      if (label) { label.style.display = "flex"; label.textContent = "Copying"; }
      try {
        if (!navigator.clipboard?.writeText) throw new Error("Clipboard unavailable");
        await navigator.clipboard.writeText(button.dataset.command ?? "");
        button.dataset.copyState = "success";
        button.setAttribute("aria-label", "Copied. Copy command again");
        if (label) label.style.display = "none";
        if (done) done.style.display = "flex";
        if (status) status.textContent = "Command copied.";
        button.dispatchEvent(new CustomEvent("command-copied", { bubbles: true }));
        timer = setTimeout(() => {
          if (label) { label.innerHTML = originalLabel; label.style.display = "flex"; }
          if (done) done.style.display = "none";
          delete button.dataset.copyState;
          button.setAttribute("aria-label", originalName);
        }, 2000);
      } catch {
        button.dataset.copyState = "error";
        button.setAttribute("aria-label", "Retry copying command");
        if (label) label.textContent = "Retry";
        if (status) status.textContent = "Copy failed. Select the command and copy it manually.";
      } finally {
        button.disabled = false;
      }
    });
  });
}
