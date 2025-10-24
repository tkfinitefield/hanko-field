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

const badgeClassForTone = (tone) => {
  const base = ["badge"];
  switch ((tone || "").toLowerCase()) {
    case "success":
      base.push("badge-success");
      break;
    case "danger":
      base.push("badge-danger");
      break;
    case "warning":
      base.push("badge-warning");
      break;
    case "info":
      base.push("badge-info");
      break;
    default:
      break;
  }
  return base.join(" ");
};

const updateBadgeTone = (element, tone) => {
  if (!(element instanceof HTMLElement)) {
    return;
  }
  element.className = badgeClassForTone(tone);
};

const updateSelectedQueryParam = (id) => {
  try {
    const url = new URL(window.location.href);
    if (typeof id === "string" && id.trim() !== "") {
      url.searchParams.set("selected", id.trim());
    } else {
      url.searchParams.delete("selected");
    }
    window.history.replaceState({}, "", url.toString());
  } catch (error) {
    // ignore invalid URLs (e.g., older browsers)
  }
};

const buildMetadataRow = (label, value) => {
  const row = document.createElement("div");
  row.className = "flex items-center justify-between gap-2";
  const dt = document.createElement("dt");
  dt.className = "font-medium text-slate-500";
  dt.textContent = label || "";
  const dd = document.createElement("dd");
  dd.textContent = value || "";
  row.appendChild(dt);
  row.appendChild(dd);
  return row;
};

const buildActionButton = (action) => {
  const anchor = document.createElement("a");
  anchor.className = "btn btn-secondary btn-sm";
  anchor.href = typeof action.url === "string" ? action.url : "#";
  anchor.textContent = action.label || "詳細";
  if (typeof action.icon === "string" && action.icon.trim() !== "") {
    anchor.textContent = `${action.icon} ${anchor.textContent}`;
  }
  return anchor;
};

const buildTimelineItem = (event) => {
  const item = document.createElement("li");
  item.className = "mb-4 last:mb-0";

  const marker = document.createElement("div");
  marker.className = "absolute -left-1.5 h-3 w-3 rounded-full border border-white bg-slate-300";
  item.appendChild(marker);

  const meta = document.createElement("p");
  meta.className = "flex items-center gap-2 text-xs text-slate-400";
  if (typeof event.icon === "string" && event.icon.trim() !== "") {
    const icon = document.createElement("span");
    icon.textContent = event.icon;
    meta.appendChild(icon);
  }
  const timeLabel = document.createElement("span");
  timeLabel.textContent = event.occurredRelative || "";
  meta.appendChild(timeLabel);
  if (typeof event.actor === "string" && event.actor.trim() !== "") {
    const actor = document.createElement("span");
    actor.textContent = `· ${event.actor}`;
    meta.appendChild(actor);
  }
  item.appendChild(meta);

  const title = document.createElement("p");
  title.className = "mt-1 font-medium text-slate-800";
  title.textContent = event.title || "";
  item.appendChild(title);

  if (typeof event.description === "string" && event.description.trim() !== "") {
    const description = document.createElement("p");
    description.className = "mt-1 text-slate-600";
    description.textContent = event.description;
    item.appendChild(description);
  }

  return item;
};

const replaceResourceElement = (scope, payload) => {
  if (!(scope instanceof HTMLElement)) {
    return;
  }
  const resource = payload && payload.resource ? payload.resource : {};
  const current = scope.querySelector("[data-notification-resource]");
  const element = document.createElement(typeof resource.url === "string" && resource.url.trim() !== "" ? "a" : "span");
  element.setAttribute("data-notification-resource", "");
  if (element instanceof HTMLAnchorElement && resource.url) {
    element.href = resource.url;
    element.className = "text-brand-600 hover:text-brand-500";
  }
  element.textContent = resource.label || "";
  if (current instanceof HTMLElement) {
    current.replaceWith(element);
  } else {
    const placeholder = scope.querySelector("[data-notification-resource-placeholder]");
    if (placeholder instanceof HTMLElement) {
      placeholder.replaceWith(element);
    }
  }
};

const renderNotificationDetail = (payload) => {
  const container = document.querySelector("[data-notification-detail]");
  if (!(container instanceof HTMLElement) || !payload) {
    return;
  }

  container.dataset.notificationId = payload.id || "";

  const category = container.querySelector("[data-notification-category]");
  if (category) {
    category.textContent = payload.categoryLabel || "";
  }

  const title = container.querySelector("[data-notification-title]");
  if (title) {
    title.textContent = payload.title || "";
  }

  const summary = container.querySelector("[data-notification-summary]");
  if (summary) {
    summary.textContent = payload.summary || "";
  }

  const severity = container.querySelector("[data-notification-severity]");
  if (severity instanceof HTMLElement) {
    severity.textContent = payload.severityLabel || "";
    updateBadgeTone(severity, payload.severityTone);
  }

  const status = container.querySelector("[data-notification-status]");
  if (status instanceof HTMLElement) {
    status.textContent = payload.statusLabel || "";
    updateBadgeTone(status, payload.statusTone);
  }

  const owner = container.querySelector("[data-notification-owner]");
  if (owner) {
    owner.textContent = payload.owner || "";
    const ownerRow = owner.closest("div");
    if (ownerRow instanceof HTMLElement) {
      ownerRow.hidden = !(payload.owner && payload.owner.trim() !== "");
    }
  }

  replaceResourceElement(container, payload);
  const resourceElement = container.querySelector("[data-notification-resource]");
  if (resourceElement) {
    const label = resourceElement.parentElement && resourceElement.parentElement.previousElementSibling;
    if (label instanceof HTMLElement) {
      const kind = payload && payload.resource ? payload.resource.kind : "";
      label.textContent = kind || "リソース";
    }
  }

  const created = container.querySelector("[data-notification-created]");
  if (created instanceof HTMLElement) {
    created.textContent = payload.createdRelative || "";
    if (payload.createdAt) {
      created.title = payload.createdAt;
    }
  }

  const acknowledged = container.querySelector("[data-notification-acknowledged]");
  if (acknowledged instanceof HTMLElement) {
    acknowledged.textContent = payload.acknowledgedLabel || "";
    const row = acknowledged.closest("div");
    if (row instanceof HTMLElement) {
      row.hidden = !(payload.acknowledgedLabel && payload.acknowledgedLabel.trim() !== "");
    }
  }

  const resolved = container.querySelector("[data-notification-resolved]");
  if (resolved instanceof HTMLElement) {
    resolved.textContent = payload.resolvedLabel || "";
    const row = resolved.closest("div");
    if (row instanceof HTMLElement) {
      row.hidden = !(payload.resolvedLabel && payload.resolvedLabel.trim() !== "");
    }
  }

  const metadataContainer = container.querySelector("[data-notification-metadata-container]");
  const metadataList = container.querySelector("[data-notification-metadata]");
  if (metadataList instanceof HTMLElement) {
    metadataList.innerHTML = "";
    const list = Array.isArray(payload.metadata) ? payload.metadata : [];
    list.forEach((meta) => {
      metadataList.appendChild(buildMetadataRow(meta.label, meta.value));
    });
    if (metadataContainer instanceof HTMLElement) {
      metadataContainer.hidden = list.length === 0;
    }
  }

  const actionContainer = container.querySelector("[data-notification-actions]");
  const actionList = container.querySelector("[data-notification-action-list]");
  if (actionList instanceof HTMLElement) {
    actionList.innerHTML = "";
    const actions = Array.isArray(payload.links) ? payload.links : [];
    actions.forEach((action) => {
      actionList.appendChild(buildActionButton(action));
    });
    if (actionContainer instanceof HTMLElement) {
      actionContainer.hidden = actions.length === 0;
    }
  }

  const timelineContainer = container.querySelector("[data-notification-timeline-container]");
  const timelineList = container.querySelector("[data-notification-timeline]");
  if (timelineList instanceof HTMLElement) {
    timelineList.innerHTML = "";
    const events = Array.isArray(payload.timeline) ? payload.timeline : [];
    events.forEach((event) => {
      timelineList.appendChild(buildTimelineItem(event));
    });
    if (timelineContainer instanceof HTMLElement) {
      timelineContainer.hidden = events.length === 0;
    }
  }
};

const selectNotificationRow = (row) => {
  if (!(row instanceof HTMLElement)) {
    return;
  }
  const table = row.closest("table");
  if (table instanceof HTMLElement) {
    table.querySelectorAll("[data-notification-row]").forEach((tr) => {
      if (!(tr instanceof HTMLElement)) {
        return;
      }
      if (tr === row) {
        tr.dataset.selected = "true";
        tr.classList.add("bg-brand-50");
      } else {
        delete tr.dataset.selected;
        tr.classList.remove("bg-brand-50");
      }
    });
  } else {
    row.dataset.selected = "true";
    row.classList.add("bg-brand-50");
  }
};

const parseNotificationPayload = (row) => {
  if (!(row instanceof HTMLElement)) {
    return null;
  }
  const raw = row.getAttribute("data-notification-payload");
  if (typeof raw !== "string" || raw.trim() === "") {
    return null;
  }
  try {
    return JSON.parse(raw);
  } catch (error) {
    return null;
  }
};

const initNotificationsSelection = () => {
  const root = document.querySelector("[data-notifications-root]");
  if (!(root instanceof HTMLElement)) {
    return;
  }

  const applyDefaultSelection = (scope) => {
    const table = scope.querySelector("[data-notifications-table]");
    if (!(table instanceof HTMLElement)) {
      return;
    }
    const selectedRow =
      table.querySelector("[data-notification-row][data-selected='true']") ||
      table.querySelector("[data-notification-row]");
    const payload = selectedRow ? parseNotificationPayload(selectedRow) : null;
    if (selectedRow && payload) {
      selectNotificationRow(selectedRow);
      renderNotificationDetail(payload);
      updateSelectedQueryParam(payload.id);
    }
  };

  const shouldIgnoreClick = (event) => {
    if (!(event.target instanceof Element)) {
      return true;
    }
    const interactive = event.target.closest("a, button, input, textarea, select, [role='button']");
    return Boolean(interactive);
  };

  root.addEventListener("click", (event) => {
    if (!(event instanceof MouseEvent)) {
      return;
    }
    if (shouldIgnoreClick(event)) {
      return;
    }
    const row = event.target instanceof Element ? event.target.closest("[data-notification-row]") : null;
    if (!(row instanceof HTMLElement)) {
      return;
    }
    const payload = parseNotificationPayload(row);
    if (!payload) {
      return;
    }
    event.preventDefault();
    event.stopPropagation();
    selectNotificationRow(row);
    renderNotificationDetail(payload);
    updateSelectedQueryParam(payload.id);
  });

  applyDefaultSelection(root);

  if (window.htmx) {
    document.body.addEventListener("htmx:afterSwap", (event) => {
      if (!(event.target instanceof Element)) {
        return;
      }
      const table = event.target.closest?.("[data-notifications-table]") || (event.target.matches && event.target.matches("[data-notifications-table]") ? event.target : null);
      if (table instanceof HTMLElement) {
        applyDefaultSelection(table);
      }
    });
  }
};

const initDashboardRefresh = () => {
  const parseTargets = (value) => {
    if (typeof value !== "string" || value.trim() === "") {
      return [];
    }
    return value
      .split(",")
      .map((entry) => entry.trim())
      .filter((entry) => entry.length > 0);
  };

  const clearRefreshState = (button) => {
    delete button.dataset.loading;
    delete button.dataset.dashboardPending;
    button.classList.remove("btn-loading");
    button.removeAttribute("aria-busy");
  };

  document.addEventListener("click", (event) => {
    const trigger = event.target instanceof Element ? event.target.closest("[data-dashboard-refresh]") : null;
    if (!trigger) {
      return;
    }
    if (!window.htmx) {
      return;
    }
    event.preventDefault();
    const selectors = parseTargets(trigger.getAttribute("data-dashboard-targets"));
    if (selectors.length === 0) {
      return;
    }
    trigger.dataset.loading = "true";
    trigger.dataset.dashboardPending = selectors.join(",");
    trigger.classList.add("btn-loading");
    trigger.setAttribute("aria-busy", "true");
    selectors.forEach((selector) => {
      window.htmx.trigger(selector, "refresh");
    });
  });

  if (!window.htmx) {
    return;
  }

  document.body.addEventListener("htmx:afterSettle", (event) => {
    const target = event.target;
    if (!(target instanceof Element)) {
      return;
    }

    document.querySelectorAll("[data-dashboard-refresh][data-loading]").forEach((button) => {
      const pending = button.dataset.dashboardPending || "";
      if (!pending) {
        clearRefreshState(button);
        return;
      }
      const selectors = parseTargets(pending);
      if (selectors.length === 0) {
        clearRefreshState(button);
        return;
      }
      const remaining = selectors.filter((selector) => {
        if (selector === "") {
          return false;
        }
        if (target.matches(selector) || target.closest(selector) !== null) {
          return false;
        }
        return true;
      });
      if (remaining.length === selectors.length) {
        return;
      }
      if (remaining.length === 0) {
        delete button.dataset.dashboardPending;
        clearRefreshState(button);
        return;
      }
      button.dataset.dashboardPending = remaining.join(",");
    });
  });
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

const toastToneClass = (tone) => {
  const key = typeof tone === "string" ? tone.toLowerCase() : "";
  switch (key) {
    case "success":
      return "toast-success";
    case "danger":
    case "error":
      return "toast-danger";
    case "warning":
      return "toast-warning";
    case "info":
    default:
      return "toast-info";
  }
};

const normaliseToastDetail = (detail) => {
  if (detail == null) {
    return null;
  }
  if (typeof detail === "string") {
    const message = detail.trim();
    if (message === "") {
      return null;
    }
    return {
      title: "",
      message,
      tone: "info",
      duration: 6000,
      dismissible: true,
    };
  }
  if (typeof detail !== "object") {
    return null;
  }

  const readString = (...candidates) => {
    for (const candidate of candidates) {
      if (typeof candidate === "string" && candidate.trim() !== "") {
        return candidate.trim();
      }
    }
    return "";
  };

  const tone = readString(detail.tone, detail.type, detail.level, "info");
  const title = readString(detail.title, detail.heading);
  const message = readString(detail.message, detail.body, detail.description, detail.value, detail.text, detail.label);

  const readNumber = (...candidates) => {
    for (const candidate of candidates) {
      if (typeof candidate === "number" && Number.isFinite(candidate)) {
        return candidate;
      }
      if (typeof candidate === "string" && candidate.trim() !== "") {
        const parsed = Number.parseInt(candidate.trim(), 10);
        if (Number.isFinite(parsed)) {
          return parsed;
        }
      }
    }
    return null;
  };

  const durationCandidate = readNumber(detail.duration, detail.timeout, detail.delay, detail.autoHideAfter);
  const defaultDuration = 6000;
  let duration = defaultDuration;
  if (durationCandidate !== null) {
    duration = durationCandidate;
  }

  let autoHide = true;
  if (detail.autoHide === false || detail.sticky === true) {
    autoHide = false;
  } else if (detail.autoHide === true) {
    autoHide = true;
  }

  if (!autoHide) {
    duration = 0;
  } else if (!Number.isFinite(duration) || duration <= 0) {
    duration = defaultDuration;
  }

  const dismissible = detail.dismissible !== false;
  const id = readString(detail.id);
  if (title === "" && message === "") {
    return null;
  }

  return {
    id,
    tone,
    title,
    message,
    duration,
    dismissible,
  };
};

const initToastStack = () => {
  const root = document.getElementById("toast-stack");
  if (!(root instanceof HTMLElement)) {
    return {
      root: null,
      show: () => {},
      clear: () => {},
      hide: () => {},
      remove: () => {},
    };
  }

  const timers = new WeakMap();
  const maxToasts = 4;

  const clearTimer = (toast) => {
    const timerId = timers.get(toast);
    if (typeof timerId === "number") {
      window.clearTimeout(timerId);
    }
    timers.delete(toast);
  };

  const finishRemoval = (toast) => {
    clearTimer(toast);
    if (toast instanceof HTMLElement) {
      toast.remove();
    }
  };

  const hideToast = (toast, { immediate = false } = {}) => {
    if (!(toast instanceof HTMLElement)) {
      return;
    }
    if (toast.dataset.toastState === "closing") {
      return;
    }
    toast.dataset.toastState = "closing";
    clearTimer(toast);
    const remove = () => {
      toast.removeEventListener("animationend", remove);
      finishRemoval(toast);
      toast.dataset.toastState = "";
    };
    if (immediate) {
      remove();
      return;
    }
    toast.classList.remove("animate-toast-in");
    toast.classList.add("animate-toast-out");
    toast.addEventListener("animationend", remove, { once: true });
    window.setTimeout(remove, 240);
  };

  const scheduleRemoval = (toast, duration) => {
    clearTimer(toast);
    if (!(toast instanceof HTMLElement) || !Number.isFinite(duration) || duration <= 0) {
      return;
    }
    const timerId = window.setTimeout(() => hideToast(toast), duration);
    timers.set(toast, timerId);
  };

  const createCloseButton = (onClose) => {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "toast-close";
    button.setAttribute("aria-label", "通知を閉じる");
    button.innerHTML =
      '<span class="sr-only">閉じる</span><svg viewBox="0 0 20 20" fill="none" aria-hidden="true" class="h-4 w-4"><path d="M5 5l10 10M15 5L5 15" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>';
    button.addEventListener("click", (event) => {
      event.preventDefault();
      event.stopPropagation();
      onClose();
    });
    return button;
  };

  const renderToast = (config) => {
    const toneClass = toastToneClass(config.tone);
    const toast = document.createElement("div");
    toast.className = ["toast", toneClass, "animate-toast-in"].filter(Boolean).join(" ");
    toast.setAttribute("role", "status");
    toast.setAttribute("aria-live", "polite");
    if (config.id) {
      try {
        toast.id = config.id;
      } catch (error) {
        // ignore invalid IDs (e.g., containing spaces)
      }
    }
    if (config.duration > 0) {
      toast.dataset.autohide = "true";
    }

    const body = document.createElement("div");
    body.className = "flex-1 space-y-1";

    if (config.title) {
      const heading = document.createElement("p");
      heading.className = "toast-title";
      heading.textContent = config.title;
      body.appendChild(heading);
    }

    if (config.message) {
      const message = document.createElement("p");
      message.className = config.title ? "toast-body" : "toast-body";
      message.textContent = config.message;
      body.appendChild(message);
    }

    toast.appendChild(body);

    if (config.dismissible) {
      toast.appendChild(
        createCloseButton(() => {
          hideToast(toast);
        }),
      );
    }

    toast.addEventListener("mouseenter", () => {
      clearTimer(toast);
    });

    toast.addEventListener("mouseleave", () => {
      scheduleRemoval(toast, config.duration);
    });

    toast.addEventListener("focusin", () => {
      clearTimer(toast);
    });

    toast.addEventListener("focusout", () => {
      scheduleRemoval(toast, config.duration);
    });

    return toast;
  };

  const show = (detail) => {
    const config = normaliseToastDetail(detail);
    if (!config) {
      return null;
    }

    while (root.childElementCount >= maxToasts) {
      const last = root.lastElementChild;
      if (last instanceof HTMLElement) {
        hideToast(last, { immediate: true });
      } else if (last) {
        root.removeChild(last);
      } else {
        break;
      }
    }

    const toast = renderToast(config);
    root.prepend(toast);
    requestAnimationFrame(() => {
      toast.classList.add("animate-toast-in");
    });
    scheduleRemoval(toast, config.duration);
    return toast;
  };

  const clear = () => {
    Array.from(root.children).forEach((child) => hideToast(child, { immediate: true }));
  };

  const removeById = (id) => {
    if (typeof id !== "string" || id.trim() === "") {
      return;
    }
    const safeId = window.CSS && typeof window.CSS.escape === "function" ? window.CSS.escape(id) : id.replace(/[^a-zA-Z0-9_-]/g, "");
    const toast = root.querySelector(`#${safeId}`);
    if (toast instanceof HTMLElement) {
      hideToast(toast);
    }
  };

  const eventNames = ["toast", "showToast", "toast:show"];
  const handleEvent = (event) => {
    if (!(event instanceof CustomEvent)) {
      return;
    }
    show(event.detail);
  };
  eventNames.forEach((name) => {
    document.body.addEventListener(name, handleEvent);
  });

  return {
    root,
    show,
    clear,
    hide: (toast) => hideToast(toast),
    remove: removeById,
  };
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

const initGlobalSearchInteractions = () => {
  const ROOT_SELECTOR = "[data-search-root]";
  const RESULTS_SELECTOR = "[data-search-results]";
  const RESULT_ROW_SELECTOR = "[data-search-result]";
  const controllers = new WeakMap();

  class SearchController {
    constructor(root) {
      this.root = root;
      this.form = root.querySelector("[data-search-form]");
      this.input = root.querySelector("[data-search-input]");
      this.results = [];
      this.activeRow = null;
      this.detail = root.querySelector("[data-search-detail]");
      this.detailEmpty = this.detail ? this.detail.querySelector("[data-search-detail-empty]") : null;
      this.detailContent = this.detail ? this.detail.querySelector("[data-search-detail-content]") : null;
      this.detailEntity = this.detail ? this.detail.querySelector("[data-search-detail-entity]") : null;
      this.detailTitle = this.detail ? this.detail.querySelector("[data-search-detail-title]") : null;
      this.detailDescription = this.detail ? this.detail.querySelector("[data-search-detail-description]") : null;
      this.detailBadge = this.detail ? this.detail.querySelector("[data-search-detail-badge]") : null;
      this.detailMetadata = this.detail ? this.detail.querySelector("[data-search-detail-metadata]") : null;
      this.detailOpen = this.detail ? this.detail.querySelector("[data-search-detail-open]") : null;
      this.detailCopy = this.detail ? this.detail.querySelector("[data-search-detail-copy]") : null;
      this.bindEvents();
    }

    bindEvents() {
      this.root.addEventListener("click", (event) => {
        const target = event.target instanceof Element ? event.target : null;
        if (!target) {
          return;
        }
        const openLink = target.closest("[data-search-open-link]");
        if (openLink) {
          return;
        }
        const row = target.closest(RESULT_ROW_SELECTOR);
        if (!row) {
          return;
        }
        event.preventDefault();
        this.setActive(row);
        row.focus({ preventScroll: true });
      });

      this.root.addEventListener("focusin", (event) => {
        const row = event.target instanceof Element ? event.target.closest(RESULT_ROW_SELECTOR) : null;
        if (!row) {
          return;
        }
        this.setActive(row);
      });

      if (this.detailCopy instanceof HTMLElement) {
        this.detailCopy.addEventListener("click", (event) => {
          event.preventDefault();
          this.copyActiveLink();
        });
      }
    }

    refresh() {
      const container = this.root.querySelector(RESULTS_SELECTOR);
      const rows = container ? Array.from(container.querySelectorAll(RESULT_ROW_SELECTOR)) : [];
      this.results = rows.filter((row) => row instanceof HTMLElement);
      if (this.results.length === 0) {
        this.setActive(null);
        return;
      }
      const current = this.activeRow && this.results.includes(this.activeRow) ? this.activeRow : null;
      if (current) {
        this.setActive(current);
        return;
      }
      this.setActive(this.results[0]);
    }

    focusInput() {
      if (!(this.input instanceof HTMLElement)) {
        return;
      }
      this.input.focus({ preventScroll: true });
      if (this.input instanceof HTMLInputElement || this.input instanceof HTMLTextAreaElement) {
        this.input.select();
      }
    }

    move(delta) {
      if (this.results.length === 0) {
        return;
      }
      const currentIndex = this.activeRow ? this.results.indexOf(this.activeRow) : -1;
      let nextIndex = currentIndex + delta;
      if (nextIndex < 0) {
        nextIndex = 0;
      }
      if (nextIndex >= this.results.length) {
        nextIndex = this.results.length - 1;
      }
      const nextRow = this.results[nextIndex];
      if (!nextRow) {
        return;
      }
      this.setActive(nextRow);
      nextRow.focus({ preventScroll: false });
      nextRow.scrollIntoView({ block: "nearest" });
    }

    openActive() {
      const row = this.activeRow;
      if (!row) {
        return;
      }
      const url = row.getAttribute("data-search-url");
      if (!url) {
        return;
      }
      window.location.assign(url);
    }

    copyActiveLink() {
      if (!(this.detailCopy instanceof HTMLElement)) {
        return;
      }
      const url = this.activeRow ? this.activeRow.getAttribute("data-search-url") : null;
      if (!url) {
        return;
      }
      const reset = () => {
        if (this.detailCopy) {
          delete this.detailCopy.dataset.searchCopyState;
        }
      };
      const applyCopiedState = () => {
        if (this.detailCopy) {
          this.detailCopy.dataset.searchCopyState = "copied";
          window.setTimeout(reset, 1500);
        }
      };
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard
          .writeText(url)
          .then(applyCopiedState)
          .catch((error) => {
            console.warn("search: clipboard write failed", error);
            fallbackCopy(url);
            applyCopiedState();
          });
        return;
      }
      fallbackCopy(url);
      applyCopiedState();
    }

    setActive(row) {
      if (this.activeRow === row) {
        return;
      }
      if (this.activeRow instanceof HTMLElement) {
        this.activeRow.classList.remove("ring-2", "ring-brand-200", "bg-slate-50");
        delete this.activeRow.dataset.searchActive;
      }
      this.activeRow = row instanceof HTMLElement ? row : null;
      if (this.activeRow) {
        this.activeRow.dataset.searchActive = "true";
        this.activeRow.classList.add("ring-2", "ring-brand-200", "bg-slate-50");
        this.populateDetail(this.activeRow);
      } else {
        this.clearDetail();
      }
    }

    populateDetail(row) {
      if (!this.detail) {
        return;
      }
      if (this.detailEmpty instanceof HTMLElement) {
        this.detailEmpty.classList.add("hidden");
      }
      if (this.detailContent instanceof HTMLElement) {
        this.detailContent.classList.remove("hidden");
      }
      if (this.detailEntity instanceof HTMLElement) {
        this.detailEntity.textContent = row.getAttribute("data-search-entity") || "";
      }
      if (this.detailTitle instanceof HTMLElement) {
        this.detailTitle.textContent = textContent(row, "[data-search-field='title']");
      }
      if (this.detailDescription instanceof HTMLElement) {
        this.detailDescription.textContent = textContent(row, "[data-search-field='description']");
        this.detailDescription.classList.toggle("hidden", this.detailDescription.textContent.trim() === "");
      }
      if (this.detailBadge instanceof HTMLElement) {
        const badge = row.querySelector("[data-search-field='badge']");
        this.detailBadge.innerHTML = badge ? badge.innerHTML : "";
        this.detailBadge.classList.toggle("hidden", !badge);
      }
      if (this.detailMetadata instanceof HTMLElement) {
        const metadata = row.querySelector("[data-search-field='metadata']");
        this.detailMetadata.innerHTML = metadata ? metadata.innerHTML : "";
        this.detailMetadata.classList.toggle("hidden", !metadata);
      }
      if (this.detailOpen instanceof HTMLElement) {
        const url = row.getAttribute("data-search-url") || "#";
        this.detailOpen.setAttribute("href", url);
      }
      if (this.detailCopy instanceof HTMLElement) {
        const url = row.getAttribute("data-search-url") || "";
        this.detailCopy.disabled = url === "";
      }
    }

    clearDetail() {
      if (!this.detail) {
        return;
      }
      if (this.detailContent instanceof HTMLElement) {
        this.detailContent.classList.add("hidden");
      }
      if (this.detailEmpty instanceof HTMLElement) {
        this.detailEmpty.classList.remove("hidden");
      }
    }
  }

  const fallbackCopy = (text) => {
    const textarea = document.createElement("textarea");
    textarea.value = text;
    textarea.setAttribute("readonly", "readonly");
    textarea.style.position = "absolute";
    textarea.style.left = "-1000px";
    textarea.style.top = "-1000px";
    document.body.appendChild(textarea);
    textarea.select();
    try {
      document.execCommand("copy");
    } catch (error) {
      console.warn("search: execCommand copy failed", error);
    }
    document.body.removeChild(textarea);
  };

  const textContent = (root, selector) => {
    const element = root.querySelector(selector);
    if (!element) {
      return "";
    }
    return element.textContent ? element.textContent.trim() : "";
  };

  const getRoot = () => document.querySelector(ROOT_SELECTOR);

  const getController = () => {
    const root = getRoot();
    if (!root) {
      return null;
    }
    let controller = controllers.get(root);
    if (!controller) {
      controller = new SearchController(root);
      controllers.set(root, controller);
      controller.refresh();
    }
    return controller;
  };

  const refreshController = () => {
    const controller = getController();
    if (controller) {
      controller.refresh();
    }
  };

  const root = getRoot();
  if (root) {
    controllers.set(root, new SearchController(root));
    refreshController();
  }

  document.addEventListener("keydown", (event) => {
    const controller = getController();
    if (!controller) {
      return;
    }
    if (event.key === "/" && !event.ctrlKey && !event.metaKey && !event.altKey) {
      if (isEditableTarget(event.target)) {
        return;
      }
      event.preventDefault();
      controller.focusInput();
      return;
    }
    if (event.key === "ArrowDown") {
      if (isEditableTarget(event.target)) {
        if (event.target instanceof Element && event.target.closest(ROOT_SELECTOR)) {
          return;
        }
        return;
      }
      event.preventDefault();
      controller.move(1);
      return;
    }
    if (event.key === "ArrowUp") {
      if (isEditableTarget(event.target)) {
        if (event.target instanceof Element && event.target.closest(ROOT_SELECTOR)) {
          return;
        }
        return;
      }
      event.preventDefault();
      controller.move(-1);
      return;
    }
    if (event.key === "Enter") {
      const target = event.target;
      if (isEditableTarget(target)) {
        if (target instanceof Element && target.closest("[data-search-form]")) {
          return;
        }
      }
      const row = target instanceof Element ? target.closest(RESULT_ROW_SELECTOR) : null;
      if (!row) {
        if (target instanceof Element && target.closest(ROOT_SELECTOR)) {
          event.preventDefault();
          controller.openActive();
        } else if (!isEditableTarget(target)) {
          event.preventDefault();
          controller.openActive();
        }
      } else {
        event.preventDefault();
        controller.openActive();
      }
    }
  });

  if (window.htmx) {
    document.body.addEventListener("htmx:afterSwap", (event) => {
      if (!(event.target instanceof Element)) {
        return;
      }
      if (event.target.matches(RESULTS_SELECTOR) || event.target.querySelector(RESULT_ROW_SELECTOR)) {
        refreshController();
      }
      if (event.target.closest && event.target.closest(RESULTS_SELECTOR)) {
        refreshController();
      }
    });
    document.body.addEventListener("htmx:afterSettle", (event) => {
      if (!(event.target instanceof Element)) {
        return;
      }
      if (event.target.matches(RESULTS_SELECTOR) || event.target.querySelector(RESULT_ROW_SELECTOR)) {
        refreshController();
      }
    });
  }
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
    initNotificationsSelection();
    initDashboardRefresh();
    initHXTriggerHandlers();
    const toast = initToastStack();
    initUserMenu();
    initGlobalSearchInteractions();

    window.hankoAdmin.modal = modal;
    window.hankoAdmin.toast = toast;
  },
};

window.addEventListener("DOMContentLoaded", () => {
  window.hankoAdmin.init();
});
