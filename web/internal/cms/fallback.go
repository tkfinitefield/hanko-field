package cms

import (
	"strings"
	"time"
)

var fallbackGuides = []Guide{
	{
		Slug:    "materials",
		Lang:    "ja",
		Title:   "はんこ素材の選び方",
		Summary: "用途別に最適な素材を解説します。",
		Body: `<p>はんこを長く使い続けるためには、用途に合わせて素材を選ぶことが大切です。仕事用か個人用か、押印頻度や保管環境などを踏まえて最適な素材を選びましょう。</p>
  <h2>使用シーンから素材を選ぶ</h2>
  <p>日常利用が多い方には耐久性の高い素材がおすすめです。儀礼的な場面では仕上がりの美しさを重視しましょう。</p>
  <h3>日常利用におすすめの素材</h3>
  <ul>
    <li>檜：軽くて手馴染みが良く、温かみのある印影が特徴です。</li>
    <li>金属：耐久性が高く、毎日の押印でも摩耗しにくい素材です。</li>
    <li>ゴム：軽量でコストパフォーマンスに優れ、社内回覧などに最適です。</li>
  </ul>
  <h2>お手入れの基本</h2>
  <p>使用後は表面の朱肉を柔らかい布で軽く拭き取り、湿気の少ない場所で保管しましょう。</p>
  <h3>朱肉を選ぶポイント</h3>
  <p>朱肉の粘度によって印影の濃さが変わります。公的書類では粘度が高いものを、イラストなどの装飾用では薄付きのものが人気です。</p>`,
		Category:           "howto",
		Personas:           []string{"maker", "manager"},
		Tags:               []string{"素材", "ケア", "朱肉"},
		HeroImageURL:       "https://placehold.co/1200x720?text=Materials",
		ReadingTimeMinutes: 6,
		Author:             Author{Name: "Hanko Field 編集部"},
		Sources:            []string{"https://cdn.hanko-field.jp/guides/materials-ja.pdf"},
		PublishAt:          time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC),
		UpdatedAt:          time.Date(2025, 1, 12, 0, 0, 0, 0, time.UTC),
		SEO: GuideSEO{
			MetaTitle:       "はんこ素材の選び方 | Hanko Field",
			MetaDescription: "用途に応じて選べるはんこ素材の特徴とお手入れ方法を紹介します。",
			OGImage:         "https://placehold.co/1200x720?text=Materials",
		},
	},
	{
		Slug:    "materials",
		Lang:    "en",
		Title:   "How to Choose Materials",
		Summary: "Pick the right material for your everyday or ceremonial seal.",
		Body: `<p>Selecting the right stamp material ensures crisp impressions and long-term durability. Start by thinking about how often you stamp and the atmosphere where the seal will live.</p>
  <h2>Match the material to your workflow</h2>
  <p>Frequent stamping calls for robust materials, while ceremonial uses lean toward finishes that highlight craftsmanship.</p>
  <h3>Daily-use favorites</h3>
  <ul>
    <li>Hinoki Cypress: Lightweight with a warm grain that softens every impression.</li>
    <li>Stainless Steel: Built for longevity—ideal for high-volume office counters.</li>
    <li>Eco Rubber: Budget-friendly and perfect for routing slips or internal memos.</li>
  </ul>
  <h2>Maintenance essentials</h2>
  <p>Wipe residual ink with a soft cloth after stamping and store the seal away from humidity or direct sunlight.</p>
  <h3>Choosing the right ink pad</h3>
  <p>Higher-viscosity ink keeps official documents sharp, while lighter formulas add nuance to creative projects.</p>`,
		Category:           "howto",
		Personas:           []string{"maker", "newcomer"},
		Tags:               []string{"materials", "care", "ink"},
		HeroImageURL:       "https://placehold.co/1200x720?text=Materials",
		ReadingTimeMinutes: 6,
		Author:             Author{Name: "Hanko Field Editorial"},
		Sources:            []string{"https://cdn.hanko-field.jp/guides/materials-en.pdf"},
		PublishAt:          time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC),
		UpdatedAt:          time.Date(2025, 1, 12, 0, 0, 0, 0, time.UTC),
		SEO: GuideSEO{
			MetaTitle:       "How to Choose Materials | Hanko Field",
			MetaDescription: "A practical walkthrough for selecting stamp materials and caring for them.",
			OGImage:         "https://placehold.co/1200x720?text=Materials",
		},
	},
	{
		Slug:    "design-basics",
		Lang:    "ja",
		Title:   "印影デザインの基本",
		Summary: "読みやすさと個性を両立するデザインのコツを紹介します。",
		Body: `<p>印影はブランドや個人を表現する大切な要素です。バランスの取れたデザインは読みやすさと個性の両立がポイントです。</p>
  <h2>レイアウトの基本</h2>
  <p>外周と内側の余白を意識し、文字ごとの重心をそろえることで安定感が生まれます。</p>
  <h3>よくあるレイアウト</h3>
  <ul>
    <li>縦書き：伝統的で公的な印象を与えます。</li>
    <li>横書き：住所や英数字を含むデザインで視認性が高まります。</li>
    <li>弧状配置：ロゴやキャッチコピーを印象的に見せたい時に活用します。</li>
  </ul>
  <h2>フォント選びと可読性</h2>
  <p>可読性を最優先に、画数の多い漢字は太めのフォントを選ぶとバランスが整います。</p>`,
		Category:           "culture",
		Personas:           []string{"maker", "manager"},
		Tags:               []string{"デザイン", "フォント", "バランス"},
		HeroImageURL:       "https://placehold.co/1200x720?text=Design",
		ReadingTimeMinutes: 5,
		Author:             Author{Name: "Creative Team"},
		Sources:            []string{"https://cdn.hanko-field.jp/guides/design-basics-ja.pdf"},
		PublishAt:          time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC),
		UpdatedAt:          time.Date(2025, 1, 6, 0, 0, 0, 0, time.UTC),
		SEO: GuideSEO{
			MetaTitle:       "印影デザインの基本 | Hanko Field",
			MetaDescription: "読みやすさを保ちながら個性を出すデザインの基本をまとめました。",
			OGImage:         "https://placehold.co/1200x720?text=Design",
		},
	},
	{
		Slug:    "design-basics",
		Lang:    "en",
		Title:   "Seal Design Basics",
		Summary: "Balance legibility with personality in every seal design.",
		Body: `<p>A well-balanced seal impression communicates trust and personality. Start with clear spacing, then layer stylistic touches.</p>
  <h2>Layout foundations</h2>
  <p>Even spacing around the border keeps the stamp feeling grounded. Align character centerlines to avoid drifting shapes.</p>
  <h3>Common arrangements</h3>
  <ul>
    <li>Vertical stacks emphasise tradition and work well for kanji names.</li>
    <li>Horizontal lines make mixed text (addresses, numbers) easy to scan.</li>
    <li>Arc layouts spotlight slogans or logotypes within corporate identities.</li>
  </ul>
  <h2>Type choices</h2>
  <p>Pick heavier cuts for dense kanji, and use rounded styles to soften a formal layout.</p>`,
		Category:           "culture",
		Personas:           []string{"maker", "creative"},
		Tags:               []string{"design", "typography", "layout"},
		HeroImageURL:       "https://placehold.co/1200x720?text=Design",
		ReadingTimeMinutes: 5,
		Author:             Author{Name: "Creative Team"},
		Sources:            []string{"https://cdn.hanko-field.jp/guides/design-basics-en.pdf"},
		PublishAt:          time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC),
		UpdatedAt:          time.Date(2025, 1, 6, 0, 0, 0, 0, time.UTC),
		SEO: GuideSEO{
			MetaTitle:       "Seal Design Basics | Hanko Field",
			MetaDescription: "Practical guidance for crafting a balanced seal layout.",
			OGImage:         "https://placehold.co/1200x720?text=Design",
		},
	},
	{
		Slug:    "size-guide",
		Lang:    "ja",
		Title:   "サイズ比較ガイド",
		Summary: "丸・角・楕円のサイズ感を比較し、用途に合った大きさを見つけましょう。",
		Body: `<p>サイズによって印影の印象や用途が大きく変わります。利用シーンを想像しながら最適なサイズを選びましょう。</p>
  <h2>代表的なサイズ一覧</h2>
  <p>Hanko Field の標準ラインナップでは丸型12mmから長方形60mmまで対応しています。</p>
  <h3>おすすめの選び方</h3>
  <ul>
    <li>丸型12mm：個人の認印として携帯しやすいサイズ。</li>
    <li>角型18mm：社内回覧や部署印に。社名ロゴを配置しても視認性が保てます。</li>
    <li>長方形40×14mm：住所や社名を入れた角印として人気です。</li>
  </ul>
  <h2>印影のシミュレーション</h2>
  <p>テンプレート機能を使えば、同じデザインで複数サイズを比較できます。</p>`,
		Category:           "faq",
		Personas:           []string{"manager", "newcomer"},
		Tags:               []string{"サイズ", "比較", "選び方"},
		HeroImageURL:       "https://placehold.co/1200x720?text=Sizes",
		ReadingTimeMinutes: 4,
		Author:             Author{Name: "Support Desk"},
		Sources:            []string{"https://cdn.hanko-field.jp/guides/size-guide-ja.pdf"},
		PublishAt:          time.Date(2024, 12, 20, 0, 0, 0, 0, time.UTC),
		UpdatedAt:          time.Date(2024, 12, 22, 0, 0, 0, 0, time.UTC),
		SEO: GuideSEO{
			MetaTitle:       "サイズ比較ガイド | Hanko Field",
			MetaDescription: "丸・角・長方形の定番サイズを用途別に比較します。",
			OGImage:         "https://placehold.co/1200x720?text=Sizes",
		},
	},
	{
		Slug:    "size-guide",
		Lang:    "en",
		Title:   "Size Comparison Guide",
		Summary: "Compare round, square, and rectangular seals to find the right fit.",
		Body: `<p>Stamp size shapes the presence of every impression. Visualise the document you’ll sign and pick the size that complements it.</p>
  <h2>Popular sizes</h2>
  <p>Our catalog spans from compact 12&nbsp;mm round seals to 60&nbsp;mm long rectangles.</p>
  <h3>Recommendations</h3>
  <ul>
    <li>Round 12&nbsp;mm: Portable for personal approvals and mail.</li>
    <li>Square 18&nbsp;mm: Bold enough for department marks with room for a logotype.</li>
    <li>Rectangular 40×14&nbsp;mm: Ideal for address blocks and invoice authentication.</li>
  </ul>
  <h2>Preview before ordering</h2>
  <p>Use the template gallery to generate side-by-side previews of multiple sizes.</p>`,
		Category:           "faq",
		Personas:           []string{"manager", "newcomer"},
		Tags:               []string{"size", "comparison", "selection"},
		HeroImageURL:       "https://placehold.co/1200x720?text=Sizes",
		ReadingTimeMinutes: 4,
		Author:             Author{Name: "Support Desk"},
		Sources:            []string{"https://cdn.hanko-field.jp/guides/size-guide-en.pdf"},
		PublishAt:          time.Date(2024, 12, 20, 0, 0, 0, 0, time.UTC),
		UpdatedAt:          time.Date(2024, 12, 22, 0, 0, 0, 0, time.UTC),
		SEO: GuideSEO{
			MetaTitle:       "Size Comparison Guide | Hanko Field",
			MetaDescription: "A quick reference for the most common seal sizes and when to use them.",
			OGImage:         "https://placehold.co/1200x720?text=Sizes",
		},
	},
}

func fallbackGuidesForLang(lang string) []Guide {
	lang = normalizeLang(lang)
	results := make([]Guide, 0, len(fallbackGuides))
	for _, g := range fallbackGuides {
		if strings.EqualFold(g.Lang, lang) {
			results = append(results, cloneGuide(g))
		}
	}
	if len(results) == 0 && lang != "en" {
		for _, g := range fallbackGuides {
			if strings.EqualFold(g.Lang, "en") {
				results = append(results, cloneGuide(g))
			}
		}
	}
	sortGuides(results)
	return results
}

func fallbackGuide(slug, lang string) (Guide, error) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	if slug == "" {
		return Guide{}, ErrNotFound
	}
	lang = normalizeLang(lang)
	candidates := []string{lang}
	if lang != "en" {
		candidates = append(candidates, "en")
	}
	if lang != "ja" {
		candidates = append(candidates, "ja")
	}
	for _, target := range candidates {
		for _, g := range fallbackGuides {
			if strings.ToLower(g.Slug) == slug && strings.EqualFold(g.Lang, target) {
				return cloneGuide(g), nil
			}
		}
	}
	return Guide{}, ErrNotFound
}
