# API モデル設計メモ

各ドメインで共通的に利用する API モデル／DTO の取り扱い指針を整理する。

## 共通ポリシー
- API のバージョンは `api/v1` を前提とし、レスポンスの ISO8601 文字列はすべて `DateTime` に正規化する。
- DTO 層では JSON のキー名をそのまま保持し、ドメイン層では列挙型・値オブジェクトに変換する。
- 更新系エンドポイントは Idempotency-Key で守られるため、`copyWith` を使って差分を作り直してから DTO に戻す。
- 将来のバージョン追加時は DTO の `factory fromJson` にフォールバック値を追加し、ドメインの `copyWith` で既定値を補完する。

## Users
- 必須フィールド: `persona`, `preferredLang`, `isActive`, `piiMasked`, `createdAt`, `updatedAt`.
- サブリソース: 住所帳（`recipient`, `line1`, `city`, `postalCode`, `country`）、支払手段（`provider`, `methodType`, `providerRef`）、お気に入り（`designRef`, `addedAt`）。
- バージョン戦略: `role`・`onboarding` のような拡張フィールドは Map をそのまま保持し、未定義の場合でもクラッシュしないよう `Map<String, dynamic>?` で扱う。

## Designs
- 必須フィールド: `ownerRef`, `status`, `shape`, `size.mm`, `style.writing`, `version`, `createdAt`, `updatedAt`.
- 版履歴は `version`（int）で管理し、クライアントは `fetchVersions` で差分を要求する。
- AI メタ情報やアセットは任意。null の場合は `DesignAiMetadata` / `DesignAssets` を省略して表現。

## Catalog（素材・SKU・テンプレ・フォント）
- 素材: `name`, `type`, `isActive`, `createdAt` が必須。環境情報は `CatalogMaterialSustainability` に集約。
- SKU: `sku`, `materialRef`, `shape`, `size.mm`, `basePrice`, `stockPolicy`, `isActive`, `createdAt` が必須。
- テンプレ: `name`, `shape`, `writing`, `constraints`, `isPublic`, `sort`, `createdAt`, `updatedAt`。
- フォント: `family`, `writing`, `license.type`, `isPublic`, `createdAt`。`unicodeRanges` や `metrics` は任意だが DTO 側で null 許容。
- バージョン戦略: テンプレ/フォントは `version` と `isDeprecated` を併用し、UI では `isDeprecated` を元に絞り込む。

## Orders
- 必須フィールド: `orderNumber`, `userRef`, `status`, `currency`, `totals`, `lineItems`, `createdAt`, `updatedAt`。
- `OrderTotals` は `subtotal/discount/shipping/tax/total` が揃っている前提でハンドリング。
- 履歴系: 支払 (`OrderPayment`)、配送 (`OrderShipment`)、制作イベント (`ProductionEvent`) は個別 API で取得する。
- バージョン戦略: ステータスは文字列列挙で、追加ステータスが来ても `switch` でフォールバックできるよう `enum` の未定義値は例外化→実装側でハンドリング。

## Promotions
- 必須フィールド: `code`, `kind`, `value`, `isActive`, `startsAt`, `endsAt`, `usageLimit`, `usageCount`, `limitPerUser`, `createdAt`。
- `kind` によって `value` の意味が変わる（percent は 0–100、fixed は金額）。クライアント側で単位を誤らないよう DTO で double に統一。
- 条件や併用ルールはオプショナル Map として拡張可能。

## Content（ガイド/固定ページ）
- ガイド: `slug`, `category`, `isPublic`, `translations`, `createdAt`, `updatedAt` が必須。翻訳は locale ごとに必ず `title` と `body` を保持。
- 固定ページ: `slug`, `type`, `isPublic`, `createdAt`, `updatedAt`。翻訳ブロックは `ContentBlock` の配列で柔軟に拡張する。
- バージョン戦略: `version` + `isDeprecated` + `publishAt` を併用。クライアントは `publishAt` が未来なら事前ダウンロードだけ行い UI 表示はしない。
