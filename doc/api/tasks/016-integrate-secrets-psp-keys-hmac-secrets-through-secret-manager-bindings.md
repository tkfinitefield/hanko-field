# Integrate secrets (PSP keys, HMAC secrets) through Secret Manager bindings.

**Parent Section:** 2. Core Platform Services
**Task ID:** 016

## Goal
Integrate Google Secret Manager for sensitive values and expose ergonomic API for runtime consumption.

## Design
- `internal/platform/secrets.Fetcher` reads `secret://` URIs, caches values, and supports reload notifications.
- Provide local development fallback to read from `.secrets.local` file when Secret Manager unavailable.
- Expose metrics for fetch latency and cache hits.

## Steps
1. Implement fetcher with environment-specific project IDs and version pins (latest by default).
2. Integrate with config loader to resolve secret references automatically.
3. Add panic-on-start option for required secrets missing; log redacted names.
4. Provide rotation playbook and optional Pub/Sub push trigger for hot reload.

## 作業内容
- 実装: `internal/platform/secrets.Fetcher` を追加し、`secret://` 参照の取得・キャッシュ・リロード通知・ローカルフォールバック・メトリクスを実装。
- 設定: `config.Load` に秘密情報解決の強化、必須シークレット検査、欠損時パニックオプション、`secret://` 正規化を追加し、Go テストを更新。
- 起動フロー: `cmd/api/main.go` でフェッチャー初期化・環境変数連携・必須シークレット強制を行うよう変更。
- ドキュメント: `doc/api/configuration.md` に新しいシークレットスキーム、環境変数、回転手順、Pub/Sub ホットリロードの手順を追記。
