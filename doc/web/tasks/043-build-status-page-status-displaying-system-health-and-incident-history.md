# Build status page (`/status`) displaying system health and incident history.

**Parent Section:** 7. Support & Legal
**Task ID:** 043

## Goal
Build status page showing system health.

## Implementation Steps
1. Integrate with status API to fetch incidents.
2. Display current status, component status, incident history.
3. Provide subscription instructions.

## UI構成（Material Design 3）
- **レイアウト**
  - Scaffold（画面骨格）
  - Center-aligned top app bar（タイトル「ステータス」、戻る/ヘルプ）
  - 既存グローバルナビに準拠（必要に応じて）

- **現在の稼働状況（ヘッダー）**
  - Banner：全体状態（正常/一部障害/障害）を色で示す
  - Assist chips：サービス別ステータス（API/Auth/Storage 等）
  - Supporting text：最終更新時刻、計測期間

- **コンポーネントステータス**
  - Cards：各コンポーネントの要約（稼働率、MTTR、イベント数）
  - List items：詳細行（稼働/劣化/停止）＋ Trailing icon（外部リンク/詳細）
  - Badges：重大度（minor/major）や影響範囲の表示

- **稼働率サマリー**
  - Cards：直近 7/30/90 日の稼働率サマリー
  - 画像/簡易グラフ（Card 内の Image/Canvas 相当）で履歴トレンド
  - Assist chips：期間切替（7d/30d/90d）

- **インシデント履歴**
  - SearchBar：キーワード/期間検索
  - Filter chips：重大度/状態（open/resolved）/影響サービス
  - List：インシデント行（タイトル、期間、影響範囲、状態 Badge）
  - Expansion panels：詳細（タイムライン、対処、リンク、ポストモーテム）
  - Buttons：Text button（詳細を開く）、Filled tonal button（購読/サブスク）

- **通知/購読**
  - Cards：購読方法（メール、Webhook、RSS）
  - Buttons：Filled button（購読する）、Outlined button（RSS をコピー）
  - Dialog：メール購読設定（アドレス、カテゴリ選択）

- **ローディング/エラー状態**
  - Circular progress indicator（全体/セクション単位）
  - Skeleton（List/Card のプレースホルダ）
  - Error Card：Supporting text＋Retry（Filled tonal button）

- **アクセシビリティ/フィードバック**
  - Semantics/Focus order/コントラスト比の遵守
  - Snackbar：通信失敗/成功時の非モーダル通知

## UI Components
- **Layout:** `StatusLayout` with `StatusHeader` showing overall indicator and last updated timestamp.
- **Current status:** `StatusBanner` summarizing platform state with `StatusBadge` per service.
- **Component grid:** `ComponentList` cards listing uptime, MTTR, incident count.
- **Incident timeline:** `IncidentTimeline` accordion with collapsible incident entries.
- **Subscription panel:** `SubscriptionCard` with email/webhook/RSS options.
- **Support links:** `SupportFooter` referencing runbooks, contact info.
