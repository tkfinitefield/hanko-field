# ウェブ

ウェブは Go + htmx で開発します。
Cloud Run で実行します。

---

# 1. 情報設計（サイトマップ / URL 体系）

## 1.1 上位ナビ（公開）

* `/` トップ/ランディング
* `/shop` カテゴリ一覧（素材×形×サイズフィルタ）
* `/products/{productId}` SKU詳細
* `/templates` / `/templates/{templateId}`
* `/guides` / `/guides/{slug}`
* `/content/{slug}` 固定ページ（特商法/規約/プライバシー 等）
* `/status` ステータス

## 1.2 作成〜AI〜プレビュー

* `/design/new` 作成タイプ選択（文字入力/画像/ロゴ）
* `/design/editor` **2ペイン**（フォーム＋ライブプレビュー）
* `/design/ai` AI候補ギャラリー
* `/design/preview` 実寸/朱肉モック
* `/design/versions` バージョン履歴

## 1.3 ショッピング

* `/cart` カート
* `/checkout/address` 配送先/請求先
* `/checkout/shipping` 配送方法
* `/checkout/payment` 支払
* `/checkout/review` 確認
* `/checkout/complete` 完了

## 1.4 アカウント

* `/account` プロフィール
* `/account/addresses` 住所帳
* `/account/orders` / `/account/orders/{orderId}`
* `/account/library` マイ印鑑（デザイン一覧）
* `/account/security` 連携/2FA

## 1.5 サポート/法務

* `/support` 問い合わせ
* `/legal/{slug}` 規約類

> すべて **SSR（初回）＋ htmx 部分差し替え** を基本とします。

---

# 2. 画面仕様（フルページ / フラグメント / モーダル）

> 記法：
>
> * **FP**: フルページ（SSR）
> * **FRAG**: フラグメント（`hx-get`/`hx-post`で差し替え）
> * **MODAL**: モーダル（`#modal`に差し込み）
> * **API**: OpenAPI の該当エンドポイント

---

## 2.1 ランディング / 探索

### `/` トップ（FP）

* ヒーロー（CTA：印影を作る / テンプレを探す）
* 比較表（素材×サイズ×納期/価格） → **FRAG** `/frags/compare/sku-table?shape=&sizeMm=`
* ガイドの新着 → **FRAG** `/frags/guides/latest?lang=ja`
* SEO：OGP/構造化 Product & Article（静的テンプレに注入）

### `/shop`（FP）

* フィルタフォーム（素材/形/サイズ/価格レンジ/セール有無）
* 結果テーブル → **FRAG** `/shop/table?…`（tbody差し替え）
* **API**：`GET /products`（絞込）

### `/products/{productId}`（FP）

* 商品ヘッダ（素材/サイズ/形/価格/セール/納期）
* 画像ギャラリー（サムネ→メイン画像 **FRAG** 差し替え）
* レビュー抜粋 **FRAG** `/frags/reviews/snippets?productId=…`
* 「カートに追加」フォーム（数量/オプション）
* **API**：`GET /products/{id}`、カート追加は `POST /cart/items`

### `/templates` / `/templates/{id}`（FP）

* 一覧フィルタ（書体/形/登録可否）→ **FRAG** `/templates/table`
* 詳細：推奨サイズ/制約/プレビュー
* **API**：`GET /templates`, `GET /templates/{id}`

### `/guides` / `/guides/{slug}`（FP）

* 記事カード一覧（カテゴリ/言語フィルタ）→ **FRAG** `/guides/table`
* 記事本文は目次・関連記事・OGP 最適化
* **API**：`GET /content/guides`, `GET /content/guides/{slug}`

---

## 2.2 デザイン作成〜AI

### `/design/new`（FP）

* 3 カード：文字入力/画像アップ/ロゴ刻印
* 次へ：`/design/editor`

### `/design/editor`（FP ＝ 2ペイン）

* 左：フォーム（名前/漢字変換/書体/テンプレ/線の太さ/余白/格子/サイズ）

  * **FRAG** `/design/editor/form`（フォーム本体）
  * **MODAL** `/modal/pick/font`, `/modal/pick/template`, `/modal/kanji-map`
* 右：ライブプレビュー領域（SVG/PNG）

  * **FRAG** `/design/editor/preview?stateHash=…`（`hx-trigger="change from:#editor-form delay:250ms"`）
* アクション：保存（下書き）/AI修正/プレビュー/バージョン履歴
* **API**：`POST/PUT /designs`, `POST /designs/{id}:registrability-check`

### `/design/ai`（FP）

* グリッド：AI候補カード（score/タグ/差分）

  * **FRAG** `/design/ai/table?designId=…`
  * 受入ボタン → **POST** `/designs/{id}/ai-suggestions/{sid}:accept`（受入後 プレビュー差し替え）
* **API**：`POST /designs/{id}/ai-suggestions`（生成要求）, `GET /designs/{id}/ai-suggestions`

### `/design/preview`（FP）

* 実寸プレビュー（mmスケール/和紙背景/朱肉モック）
* **FRAG** `/design/preview/image?designId=…&bg=washi&dpi=…`
* ダウンロード（PNG/SVG 署名URL）
* **API**：`POST /assets:{signed-download}`（一時URL発行）

### `/design/versions`（FP）

* バージョン一覧 → **FRAG** `/design/versions/table?designId=…`
* ロールバック **MODAL** `/modal/design/version/rollback?v=…`
* **API**：`GET /designs/{id}/versions`

---

## 2.3 カート〜チェックアウト

### `/cart`（FP）

* 明細テーブル → **FRAG** `/cart/table`
* クーポン適用 **MODAL** `/modal/cart/promo` → `POST /cart:apply-promo`
* 見積リフレッシュ → **FRAG** `/cart/estimate`（合計/税/送料/割引）
* **API**：`GET /cart`, `GET/POST/PUT/DELETE /cart/items`, `POST /cart:estimate`

### `/checkout/address`（FP）

* 配送先/請求先フォーム（会社名・部署・領収書宛名対応）
* 住所帳から選択 **FRAG** `/account/addresses/table?select=1`
* **API**：`GET/POST /me/addresses`

### `/checkout/shipping`（FP）

* 方法比較（国内/国際/ETA/費用） → **FRAG** `/checkout/shipping/table?country=…&weight=…`
* **API**：内部見積 or `POST /cart:estimate` の拡張

### `/checkout/payment`（FP）

* PSP セッションボタン（Stripe/PayPal）→ `POST /checkout/session`
* 完了後 `POST /checkout/confirm`（表示は `/checkout/complete` へ）

### `/checkout/review`（FP）

* 注文最終確認（デザイン**スナップショット**表示）
* 「注文する」→ PSP へ（or 直接 `POST /orders` → 決済遷移）

### `/checkout/complete`（FP）

* 注文番号/次アクション（追跡/領収書）

---

## 2.4 アカウント

### `/account`（FP）

* プロフィール編集（表示名/言語/国） → **FRAG** `/account/profile/form`
* **API**：`GET/PUT /me`

### `/account/addresses`（FP）

* 一覧 → **FRAG** `/account/addresses/table`
* 新規/編集 → **MODAL** `/modal/address/edit` → `POST/PUT /me/addresses`

### `/account/orders`（FP）

* 注文一覧 → **FRAG** `/account/orders/table?status=&from=&to=`
* **API**：`GET /orders`

### `/account/orders/{orderId}`（FP）

* タブ：概要/明細/支払い/制作/配送/領収書

  * それぞれ **FRAG** `/account/orders/{id}/tab/{name}`
* キャンセル **MODAL** `/modal/order/cancel` → `POST /orders/{id}:cancel`
* 領収書リク **MODAL** `/modal/order/invoice` → `POST /orders/{id}:request-invoice`
* **API**：`GET /orders/{id}`, `GET /orders/{id}/payments|shipments|production-events`

### `/account/library`（FP）

* デザイン一覧（AIスコア/登録可否/再注文ボタン）

  * **FRAG** `/account/library/table`
* **API**：`GET /designs`

---

## 2.5 サポート/法務

* `/support`（FP）：問い合わせフォーム **FRAG** `/support/form`
* `/legal/{slug}`（FP）：規約本文（静的/Headless CMS 併用可）

---

# 3. フラグメント/モーダル（代表仕様）

## 3.1 一覧テーブル（共通）

* **URL**：`/frags/{resource}/table`
* **入出力**：クエリでフィルタ/ソート/ページ、HTML（`<tbody>`）を返す
* **htmx**：

  ```html
  <form hx-get="/shop/table" hx-target="#sku-tbody" hx-push-url="true">
    <select name="material">…</select>
    <select name="shape">…</select>
    <input type="number" name="sizeMm" min="6" max="30">
    <button>適用</button>
  </form>
  <table>
    <thead>…</thead>
    <tbody id="sku-tbody"><!-- サーバが差し込む --></tbody>
  </table>
  <nav hx-get="/shop/table?page={{.Next}}" hx-target="#sku-tbody" class="pager">次へ</nav>
  ```

## 3.2 モーダル（共通）

* **構造**：`<div id="modal" hx-target="this" hx-swap="innerHTML">`
* **起動**：`hx-get="/modal/{kind}"`、**送信**：`hx-post|put`
* **CSRF**：`hx-headers='{"X-CSRF-Token":"{{.CSRF}}"}'`

---

# 4. 主要フォーム定義（抜粋）

## 4.1 デザインエディタ（左ペイン）

* 項目：

  * 名前（rawName）・漢字変換（外国人向け：候補リスト）
  * 形（丸/角）、サイズmm
  * 書体（tensho/reisho/…）・フォント・テンプレ
  * 線の太さ（0.5–1.5）・余白（0–0.3）・格子/中央寄せ
* バリデーション：サイズ（6–30mm）、禁則（テンプレ.constraints）
* 送信先：`PUT /designs/{id}`（保存）、`POST /designs/{id}:registrability-check`

## 4.2 チェックアウト

* `/checkout/address`：配送先・請求先（会社/部署/領収書宛名）
* `/checkout/shipping`：選択（国内/国際、費用/ETA）
* `/checkout/payment`：PSP セッション作成／戻り

---

# 5. 状態/フィードバック

* ローディング：`hx-indicator`（スピナー）、`aria-busy`
* 成功：行差し替え＋トースト（“保存しました”）
* 失敗：エラー行を **FRAG** で差し込み（`role="alert"`）
* 空状態：表の`<tbody>`に “該当なし” 行（1行）
* オフライン：`/offline` へ誘導（必要なら PWA 併用）

---

# 6. セキュリティ / 認証

* 認証：`Authorization: Bearer <Firebase ID Token>`（SSRではクッキー→サーバ検証）
* CSRF：全 `POST/PUT/PATCH/DELETE` は `X-CSRF-Token` 必須（テンプレに埋め込み、htmxヘッダ常設）
* 署名URL：ダウンロード/プレビューは**短寿命**（Assets API）
* RBAC：`staff/admin` はサーバ側テンプレでメニュー制御＋APIで二重チェック
* Idempotency：書込み系に `Idempotency-Key`（ヘッダ）を許可（重複防止）

---

# 7. 国際化 / アクセシビリティ / パフォーマンス

* i18n：サーバテンプレの文言辞書（`{{ T "key" . }}`）、URLは共通（`?lang=ja` で切替）
* A11y：フォーム `<label for>`、`aria-live="polite"`、モーダルはフォーカストラップ＋`aria-modal="true"`
* パフォーマンス：

  * 初回 SSR（TTFB 最優先）
  * 画像 `srcset`/`loading="lazy"`、`accept-ch` で DPR 最適化
  * htmx 差し替えは **部分 DOM** のみ（tbody/右ペイン/行）
  * CSS は 1 ファイル（critical は `<style>` inline）

---

# 8. 計測 / SEO

* 計測イベント（サーバ側）：`/shop filter`, `/product view`, `/design save`, `/ai accept`, `/checkout progress`, `/purchase`
* SEO：静的プレレンダ（guides/products/templates）、JSON-LD（Product/Article/Breadcrumb）

---

# 9. 画面 ↔ API マッピング（要点）

| 画面/FRAG                  | 主API                                             |           |                    |
| ------------------------ | ------------------------------------------------ | --------- | ------------------ |
| `/shop/table`            | `GET /products`                                  |           |                    |
| `/products/{id}`         | `GET /products/{id}`                             |           |                    |
| `/design/editor/preview` | 生成なし：サーバ描画 or `POST /assets:signed-download`     |           |                    |
| `/design/ai/table`       | `GET /designs/{id}/ai-suggestions`               |           |                    |
| 受入ボタン                    | `POST /designs/{id}/ai-suggestions/{sid}:accept` |           |                    |
| `/cart/table`            | `GET /cart`, `GET /cart/items`                   |           |                    |
| クーポン適用                   | `POST /cart:apply-promo`                         |           |                    |
| `/checkout/session`      | `POST /checkout/session`                         |           |                    |
| `/account/orders/table`  | `GET /orders`                                    |           |                    |
| 注文詳細タブ                   | `GET /orders/{id}`, `GET /orders/{id}/payments   | shipments | production-events` |

---

# 10. ディレクトリ（提案）

```
/web
  /layouts/_base.html        # <head>, header, footer, #content
  /layouts/_modal.html       # #modal, トースト
  /partials/_table_empty.html
  /partials/_pager.html
  /frags
    shop_table.html
    product_gallery.html
    design_form.html
    design_preview.html
    ai_table.html
    cart_table.html
    guides_table.html
    orders_table.html
  /pages
    home.html
    shop.html
    product_show.html
    design_editor.html
    design_ai.html
    design_preview.html
    cart.html
    checkout_address.html
    checkout_shipping.html
    checkout_payment.html
    checkout_review.html
    checkout_complete.html
    account_index.html
    account_orders.html
    account_order_show.html
    account_addresses.html
    account_library.html
    guide_index.html
    guide_show.html
    legal.html
```
