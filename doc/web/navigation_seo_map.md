# Web ナビゲーション & SEO マップ（Task 004）

## 1. グローバルナビゲーション構造
```
/                                  (ランディング)
├─ /shop                           (素材×形×サイズフィルタ)
│   └─ /products/{productId}       (SKU 詳細)
├─ /templates                      (印影テンプレ)
│   └─ /templates/{templateId}
├─ /guides                         (ガイド一覧)
│   └─ /guides/{slug}              (記事)
├─ /content/{slug}                 (静的ページ：特商法/FAQ等)
├─ /status                         (システムステータス)
├─ /design/new                     (作成タイプ選択)
│   ├─ /design/editor              (エディタ 2 ペイン)
│   ├─ /design/ai                  (AI 候補)
│   ├─ /design/preview             (実寸プレビュー)
│   └─ /design/versions            (履歴)
├─ /cart
└─ /checkout
    ├─ /checkout/address
    ├─ /checkout/shipping
    ├─ /checkout/payment
    ├─ /checkout/review
    └─ /checkout/complete

アカウントサブサイト
/account
├─ /account/addresses
├─ /account/orders
│   └─ /account/orders/{orderId}
├─ /account/library
└─ /account/security

サポート/法務
/support
/legal/{slug}
```
- `SiteNav`（公開）と `AccountNav`（ログイン後）を分離。`DesignNav` はエディタ系導線でパンくずに包含。
- `SiteNav` 項目順：`Shop`, `Templates`, `Guides`, `Status`, `Support`. CTA ボタンは `/design/new`.

## 2. パンくず定義
| 画面 | パンくずパス | 備考 |
| --- | --- | --- |
| `/` | ホーム | パンくず表示なし。 |
| `/shop` | ホーム › 素材・ラインナップ | `Shop` ラベル。 |
| `/products/{id}` | ホーム › 素材・ラインナップ › {商品名} | 商品名は viewmodel で注入。 |
| `/templates` | ホーム › テンプレート | |
| `/templates/{id}` | ホーム › テンプレート › {テンプレ名} | |
| `/guides` | ホーム › ガイド | |
| `/guides/{slug}` | ホーム › ガイド › {タイトル} | `BreadcrumbList` JSON-LD を生成。 |
| `/content/{slug}` | ホーム › {ページ名} | 特商法など。 |
| `/status` | ホーム › ステータス | |
| `/design/new` | ホーム › デザイン › 作成タイプ | |
| `/design/editor` | ホーム › デザイン › エディタ | バージョンID存在時は “エディタ (vX)”。 |
| `/design/ai` | ホーム › デザイン › AI 候補 | |
| `/design/preview` | ホーム › デザイン › プレビュー | |
| `/design/versions` | ホーム › デザイン › バージョン履歴 | |
| `/cart` | ホーム › カート | |
| `/checkout/*` | ホーム › チェックアウト › {ステップ名} | ステップごとに `CheckoutStepper` と同期。 |
| `/account` | ホーム › マイアカウント | |
| `/account/addresses` | ホーム › マイアカウント › 住所帳 | |
| `/account/orders` | ホーム › マイアカウント › 注文履歴 | |
| `/account/orders/{id}` | ホーム › マイアカウント › 注文履歴 › 注文 {短縮ID} | タブ構造はパンくずに含めない。 |
| `/account/library` | ホーム › マイアカウント › マイ印鑑 | |
| `/account/security` | ホーム › マイアカウント › セキュリティ | |
| `/support` | ホーム › サポート | |
| `/legal/{slug}` | ホーム › 法務 › {タイトル} | 改定日メタ情報を付与。 |

- パンくずテキストは i18n 辞書キー（例：`nav.shop`, `breadcrumb.design.editor`）で管理。
- htmx フラグメントでタブ切り替え時はパンくずに影響を与えず、タブ固有の `aria-label` で補完。

## 3. SEO / メタデータ要件
| 画面 | `<title>` / `<meta description>` | OGP | JSON-LD | その他 |
| --- | --- | --- | --- | --- |
| `/` | ブランドタグライン + CTA / 160 文字以内 | `og:type=website`, キービジュアル | Organization + WebSite + BreadcrumbList | `link rel=alternate hreflang` (`ja`, `en`) |
| `/shop` | 「素材・形から探す」系キーワード | `og:type=website` | BreadcrumbList | Facet 状態は `?material=` 等クエリで canonical を談合 |
| `/products/{id}` | 商品名 + 素材/サイズ | `og:type=product`, `og:image` | Product（価格/在庫/レビュー） | `canonical` を SKU 固有 URL で固定 |
| `/templates` | 印影テンプレート一覧 | `og:type=website` | ItemList | |
| `/guides/{slug}` | 記事タイトル | `og:type=article` | Article（author/date）, BreadcrumbList | `rel=prev/next` をカテゴリ内で設定 |
| `/design/editor` | 「印影エディタ」 + 現在の名前 | `og:type=website` | 生成不要（認証後専用） | `noindex`（認証領域） |
| `/design/ai` | 同上 | 同上 | 同上 | `noindex` |
| `/cart` / `/checkout/*` | ステップ名 + ブランド | 共有 OG（プライバシー保護） | なし | `noindex` |
| `/account/*` | セクション名 + ブランド | なし | なし | `noindex`, `Cache-Control: no-store` |
| `/support` | サポート窓口案内 | `og:type=website` | FAQPage (必要に応じて) | |
| `/legal/{slug}` | 利用規約/特商法タイトル | `og:type=article` | Legislation (可能なら) | バージョン差分に `lastmod` |
| `/status` | システムステータス | `og:type=website` | ServiceStatus | RSS/Atom feed 生成 |

- canonical URL は言語に関係なく共通パス、`hreflang` で言語差別化。`?lang=en` は `rel=alternate` 指定。
- `SiteMap`：公開ページ（`/`, `/shop`, `/products/{id}`, `/templates`, `/guides`, `/content`, `/status`, `/support`, `/legal`）を対象。`/guides/{slug}` は 1 日 1 回更新。

## 4. 実装メタデータ表現
- ルート登録時に `RouteMeta` 構造体を付与：
  ```go
  type RouteMeta struct {
      Name        string // i18n キー
      Title       string // templ 用タイトルテンプレ
      Description string
      Canonical   string
      Robots      string
      JSONLD      func(vm any) template.HTML
  }
  ```
  - `page.Attach` で `RouteMeta` を登録し、`layout` から参照。Breadcrumb は `meta.Name` 連鎖で構築。
- `templ` コンポーネント `LayoutBase` は `Meta` slot に `<title>`, `<meta>`, `<link rel="canonical">`, `JSON-LD script` を描画。
- htmx フラグメント応答時はメタデータ更新を行わない。フルページ遷移時のみ `HX-Push-Url="true"` と合わせて更新。

## 5. 多言語対応
- `hreflang` サポート：`/guides` など多言語ページは `LocaleSwitcher` から `hx-get` で言語切替→`?lang=` 付き URL を生成。`canonical` は言語なしのベース URL。
- パンくず JSON-LD も `@language` に現在言語を指定。
- ナビゲーションラベル、メタデータ文言、パンくずテキストは `i18n` 辞書 (`doc/web/tailwind_components_guidelines.md` で触れた `hf-` クラスとは独立) に集約。

## 6. 次ステップ / 依存
- Task 012 で `CompBreadcrumb`, `CompNavPrimary` 実装時にこのマップを参照し、`RouteMeta` から階層を取得。
- Task 015 で SEO ヘルパを構築する際、本マップの表をテンプレ化し `<script type="application/ld+json">` を生成。
- Sitemap 生成は Task 007（ビルド/デプロイ）または別タスクで Cloud Build ステップに組み込む。
