# DB Schema

データベースには Firestore を使用します。

## コレクション一覧

/users/{uid}
/users/{uid}/addresses/{addressId}
/users/{uid}/paymentMethods/{pmId}
/users/{uid}/favorites/{designId}

/designs/{designId}
/designs/{designId}/versions/{versionId}
/designs/{designId}/aiSuggestions/{suggestionId}

/aiJobs/{jobId}
/nameMappings/{mappingId}

/templates/{templateId}
/fonts/{fontId}
/materials/{materialId}
/products/{productId}

/carts/{uid}/items/{itemId}

/orders/{orderId}
/orders/{orderId}/payments/{paymentId}
/orders/{orderId}/shipments/{shipmentId}
/orders/{orderId}/productionEvents/{eventId}

/assets/{assetId}

/content/guides/{guideId}
/content/pages/{pageId}

/promotions/{promoId}
/promotions/{promoId}/usages/{uid}

/reviews/{reviewId}

/productionQueues/{queueId}

/auditLogs/{logId}

/counters/{counterId}           // 連番用（請求書番号など）
/stockReservations/{orderId}    // 在庫引当の一時記録


---

### 参考：最小サンプル

**template**

```json
{
  "name": "Classic Tensho Round",
  "shape": "round",
  "writing": "tensho",
  "defaults": {
    "sizeMm": 15,
    "layout": { "grid": "3x3", "margin": 0.1, "autoKern": true, "centerBias": 0.0 },
    "stroke": { "weight": 0.9, "contrast": 0.2 },
    "fontRef": "/fonts/f_tensho_001"
  },
  "constraints": {
    "sizeMm": { "min": 10, "max": 21, "step": 1 },
    "strokeWeight": { "min": 0.6, "max": 1.4 },
    "margin": { "min": 0.05, "max": 0.2 },
    "glyph": {
      "maxChars": 4,
      "allowRepeat": true,
      "allowedScripts": ["kanji", "kana"],
      "prohibitedChars": ["・", "※"]
    },
    "registrability": { "jpJitsuinAllowed": true, "bankInAllowed": true }
  },
  "previewUrl": "https://example.com/preview.png",
  "isPublic": true,
  "sort": 100,
  "createdAt": "2025-10-03T00:00:00Z",
  "updatedAt": "2025-10-03T00:00:00Z"
}
```

**font**

```json
{
  "family": "Ryumin Tensho",
  "subfamily": "Regular",
  "vendor": "Finite Field Type",
  "version": "1.0.0",
  "writing": "tensho",
  "designClass": "seal",
  "license": {
    "type": "commercial",
    "uri": "https://example.com/license",
    "restrictions": ["export_svg"],
    "embeddable": true,
    "exportPermission": "render_png"
  },
  "glyphCoverage": ["A-Z", "a-z", "0-9", "常用漢字", "ひらがな", "カタカナ"],
  "metrics": { "unitsPerEm": 1000, "ascent": 800, "descent": -200, "weightRange": { "min": 400, "max": 700 } },
  "opentype": { "features": ["liga", "kern"] },
  "previewUrl": "https://example.com/font-preview.png",
  "isPublic": true,
  "sort": 50,
  "createdAt": "2025-10-03T00:00:00Z",
  "updatedAt": "2025-10-03T00:00:00Z"
}
```



### 運用ポイント

* **計算責務**は Cloud Run 側（`recalcCartEstimates`）でヘッダの `estimates` と `itemsCount` を更新。
* **プロモ適用**はトランザクションで検証→ヘッダ `promo` を更新。
* **チェックアウト開始**時に `checkout.sessionId/status=pending` を書き、**成功**後は `/orders/{orderId}` 作成→**カートを空にする/アーカイブ**。
* **payments**：作成/更新は **Cloud Functions（Webhook/Callable）** 経由に限定し、クライアントからの書込は禁止（Rules）。
* **shipments**：キャリア Webhook で `events[]` を追記・`status` を更新。`delivered` 到達時に `/orders/{id}.status = "delivered"` へ遷移させる自動処理を推奨。
* **productionEvents**：工房端末からの記録は Cloud Run 経由で、作業者の `operatorRef` とステーション `station` を付与。`packed` イベントで自動ラベル生成→`/shipments` 作成、のオーケストレーションが定石です。

## スキーマファイル

データベーススキーマは `doc/db_schema/` フォルダ内の個別JSONファイルに分離されています：

- `users.schema.json` - ユーザー基本情報
- `users.address.schema.json` - ユーザー住所
- `users.paymentMethod.schema.json` - 支払い方法
- `users.favorites.schema.json` - お気に入り
- `designs.schema.json` - デザイン
- `designs.version.schema.json` - デザインバージョン
- `designs.aiSuggestion.schema.json` - AI提案
- `templates.schema.json` - テンプレート
- `fonts.schema.json` - フォント
- `materials.schema.json` - 素材
- `products.schema.json` - 商品（SKU）
- `carts.header.schema.json` - カートヘッダー
- `orders.payment.schema.json` - 注文決済
- `orders.shipment.schema.json` - 注文配送
- `orders.productionEvent.schema.json` - 制作イベント
- `assets.schema.json` - アセット
- `content.guides.schema.json` - ガイド記事
- `content.pages.schema.json` - コンテンツページ
- `promotions.schema.json` - プロモーション
- `promotions.usage.schema.json` - プロモーション利用
- `reviews.schema.json` - レビュー
- `productionQueues.schema.json` - 制作キュー
- `auditLogs.schema.json` - 監査ログ
- `counters.schema.json` - カウンター
- `stockReservations.schema.json` - 在庫予約
