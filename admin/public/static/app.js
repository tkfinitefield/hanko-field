"use strict";

// Minimal bootstrap for the admin shell.
document.documentElement.classList.add("js-enabled");

if (window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
  document.documentElement.classList.add("reduced-motion");
}

// Expose a hook for future htmx/alpine wiring without blocking initial scaffold.
window.hankoAdmin = window.hankoAdmin || {
  init() {
    const modalRoot = () => document.getElementById("modal-root");

    document.addEventListener("click", (event) => {
      const trigger = event.target instanceof Element ? event.target.closest("[data-modal-close]") : null;
      if (!trigger) {
        return;
      }
      event.preventDefault();
      const root = modalRoot();
      if (root) {
        root.innerHTML = "";
      }
    });

    document.addEventListener("keydown", (event) => {
      if (event.key !== "Escape") {
        return;
      }
      const root = modalRoot();
      if (root && root.firstChild) {
        root.innerHTML = "";
      }
    });
  },
};

window.addEventListener("DOMContentLoaded", () => {
  window.hankoAdmin.init();
});
