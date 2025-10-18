# Web スコープ / ペルソナ / リリース検証（Task 001）

## スコープ確認
- `doc/web/web_design.md` で定義された公開導線（`/`, `/shop`, `/products/{id}`, `/templates`, `/guides`, `/content`, `/legal`, `/status`）により、探索→教材→静的ページまでの SSR+htmx 構成が網羅されている。
- デザイン作成フロー（`/design/new`→`/design/editor`→`/design/ai|preview|versions`）が明示され、フォーム/プレビュー/AI 候補/履歴などの UI 分割と FRAG/MODAL の責務も具体化されている。
- EC/チェックアウト（`/cart`→`/checkout/*`→`/checkout/complete`）、アカウント領域（プロフィール/住所/注文/ライブラリ/セキュリティ）が揃っており、購入後の継続利用までサポートされている。
- サポート・法務（`/support`, `/legal/{slug}`）とシステム状態表示（`/status`）が含まれており、運用/コンプライアンス観点も最低限カバー。
- 7章で i18n（`?lang=` 切替）、A11y、パフォーマンス方針が整理済み。計測/SEO、セキュリティ（Firebase Auth、CSRF、RBAC、短寿命署名 URL）も明記されており、非機能要件の初期指針として妥当。
- 不足点：ペルソナ別のメッセージング（LP/ガイドの言語・コピー差分）、海外配送・関税に関する UI 要件、AI サービスの SLA/バックエンド準備状況が未記載。これらは別途要件化が必要。

## ペルソナ適合性
### ペルソナ1: 日本文化ファン外国人
- 多言語 SSR（7章）と `/design/editor` の漢字マッピングモーダル（`/modal/kanji-map`）、AI 候補（`/design/ai`）により「漢字変換体験」「バリエーション提案」が可能。
- `/checkout/address` の住所帳、`/checkout/shipping` の国内/国際比較が国際配送ニーズに合致。ただし送料・関税表示、海外通貨対応、ガイドの英語コンテンツ方針は未定義。
- 文化体験訴求のコンテンツは `/guides` で構成できそうだが、トーン&マナー、UGC 連携等のマーケ要素は別設計が必要。

### ペルソナ2: こだわり派の日本人ユーザー
- `/design/editor` の細かなパラメータ（線幅/余白/格子）と `/design/versions` の履歴管理がカスタマイズ要求に応える。
- `/templates`・`/products/{id}` で素材・書体・登録制約を提示し、`/account/library`、`/account/orders` が再注文・運用性を確保。
- 法的要件（実印登録可否、領収書、法人情報）については `/modal/order/invoice` や `/checkout/address` の項目で対応可能だが、電子印鑑ガイドラインや印鑑証明連携は未定義。

## リリースマイルストーン案
| マイルストーン | 対象範囲 | 主要依存 API/サービス | ペルソナ対応 |
| --- | --- | --- | --- |
| **M0 Foundation**<br>（デザイン・コンテンツ検証） | Static/SSR 基盤、`/`, `/shop`, `/products/{id}`, `/guides`, `/content`, `/legal`, `/status`。htmx テーブル/モーダル共通部品、Firebase Auth SSR 連携。 | `GET /products`, `GET /content/*`, Firestore カタログ。 | いずれのペルソナも探索段階の検証が可能。 |
| **M1 Global MVP**<br>（購入までの最小導線） | `/design/new`, `/design/editor` の基本フォーム & プレビュー、`/cart`, `/checkout/address|shipping|payment|review|complete`, `/account`（プロフィール・住所・注文一覧）。 | `POST /designs`, `POST /designs/{id}:registrability-check`, `GET/POST /cart*`, `POST /checkout/session`, `POST /orders`, PSP/配送見積。 | ペルソナ1の体験（漢字変換、海外配送、英語 UI）、ペルソナ2の基本購入をカバー。 |
| **M2 Pro & AI**<br>（差別化・運用強化） | `/design/ai`, `/design/versions`, `/account/library`, `/account/security`, `/support`, AI サジェスト、バージョン管理、通知/セキュリティ機能、SEO拡張。 | `GET/POST /designs/{id}/ai-suggestions`, `POST /designs/{id}/ai-suggestions/{sid}:accept`, `GET /designs`, 2FA/Firebase 拡張、サポート問い合わせ送信。 | ペルソナ2の高度なカスタマイズ、継続利用ニーズを満たし、AI・CRM 機能で差別化。 |

前提：Cloud Run デプロイと Tailwind ビルドパイプラインは M0 時点で整備、M1 で PSP・物流連携、M2 で AI/通知の SLA とモニタリング（`/status`/RUM）。

## リスクと前提条件
| 項目 | 内容 | オーナー | 緩和策 |
| --- | --- | --- | --- |
| i18n リソース不足 | 言語切替は設計済みだが、文言辞書/翻訳が未手配。 | Product/Marketing | M0 終了までにコピー設計と翻訳ベンダ確保、`{{ T "key" . }}` テンプレに差し込みテスト。 |
| 海外配送 & 税制 | `/checkout/shipping` で比較 UI はあるが、国際料金・関税計算 API が未定。 | Ops | サードパーティ配送サービス選定、M1 前に料金テーブル or API 接続を確定。 |
| AI サービス安定性 | `/design/ai` は中核差別化だが、生成 API の SLA / コスト試算が不透明。 | Engineering | M1 中にスパイク、推論待ち時間に対する UX（スピナー/リトライ）を FRAG 仕様へ追加。 |
| 法的文書の整備 | `/legal/{slug}` に載せる利用規約/特商法/プラポリ文面が未確定。 | Legal/Compliance | M0 でドラフト策定、htmx での版管理ポリシー（改定日表示）をテンプレに追加。 |
| PSP 導入スケジュール | `POST /checkout/session` が Stripe 依存、審査遅延リスクあり。 | Finance/Ops | 早期にアカウント申請、M1 ではテストモードで E2E を確保し、代替 PSP を比較。 |

## アクションアイテム
- マーケ/コンテンツと連携し、LP・ガイドの言語/コピー差分とガイドロードマップを策定（M0 内）。
- Ops と物流要件（送料、納期 SLA、リターンポリシー）を詰め、`/checkout/shipping` UI 仕様に反映（M1 前）。
- AI チームと `GET/POST /designs/{id}/ai-suggestions` のレスポンス SLA / フォールバック（手動テンプレ提案）の要件を詰める（M2 前）。
