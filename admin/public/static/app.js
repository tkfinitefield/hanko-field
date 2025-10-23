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

const isVisible = (element) => {
  if (!(element instanceof HTMLElement)) {
    return false;
  }
  if (element.offsetParent !== null) {
    return true;
  }
  return element.getClientRects().length > 0;
};

const getFocusableElements = (root) => {
  if (!(root instanceof Element)) {
    return [];
  }
  return Array.from(root.querySelectorAll(focusableSelectors)).filter(
    (el) => el instanceof HTMLElement && isVisible(el) && !el.hasAttribute("disabled"),
  );
};

const lockBodyScroll = (locked) => {
  document.documentElement.classList.toggle("modal-open", locked);
  if (document.body) {
    document.body.classList.toggle("has-open-modal", locked);
    if (locked) {
      document.body.dataset.modalOpen = "true";
    } else {
      delete document.body.dataset.modalOpen;
    }
  }
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

const createModalController = () => {
  const getRoot = () => {
    const element = document.getElementById("modal");
    return element instanceof HTMLElement ? element : null;
  };

  const ensureRootDefaults = () => {
    const modal = getRoot();
    if (!modal) {
      return null;
    }
    modal.classList.add("modal");
    if (!modal.dataset.modalState) {
      modal.dataset.modalState = "closed";
    }
    if (!modal.dataset.modalOpen) {
      modal.dataset.modalOpen = "false";
    }
    if (!modal.hasAttribute("aria-hidden")) {
      modal.setAttribute("aria-hidden", "true");
    }
    return modal;
  };

  let root = ensureRootDefaults();
  if (!root) {
    return {
      get root() {
        return null;
      },
      close: () => {},
      clear: () => {},
      isOpen: () => false,
    };
  }

  let isOpen = false;
  let lastActiveElement = null;

  const getPanel = () => {
    const modal = getRoot();
    return modal ? modal.querySelector("[data-modal-panel]") : null;
  };

  const getOverlay = () => {
    const modal = getRoot();
    return modal ? modal.querySelector("[data-modal-overlay]") : null;
  };

  const ensurePanelFocusable = () => {
    const panel = getPanel();
    if (panel instanceof HTMLElement && !panel.hasAttribute("tabindex")) {
      panel.setAttribute("tabindex", "-1");
    }
  };

  const focusInitialElement = () => {
    ensurePanelFocusable();
    const modal = getRoot();
    if (!modal) {
      return;
    }
    const panel = getPanel();
    const focusTarget =
      modal.querySelector("[data-modal-autofocus]") ||
      modal.querySelector("[autofocus]") ||
      modal.querySelector("[data-autofocus]");
    const focusables = getFocusableElements(panel || modal);
    const target = focusTarget instanceof HTMLElement ? focusTarget : focusables[0];

    queueMicrotask(() => {
      if (target instanceof HTMLElement) {
        target.focus({ preventScroll: true });
        if (target instanceof HTMLInputElement || target instanceof HTMLTextAreaElement) {
          target.select();
        }
        return;
      }
      if (panel instanceof HTMLElement) {
        panel.focus({ preventScroll: true });
      }
    });
  };

  const finishClose = (restoreFocus) => {
    const modal = ensureRootDefaults();
    if (!modal) {
      return;
    }
    root = modal;
    const overlay = modal.querySelector("[data-modal-overlay]");
    if (overlay instanceof HTMLElement) {
      overlay.classList.remove("is-closing");
      overlay.removeAttribute("data-overlay-state");
    }
    modal.innerHTML = "";
    modal.classList.add("hidden");
    modal.setAttribute("aria-hidden", "true");
    modal.dataset.modalOpen = "false";
    modal.dataset.modalState = "closed";
    lockBodyScroll(false);
    isOpen = false;
    if (restoreFocus && lastActiveElement instanceof HTMLElement) {
      const elementToFocus = lastActiveElement;
      queueMicrotask(() => {
        if (elementToFocus instanceof HTMLElement) {
          elementToFocus.focus({ preventScroll: true });
        }
      });
    }
    lastActiveElement = null;
  };

  const close = ({ restoreFocus = true, skipAnimation = false } = {}) => {
    const modal = ensureRootDefaults();
    if (!modal) {
      return;
    }
    root = modal;

    if (!isOpen && modal.innerHTML.trim() === "") {
      finishClose(restoreFocus);
      return;
    }

    const panel = getPanel();
    const overlay = getOverlay();

    if (skipAnimation || !(panel instanceof HTMLElement)) {
      finishClose(restoreFocus);
      return;
    }

    modal.dataset.modalState = "closing";
    if (overlay instanceof HTMLElement) {
      overlay.classList.add("is-closing");
      overlay.setAttribute("data-overlay-state", "closing");
    }

    panel.classList.remove("animate-dialog-in");
    panel.classList.add("animate-dialog-out");
    panel.addEventListener(
      "animationend",
      () => {
        panel.classList.remove("animate-dialog-out");
        finishClose(restoreFocus);
      },
      { once: true },
    );
  };

  const open = () => {
    const modal = ensureRootDefaults();
    if (!modal) {
      return;
    }
    root = modal;

    if (isOpen) {
      focusInitialElement();
      return;
    }

    isOpen = true;
    lastActiveElement = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    modal.classList.remove("hidden");
    modal.setAttribute("aria-hidden", "false");
    modal.dataset.modalOpen = "true";
    modal.dataset.modalState = "opening";
    lockBodyScroll(true);

    const overlay = getOverlay();
    if (overlay instanceof HTMLElement) {
      overlay.classList.remove("is-closing");
      overlay.setAttribute("data-overlay-state", "opening");
    }

    requestAnimationFrame(() => {
      const currentModal = ensureRootDefaults();
      if (!currentModal) {
        return;
      }
      currentModal.dataset.modalState = "open";
      const currentOverlay = getOverlay();
      if (currentOverlay instanceof HTMLElement) {
        currentOverlay.setAttribute("data-overlay-state", "open");
        currentOverlay.classList.remove("is-closing");
      }
      const panel = getPanel();
      if (panel instanceof HTMLElement) {
        panel.classList.remove("animate-dialog-out");
      }
      focusInitialElement();
    });
  };

  const handleKeydown = (event) => {
    if (!isOpen) {
      return;
    }
    if (event.key === KEY_ESCAPE) {
      event.preventDefault();
      event.stopPropagation();
      close();
      return;
    }
    if (event.key !== KEY_TAB) {
      return;
    }

    const modal = getRoot();
    if (!modal) {
      return;
    }

    const panel = getPanel();
    const focusable = getFocusableElements(panel || modal);
    if (focusable.length === 0) {
      event.preventDefault();
      event.stopPropagation();
      if (panel instanceof HTMLElement) {
        panel.focus({ preventScroll: true });
      }
      return;
    }

    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    const active = document.activeElement;

    if (event.shiftKey) {
      if (active === first || !modal.contains(active)) {
        event.preventDefault();
        event.stopPropagation();
        last.focus({ preventScroll: true });
      }
      return;
    }

    if (active === last) {
      event.preventDefault();
      event.stopPropagation();
      first.focus({ preventScroll: true });
    }
  };

  const handleDocumentClick = (event) => {
    const modal = getRoot();
    if (!modal || !(event.target instanceof Element)) {
      return;
    }
    if (!modal.contains(event.target)) {
      return;
    }
    const closeTrigger = event.target.closest("[data-modal-close]");
    if (closeTrigger) {
      event.preventDefault();
      close();
      return;
    }
    const overlay = getOverlay();
    if (overlay instanceof HTMLElement && event.target === overlay) {
      event.preventDefault();
      close();
    }
  };

  document.addEventListener("keydown", handleKeydown);
  document.addEventListener("click", handleDocumentClick);

  const handleModalSwap = (event) => {
    const target = event.target;
    if (target instanceof HTMLElement && target.id === "modal") {
      root = ensureRootDefaults();
    }
  };

  document.body.addEventListener("htmx:oobAfterSwap", (event) => {
    handleModalSwap(event);
    const detailTarget = event.detail && event.detail.target;
    if (detailTarget instanceof HTMLElement && detailTarget.id === "modal") {
      root = ensureRootDefaults();
    }
  });

  document.body.addEventListener("htmx:afterSwap", (event) => {
    const target = event.target;
    if (!(target instanceof HTMLElement) || target.id !== "modal") {
      return;
    }
    root = ensureRootDefaults();
    const hasContent = target.innerHTML.trim().length > 0;
    if (!hasContent) {
      close({ skipAnimation: true });
      return;
    }
    open();
  });

  document.body.addEventListener("htmx:beforeSwap", (event) => {
    const target = event.target;
    if (!(target instanceof HTMLElement) || target.id !== "modal") {
      return;
    }
    root = ensureRootDefaults();
    const panel = getPanel();
    if (panel instanceof HTMLElement) {
      panel.classList.remove("animate-dialog-out");
    }
  });

  document.body.addEventListener("modal:close", (event) => {
    const detail = event instanceof CustomEvent ? event.detail || {} : {};
    close({
      restoreFocus: detail.restoreFocus !== false,
      skipAnimation: detail.skipAnimation === true,
    });
  });

  document.body.addEventListener("modal:clear", () => {
    close({ skipAnimation: true, restoreFocus: false });
  });

  document.body.addEventListener("modal:open", () => {
    const modal = ensureRootDefaults();
    if (modal && modal.innerHTML.trim().length > 0) {
      root = modal;
      open();
    }
  });

  return {
    get root() {
      return getRoot();
    },
    close,
    clear: () => close({ skipAnimation: true, restoreFocus: false }),
    isOpen: () => isOpen,
  };
};

const normaliseRefreshDetail = (detail, defaultEvent = "refresh") => {
  const selectors = new Set();
  let eventName = defaultEvent;
  let delay = 0;

  const push = (value) => {
    if (typeof value === "string" && value.trim() !== "") {
      selectors.add(value.trim());
    }
  };

  if (typeof detail === "string") {
    push(detail);
  } else if (Array.isArray(detail)) {
    detail.forEach(push);
  } else if (detail && typeof detail === "object") {
    if (Array.isArray(detail.targets)) {
      detail.targets.forEach(push);
    }
    if (Array.isArray(detail.selectors)) {
      detail.selectors.forEach(push);
    }
    if (typeof detail.target === "string") {
      push(detail.target);
    }
    if (typeof detail.selector === "string") {
      push(detail.selector);
    }
    if (typeof detail.event === "string" && detail.event.trim() !== "") {
      eventName = detail.event.trim();
    } else if (typeof detail.trigger === "string" && detail.trigger.trim() !== "") {
      eventName = detail.trigger.trim();
    }
    if (typeof detail.delay === "number" && Number.isFinite(detail.delay) && detail.delay > 0) {
      delay = detail.delay;
    }
  }

  return {
    selectors: Array.from(selectors),
    eventName,
    delay,
  };
};

const triggerFragmentRefresh = (detail, fallbackEvent) => {
  const { selectors, eventName, delay } = normaliseRefreshDetail(detail, fallbackEvent);
  if (selectors.length === 0) {
    return;
  }
  const emit = () => {
    selectors.forEach((selector) => {
      document.querySelectorAll(selector).forEach((element) => {
        if (window.htmx && typeof window.htmx.trigger === "function") {
          window.htmx.trigger(element, eventName);
        } else {
          element.dispatchEvent(new CustomEvent(eventName, { bubbles: true }));
        }
      });
    });
  };
  if (delay > 0) {
    window.setTimeout(emit, delay);
  } else {
    emit();
  }
};

const initHXTriggerHandlers = () => {
  const bus = document.body;
  if (!bus) {
    return;
  }

  const handler = (eventName) => (event) => {
    const detail = event instanceof CustomEvent ? event.detail : undefined;
    triggerFragmentRefresh(detail, eventName);
  };

  bus.addEventListener("refresh:fragment", handler("refresh"));
  bus.addEventListener("refresh:fragments", handler("refresh"));
  bus.addEventListener("refresh:targets", handler("refresh"));
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
    const modal = createModalController();
    const modalRoot = () => modal.root;
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
        if (document.body) {
          document.body.classList.add("overflow-hidden");
        }
      } else {
        sidebar.classList.add("hidden");
        sidebar.classList.remove("flex");
        sidebar.setAttribute("aria-hidden", "true");
        if (document.body) {
          document.body.classList.remove("overflow-hidden");
        }
      }
      document.querySelectorAll("[data-sidebar-toggle]").forEach((el) => {
        el.setAttribute("aria-expanded", open ? "true" : "false");
      });
    };

    const closeSidebar = () => setSidebarState(false);
    const openSidebar = () => setSidebarState(true);

    document.addEventListener("click", (event) => {
      const target = event.target instanceof Element ? event.target : null;
      if (!target) {
        return;
      }
      const sidebarToggle = target.closest("[data-sidebar-toggle]");
      if (sidebarToggle) {
        event.preventDefault();
        openSidebar();
        return;
      }
      const sidebarDismiss = target.closest("[data-sidebar-dismiss]");
      if (sidebarDismiss) {
        event.preventDefault();
        closeSidebar();
      }
    });

    document.addEventListener("keydown", (event) => {
      if (event.key !== KEY_ESCAPE) {
        return;
      }
      if (modal.isOpen()) {
        return;
      }
      const sidebar = sidebarRoot();
      const isSidebarOpen = sidebar && !sidebar.classList.contains("hidden");
      if (isSidebarOpen) {
        event.preventDefault();
        closeSidebar();
      }
    });

    initSearchShortcut(modalRoot);
    initNotificationsBadge();
    initHXTriggerHandlers();
    initUserMenu();

    window.hankoAdmin.modal = modal;
  },
};

window.addEventListener("DOMContentLoaded", () => {
  window.hankoAdmin.init();
});
