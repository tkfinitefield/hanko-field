package navigation

import (
	"strings"

	"finitefield.org/hanko-admin/internal/admin/rbac"
)

// Group represents a sidebar section.
type Group struct {
	Key        string
	Label      string
	Capability rbac.Capability
	Items      []Item
}

// Item represents a navigable entry.
type Item struct {
	Key          string
	Label        string
	Icon         string
	Capability   rbac.Capability
	Path         string
	Pattern      string
	MatchPrefix  bool
	External     bool
	OpenInNewTab bool
}

// MenuGroup is the resolved output used by templates.
type MenuGroup struct {
	Key        string
	Label      string
	Capability rbac.Capability
	Items      []MenuItem
}

// MenuItem is the resolved navigation entry with absolute paths.
type MenuItem struct {
	Key          string
	Label        string
	Icon         string
	Capability   rbac.Capability
	Href         string
	Pattern      string
	MatchPrefix  bool
	External     bool
	OpenInNewTab bool
}

// BuildMenu returns the sidebar configuration resolved for the provided base path.
func BuildMenu(basePath string) []MenuGroup {
	base := normaliseBase(basePath)
	menu := make([]MenuGroup, 0, len(defaultMenu))
	for _, group := range defaultMenu {
		items := make([]MenuItem, 0, len(group.Items))
		for _, raw := range group.Items {
			href := join(base, raw.Path)
			pattern := raw.Pattern
			if pattern == "" {
				pattern = raw.Path
			}
			pattern = join(base, pattern)
			items = append(items, MenuItem{
				Key:          raw.Key,
				Label:        raw.Label,
				Icon:         raw.Icon,
				Capability:   raw.Capability,
				Href:         href,
				Pattern:      pattern,
				MatchPrefix:  raw.MatchPrefix,
				External:     raw.External,
				OpenInNewTab: raw.OpenInNewTab,
			})
		}
		menu = append(menu, MenuGroup{
			Key:        group.Key,
			Label:      group.Label,
			Capability: group.Capability,
			Items:      items,
		})
	}
	return menu
}

var defaultMenu = []Group{
	{
		Key:        "overview",
		Capability: rbac.CapDashboardOverview,
		Items: []Item{
			{
				Key:         "dashboard",
				Label:       "ダッシュボード",
				Icon:        "🏠",
				Capability:  rbac.CapDashboardOverview,
				Path:        "/",
				Pattern:     "/",
				MatchPrefix: false,
			},
			{
				Key:         "search",
				Label:       "横断検索",
				Icon:        "🔍",
				Capability:  rbac.CapSearchGlobal,
				Path:        "/search",
				Pattern:     "/search",
				MatchPrefix: true,
			},
			{
				Key:         "notifications",
				Label:       "通知",
				Icon:        "🔔",
				Capability:  rbac.CapNotificationsFeed,
				Path:        "/notifications",
				Pattern:     "/notifications",
				MatchPrefix: true,
			},
		},
	},
	{
		Key:        "operations",
		Label:      "受注管理",
		Capability: rbac.CapOrdersList,
		Items: []Item{
			{
				Key:         "orders",
				Label:       "注文一覧",
				Icon:        "📦",
				Capability:  rbac.CapOrdersList,
				Path:        "/orders",
				Pattern:     "/orders",
				MatchPrefix: true,
			},
			{
				Key:         "shipments",
				Label:       "出荷バッチ",
				Icon:        "🚚",
				Capability:  rbac.CapShipmentsMonitor,
				Path:        "/shipments/batches",
				Pattern:     "/shipments/batches",
				MatchPrefix: true,
			},
			{
				Key:         "shipments-tracking",
				Label:       "配送トラッキング",
				Icon:        "🛰",
				Capability:  rbac.CapShipmentsMonitor,
				Path:        "/shipments/tracking",
				Pattern:     "/shipments/tracking",
				MatchPrefix: true,
			},
			{
				Key:         "production",
				Label:       "制作カンバン",
				Icon:        "🛠",
				Capability:  rbac.CapProductionQueues,
				Path:        "/production/queues",
				Pattern:     "/production",
				MatchPrefix: true,
			},
		},
	},
	{
		Key:        "catalog",
		Label:      "カタログ",
		Capability: rbac.CapCatalogManage,
		Items: []Item{
			{
				Key:         "catalog-products",
				Label:       "SKU管理",
				Icon:        "🧾",
				Capability:  rbac.CapCatalogManage,
				Path:        "/catalog/products",
				Pattern:     "/catalog",
				MatchPrefix: true,
			},
			{
				Key:         "catalog-templates",
				Label:       "テンプレート",
				Icon:        "📐",
				Capability:  rbac.CapCatalogManage,
				Path:        "/catalog/templates",
				Pattern:     "/catalog/templates",
				MatchPrefix: true,
			},
			{
				Key:         "catalog-fonts",
				Label:       "フォント",
				Icon:        "🔤",
				Capability:  rbac.CapCatalogFonts,
				Path:        "/catalog/fonts",
				Pattern:     "/catalog/fonts",
				MatchPrefix: true,
			},
		},
	},
	{
		Key:        "content",
		Label:      "コンテンツ",
		Capability: rbac.CapContentManage,
		Items: []Item{
			{
				Key:         "content-guides",
				Label:       "ガイド",
				Icon:        "📚",
				Capability:  rbac.CapContentManage,
				Path:        "/content/guides",
				Pattern:     "/content/guides",
				MatchPrefix: true,
			},
			{
				Key:         "content-pages",
				Label:       "固定ページ",
				Icon:        "📄",
				Capability:  rbac.CapContentManage,
				Path:        "/content/pages",
				Pattern:     "/content/pages",
				MatchPrefix: true,
			},
		},
	},
	{
		Key:        "marketing",
		Label:      "マーケ",
		Capability: rbac.CapPromotionsManage,
		Items: []Item{
			{
				Key:         "promotions",
				Label:       "プロモーション",
				Icon:        "🎯",
				Capability:  rbac.CapPromotionsManage,
				Path:        "/promotions",
				Pattern:     "/promotions",
				MatchPrefix: true,
			},
			{
				Key:         "reviews",
				Label:       "レビュー審査",
				Icon:        "⭐",
				Capability:  rbac.CapReviewsModerate,
				Path:        "/reviews",
				Pattern:     "/reviews",
				MatchPrefix: true,
			},
		},
	},
	{
		Key:        "customers",
		Label:      "顧客",
		Capability: rbac.CapCustomersView,
		Items: []Item{
			{
				Key:         "customers",
				Label:       "顧客一覧",
				Icon:        "👥",
				Capability:  rbac.CapCustomersView,
				Path:        "/customers",
				Pattern:     "/customers",
				MatchPrefix: true,
			},
		},
	},
	{
		Key:        "system",
		Label:      "システム",
		Capability: rbac.CapSystemTasks,
		Items: []Item{
			{
				Key:         "audit-logs",
				Label:       "監査ログ",
				Icon:        "📝",
				Capability:  rbac.CapAuditLogView,
				Path:        "/audit-logs",
				Pattern:     "/audit-logs",
				MatchPrefix: true,
			},
			{
				Key:         "system-tasks",
				Label:       "タスク/ジョブ",
				Icon:        "⏱",
				Capability:  rbac.CapSystemTasks,
				Path:        "/system/tasks",
				Pattern:     "/system/tasks",
				MatchPrefix: true,
			},
			{
				Key:         "system-counters",
				Label:       "カウンタ",
				Icon:        "🔢",
				Capability:  rbac.CapSystemCounters,
				Path:        "/system/counters",
				Pattern:     "/system/counters",
				MatchPrefix: true,
			},
			{
				Key:         "org-staff",
				Label:       "スタッフ管理",
				Icon:        "🧑‍🤝‍🧑",
				Capability:  rbac.CapStaffManage,
				Path:        "/org/staff",
				Pattern:     "/org/staff",
				MatchPrefix: true,
			},
		},
	},
	{
		Key:        "account",
		Label:      "アカウント",
		Capability: rbac.CapProfileSelf,
		Items: []Item{
			{
				Key:         "profile",
				Label:       "プロフィール",
				Icon:        "👤",
				Capability:  rbac.CapProfileSelf,
				Path:        "/profile",
				Pattern:     "/profile",
				MatchPrefix: true,
			},
			{
				Key:         "logout",
				Label:       "ログアウト",
				Icon:        "↩",
				Capability:  "",
				Path:        "/logout",
				Pattern:     "/logout",
				MatchPrefix: false,
			},
		},
	},
}

func normaliseBase(base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		return "/"
	}
	if !strings.HasPrefix(base, "/") {
		base = "/" + base
	}
	if base != "/" {
		base = strings.TrimRight(base, "/")
		if base == "" {
			return "/"
		}
	}
	return base
}

func join(base, suffix string) string {
	base = normaliseBase(base)
	suffix = strings.TrimSpace(suffix)
	if suffix == "" || suffix == "/" {
		if base == "" {
			return "/"
		}
		return base
	}
	if !strings.HasPrefix(suffix, "/") {
		suffix = "/" + suffix
	}
	if base == "/" {
		res := suffix
		if res == "" {
			return "/"
		}
		return normalizePath(res)
	}
	return normalizePath(base+suffix, base)
}

func normalizePath(path string, bases ...string) string {
	path = strings.ReplaceAll(path, "//", "/")
	if len(path) > 1 {
		path = strings.TrimRight(path, "/")
		if path == "" {
			return "/"
		}
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if len(bases) > 0 && bases[0] == "/" && path == "" {
		return "/"
	}
	return path
}
