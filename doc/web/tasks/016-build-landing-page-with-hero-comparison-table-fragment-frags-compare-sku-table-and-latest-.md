# Build landing page (`/`) with hero, comparison table fragment (`/frags/compare/sku-table`), and latest guides fragment.

**Parent Section:** 3. Landing & Exploration
**Task ID:** 016

## Goal
Build landing page with hero, comparison fragment, latest guides.

## Implementation Steps
1. Create hero section with CTA buttons linking to design flow/shop.
2. Implement comparison fragment endpoint `/frags/compare/sku-table` with filters (shape, size) and caching.
3. Implement guides fragment `/frags/guides/latest` showing localized latest guides.
4. Inject structured data (Product, Article) into page.

## UI Components
- **Layout:** `SiteLayout` with sticky `PrimaryNav` and footer `SiteFooter`.
- **Hero:** `HeroSection` featuring headline, supporting copy, CTA `PrimaryButton`, and background illustration.
- **Social proof:** `LogoStrip` carousel for partner brands right beneath hero.
- **Comparison:** `ComparisonTable` fragment mounted via htmx for `/frags/compare/sku-table` with `TableCard` wrapper.
- **Guides:** `ContentCarousel` highlighting latest guides with `GuideCard` tiles.
- **CTA band:** `CalloutBanner` with secondary CTA and trust badges.
