"use strict";

// Minimal bootstrap for the admin shell.
document.documentElement.classList.add("js-enabled");

if (window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
  document.documentElement.classList.add("reduced-motion");
}

const KEY_ESCAPE = "Escape";
const KEY_TAB = "Tab";
const focusableSelectors = [
  "a[href]",
  "area[href]",
  "button:not([disabled])",
  "input:not([disabled]):not([type='hidden'])",
  "select:not([disabled])",
  "textarea:not([disabled])",
  "[tabindex]:not([tabindex='-1'])",
].join(",");

const isEditableTarget = (element) => {
  if (!(element instanceof Element)) {
    return false;
  }
  const tag = element.tagName ? element.tagName.toLowerCase() : "";
  if (tag === "input" || tag === "textarea" || tag === "select") {
    return true;
  }
  if (element.isContentEditable) {
    return true;
  }
  return Boolean(element.closest("input, textarea, select, [contenteditable='true'], [contenteditable='']"));
};

const formatNotificationCount = (value) => {
  const raw = typeof value === "string" ? value.trim() : String(value ?? "").trim();
  if (raw === "") {
    return { display: "0", empty: true };
  }
  if (raw.includes("+")) {
    return { display: raw, empty: false };
  }
  const parsed = Number.parseInt(raw, 10);
  if (!Number.isFinite(parsed) || parsed <= 0) {
    return { display: "0", empty: true };
  }
  if (parsed > 99) {
    return { display: "99+", empty: false };
  }
  return { display: String(parsed), empty: false };
};

const applyNotificationBadgeState = (root) => {
  if (!(root instanceof Element)) {
    return;
  }
  const badge = root.querySelector("[data-notification-count]");
  if (!badge) {
    return;
  }
  const payload = formatNotificationCount(badge.textContent);
  badge.textContent = payload.display;
  const emptyAttr = badge.getAttribute("data-empty");
  const isEmpty = emptyAttr === "true" || payload.empty;
  badge.dataset.empty = isEmpty ? "true" : "false";
  badge.classList.toggle("opacity-0", isEmpty);
  badge.classList.toggle("opacity-100", !isEmpty);
  badge.classList.toggle("pointer-events-none", isEmpty);
  badge.classList.toggle("pointer-events-auto", !isEmpty);
};

const initNotificationsBadge = () => {
  const initialize = (root) => {
    if (!(root instanceof Element)) {
      return;
    }
    applyNotificationBadgeState(root);
  };

  document.querySelectorAll("[data-notifications-root]").forEach(initialize);

  if (window.htmx) {
    document.body.addEventListener("htmx:afterSwap", (event) => {
      if (!(event.target instanceof Element)) {
        return;
      }
      if (event.target.matches("[data-notifications-root]")) {
        initialize(event.target);
        return;
      }
      event.target.querySelectorAll?.("[data-notifications-root]").forEach(initialize);
    });
  }
};

const initSearchShortcut = (getModalRoot) => {
  const trigger = document.querySelector("[data-topbar-search-trigger]");
  if (!trigger) {
    return;
  }

  const openSearch = () => {
    trigger.focus({ preventScroll: true });
    trigger.click();
  };

  trigger.addEventListener("click", (event) => {
    if (window.htmx) {
      return;
    }
    const fallbackHref = trigger.getAttribute("data-search-href");
    if (fallbackHref) {
      event.preventDefault();
      window.location.href = fallbackHref;
    }
  });

  document.addEventListener("keydown", (event) => {
    if (event.key !== "/" || event.altKey || event.ctrlKey || event.metaKey) {
      return;
    }
    if (event.target instanceof Element && isEditableTarget(event.target)) {
      return;
    }
    event.preventDefault();
    openSearch();
  });

  if (window.htmx) {
    document.body.addEventListener("htmx:afterSwap", (event) => {
      if (!(event.target instanceof Element)) {
        return;
      }
      const modalRoot = getModalRoot();
      if (!modalRoot || event.target !== modalRoot) {
        return;
      }
      const field =
        modalRoot.querySelector("[data-search-input]") ||
        modalRoot.querySelector("input[type='search']") ||
        modalRoot.querySelector("input[name='q']");
      if (field instanceof HTMLElement) {
        queueMicrotask(() => {
          field.focus({ preventScroll: true });
          if (field instanceof HTMLInputElement || field instanceof HTMLTextAreaElement) {
            field.select();
          }
        });
      }
    });
  }
};

const initUserMenu = () => {
  const root = document.querySelector("[data-user-menu]");
  if (!root) {
    return;
  }
  const trigger = root.querySelector("[data-user-menu-trigger]");
  const panel = root.querySelector("[data-user-menu-panel]");
  if (!(trigger instanceof HTMLElement) || !(panel instanceof HTMLElement)) {
    return;
  }

  let open = false;

  const focusable = () =>
    Array.from(panel.querySelectorAll(focusableSelectors)).filter(
      (el) => el instanceof HTMLElement && !el.hasAttribute("disabled"),
    );

  const setOpen = (state) => {
    if (open === state) {
      return;
    }
    open = state;
    trigger.setAttribute("aria-expanded", state ? "true" : "false");
    panel.setAttribute("aria-hidden", state ? "false" : "true");
    panel.classList.toggle("opacity-0", !state);
    panel.classList.toggle("pointer-events-none", !state);
    panel.classList.toggle("opacity-100", state);
    panel.classList.toggle("pointer-events-auto", state);

    if (state) {
      const items = focusable();
      (items[0] || panel).focus({ preventScroll: true });
    } else {
      trigger.focus({ preventScroll: true });
    }
  };

  const closeMenu = () => setOpen(false);

  trigger.addEventListener("click", (event) => {
    event.preventDefault();
    setOpen(!open);
  });

  document.addEventListener("click", (event) => {
    if (!open) {
      return;
    }
    if (event.target instanceof Node && root.contains(event.target)) {
      return;
    }
    closeMenu();
  });

  document.addEventListener("keydown", (event) => {
    if (!open) {
      return;
    }
    if (event.key === KEY_ESCAPE) {
      event.preventDefault();
      closeMenu();
      return;
    }
    if (event.key !== KEY_TAB) {
      return;
    }
    const items = focusable();
    if (items.length === 0) {
      event.preventDefault();
      panel.focus({ preventScroll: true });
      return;
    }
    const first = items[0];
    const last = items[items.length - 1];
    if (event.shiftKey) {
      if (document.activeElement === first) {
        event.preventDefault();
        last.focus({ preventScroll: true });
      }
      return;
    }
    if (document.activeElement === last) {
      event.preventDefault();
      first.focus({ preventScroll: true });
    }
  });
};

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
      if (event.key !== KEY_ESCAPE) {
        return;
      }

      let handled = false;
      const root = modalRoot();
      if (root && root.firstChild) {
        root.innerHTML = "";
        handled = true;
      }

      if (!handled) {
        const sidebar = sidebarRoot();
        const isSidebarOpen = sidebar && !sidebar.classList.contains("hidden");
        if (isSidebarOpen) {
          closeSidebar();
        }
      }
    });

    initSearchShortcut(modalRoot);
    initNotificationsBadge();
    initUserMenu();
  },
};

window.addEventListener("DOMContentLoaded", () => {
  window.hankoAdmin.init();
});
