# Tailwind デザイン・コンポーネント指針（Task 003）

## 1. Tailwind 基本方針
- プロジェクト共通プレフィックスを `hf-` に統一し、外部スタイルとの競合を防ぐ。
- テーマ値は `tailwind.config.js` で定義し、ユーティリティクラスはテンプレ内で直接使用。複雑な UI は `@apply` に頼らず templ コンポーネントで構造化。
- ベーススタイルは `@tailwind base` 上に `@layer base` を用いて、`body`, `a`, `button`, `input` のデフォルトを設定。フォームは `@tailwindcss/forms` プラグインを併用。
- ダークモードは `class` 切替方式を採用（`<html class="dark">`）し、MVP ではライトテーマをデフォルトとする。

```js
// assets/tailwind.config.js（抜粋）
module.exports = {
  content: ["./internal/ui/**/*.templ", "./internal/ui/**/*.go"],
  prefix: "hf-",
  darkMode: "class",
  theme: {
    extend: {
      colors: {
        brand: {
          primary: "#D93025",       // 朱（印肉イメージ）
          secondary: "#1F2933",     // 墨色
          accent: "#F28F16",        // 金赤（アクセント）
        },
        neutral: {
          50: "#F8FAFC",
          100: "#EEF2F6",
          200: "#D7DDE5",
          300: "#B8C1CE",
          400: "#8E99AB",
          500: "#667085",
          600: "#4D5565",
          700: "#3C4250",
          800: "#2B2F38",
          900: "#1F232B",
        },
        success: "#2BA24C",
        warning: "#F59E0B",
        danger: "#D61F1F",
        info: "#0F62FE",
      },
      fontFamily: {
        sans: ["'Noto Sans JP'", "Inter", "system-ui", "sans-serif"],
        serif: ["'Noto Serif JP'", "ui-serif", "Georgia", "serif"],
        mono: ["'JetBrains Mono'", "ui-monospace", "SFMono-Regular"],
      },
      fontSize: {
        xs: ["0.75rem", { lineHeight: "1.5" }],
        sm: ["0.875rem", { lineHeight: "1.6" }],
        base: ["1rem", { lineHeight: "1.6" }],
        lg: ["1.125rem", { lineHeight: "1.6" }],
        xl: ["1.25rem", { lineHeight: "1.5" }],
        "2xl": ["1.5rem", { lineHeight: "1.5" }],
        "3xl": ["1.875rem", { lineHeight: "1.4" }],
        "4xl": ["2.25rem", { lineHeight: "1.25" }],
      },
      spacing: {
        13: "3.25rem",
        18: "4.5rem",
        22: "5.5rem",
        30: "7.5rem",
      },
      borderRadius: {
        sm: "0.25rem",
        DEFAULT: "0.5rem",
        lg: "0.75rem",
        full: "9999px",
      },
      boxShadow: {
        card: "0 12px 24px -10px rgba(31, 41, 51, 0.15)",
        focus: "0 0 0 4px rgba(217, 48, 37, 0.18)",
      },
      zIndex: {
        modal: 40,
        toast: 45,
        overlay: 50,
      },
      screens: {
        xs: "480px",
        sm: "640px",
        md: "768px",
        lg: "1024px",
        xl: "1280px",
        "2xl": "1536px",
      },
      transitionDuration: {
        fast: "120ms",
        normal: "200ms",
        slow: "320ms",
      },
    },
  },
  plugins: [require("@tailwindcss/forms"), require("@tailwindcss/typography")],
};
```

## 2. デザイントークン
- **カラー**：`brand.primary` を CTA・プライマリアクションに使用。コンテキストによっては `danger` をフォームエラー、`success` を完了ステータスに適用。アクセントカラーは少量に留め、背景は `neutral.50/100` が基本。
- **タイポグラフィ**：本文は `hf-text-base hf-text-neutral-700`。LP ヒーローは `hf-text-3xl` 以上、ガイド記事は `@tailwindcss/typography` の `prose` を使う。
- **スペーシング**：ページ内レイアウトは 8px グリッド（`hf-p-2`, `hf-p-4`）。ヒーローやセクションは `hf-py-18`, `hf-py-22` で余白を確保。
- **シャドウ/境界**：カード/モーダルは `hf-shadow-card hf-rounded-lg`。フォーカスは `hf-shadow-focus` によりブランドカラーのアウトラインを表示。
- **レスポンシブ**：`xs` ブレークポイントで 2カラム→1カラム切替など柔軟に設計。`md` をデザインエディタの分岐基準とし、`lg` 以上で 2ペイン表示。

## 3. 共有コンポーネントカタログ
- **レイアウト系**：`LayoutBase`, `LayoutAccount`, `LayoutCheckout`。ヘッダー/フッター/モーダルコンテナを内包。レスポンシブナビは `CompNavPrimary` と `CompNavMobile`.
- **ナビゲーション**：`CompBreadcrumb`, `CompTabBar`, `CompStepper`. htmx ページャは `CompPager` で `/frags/**/table` を制御。
- **データ表示**：`CompCard`, `CompTable`, `CompEmptyState`, `CompBadge`. `CompBadge` では色バリエーションを `variant` プロパティで切替。
- **フォーム**：`CompInput`, `CompTextarea`, `CompSelect`, `CompRadioGroup`, `CompToggle`. エラー時は `hf-border-danger hf-text-danger`、補助テキストは `hf-text-neutral-500`.
- **フィードバック**：`CompToast`, `CompAlert`, `CompLoadingSpinner`, `CompSkeleton`. スケルトンは `hf-animate-pulse` + グラデーション背景を用意。
- **モーダル/ダイアログ**：`ModalFrame`（タイトル、本文、アクション slot）、`ModalConfirm`, `ModalForm`. ESC/オーバーレイ閉じ挙動は Task 014 で JS 実装。
- **CTA/ボタン**：`CompButton` が `variant (primary|secondary|ghost|danger)` と `size (sm|md|lg)` を受け取り Tailwind クラスを切替。
- **アイコン/メディア**：`CompAvatar`, `CompIcon`（SVG スプライト呼び出し）, `CompImage`（`loading="lazy"` と `srcset` を適用）。

### コンポーネント命名
- templ ファイル名は `components/{group}/{name}.templ`。`group` は `Form`, `Data`, `Layout` 等。
- フラグメント内で再利用する場合、`Frag*` から `Comp*` を呼び出す。ロジックは handler 側、スタイルはコンポーネント側に集約。

## 4. CSS/JS ガイドライン
- Tailwind ユーティリティは `hf-` プレフィックスで揃え、`class="hf-flex hf-items-center hf-gap-3"` のように読みやすい順序（レイアウト→スペーシング→タイポ→色→装飾）で並べる。
- `@apply` は `components.css` で `hf-btn-base`, `hf-input-base` など抽象化する最小限の用途に限定。大規模なレイアウトは templ 側でコンポーネント化。
- フォームバリデーションメッセージは `hf-text-sm hf-text-danger` を基本に、スクリーンリーダー向けに `role="alert"` を付与。
- htmx で差し替えられる要素に対しては `hf-transition-opacity hf-duration-normal` 等で自然にフェードさせる。長処理の場合は `hx-indicator="#spinner-id"` として `CompLoadingSpinner` を表示。
- Alpine.js は必要最小限（モーダル制御、ショートカット）に留め、複雑な状態管理は Go 側 SSR/htmx で処理。
- 静的 JS は `assets/js/{feature}.ts` に分割し、esbuild で `static/dist` へバンドル。ファイルごとに `init()` を export し、`data-module="cart"` 等で遅延初期化。
- CSS/JS の命名・構成は Task 006 で用意する npm scripts（`npm run lint:css`, `npm run lint:js`）で検証。Prettier 設定でクラス順序は手動維持とする。

## 5. コンポーネント導入手順
1. `tailwind.config.js` に token を追加し、`npm run dev:css` でビルド確認。
2. `internal/ui/components/{Group}/{Name}.templ` を作成、`Comp` プレフィックスでエクスポート。
3. `internal/handler` で ViewModel を組み立て、`Page*`/`Frag*` テンプレから呼び出してレイアウトに反映。
4. Story-like な確認として `/styleguide` ページ（後続タスク）に各コンポーネントを並べ、ビジュアル QA を可能にする。
5. 共有クラス/JS 追加時は `doc/web/tailwind_components_guidelines.md` を更新し、チームへ変更周知。

## 6. 今後の課題
- デザインシステムのバージョニング管理（`CHANGELOG`）を導入し、破壊的変更の通知を徹底。
- ダークテーマ、ハイコントラストテーマの色セットを M2 以降に検討。
- アイコンセット（Lucide など）を確定し、`CompIcon` に統一した呼び出し API を定義。
- `CompChart` 等のデータ可視化コンポーネントは必要に応じて別ライブラリ検討（軽量な Charts.js もしくは SVG テンプレ）。
