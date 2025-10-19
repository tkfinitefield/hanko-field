"use strict";

// Minimal bootstrap for the admin shell.
document.documentElement.classList.add("js-enabled");

if (window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
  document.documentElement.classList.add("reduced-motion");
}

// Expose a hook for future htmx/alpine wiring without blocking initial scaffold.
window.hankoAdmin = window.hankoAdmin || {
  init() {
    // no-op placeholder; real hooks populate here.
  },
};

window.addEventListener("DOMContentLoaded", () => {
  window.hankoAdmin.init();
});
