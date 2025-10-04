**Go + htmx**（SSR＋部分更新）前提で、管理画面（Admin Console）の**完全な画面一覧**を、

* 画面URL（人が開くフルページ）
* 部分描画用のフラグメントURL（`hx-get/hx-post`で差し替え）
* 主アクション（並べ替え・フィルタ・一括操作）
* ひも付くAPI（先にお渡しした OpenAPI v1 の該当エンドポイント）
  まで紐づけて提示します。

> UIフレームの原則（共通）
>
> * レイアウト：`/admin/_layout.html`（サイドバー＋トップバー＋`<main id="content">`）
> * 各一覧テーブルは **SSR**（初回）→ **htmxで tbody 差し替え**
> * 検索・フィルタは `<form hx-get=".../table" hx-target="#table-body" hx-push-url="true">`
> * モーダル：`<div id="modal" hx-target="this" class="hidden">` に `hx-get="/admin/.../modal/..."`
> * CSRF：`<meta name="csrf-token">`＋`hx-headers='{"X-CSRF-Token":"{{.CSRF}}"}'`
> * RBAC：`staff/admin` でサイドバー表示制御。
> * i18n：テキストはテンプレート関数経由。

---

# 0. フレーム / 共通ユーティリティ

* **/admin/login**（ログイン）
* **/admin**（ダッシュボード）

  * フラグメント：`/admin/fragments/kpi`, `/admin/fragments/alerts`
* **/admin/search**（横断検索：注文/ユーザー/レビュー等）

  * フラグメント：`/admin/search/table`（結果テーブル）
* **/admin/notifications**（失敗ジョブ/在庫警告/配送例外）

  * フラグメント：`/admin/notifications/table`
* **/admin/profile**（自分の2FA/APIキー）

---

# 1. 受注・出荷（オペレーション中核）

## 注文一覧 / 詳細

* 画面：`/admin/orders`

  * フラグメント：`/admin/orders/table`（絞込・ソート・ページング）
  * フィルタ：`status,since,currency,amountMin,max,hasRefund`
  * 一括：**ステータス遷移**, **出荷ラベル生成**, **CSV出力**
  * API：`GET /admin/orders`
* 詳細：`/admin/orders/{orderId}`

  * タブ：`/admin/orders/{id}/tab/summary|lines|payments|production|shipments|invoice|audit`
  * アクション（モーダル）

    * ステータス更新：`/admin/orders/{id}/modal/status` → `PUT /admin/orders/{id}:status`
    * 返金：`/admin/orders/{id}/modal/refund` → `POST /orders/{id}/payments:refund`
    * 領収書発行：`/admin/orders/{id}/modal/invoice` → `POST /admin/invoices:issue`
  * API：`GET /orders/{id}`, `GET /orders/{id}/payments`, `GET /orders/{id}/shipments`, `GET /orders/{id}/production-events`

## 出荷（バッチ / 追跡）

* 出荷バッチ：`/admin/shipments/batches`（任意。なければ注文詳細から発行）

  * ラベル生成：`/admin/orders/{id}/shipments`（POST）
* 追跡モニタ：`/admin/shipments/tracking`

  * フラグメント：`/admin/shipments/tracking/table`
  * API：キャリアWebhookで更新（閲覧は Firestore 経由の集計 or `GET /orders/{id}/shipments`）

## 制作（工房）

* カンバン：`/admin/production/queues`

  * フラグメント：`/admin/production/queues/board`（各列 `queued|engraving|polishing|qc|packed`）
  * D&D更新：`hx-post="/admin/orders/{id}/production-events"`（`{type:"engraving"}` 等）
  * API：`POST /admin/orders/{id}/production-events`
* 作業指示書：`/admin/production/workorders/{orderId}`
* QC：`/admin/production/qc`（`qc.pass/fail` 送信）

---

# 2. カタログ（商品・素材・テンプレ）

* 一覧：`/admin/catalog/templates|fonts|materials|products`

  * テーブル：`/admin/catalog/{kind}/table`
  * 新規：`/admin/catalog/{kind}/modal/new` → `POST /admin/catalog/{kind}`
  * 編集：`/admin/catalog/{kind}/{id}/modal/edit` → `PUT /admin/catalog/{kind}/{id}`
  * 削除：`/admin/catalog/{kind}/{id}/modal/delete` → `DELETE ...`
  * API：OpenAPI の `admin/catalog/*` 系

---

# 3. CMS（ガイド・固定ページ）

* ガイド：`/admin/content/guides`（一覧/公開フラグ/予約公開）

  * プレビュー：`/admin/content/guides/{id}/preview?lang=ja`
  * 編集：モーダル or 2ペイン編集（左フォーム、右ライブプレビュー）
  * API：`POST/PUT/DELETE /admin/content/guides*`
* 固定ページ：`/admin/content/pages`（ブロック編集UI）

---

# 4. プロモーション

* クーポン一覧：`/admin/promotions`

  * テーブル：`/admin/promotions/table`
  * 新規/編集モーダル：`/admin/promotions/modal/{new|edit}`
  * 利用状況：`/admin/promotions/{promoId}/usages`（ユーザー別）
  * API：`/admin/promotions*`, `GET /admin/promotions/{id}/usages`

---

# 5. 顧客 / レビュー / KYC（任意）

* 顧客一覧：`/admin/customers` → 詳細：`/admin/customers/{uid}`（注文・住所・支払い）

  * 「退会＋PIIマスク」：モーダル → `POST /users/{uid}:deactivate-and-mask`
* レビュー審査：`/admin/reviews?moderation=pending`

  * 承認/却下：`PUT /admin/reviews/{id}:moderate`
  * 店舗返信：`POST /admin/reviews/{id}:store-reply`

---

# 6. 制作キュー / 組織 / 権限

* 制作キュー設定：`/admin/production-queues`

  * 新規/編集：モーダル → `POST/PUT /admin/production-queues*`
* スタッフ・ロール：`/admin/org/staff`, ` /admin/org/roles`（RBAC UI）

  * （APIは今後追加でもOK。先行はFirebase Console管理でも可）

---

# 7. 決済・会計・対帳

* 取引一覧：`/admin/payments/transactions`（PSP検索）

  * 手動キャプチャ：`POST /orders/{id}/payments:manual-capture`
  * 返金：`POST /orders/{id}/payments:refund`
* 税設定：`/admin/finance/taxes`（任意）

---

# 8. ログ / 監査 / システム

* 監査ログ：`/admin/audit-logs?targetRef=...`

  * テーブル：`/admin/audit-logs/table`（差分を折りたたみで表示）
* エラーログ：`/admin/system/errors`（Functions/Run/Webhook失敗）
* ジョブ/タスク：`/admin/system/tasks`（清掃/在庫予約解放など）
* 連番カウンタ：`/admin/system/counters` → `POST /admin/counters/{name}:next` テストUI

---

## サイドバー（推奨構成）

```
ダッシュボード
受注管理
  ├ 注文一覧
  ├ 出荷追跡
  └ 制作カンバン
カタログ
  ├ SKU
  ├ 素材
  ├ テンプレ
  └ フォント
コンテンツ
  ├ ガイド
  └ 固定ページ
マーケ
  ├ プロモーション
  └ レビュー審査
顧客
  └ 顧客一覧
システム
  ├ 監査ログ
  ├ カウンタ
  ├ タスク/ジョブ
  └ 設定
```

---

## htmx フラグメント設計（代表例）

### 1) 一覧テーブル（注文）

* **フルページ**：`GET /admin/orders`

  * `{{ template "orders_index.html" . }}`
* **tbody差し替え**：`GET /admin/orders/table?status=paid&page=2`

  ```html
  <form id="order-filter" hx-get="/admin/orders/table" hx-target="#orders-tbody" hx-push-url="true" class="flex gap-2">
    <select name="status"><option value="">All</option>...</select>
    <input type="datetime-local" name="since">
    <button class="btn">適用</button>
  </form>

  <table class="tbl">
    <thead>…</thead>
    <tbody id="orders-tbody">
      {{/* ここに /admin/orders/table の partial を挿入 */}}
    </tbody>
  </table>
  <nav id="pager" hx-get="/admin/orders/table?page={{.Next}}" hx-target="#orders-tbody" class="pager">次へ</nav>
  ```

### 2) モーダル（注文ステータス）

* 起動：`<button hx-get="/admin/orders/{{.ID}}/modal/status" hx-target="#modal" hx-trigger="click">変更</button>`
* フォーム送信：

  ```html
  <form hx-put="/admin/orders/{{.ID}}:status"
        hx-headers='{"X-CSRF-Token":"{{.CSRF}}"}'
        hx-target="#order-status-cell"
        hx-swap="outerHTML">
    <select name="status">
      <option value="in_production">制作へ</option>…
    </select>
    <textarea name="note"></textarea>
    <button class="btn-primary">更新</button>
  </form>
  ```

### 3) カンバンD&D（簡易）

* カード：`<div class="card" draggable="true" hx-post="/admin/orders/{{.ID}}/production-events" hx-vals='{"type":"engraving"}'></div>`

---

## 画面 ↔ API 対応早見表（主要）

| 画面                 | 主フラグメント                             | 主API                                                 |
| ------------------ | ----------------------------------- | ---------------------------------------------------- |
| /admin/orders      | /admin/orders/table                 | GET /admin/orders                                    |
| /admin/orders/{id} | /admin/orders/{id}/tab/*            | GET /orders/{id}, GET sub-resources                  |
| ステータス更新モーダル        | /admin/orders/{id}/modal/status     | PUT /admin/orders/{id}:status                        |
| 出荷ラベル作成            | /admin/orders/{id}/shipments (POST) | POST /admin/orders/{id}/shipments                    |
| 制作イベント追加           | —                                   | POST /admin/orders/{id}/production-events            |
| カタログ（SKU等）         | /admin/catalog/{kind}/table         | POST/PUT/DELETE /admin/catalog/*                     |
| プロモ一覧              | /admin/promotions/table             | GET/POST/PUT/DELETE /admin/promotions*               |
| プロモ利用              | —                                   | GET /admin/promotions/{id}/usages                    |
| レビュー審査             | /admin/reviews/table                | GET /admin/reviews, PUT /admin/reviews/{id}:moderate |
| ガイド/ページ            | /admin/content/*/table              | POST/PUT/DELETE /admin/content/*                     |
| 監査ログ               | /admin/audit-logs/table             | GET（内部実装 or Firestore読み）                             |
| カウンタ               | /admin/system/counters/table        | POST /admin/counters/{name}:next                     |

---

## ディレクトリ構成（例・Go `html/template`）

```
/web
  /layouts/_base.html
  /layouts/_modal.html
  /partials/table_*.html
  /admin
    dashboard.html
    orders_index.html
    orders_show.html
    orders_modal_status.html
    shipments_tracking.html
    production_board.html
    catalog_{templates,fonts,materials,products}.html
    content_{guides,pages}.html
    promotions_index.html
    reviews_index.html
    customers_index.html
    audit_logs.html
    counters.html
```

---

## キーボード & UX（推奨）

* `/` で全体検索フォーカス、`f`でフィルタ、`j/k`で行移動、`o`で詳細、`g`でタブ切替
* モーダルは `Esc` で閉じる。`hx-trigger="keyup[key=='Escape']"`
* 長処理は `hx-indicator` とトーストで可視化。

---

## セキュリティ

* 全POST系は `X-CSRF-Token` 必須（テンプレに埋め込み）
* `Authorization: Bearer`（Firebase ID Token）検証→RBAC中間層
* Webhookは `/webhooks/*` で HMAC 検証＋IP制限（Ingress）
