# API

API は Cloud Run (Go) で実装します。

> 共通仕様（推奨）
>
> * **Base URL**：`https://{service-name}-{hash}-a.run.app/api/v1`
> * **認証**：
>
>   * ユーザー/スタッフ：`Authorization: Bearer <Firebase ID Token>`
>   * サーバ間/Webhook保護：**OIDC IAP**または**署名付きヘッダ（HMAC）**
> * **Idempotency**：`Idempotency-Key` ヘッダ（POST/PUT/PATCH/DELETE に必須）
> * **ページング**：`?pageSize=50&pageToken=...`（レスポンスに `nextPageToken`）
> * **並び替え/検索**：`?orderBy=createdAt desc&filter=field op value`（必要箇所）

---

# Public（認証不要）

* `GET /healthz` / `GET /readyz` … ライフチェック
* `GET /templates` / `GET /templates/{templateId}`
* `GET /fonts` / `GET /fonts/{fontId}`
* `GET /materials` / `GET /materials/{materialId}`
* `GET /products` / `GET /products/{productId}` … 形状/サイズ/素材でフィルタ可
* `GET /content/guides?lang=ja&category=culture` / `GET /content/guides/{slug}?lang=ja`
* `GET /content/pages/{slug}?lang=ja`
* `GET /promotions/{code}/public` … 公開可否/期間のみ返す（割引計算はユーザー API 内）

---

# Authenticated User（ユーザー）

## Profile / Account

* `GET /me` … `/users/{uid}` の取得
* `PUT /me` … 表示名・言語など（`role/isActive/piiMasked` は不可）
* `GET /me/addresses` / `POST /me/addresses` / `PUT /me/addresses/{addressId}` / `DELETE …`
* `GET /me/payment-methods` / `POST …` / `DELETE …`
  ※ 実カードは PSP 側。ここは参照 ID のみ。
* `GET /me/favorites` / `PUT /me/favorites/{designId}`（登録）/ `DELETE …`

## Designs（印影）

* `POST /designs` … 新規（typed/upload/logo）
* `GET /designs` / `GET /designs/{designId}`
* `PUT /designs/{designId}` / `DELETE /designs/{designId}`
* `GET /designs/{designId}/versions` / `GET /designs/{designId}/versions/{versionId}`
* `POST /designs/{designId}/duplicate`
* **AI 提案**

  * `POST /designs/{designId}/ai-suggestions` … 生成要求（balance/generateCandidates 等）
  * `GET /designs/{designId}/ai-suggestions` / `GET /designs/{designId}/ai-suggestions/{suggestionId}`
  * `POST /designs/{designId}/ai-suggestions/{suggestionId}:accept`
  * `POST /designs/{designId}/ai-suggestions/{suggestionId}:reject`
  * `POST /designs/{designId}:registrability-check` … 実印/銀行印チェック

## Name Mapping（外国人の漢字変換）

* `POST /name-mappings:convert` … `{ latin:"Matsumoto", locale:"en" }` → 候補一覧
* `POST /name-mappings/{mappingId}:select` … 採用候補確定

## Cart / Checkout

* `GET /cart`（ヘッダ） / `PATCH /cart`（通貨・プロモ適用等）
* `GET /cart/items` / `POST /cart/items` / `PUT /cart/items/{itemId}` / `DELETE …`
* `POST /cart:estimate` … 税・送料・割引見積
* `POST /cart:apply-promo` / `DELETE /cart:remove-promo`
* `POST /checkout/session` … PSP セッション作成（Stripe）
* `POST /checkout/confirm` … クライアント側で完了通知（最終確定は Webhook）

## Orders / Payments / Shipments

* `GET /orders` / `GET /orders/{orderId}`
* `POST /orders/{orderId}:cancel` … 出荷前のみ
* `POST /orders/{orderId}:request-invoice` … 領収書発行要求（後続で PDF 生成）
* `GET /orders/{orderId}/payments`
* `GET /orders/{orderId}/shipments` / `GET /orders/{orderId}/shipments/{shipmentId}`
* `GET /orders/{orderId}/production-events`
* `POST /orders/{orderId}:reorder` … designSnapshot から再注文作成

## Reviews

* `POST /reviews` … 注文に対するレビュー作成
* `GET /reviews?orderId=...`（自分の分）

## Assets（安全な入出力）

* `POST /assets:signed-upload` … 署名アップロード URL 発行（kind/purpose 指定）
* `POST /assets/{assetId}:signed-download` … 一時ダウンロード URL

---

# Admin / Staff（管理・運用）

## カタログ・CMS

* `POST /templates` / `PUT /templates/{id}` / `DELETE …`
* `POST /fonts` / `PUT /fonts/{id}` / `DELETE …`
* `POST /materials` / `PUT /materials/{id}` / `DELETE …`
* `POST /products` / `PUT /products/{id}` / `DELETE …`
* `POST /content/guides` / `PUT /content/guides/{id}` / `DELETE …`
* `POST /content/pages` / `PUT /content/pages/{id}` / `DELETE …`

## プロモーション

* `GET /promotions` / `POST /promotions` / `PUT /promotions/{promoId}` / `DELETE …`
* `GET /promotions/{promoId}/usages`（ユーザー別）
* `POST /promotions:validate` … 管理用に条件検証（カート不要）

## 受注・決済・在庫

* `GET /orders?status=in_production&since=...`（運用向け絞込）
* `PUT /orders/{orderId}:status` … `paid → in_production → shipped …`
* `POST /orders/{orderId}/payments:manual-capture` / `…:refund`
* `POST /orders/{orderId}/shipments` … ラベル生成（キャリア連携）
* `PUT /orders/{orderId}/shipments/{shipmentId}` … 追跡ステータス訂正
* `POST /orders/{orderId}/production-events` … 工程イベント追加
* `GET /stock/low` … 安全在庫割れの一覧
* `POST /stock/reservations:release-expired` … 期限切れ解放（手動キック）

## 制作キュー

* `GET /production-queues` / `POST /production-queues` / `PUT /production-queues/{queueId}` / `DELETE …`
* `GET /production-queues/{queueId}/wip` … 進捗サマリ
* `POST /production-queues/{queueId}:assign-order` … 受注の割当

## ユーザー・レビュー・監査

* `GET /users?query=...` / `GET /users/{uid}`
* `POST /users/{uid}:deactivate-and-mask` … 退会＋PIIマスク
* `GET /reviews?moderation=pending` / `PUT /reviews/{id}:moderate`（approve/reject）
* `POST /reviews/{id}:store-reply`
* `GET /audit-logs?targetRef=/orders/{id}`

## 運用ユーティリティ

* `POST /invoices:issue` … 請求書番号採番＋PDF 生成バッチ（`orderId` or 配列）
* `POST /counters/{name}:next` … 連番採番（site/currency 等の scope 可）
* `POST /exports:bigquery-sync`（必要なら）
* `GET /system/errors` / `GET /system/tasks`（失敗ジョブ可視化）

---

# Webhooks（外部→自社 Cloud Run）

> すべて **署名検証** 必須（例：Stripe-Signature）。**/webhooks/** は公開だが FW/IP 制限 & OIDC/IAP で二重防御推奨。

* `POST /webhooks/payments/stripe` … 支払い成功/失敗/返金

  * 主要イベント：`payment_intent.succeeded|payment_intent.payment_failed|charge.refunded`
  * 処理：`/orders/{id}` を `paid`、`refund` セクション更新、プロモ usage 増分、予約在庫 commit など
* `POST /webhooks/payments/paypal` … 返金含む
* `POST /webhooks/shipping/{carrier}` … `dhl|jp-post|yamato|ups|fedex`

  * 処理：`/orders/{id}/shipments/{shipmentId}` の `events[]` 追記 → `delivered` 反映
* `POST /webhooks/ai/worker` … キューワーカー（Pull/Push どちらでも）。`aiJobs` 実行・`aiSuggestions` 生成

---

# Internal（サーバ間・Scheduler 起動用）

> **IAP/OIDC** トークン必須。クライアントからは叩かない。

* `POST /internal/checkout/reserve-stock` … カートから `/stockReservations` 作成＋ `products.stockQuantity--`（Tx）
* `POST /internal/checkout/commit` … 決済成功後、予約を `committed` に／`orders.status=paid`
* `POST /internal/checkout/release` … 決済失敗/タイムアウトで在庫戻し
* `POST /internal/promotions/apply` … `usageCount`/`limitPerUser` のアトミック検証＋増分
* `POST /internal/invoices/issue-one` … `/counters/{counterId}` を Tx で進め、PDF 生成 → `/orders.invoice` 更新
* `POST /internal/maintenance/cleanup-reservations` … `expiresAt < now && status=reserved` の解放
* `POST /internal/maintenance/stock-safety-notify` … 安全在庫割れ通知
* `POST /internal/audit-log` … 任意イベントの追記（他エンドポイントからも都度書込）

---

# 代表リクエスト/レスポンス（抜粋）

**POST /designs/{id}/ai-suggestions**（生成要求）

```json
// req
{ "method": "balance", "model": "glyph-balancer@2025-09" }
// res
{ "suggestionId": "s_abc", "status": "queued" }
```

**POST /cart:estimate**

```json
// res
{
  "currency": "JPY",
  "subtotal": 7800, "discount": 500, "tax": 780, "shipping": 900, "total": 8980,
  "promotion": { "code": "SAKURA10", "applied": true }
}
```

**POST /webhooks/payments/stripe**

```json
// res
{ "ok": true, "orderId": "o_123", "newStatus": "paid" }
```

**POST /internal/checkout/reserve-stock**

```json
// req
{ "orderId":"o_tmp123", "userRef":"/users/u_1", "lines":[{"productRef":"/products/p1","sku":"BWB-R15","qty":1}], "ttlSec": 900 }
// res
{ "reservationId":"o_tmp123", "status":"reserved", "expiresAt":"2025-10-04T12:00:00Z" }
```

---

# ルーティング実装メモ（Go／Cloud Run）

* ルータ：`chi` / `gorilla/mux` / `echo` など。`/api/v1` にグループ化。
* **ミドルウェア**：

  * Firebase ID Token 検証（`Authorization`）
  * RBAC（`role=staff|admin` クレーム）
  * Idempotency-Key の重複検出（Firestore/Redis にキー保存）
  * Request Logging（構造化）／Tracing（Cloud Trace）
  * 署名検証（Stripe/Webhook HMAC）
* **バックグラウンド**：

  * **Cloud Scheduler → Cloud Run (Jobs/Service)** で `cleanup-reservations`, `stock-safety-notify` 実行
  * AI ワーカーは Pull 方式なら **Pub/Sub** 経由 `/internal` で起動

---

# エンドポイント一覧（タグ別・サマリ）

**Public**：healthz, templates, fonts, materials, products, guides, pages, promotions/public
**User**：me(+addresses/payments/favorites), designs(+versions/ai-suggestions), name-mappings, cart(+items/promo/estimate/checkout), orders(+cancel/request-invoice/reorder, payments, shipments, production-events), reviews, assets(signed-upload/download)
**Admin**：catalog/templates/fonts/materials/products, CMS/guides/pages, promotions(+usages/validate), orders ops(+status/payments/shipments/production), stock, production-queues, users ops(+deactivate), reviews moderation, audit-logs, invoices, counters, exports, system
**Webhooks**：payments(stripe/paypal), shipping(carrier), ai/worker
**Internal**：checkout(reserve/commit/release), promotions/apply, invoices/issue-one, maintenance(cleanup…), audit-log

---
