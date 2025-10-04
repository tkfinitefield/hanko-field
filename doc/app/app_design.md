# アプリ
モバイルアプリは Flutter で開発します。MVVMパターンを採用します。
状態管理には riverpod を使用します。ただし、コード生成は使いません。また、StateProvider は使用しません。NotifierやAsyncNotifierを使用します。


# 画面一覧

# アプリ全体ナビ

* ボトムタブ：**作成** / **ショップ** / **注文** / **マイ印鑑** / **プロフィール**
* 共通：通知ベル、検索、ヘルプ（AppBar）

---

## 0) 起動・初期設定

* `/splash` スプラッシュ
* `/onboarding` 初回チュートリアル
* `/locale` 言語・地域選択
* `/persona` ペルソナ選択（日本人/外国人）
* `/auth` ログイン/ゲスト選択（Apple/Google/Email）

---

## 1) ホーム/探索

* `/home` ホーム（特集・最近のデザイン・おすすめテンプレ）
* `/search` 検索（テンプレ/素材/記事/FAQ）
* `/notifications` お知らせ一覧

---

## 2) 印影作成フロー（作成タブ）

* `/design/new` 作成タイプ選択（文字入力/画像アップ/ロゴ刻印）
* `/design/input` 名前入力

  * 外国人モード：`/design/input/kanji-map`（漢字候補と意味）
* `/design/style` 書体/テンプレ選択（篆書/隷書/楷書/古印体、丸/角）
* `/design/editor` エディタ（配置・太さ・余白・回転・格子）
* `/design/ai` AI修正（プリセット提案・候補比較）
* `/design/check` 実印/銀行印チェック（日本人向け）
* `/design/preview` プレビュー（実寸/和紙モック/背景切替）
* `/design/export` デジタル出力（PNG/SVG）
* `/design/versions` バージョン履歴（差分比較/ロールバック）
* `/design/share` 共有（SNSモック/透かし）

---

## 3) ショップ（素材・商品・オプション）

* `/shop` ショップトップ（素材カテゴリ）
* `/materials/:materialId` 素材詳細（硬度/質感/写真）
* `/products/:productId` SKU詳細（形状/サイズ/価格/在庫・受注生産）
* `/products/:productId/addons` オプション（ケース/桐箱/朱肉）

---

## 4) カート〜チェックアウト

* `/cart` カート（行編集/クーポン/見積）
* `/checkout/address` 配送先選択・追加
* `/checkout/shipping` 配送方法（国内/国際）
* `/checkout/payment` 支払方法（トークン参照）
* `/checkout/review` 注文最終確認（デザイン**スナップショット**表示）
* `/checkout/complete` 注文完了（注文番号/次アクション）

---

## 5) 注文・制作・配送

* `/orders` 注文一覧
* `/orders/:orderId` 注文詳細（金額/配送先/デザインスナップショット）
* `/orders/:orderId/production` 制作進捗（刻印→研磨→検品→梱包）
* `/orders/:orderId/tracking` 配送トラッキング（イベント時系列）
* `/orders/:orderId/invoice` 領収書/請求書（表示/ダウンロード）
* `/orders/:orderId/reorder` 再注文（同スナップショット）

---

## 6) マイ印鑑（ライブラリ）

* `/library` マイ印鑑一覧（並び替え/フィルタ）
* `/library/:designId` デザイン詳細（AIスコア/登録可否/使用履歴）
* `/library/:designId/versions` バージョン履歴
* `/library/:designId/duplicate` 複製して新規作成
* `/library/:designId/export` デジタル印影DL
* `/library/:designId/shares` 共有リンク管理（期限付き）

---

## 7) 文化・ガイド（外国人向け強化）

* `/guides` 文化ガイド一覧
* `/guides/:slug` 記事詳細（i18n）
* `/kanji/dictionary` 漢字の意味辞書
* `/howto` 使い方ガイド/動画

---

## 8) プロフィール/設定

* `/profile` プロフィール（アイコン/表示名/モード切替）
* `/profile/addresses` 住所帳
* `/profile/payments` 支払手段
* `/profile/notifications` 通知設定
* `/profile/locale` 言語と通貨
* `/profile/legal` 法務（特商法/規約/プライバシー）
* `/profile/support` ヘルプ&問い合わせ
* `/profile/linked-accounts` 連携（Apple/Google）
* `/profile/export` データエクスポート
* `/profile/delete` アカウント削除

---

## 9) サポート/ステータス

* `/support/faq` FAQ
* `/support/contact` 問い合わせフォーム
* `/support/chat` チャットサポート（ボット→有人）
* `/status` システムステータス

---

## 10) システム/ユーティリティ

* `/permissions` 権限許可（写真/ファイル/通知）
* `/updates/changelog` 更新履歴
* `/app-update` 強制アップデート
* `/offline` オフライン画面
* `/error` エラー汎用

---
