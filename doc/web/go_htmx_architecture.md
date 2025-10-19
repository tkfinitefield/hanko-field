# Go + htmx アーキテクチャ指針（Task 002）

## 技術スタック概要
- **言語/フレームワーク**：Go 1.23、`github.com/go-chi/chi/v5`（ルータ）、`github.com/a-h/templ`（テンプレートコンパイラ）、htmx、Tailwind CSS、Alpine/vanilla JS ヘルパ最低限。
- **ビルド**：`templ generate` による Go コード生成、esbuild（JS）＋ Tailwind CLI（CSS）。Cloud Run デプロイ前提で 12Factor な環境変数管理を想定。
- **レンダリング方針**：初回 SSR、`hx-get`/`hx-post` による部分更新（FRAG/MODAL）。レイテンシ最小化のため SSR レイヤを単一プロセスで完結させる。

## ディレクトリ / パッケージ規約
```
web/
 ├── cmd/web/                # アプリケーションエントリ（main.go）
 ├── internal/
 │   ├── app/                # 設定ロード、依存解決、サーバの起動
 │   ├── server/             # ルータ構築、ミドルウェア束ねる
 │   │   ├── middleware/     # auth/csrf/logging 等（Task 008）
 │   │   └── router.go
 │   ├── handler/
 │   │   ├── page/           # SSR フルページ (`/`, `/shop` ...)
 │   │   ├── frag/           # htmx フラグメント (`/frags/...`)
 │   │   ├── modal/          # モーダル (`/modal/...`)
 │   │   └── asset/          # 静的ファイル署名URL/manifest API
 │   ├── ui/
 │   │   ├── layouts/        # 基本レイアウト templ（`base.templ`）
 │   │   ├── pages/          # ページ用 templ (`home.templ`)
 │   │   ├── frags/          # 部品 templ (`shop_table.templ`)
 │   │   ├── modals/         # モーダル templ
 │   │   └── components/     # 再利用コンポーネント（`Button.templ` 等）
 │   ├── viewmodel/          # DTO/変換ロジック（UI 表示用）
 │   └── util/               # htmx ヘルパ、レスポンスユーティリティ
 ├── assets/
 │   ├── tailwind.config.js
 │   ├── postcss.config.js
 │   ├── css/
 │   └── js/
 └── tools/                  # go:generate、lint/formatスクリプト
```
- `internal/ui/...` 配下は `.templ` ファイルと生成された `.go` をペアで管理。`go:generate templ generate` を `internal/ui` 直下で実行。
- FRAG/モーダル等の Path はディレクトリとの 1:1 対応を原則とし、保守性を高める。

## ルーティング方針（chi）
```go
func NewRouter(deps *app.Dependencies) chi.Router {
    r := chi.NewRouter()
    r.Use(middleware.Recoverer, telemetry.HTTP, security.Headers)

    r.Route("/", func(r chi.Router) {
        r.Use(session.LoadOptional, auth.InjectUser)
        page.Attach(r, deps)        // FP
    })

    r.Route("/frags", func(r chi.Router) {
        r.Use(hx.MustHXRequest, cache.ETagAware)
        frag.Attach(r, deps)        // FRAG
    })

    r.Route("/modal", func(r chi.Router) {
        r.Use(hx.MustHXRequest, security.CSRF)
        modal.Attach(r, deps)       // MODAL
    })

    r.Mount("/assets", asset.Router(deps.Static))
    r.NotFound(page.NotFound)

    return r
}
```
- **FP（Full Page）**：SSR + BaseLayout。`page.Attach` 内で `r.Get("/", Home)` 等を定義。
- **FRAG**：`/frags/resource/action`。`Accept`ヘッダで `text/html` を強制し、`hx-request` チェックで直接アクセス時には 404。
- **MODAL**：`/modal/{topic}`。`hx-target="#modal"` 前提。CSRF トークンを `HX-Request` 毎に埋め込む。
- **API**：アプリケーション API は `api/` サービスで提供済み。Web では UI 補助 API（署名URL 等）のみに限定。
- **ミドルウェア順序**：`Recoverer` → `RequestID` → `Logger` → `RealIP` → `SecurityHeaders` → 認証/セッション → ドメイン固有。

## テンプレート命名規約（templ）
- **レイアウト**：`layouts/base.templ` → `LayoutBase`. 子レイアウトは `LayoutAccount` 等。
- **ページ**：`pages/{section}/{name}.templ`。コンポーネント名は `Page{Section}{Name}`（例：`PageShopIndex`）。`LayoutBase` を埋め込み、`Title`, `Meta`, `Body` を slot として受ける。
- **フラグメント**：`frags/{namespace}/{component}.templ` → `Frag{Namespace}{Component}`。`<tbody>` 等のルート要素で返し、`hx-target` で明確にスコープ。
- **モーダル**：`modals/{namespace}/{name}.templ` → `Modal{Namespace}{Name}`。`ModalFrame` コンポーネントに `Title`, `Body`, `Actions` を slot で流す。
- **コンポーネント**：`components/{kind}/{name}.templ` → `Comp{Kind}{Name}`。小文字/スネークケースは避け、再利用を明確化。
- **ViewModel**：`internal/viewmodel` で `type ShopTable struct` など UI 専用構造体を定義。handler でドメインオブジェクトから変換。
- **国際化**：`templ` に `{{ T "key" .Locale }}` を埋め込むヘルパを配置。`viewmodel` で `Locale string` を必ずセット。

## htmx 運用ルール
- **エンドポイント**：`/frags/**` と `/modal/**` は htmx 専用。`HX-Request != true` の場合は 404 または `HX-Redirect` を返し誤用防止。
- **ターゲット命名**：`id="shop-table"` → `/frags/shop/table` のように ID と Path を揃える。触る範囲は最小 DOM。
- **トリガ**：フォームは `hx-trigger="change delay:250ms"`、ページャは `hx-get`. 長時間処理は `hx-trigger="revealed"` または `hx-post` + `hx-indicator`.
- **スワップ**：リスト系は `hx-swap="outerHTML"`, モーダルは `innerHTML`, トーストは `afterbegin`.
- **エラーハンドリング**：`hx-on::afterRequest` で 401/419 → `window.location = "/signin"`。`HX-Retarget` を使いエラー領域更新。`util/hx` パッケージで共通実装。
- **プログレッシブエンハンス**：`hx-boost="true"` を `<body>` に設定し、JS オフライン時でも `<a>` `<form>` は SSR で動作。

## アセット / キャッシュ戦略
- **Tailwind**：`assets/css/app.css` → `tailwind.config.js`（プレフィックス `hf-`）。`npm run dev` で `--watch`、`npm run build` で minify + purge。
- **JS**：`assets/js` は最小限（htmx, Alpine optional, util scripts）。esbuild でバンドルし `static/dist/app.js` へ。
- **静的配信**：ビルド済み CSS/JS は Cloud Storage へアップロードし `Cache-Control: public, max-age=31536000` + ハッシュファイル名。HTML/FRAG は `no-store`。
- **署名URL**：画像等は API サービスの署名 URL を用いる。テンプレからは `data-src` で遅延ロード。
- **テンプレキャッシュ**：templ はコンパイル済なのでランタイムキャッシュ不要。`etag` ミドルウェアで FRAG 応答の差分検出（`HX-Reswap:none` と組み合わせて最適化）。

## データフロー & レンダリング
1. handler がドメインサービス（別プロジェクト or API クライアント）を呼び出し。
2. `viewmodel` で UI 表現に整形し、`templ` コンポーネントへ渡す。
3. レスポンスユーティリティが `Content-Type` 固定（ページ: `text/html; charset=utf-8`, JSON: `application/json`）。
4. htmx 応答時に `HX-Trigger` でトーストイベント、`HX-Redirect` で遷移を制御。

## 追加ガイドライン
- **テスト**：`internal/handler/**` ごとに `*_test.go` を配置し、`httptest` + `github.com/PuerkitoBio/goquery` で DOM アサート。FRAG は Task 010 で拡張。
- **ログ/トレーシング**：`util/log` で構造化ログ。htmx の `HX-Request` ヘッダをログに含め、プロファイルしやすくする。
- **エラー表示**：`PageError` / `FragError` コンポーネントを統一し、ユーザー向け・開発者向けメッセージを分離。
- **アクセシビリティ**：モーダルは `aria-modal="true"` + 焦点トラップ js。フォームバリデーションは FRAG でエラーメッセージ領域に `role="alert"`.

この指針を基に、Task 006（プロジェクトスキャフォールド）で実装開始し、Task 008/010/014 等でミドルウェア・モーダル・テスト戦略を具体化する。***
