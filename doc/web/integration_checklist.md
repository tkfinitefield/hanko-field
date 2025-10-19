# Web フラグメント × API 連携チェックリスト（Task 005）

## 共通方針
- Web レイヤは `api/` サービスの REST を消費。認証済セッションは Firebase ID Token（Cookie）＋ CSRF トークンで保護。
- htmx リクエストは `HX-Request: true` を付与し、`HX-Trigger` で成功/失敗イベントを発火。POST 系は `Idempotency-Key` をヘッダに追加。
- 以下の表で `✓` は現時点の API 仕様で要件が満たされている、`△` は追加フィールド/エンドポイント調整が必要な項目。

## 1. 公開領域 / 探索
| UI（ページ/FRAG/MODAL） | API エンドポイント | リクエスト内容 | 主要レスポンス項目 | 状態 |
| --- | --- | --- | --- | --- |
| `/` ホーム › `FragCompareSkuTable` | `GET /products?material=&shape=&sizeMm=` | クエリ：フィルタ（任意） | `products[]`（`id`,`name`,`material`,`shape`,`price`,`leadTimeDays`） | ✓ |
| `/` ホーム › `FragGuidesLatest` | `GET /content/guides?lang=ja&limit=3&sort=publishedAt desc` | 言語/件数/ソート | `guides[]`（`slug`,`title`,`summary`,`publishedAt`,`heroImage`） | ✓ |
| `/shop` 本体 | `GET /products` | Facet フィルタ, ページング, ソート | `pageInfo`,`products[]`（`sku`,`badge`,`price`,`inStock`） | △ ページング情報の `prevPageToken` が未定義 → API で追加 |
| `/products/{id}` | `GET /products/{id}` | パス：`productId` | `name`,`description`,`materials[]`,`images[]`,`price`,`stock`,`leadTimeDays`,`legalNotes` | ✓ |
| `/products/{id}` › レビュー抜粋 `FragReviewsSnippets` | `GET /reviews?productId=...&public=true&limit=3` | クエリ：`public=true` | `reviews[]`（`rating`,`headline`,`body`,`authorName`） | △ API に `public=true` フラグが無いため追加検討 |
| `/templates` › `FragTemplatesTable` | `GET /templates?style=&registrable=` | フィルタ（書体/登録可否） | `templates[]`（`id`,`name`,`style`,`registrable`,`previewUrl`） | ✓ |
| `/guides` リスト | `GET /content/guides` | カテゴリ/言語/ページング | `guides[]`,`pageInfo` | △ ページングパラメータを `/shop` と合わせる |
| `/guides/{slug}` | `GET /content/guides/{slug}` | パス：`slug`,`?lang` | `title`,`body`,`seo`, `related[]` | ✓ |
| `/content/{slug}` | `GET /content/pages/{slug}` | | `title`,`body`,`effectiveDate` | ✓ |
| `/status` | `GET /status` or RUM feed | 未定（Ops 合意） | `services[]`（`name`,`status`,`lastIncident`） | △ API 設計必要 |

## 2. デザイン作成フロー
| UI | API | リクエスト | レスポンス | 状態 |
| --- | --- | --- | --- | --- |
| `/design/new` | - | - | - | - 静的（API 無し） |
| `/design/editor/form` `FragDesignEditorForm` | `GET /designs/{id}` or 初期は空 | 既存デザイン取得 | `design`（`rawName`,`kanji`,`font`,`templateId`,`strokeWidth` 等） | ✓ |
| `/design/editor/form` 保存 | `PUT /designs/{id}` | Body：フォーム全項目 + `idempotency-key` | 更新後 `design` | ✓ |
| `/design/editor/preview` | `POST /assets:signed-download` or internal render | プレビュー用 `designStateHash` | `signedUrl`,`expiresAt` | △ プレビュー描画 API（画像/SVG生成）の仕様調整必要 |
| `/modal/kanji-map` | `POST /name-mappings:convert` | `{ latinName, locale }` | `candidates[]`（`kanji`,`meaning`） | ✓ |
| `/modal/pick/font` | `GET /fonts` | フィルタ：書体等 | `fonts[]`（`id`,`previewUrl`,`license`） | ✓ |
| `/modal/pick/template` | `GET /templates` | `?type=round|square` | `templates[]` | ✓ |
| `/design/ai/table` | `GET /designs/{id}/ai-suggestions` | `designId`, `?status` | `suggestions[]`（`id`,`previewUrl`,`score`,`diff`） | △ レスポンスに `diff` 欄が未定義 → API 担当と調整 |
| `/design/ai` 生成リクエスト | `POST /designs/{id}/ai-suggestions` | `{ method, promptModifiers }` | `suggestionId`,`status` | ✓ |
| `/design/preview` | `POST /assets:{signed-download}` | `designId`,`bg`,`dpi` | `signedUrl` | ✓ (プレビューAPIと同一) |
| `/design/versions/table` | `GET /designs/{id}/versions` | | `versions[]`（`id`,`createdAt`,`diffSummary`,`actor`） | △ `diffSummary` 欄追加要 |
| `/modal/design/version/rollback` | `POST /designs/{id}/versions/{versionId}:rollback` | | `design`（最新状態） | ✓ |

## 3. カート / チェックアウト
| UI | API | リクエスト | レスポンス | 状態 |
| --- | --- | --- | --- | --- |
| `/cart/table` | `GET /cart` + `GET /cart/items` | カートヘッダ/アイテム | `cart`（`currency`,`totals`）/`items[]`（`designSnapshot`,`price`,`quantity`） | ✓ |
| `/cart/estimate` | `POST /cart:estimate` | `{ shippingAddress, items }` | `totals`（`subtotal`,`discount`,`tax`,`shipping`,`total`） | ✓ |
| `/modal/cart/promo` | `POST /cart:apply-promo` | `{ code }` | `promotion`（`code`,`applied`,`amount`） | ✓ |
| `/modal/cart/promo` 削除 | `DELETE /cart:remove-promo` | | `promotion` | ✓ |
| `/checkout/address` › 住所帳 `FragAccountAddressesTable` | `GET /me/addresses` | - | `addresses[]`（`id`,`label`,`country`,`isDefault`） | ✓ |
| `/checkout/address` 追加 | `POST /me/addresses` | 入力値 | `address` | ✓ |
| `/checkout/shipping/table` | `POST /cart:estimate`(拡張) | `{ country, postalCode, weight, currency }` | `methods[]`（`id`,`name`,`eta`,`fee`,`carrier`） | △ API で配送選択レスポンスを拡張 |
| `/checkout/payment` | `POST /checkout/session` | `{ returnUrl, cancelUrl, locale }` | `sessionUrl`,`clientSecret` | ✓ |
| `/checkout/payment` 完了 | `POST /checkout/confirm` | `{ sessionId }` | `orderId` | ✓ (`Idempotency-Key` 必須) |
| `/checkout/review` | `GET /cart` | - | `cart` + `shippingMethod` + `paymentSummary` | △ `paymentSummary` 情報が無い → API で整備 |
| `/checkout/complete` | `GET /orders/{id}` | `orderId` | `order`（`number`,`tracking`, `downloadLinks`） | ✓ |

## 4. アカウント領域
| UI | API | リクエスト | レスポンス | 状態 |
| --- | --- | --- | --- | --- |
| `/account/profile/form` | `GET /me` / `PUT /me` | ユーザープロフィール | `displayName`,`locale`,`country`,`marketingOptIn` | ✓ |
| `/account/addresses/table` | `GET /me/addresses` | | `addresses[]` | ✓ |
| `/modal/address/edit` | `POST/PUT /me/addresses` | | `address` | ✓ |
| `/account/orders/table` | `GET /orders?status=&from=&to=` | クエリ：期間/状態 | `orders[]`（`id`,`number`,`placedAt`,`total`,`status`） | ✓ |
| `/account/orders/{id}` タブ | `GET /orders/{id}` / `GET /orders/{id}/payments|shipments|production-events` | パス：`orderId` | 各詳細配列 | ✓ |
| `/modal/order/cancel` | `POST /orders/{id}:cancel` | `{ reason, notes }` | `order`（更新後） | ✓ |
| `/modal/order/invoice` | `POST /orders/{id}:request-invoice` | `{ format }` | `invoiceRequestId`,`status` | △ `status` 欄の定義が必要 |
| `/account/library/table` | `GET /designs` | フィルタ：`?aiScore>=`, `?registrable=` | `designs[]`（`id`,`thumbnail`,`aiScore`,`registrable`) | △ `aiScore` フィルタ対応が API に未定 |
| `/account/security` | `GET /me/security` (要定義) | | `providers[]`,`mfaEnabled`,`lastLoginAt` | △ API 仕様未策定 |
| 通知ドロップダウン `/frags/notifications/list` | `GET /notifications?limit=10` | | `notifications[]`（`id`,`title`,`body`,`read`,`href`） | △ エンドポイント未定義 |

## 5. サポート / 法務 / ユーティリティ
| UI | API | リクエスト | レスポンス | 状態 |
| --- | --- | --- | --- | --- |
| `/support/form` | `POST /support/contact` (要定義) | `{ name,email,topic,message }` | `ticketId`,`status` | △ API 未実装 |
| `/legal/{slug}` | `GET /content/pages/{slug}` | `?lang` | `title`,`body`,`version`,`publishedAt` | ✓ |
| `/status` フィード | `GET /status` | | `services[]` | △ (上記と同じ) |
| `/offline` | - | - | - | 静的 |

## 6. アクションチェックリスト
- [x] 公開ページの主要データ取得 API を整理（プロダクト/テンプレ/ガイド/静的ページ）。
- [x] デザイン作成フローで必要な CRUD + AI 提案 API を特定。
- [x] カート〜チェックアウトの見積・支払・注文確定エンドポイントを列挙。
- [x] アカウント領域の履歴参照・キャンセル等の API マッピングを洗い出し。
- [ ] API チームと共有し、`△` 項目のスキーマ/レスポンス拡張を決める。
- [ ] Task 008/015 でミドルウェアと SEO ヘルパを実装する際、本表の `Idempotency-Key`・`JSON-LD` 要件を反映。
- [ ] Sitemap/構造化データ生成時に公開ページ一覧（セクション 1）を流用。

## 7. フォローアップ
- `△` とした項目は `doc/api/tasks/` にスピンアウトして仕様確定を依頼。特に配送見積 (`POST /cart:estimate`) 拡張と `notifications` API を早期に決める。
- htmx フラグメントキャッシュ制御（ETag/Last-Modified）は API レスポンス `etag` or `updatedAt` を活用。Task 008 に連携。
- クライアントで利用するデータモデルは `internal/viewmodel` に型定義し、API レスポンスとの差分を明示。
