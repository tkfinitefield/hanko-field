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
    const sidebarRoot = () => document.getElementById("mobile-sidebar");

    const setSidebarState = (open) => {
      const sidebar = sidebarRoot();
      if (!sidebar) {
        return;
      }
      if (open) {
        sidebar.classList.remove("hidden");
        sidebar.classList.add("flex");
        sidebar.setAttribute("aria-hidden", "false");
        document.body.classList.add("overflow-hidden");
      } else {
        sidebar.classList.add("hidden");
        sidebar.classList.remove("flex");
        sidebar.setAttribute("aria-hidden", "true");
        document.body.classList.remove("overflow-hidden");
      }
      document.querySelectorAll("[data-sidebar-toggle]").forEach((el) => {
        el.setAttribute("aria-expanded", open ? "true" : "false");
      });
    };

    const closeSidebar = () => setSidebarState(false);
    const openSidebar = () => setSidebarState(true);

    document.addEventListener("click", (event) => {
      const trigger = event.target instanceof Element ? event.target.closest("[data-modal-close]") : null;
      if (!trigger) {
        const sidebarToggle = event.target instanceof Element ? event.target.closest("[data-sidebar-toggle]") : null;
        if (sidebarToggle) {
          event.preventDefault();
          openSidebar();
          return;
        }
        const sidebarDismiss = event.target instanceof Element ? event.target.closest("[data-sidebar-dismiss]") : null;
        if (sidebarDismiss) {
          event.preventDefault();
          closeSidebar();
        }
        return;
      }
      event.preventDefault();
      const root = modalRoot();
      if (root) {
        root.innerHTML = "";
      }
    });

    document.addEventListener("keydown", (event) => {
      if (event.key === "Escape") {
        const root = modalRoot();
        if (root && root.firstChild) {
          root.innerHTML = "";
        }
        if (!event.defaultPrevented) {
          closeSidebar();
        }
      }
    });
  },
};

window.addEventListener("DOMContentLoaded", () => {
  window.hankoAdmin.init();
});
