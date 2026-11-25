"use strict";

// copy button of the personal page; navigator.clipboard needs HTTPS, so there is a fallback
document.addEventListener("DOMContentLoaded", () => {
  document.querySelectorAll("[data-copy]").forEach((btn) => {
    btn.addEventListener("click", async () => {
      const text = btn.dataset.copy;
      const label = btn.textContent;
      try {
        if (navigator.clipboard && window.isSecureContext) {
          await navigator.clipboard.writeText(text);
        } else {
          const area = document.createElement("textarea");
          area.value = text;
          document.body.append(area);
          area.select();
          document.execCommand("copy");
          area.remove();
        }
        btn.textContent = btn.dataset.done;
        setTimeout(() => { btn.textContent = label; }, 1500);
      } catch {
        /* the link is shown on the page anyway */
      }
    });
  });
});
