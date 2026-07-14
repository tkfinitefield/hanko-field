use std::{io::Cursor, sync::LazyLock};

use anyhow::{Context, Result, anyhow, bail};
use fontdue::{Font, FontSettings};
use image::{DynamicImage, ImageFormat, Rgba, RgbaImage};

use crate::seal_fonts::{SealFontProfileAsset, seal_font_profile_assets};

use super::{SealRecipeFontProfile, SealRecipeSpacing, SealShape, is_cjk_han_character};

const SEAL_RENDER_CANVAS_SIZE: u32 = 1024;
const SEAL_RENDER_CONTENT_TYPE: &str = "image/png";
const SEAL_RENDER_RED: [u8; 4] = [0x9D, 0x1F, 0x22, 0xFF];
const SEAL_RENDER_WHITE: [u8; 4] = [0xFF, 0xFF, 0xFF, 0xFF];
const SQUARE_FRAME_OUTER_INSET: u32 = 96;
const ROUND_FRAME_OUTER_INSET: u32 = 96;
const FIXED_FRAME_STROKE_WIDTH: u32 = 34;
const AIRY_GLYPH_FRAME_INNER_MARGIN: u32 = 140;
const BALANCED_GLYPH_FRAME_INNER_MARGIN: u32 = 96;
const DENSE_GLYPH_FRAME_INNER_MARGIN: u32 = 56;
const ROUND_AIRY_GLYPH_LAYOUT_HALF_SIZE: i32 = 220;
const ROUND_BALANCED_GLYPH_LAYOUT_HALF_SIZE: i32 = 248;
const ROUND_DENSE_GLYPH_LAYOUT_HALF_SIZE: i32 = 268;
const GLYPH_SIZE_FIT_STEP_PX: f32 = 24.0;
const INK_BOUNDS_ALPHA_THRESHOLD: u8 = 16;
const DEFAULT_GLYPH_FONT_SIZE_PX: f32 = 720.0;
const MIN_GLYPH_FONT_SIZE_PX: f32 = 32.0;
const MAX_GLYPH_FONT_SIZE_PX: f32 = 1600.0;

struct LoadedSealFontProfileAsset {
    key: &'static str,
    font_family: &'static str,
    font: Font,
}

static LOADED_SEAL_FONT_PROFILE_ASSETS: LazyLock<Vec<LoadedSealFontProfileAsset>> =
    LazyLock::new(|| {
        seal_font_profile_assets()
            .iter()
            .map(|asset| LoadedSealFontProfileAsset {
                key: asset.key,
                font_family: asset.font_family,
                font: load_font(asset).unwrap_or_else(|err| {
                    panic!("failed to load bundled seal font {}: {err:#}", asset.key)
                }),
            })
            .collect()
    });

#[derive(Debug, Clone)]
#[allow(dead_code)]
pub(crate) struct RenderedFixedRuleSealImage {
    pub(crate) content_type: &'static str,
    pub(crate) bytes: Vec<u8>,
    pub(crate) width: u32,
    pub(crate) height: u32,
    pub(crate) shape: SealShape,
    pub(crate) spacing: SealRecipeSpacing,
    pub(crate) font_profile: &'static str,
}

#[derive(Debug, Clone)]
#[allow(dead_code)]
pub(crate) struct RenderedSealGlyphRun {
    pub(crate) source_text: String,
    pub(crate) requested_font_profile: &'static str,
    pub(crate) font_profile: &'static str,
    pub(crate) font_family: &'static str,
    pub(crate) glyphs: Vec<RenderedSealGlyph>,
}

#[derive(Debug, Clone)]
#[allow(dead_code)]
pub(crate) struct RenderedSealGlyph {
    pub(crate) character: char,
    pub(crate) glyph_index: u16,
    pub(crate) bitmap_width: usize,
    pub(crate) bitmap_height: usize,
    pub(crate) x_min: i32,
    pub(crate) y_min: i32,
    pub(crate) advance_width: f32,
    pub(crate) bitmap: Vec<u8>,
}

#[allow(dead_code)]
pub(crate) fn render_seal_glyphs_from_real_font(
    kanji: &str,
    requested_profile: SealRecipeFontProfile,
) -> Result<RenderedSealGlyphRun> {
    render_seal_glyphs_from_real_font_at_px(kanji, requested_profile, DEFAULT_GLYPH_FONT_SIZE_PX)
}

#[allow(dead_code)]
pub(crate) fn render_fixed_rule_seal_png(
    kanji: &str,
    requested_profile: SealRecipeFontProfile,
    shape: SealShape,
) -> Result<RenderedFixedRuleSealImage> {
    render_fixed_rule_seal_png_with_spacing(
        kanji,
        requested_profile,
        shape,
        SealRecipeSpacing::Balanced,
    )
}

#[allow(dead_code)]
pub(crate) fn render_fixed_rule_seal_png_with_spacing(
    kanji: &str,
    requested_profile: SealRecipeFontProfile,
    shape: SealShape,
    spacing: SealRecipeSpacing,
) -> Result<RenderedFixedRuleSealImage> {
    let glyph_run = render_glyph_run_fitting_fixed_frame(kanji, requested_profile, shape, spacing)?;
    let mut canvas = RgbaImage::from_pixel(
        SEAL_RENDER_CANVAS_SIZE,
        SEAL_RENDER_CANVAS_SIZE,
        Rgba(SEAL_RENDER_WHITE),
    );

    draw_fixed_single_frame(&mut canvas, shape);
    draw_red_glyph_run(&mut canvas, &glyph_run, shape, spacing)?;

    let mut bytes = Vec::new();
    DynamicImage::ImageRgba8(canvas)
        .write_to(&mut Cursor::new(&mut bytes), ImageFormat::Png)
        .context("failed to encode fixed-rule seal png")?;

    Ok(RenderedFixedRuleSealImage {
        content_type: SEAL_RENDER_CONTENT_TYPE,
        bytes,
        width: SEAL_RENDER_CANVAS_SIZE,
        height: SEAL_RENDER_CANVAS_SIZE,
        shape,
        spacing,
        font_profile: glyph_run.font_profile,
    })
}

fn render_glyph_run_fitting_fixed_frame(
    kanji: &str,
    requested_profile: SealRecipeFontProfile,
    shape: SealShape,
    spacing: SealRecipeSpacing,
) -> Result<RenderedSealGlyphRun> {
    let mut font_size = glyph_font_size_for_text(kanji, spacing)?;
    loop {
        let glyph_run =
            render_seal_glyphs_from_real_font_at_px(kanji, requested_profile, font_size)?;
        if glyph_run_fits_fixed_frame(&glyph_run, shape, spacing)? {
            return Ok(glyph_run);
        }

        let next_font_size = font_size - GLYPH_SIZE_FIT_STEP_PX;
        if next_font_size < MIN_GLYPH_FONT_SIZE_PX {
            bail!("glyph run does not fit fixed frame interior at minimum font size");
        }
        font_size = next_font_size;
    }
}

#[allow(dead_code)]
pub(crate) fn render_seal_glyphs_from_real_font_at_px(
    kanji: &str,
    requested_profile: SealRecipeFontProfile,
    font_size_px: f32,
) -> Result<RenderedSealGlyphRun> {
    validate_glyph_font_size(font_size_px)?;
    let selected_text = validate_selected_kanji_text(kanji)?;
    let chars = selected_text.chars().collect::<Vec<_>>();

    let mut failures = Vec::new();
    for profile_key in font_profile_fallback_order(requested_profile) {
        let asset = loaded_seal_font_profile_asset(profile_key)?;
        match render_glyph_run_with_asset(
            &selected_text,
            &chars,
            requested_profile,
            asset,
            font_size_px,
        ) {
            Ok(run) => return Ok(run),
            Err(err) => failures.push(format!("{}: {err:#}", asset.key)),
        }
    }

    bail!(
        "selected kanji could not be rendered by any approved real font: {}",
        failures.join("; ")
    )
}

fn validate_glyph_font_size(font_size_px: f32) -> Result<()> {
    if !(MIN_GLYPH_FONT_SIZE_PX..=MAX_GLYPH_FONT_SIZE_PX).contains(&font_size_px) {
        bail!(
            "font_size_px must be in range {}-{}",
            MIN_GLYPH_FONT_SIZE_PX,
            MAX_GLYPH_FONT_SIZE_PX
        );
    }
    Ok(())
}

fn glyph_font_size_for_text(kanji: &str, spacing: SealRecipeSpacing) -> Result<f32> {
    let char_count = kanji.chars().count();
    match (char_count, spacing) {
        (1, SealRecipeSpacing::Airy) => Ok(500.0),
        (1, SealRecipeSpacing::Balanced) => Ok(560.0),
        (1, SealRecipeSpacing::Dense) => Ok(640.0),
        (2, SealRecipeSpacing::Airy) => Ok(330.0),
        (2, SealRecipeSpacing::Balanced) => Ok(390.0),
        (2, SealRecipeSpacing::Dense) => Ok(460.0),
        _ => bail!("selected kanji must be 1 or 2 characters"),
    }
}

fn validate_selected_kanji_text(kanji: &str) -> Result<String> {
    if kanji.trim() != kanji {
        bail!("selected kanji must not include surrounding whitespace");
    }
    if kanji.is_empty() {
        bail!("selected kanji is required");
    }
    if kanji.chars().any(char::is_whitespace) {
        bail!("selected kanji must not contain whitespace");
    }

    let char_count = kanji.chars().count();
    if char_count == 0 || char_count > 2 {
        bail!("selected kanji must be 1 or 2 characters");
    }
    if !kanji.chars().all(is_cjk_han_character) {
        bail!("selected kanji must contain only CJK Han characters");
    }

    Ok(kanji.to_owned())
}

fn font_profile_fallback_order(requested_profile: SealRecipeFontProfile) -> Vec<&'static str> {
    let requested = requested_profile.as_str();
    let mut profiles = Vec::new();
    for candidate in [requested, "formal_serif", "soft_sans"] {
        if !profiles.contains(&candidate) {
            profiles.push(candidate);
        }
    }
    profiles
}

fn seal_font_profile_asset(profile_key: &str) -> Result<&'static SealFontProfileAsset> {
    seal_font_profile_assets()
        .iter()
        .find(|asset| asset.key == profile_key)
        .ok_or_else(|| anyhow!("seal font profile asset not found: {profile_key}"))
}

fn loaded_seal_font_profile_asset(
    profile_key: &str,
) -> Result<&'static LoadedSealFontProfileAsset> {
    LOADED_SEAL_FONT_PROFILE_ASSETS
        .iter()
        .find(|asset| asset.key == profile_key)
        .ok_or_else(|| anyhow!("loaded seal font profile asset not found: {profile_key}"))
}

fn load_font(asset: &SealFontProfileAsset) -> Result<Font> {
    Font::from_bytes(asset.bytes, FontSettings::default())
        .map_err(|err| anyhow!("failed to load {} font bytes: {err}", asset.key))
}

fn render_glyph_run_with_asset(
    selected_text: &str,
    chars: &[char],
    requested_profile: SealRecipeFontProfile,
    asset: &'static LoadedSealFontProfileAsset,
    font_size_px: f32,
) -> Result<RenderedSealGlyphRun> {
    let mut glyphs = Vec::with_capacity(chars.len());

    for character in chars {
        let glyph_index = asset.font.lookup_glyph_index(*character);
        if glyph_index == 0 {
            bail!(
                "font profile {} does not contain selected glyph {}",
                asset.key,
                format_unicode_scalar(*character)
            );
        }

        let (metrics, bitmap) = asset.font.rasterize(*character, font_size_px);
        if metrics.width == 0 || metrics.height == 0 || bitmap.iter().all(|alpha| *alpha == 0) {
            bail!(
                "font profile {} produced empty glyph bitmap for {}",
                asset.key,
                format_unicode_scalar(*character)
            );
        }

        glyphs.push(RenderedSealGlyph {
            character: *character,
            glyph_index,
            bitmap_width: metrics.width,
            bitmap_height: metrics.height,
            x_min: metrics.xmin,
            y_min: metrics.ymin,
            advance_width: metrics.advance_width,
            bitmap,
        });
    }

    Ok(RenderedSealGlyphRun {
        source_text: selected_text.to_owned(),
        requested_font_profile: requested_profile.as_str(),
        font_profile: asset.key,
        font_family: asset.font_family,
        glyphs,
    })
}

fn draw_fixed_single_frame(canvas: &mut RgbaImage, shape: SealShape) {
    match shape {
        SealShape::Square => draw_square_frame(canvas),
        SealShape::Round => draw_round_frame(canvas),
    }
}

fn draw_square_frame(canvas: &mut RgbaImage) {
    let outer_left = SQUARE_FRAME_OUTER_INSET;
    let outer_top = SQUARE_FRAME_OUTER_INSET;
    let outer_right = SEAL_RENDER_CANVAS_SIZE - SQUARE_FRAME_OUTER_INSET - 1;
    let outer_bottom = SEAL_RENDER_CANVAS_SIZE - SQUARE_FRAME_OUTER_INSET - 1;
    let inner_left = outer_left + FIXED_FRAME_STROKE_WIDTH;
    let inner_top = outer_top + FIXED_FRAME_STROKE_WIDTH;
    let inner_right = outer_right - FIXED_FRAME_STROKE_WIDTH;
    let inner_bottom = outer_bottom - FIXED_FRAME_STROKE_WIDTH;

    for y in outer_top..=outer_bottom {
        for x in outer_left..=outer_right {
            let in_outer =
                x >= outer_left && x <= outer_right && y >= outer_top && y <= outer_bottom;
            let in_inner =
                x >= inner_left && x <= inner_right && y >= inner_top && y <= inner_bottom;
            if in_outer && !in_inner {
                canvas.put_pixel(x, y, Rgba(SEAL_RENDER_RED));
            }
        }
    }
}

fn draw_round_frame(canvas: &mut RgbaImage) {
    let center = (SEAL_RENDER_CANVAS_SIZE as f32 - 1.0) / 2.0;
    let outer_radius = (SEAL_RENDER_CANVAS_SIZE - ROUND_FRAME_OUTER_INSET * 2) as f32 / 2.0;
    let inner_radius = outer_radius - FIXED_FRAME_STROKE_WIDTH as f32;
    let outer_radius_sq = outer_radius * outer_radius;
    let inner_radius_sq = inner_radius * inner_radius;

    for y in 0..SEAL_RENDER_CANVAS_SIZE {
        for x in 0..SEAL_RENDER_CANVAS_SIZE {
            let dx = x as f32 - center;
            let dy = y as f32 - center;
            let distance_sq = dx * dx + dy * dy;
            if distance_sq <= outer_radius_sq && distance_sq >= inner_radius_sq {
                canvas.put_pixel(x, y, Rgba(SEAL_RENDER_RED));
            }
        }
    }
}

fn draw_red_glyph_run(
    canvas: &mut RgbaImage,
    glyph_run: &RenderedSealGlyphRun,
    shape: SealShape,
    spacing: SealRecipeSpacing,
) -> Result<()> {
    let layout = layout_glyph_run_by_ink(glyph_run)?;
    let target_bounds = fixed_frame_layout_bounds(shape, spacing);
    if layout.ink_bounds.width() > target_bounds.width()
        || layout.ink_bounds.height() > target_bounds.height()
    {
        bail!("glyph run does not fit fixed frame interior");
    }

    let offset_x = target_bounds.left
        + ((target_bounds.width() as i32 - layout.ink_bounds.width() as i32) / 2)
        - layout.ink_bounds.left;
    let offset_y = target_bounds.top
        + ((target_bounds.height() as i32 - layout.ink_bounds.height() as i32) / 2)
        - layout.ink_bounds.top;

    for placement in layout.placements {
        let glyph = &glyph_run.glyphs[placement.glyph_index];
        draw_red_glyph_bitmap(
            canvas,
            glyph,
            placement.origin_x + offset_x,
            placement.origin_y + offset_y,
        );
    }

    Ok(())
}

fn glyph_run_fits_fixed_frame(
    glyph_run: &RenderedSealGlyphRun,
    shape: SealShape,
    spacing: SealRecipeSpacing,
) -> Result<bool> {
    let layout = layout_glyph_run_by_ink(glyph_run)?;
    let target_bounds = fixed_frame_layout_bounds(shape, spacing);
    Ok(layout.ink_bounds.width() <= target_bounds.width()
        && layout.ink_bounds.height() <= target_bounds.height())
}

#[derive(Debug, Clone, Copy)]
struct InkBounds {
    left: i32,
    top: i32,
    right: i32,
    bottom: i32,
}

impl InkBounds {
    fn width(self) -> u32 {
        (self.right - self.left) as u32
    }

    fn height(self) -> u32 {
        (self.bottom - self.top) as u32
    }

    fn union(self, other: Self) -> Self {
        Self {
            left: self.left.min(other.left),
            top: self.top.min(other.top),
            right: self.right.max(other.right),
            bottom: self.bottom.max(other.bottom),
        }
    }
}

#[derive(Debug, Clone, Copy)]
struct LayoutBounds {
    left: i32,
    top: i32,
    right: i32,
    bottom: i32,
}

impl LayoutBounds {
    fn width(self) -> u32 {
        (self.right - self.left) as u32
    }

    fn height(self) -> u32 {
        (self.bottom - self.top) as u32
    }
}

#[derive(Debug, Clone, Copy)]
struct GlyphBitmapPlacement {
    glyph_index: usize,
    origin_x: i32,
    origin_y: i32,
}

#[derive(Debug, Clone)]
struct GlyphRunInkLayout {
    placements: Vec<GlyphBitmapPlacement>,
    ink_bounds: InkBounds,
}

fn layout_glyph_run_by_ink(glyph_run: &RenderedSealGlyphRun) -> Result<GlyphRunInkLayout> {
    let glyph_gap = glyph_gap_for_len(glyph_run.glyphs.len());
    let glyph_ink_bounds = glyph_run
        .glyphs
        .iter()
        .map(glyph_ink_bounds)
        .collect::<Result<Vec<_>>>()?;
    let max_ink_height = glyph_ink_bounds
        .iter()
        .map(|bounds| bounds.height())
        .max()
        .ok_or_else(|| anyhow!("glyph run has no drawable bitmap content"))?;

    let mut placements = Vec::with_capacity(glyph_run.glyphs.len());
    let mut pen_x = 0_i32;
    let mut layout_ink_bounds: Option<InkBounds> = None;
    for (glyph_index, bounds) in glyph_ink_bounds.iter().enumerate() {
        let ink_top = ((max_ink_height - bounds.height()) / 2) as i32;
        let origin_x = pen_x - bounds.left;
        let origin_y = ink_top - bounds.top;
        let placed_bounds = InkBounds {
            left: origin_x + bounds.left,
            top: origin_y + bounds.top,
            right: origin_x + bounds.right,
            bottom: origin_y + bounds.bottom,
        };

        placements.push(GlyphBitmapPlacement {
            glyph_index,
            origin_x,
            origin_y,
        });
        layout_ink_bounds = Some(match layout_ink_bounds {
            Some(current) => current.union(placed_bounds),
            None => placed_bounds,
        });
        pen_x += bounds.width() as i32 + glyph_gap as i32;
    }

    let ink_bounds =
        layout_ink_bounds.ok_or_else(|| anyhow!("glyph run has no drawable bitmap content"))?;
    Ok(GlyphRunInkLayout {
        placements,
        ink_bounds,
    })
}

fn glyph_ink_bounds(glyph: &RenderedSealGlyph) -> Result<InkBounds> {
    let mut left = glyph.bitmap_width as i32;
    let mut top = glyph.bitmap_height as i32;
    let mut right = 0_i32;
    let mut bottom = 0_i32;

    for y in 0..glyph.bitmap_height {
        for x in 0..glyph.bitmap_width {
            let alpha = glyph.bitmap[y * glyph.bitmap_width + x];
            if alpha < INK_BOUNDS_ALPHA_THRESHOLD {
                continue;
            }
            left = left.min(x as i32);
            top = top.min(y as i32);
            right = right.max(x as i32 + 1);
            bottom = bottom.max(y as i32 + 1);
        }
    }

    if left >= right || top >= bottom {
        bail!("glyph has no ink pixels above layout threshold");
    }
    Ok(InkBounds {
        left,
        top,
        right,
        bottom,
    })
}

fn glyph_gap_for_len(len: usize) -> u32 {
    if len <= 1 { 0 } else { 24 }
}

fn fixed_frame_layout_bounds(shape: SealShape, spacing: SealRecipeSpacing) -> LayoutBounds {
    match shape {
        SealShape::Square => {
            let frame_inner_edge = SQUARE_FRAME_OUTER_INSET + FIXED_FRAME_STROKE_WIDTH;
            let inset = (frame_inner_edge + glyph_spacing_margin(spacing)) as i32;
            LayoutBounds {
                left: inset,
                top: inset,
                right: SEAL_RENDER_CANVAS_SIZE as i32 - inset,
                bottom: SEAL_RENDER_CANVAS_SIZE as i32 - inset,
            }
        }
        SealShape::Round => {
            let half_size = round_glyph_layout_half_size(spacing);
            let center = SEAL_RENDER_CANVAS_SIZE as i32 / 2;
            LayoutBounds {
                left: center - half_size,
                top: center - half_size,
                right: center + half_size,
                bottom: center + half_size,
            }
        }
    }
}

fn glyph_spacing_margin(spacing: SealRecipeSpacing) -> u32 {
    match spacing {
        SealRecipeSpacing::Airy => AIRY_GLYPH_FRAME_INNER_MARGIN,
        SealRecipeSpacing::Balanced => BALANCED_GLYPH_FRAME_INNER_MARGIN,
        SealRecipeSpacing::Dense => DENSE_GLYPH_FRAME_INNER_MARGIN,
    }
}

fn round_glyph_layout_half_size(spacing: SealRecipeSpacing) -> i32 {
    match spacing {
        SealRecipeSpacing::Airy => ROUND_AIRY_GLYPH_LAYOUT_HALF_SIZE,
        SealRecipeSpacing::Balanced => ROUND_BALANCED_GLYPH_LAYOUT_HALF_SIZE,
        SealRecipeSpacing::Dense => ROUND_DENSE_GLYPH_LAYOUT_HALF_SIZE,
    }
}

fn draw_red_glyph_bitmap(
    canvas: &mut RgbaImage,
    glyph: &RenderedSealGlyph,
    origin_x: i32,
    origin_y: i32,
) {
    for y in 0..glyph.bitmap_height {
        for x in 0..glyph.bitmap_width {
            let alpha = glyph.bitmap[y * glyph.bitmap_width + x];
            if alpha == 0 {
                continue;
            }
            let canvas_x = origin_x + x as i32;
            let canvas_y = origin_y + y as i32;
            if canvas_x < 0
                || canvas_y < 0
                || canvas_x >= SEAL_RENDER_CANVAS_SIZE as i32
                || canvas_y >= SEAL_RENDER_CANVAS_SIZE as i32
            {
                continue;
            }
            blend_red_over_white(canvas, canvas_x as u32, canvas_y as u32, alpha);
        }
    }
}

fn blend_red_over_white(canvas: &mut RgbaImage, x: u32, y: u32, alpha: u8) {
    let alpha = alpha as u16;
    let inv_alpha = 255 - alpha;
    let red =
        ((SEAL_RENDER_RED[0] as u16 * alpha) + (SEAL_RENDER_WHITE[0] as u16 * inv_alpha)) / 255;
    let green =
        ((SEAL_RENDER_RED[1] as u16 * alpha) + (SEAL_RENDER_WHITE[1] as u16 * inv_alpha)) / 255;
    let blue =
        ((SEAL_RENDER_RED[2] as u16 * alpha) + (SEAL_RENDER_WHITE[2] as u16 * inv_alpha)) / 255;
    canvas.put_pixel(x, y, Rgba([red as u8, green as u8, blue as u8, 255]));
}

fn format_unicode_scalar(character: char) -> String {
    format!("U+{:04X}", character as u32)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn m14_t04_renders_exact_selected_unicode_kanji_from_real_font() {
        let run = render_seal_glyphs_from_real_font_at_px(
            "美空",
            SealRecipeFontProfile::FormalSerif,
            192.0,
        )
        .expect("selected kanji should render");

        assert_eq!(run.source_text, "美空");
        assert_eq!(run.requested_font_profile, "formal_serif");
        assert_eq!(run.font_profile, "formal_serif");
        assert_eq!(run.font_family, "Noto Serif JP");
        assert_eq!(
            run.glyphs
                .iter()
                .map(|glyph| glyph.character)
                .collect::<String>(),
            "美空"
        );
        for glyph in run.glyphs {
            assert_ne!(glyph.glyph_index, 0);
            assert!(glyph.bitmap_width > 0);
            assert!(glyph.bitmap_height > 0);
            assert_eq!(glyph.bitmap.len(), glyph.bitmap_width * glyph.bitmap_height);
            assert!(glyph.bitmap.iter().any(|alpha| *alpha > 0));
            assert!(glyph.advance_width > 0.0);
            let _bounds_origin = (glyph.x_min, glyph.y_min);
        }
    }

    #[test]
    fn m14_t04_rejects_non_kanji_text_before_font_rendering() {
        let err =
            render_seal_glyphs_from_real_font_at_px("美A", SealRecipeFontProfile::SoftSans, 192.0)
                .expect_err("non-kanji text must fail");

        assert!(
            err.to_string()
                .contains("selected kanji must contain only CJK Han characters"),
            "unexpected error: {err:#}"
        );
    }

    #[test]
    fn m14_t04_falls_back_to_approved_font_when_profile_lacks_selected_glyph() {
        let fallback_char = find_character_missing_from_primary_but_present_in_fallback(
            "bold_brush",
            "formal_serif",
        )
        .expect("test fonts should expose a fallback-only kanji");
        let selected = fallback_char.to_string();

        let run = render_seal_glyphs_from_real_font_at_px(
            &selected,
            SealRecipeFontProfile::BoldBrush,
            192.0,
        )
        .expect("fallback font should render selected kanji");

        assert_eq!(run.source_text, selected);
        assert_eq!(run.requested_font_profile, "bold_brush");
        assert_eq!(run.font_profile, "formal_serif");
        assert_eq!(run.glyphs.len(), 1);
        assert_eq!(run.glyphs[0].character, fallback_char);
        assert_ne!(run.glyphs[0].glyph_index, 0);
    }

    #[test]
    fn m14_t04_rejects_selected_kanji_missing_from_all_approved_fonts() {
        let missing_char = find_character_missing_from_all_fonts()
            .expect("test candidate set should include one unsupported CJK character");
        let selected = missing_char.to_string();

        let err = render_seal_glyphs_from_real_font_at_px(
            &selected,
            SealRecipeFontProfile::ClassicSeal,
            192.0,
        )
        .expect_err("unsupported selected kanji must fail");

        assert!(
            err.to_string()
                .contains("selected kanji could not be rendered by any approved real font"),
            "unexpected error: {err:#}"
        );
    }

    #[test]
    fn m14_t05_renders_square_png_with_fixed_red_frame_and_white_background() {
        let first =
            render_fixed_rule_seal_png("美", SealRecipeFontProfile::FormalSerif, SealShape::Square)
                .expect("square seal should render");
        let second =
            render_fixed_rule_seal_png("空", SealRecipeFontProfile::SoftSans, SealShape::Square)
                .expect("second square seal should render");

        assert_eq!(first.shape, SealShape::Square);
        assert_eq!(first.spacing, SealRecipeSpacing::Balanced);
        assert_eq!(first.font_profile, "formal_serif");
        assert_eq!(second.shape, SealShape::Square);
        assert_eq!(second.spacing, SealRecipeSpacing::Balanced);
        assert_eq!(second.font_profile, "soft_sans");
        let first = decode_fixed_rule_image(&first);
        let second = decode_fixed_rule_image(&second);
        assert_fixed_square_frame(&first);
        assert_fixed_square_frame(&second);
        assert_red_or_white_palette(&first);
        assert_red_or_white_palette(&second);
        assert!(count_inner_red_pixels(&first) > 1_000);
        assert!(count_inner_red_pixels(&second) > 1_000);
    }

    #[test]
    fn m14_t05_renders_round_png_with_fixed_single_red_frame_and_white_background() {
        let first =
            render_fixed_rule_seal_png("美", SealRecipeFontProfile::FormalSerif, SealShape::Round)
                .expect("round seal should render");
        let second =
            render_fixed_rule_seal_png("空", SealRecipeFontProfile::SoftSans, SealShape::Round)
                .expect("second round seal should render");

        assert_eq!(first.shape, SealShape::Round);
        assert_eq!(first.spacing, SealRecipeSpacing::Balanced);
        assert_eq!(first.font_profile, "formal_serif");
        assert_eq!(second.shape, SealShape::Round);
        assert_eq!(second.spacing, SealRecipeSpacing::Balanced);
        assert_eq!(second.font_profile, "soft_sans");
        let first = decode_fixed_rule_image(&first);
        let second = decode_fixed_rule_image(&second);
        assert_fixed_round_frame(&first);
        assert_fixed_round_frame(&second);
        assert_red_or_white_palette(&first);
        assert_red_or_white_palette(&second);
        assert!(count_inner_red_pixels(&first) > 1_000);
        assert!(count_inner_red_pixels(&second) > 1_000);
    }

    #[test]
    fn m14_t05_fits_two_character_text_inside_fixed_frame() {
        let square = render_fixed_rule_seal_png(
            "美空",
            SealRecipeFontProfile::FormalSerif,
            SealShape::Square,
        )
        .expect("two-character square seal should render");
        let round = render_fixed_rule_seal_png(
            "美空",
            SealRecipeFontProfile::FormalSerif,
            SealShape::Round,
        )
        .expect("two-character round seal should render");

        let square = decode_fixed_rule_image(&square);
        let round = decode_fixed_rule_image(&round);
        assert_fixed_square_frame(&square);
        assert_fixed_round_frame(&round);
        assert_red_or_white_palette(&square);
        assert_red_or_white_palette(&round);
        assert!(count_inner_red_pixels(&square) > 1_000);
        assert!(count_inner_red_pixels(&round) > 1_000);
    }

    #[test]
    fn m14_t06_centers_ink_for_shapes_spacings_and_character_counts() {
        for shape in [SealShape::Square, SealShape::Round] {
            for spacing in [
                SealRecipeSpacing::Airy,
                SealRecipeSpacing::Balanced,
                SealRecipeSpacing::Dense,
            ] {
                for selected_text in ["美", "美空"] {
                    let rendered = render_fixed_rule_seal_png_with_spacing(
                        selected_text,
                        SealRecipeFontProfile::FormalSerif,
                        shape,
                        spacing,
                    )
                    .expect("seal should render with requested layout");
                    assert_eq!(rendered.shape, shape);
                    assert_eq!(rendered.spacing, spacing);

                    let image = decode_fixed_rule_image(&rendered);
                    match shape {
                        SealShape::Square => assert_fixed_square_frame(&image),
                        SealShape::Round => assert_fixed_round_frame(&image),
                    }
                    assert_red_or_white_palette(&image);
                    let bounds = assert_glyph_ink_centered_and_inside_target(
                        &image,
                        shape,
                        spacing,
                        selected_text,
                    );
                    assert!(
                        bounds.width() > 80 && bounds.height() > 80,
                        "rendered glyph ink should remain legible for {selected_text:?}"
                    );
                }
            }
        }
    }

    #[test]
    fn m14_t06_spacing_changes_visible_ink_size() {
        let airy = render_fixed_rule_seal_png_with_spacing(
            "美",
            SealRecipeFontProfile::FormalSerif,
            SealShape::Square,
            SealRecipeSpacing::Airy,
        )
        .expect("airy seal should render");
        let balanced = render_fixed_rule_seal_png_with_spacing(
            "美",
            SealRecipeFontProfile::FormalSerif,
            SealShape::Square,
            SealRecipeSpacing::Balanced,
        )
        .expect("balanced seal should render");
        let dense = render_fixed_rule_seal_png_with_spacing(
            "美",
            SealRecipeFontProfile::FormalSerif,
            SealShape::Square,
            SealRecipeSpacing::Dense,
        )
        .expect("dense seal should render");

        let airy_bounds = assert_glyph_ink_centered_and_inside_target(
            &decode_fixed_rule_image(&airy),
            SealShape::Square,
            SealRecipeSpacing::Airy,
            "美",
        );
        let balanced_bounds = assert_glyph_ink_centered_and_inside_target(
            &decode_fixed_rule_image(&balanced),
            SealShape::Square,
            SealRecipeSpacing::Balanced,
            "美",
        );
        let dense_bounds = assert_glyph_ink_centered_and_inside_target(
            &decode_fixed_rule_image(&dense),
            SealShape::Square,
            SealRecipeSpacing::Dense,
            "美",
        );

        assert!(airy_bounds.height() < balanced_bounds.height());
        assert!(balanced_bounds.height() < dense_bounds.height());
        assert!(airy_bounds.width() < balanced_bounds.width());
        assert!(balanced_bounds.width() < dense_bounds.width());
    }

    fn find_character_missing_from_primary_but_present_in_fallback(
        primary_key: &str,
        fallback_key: &str,
    ) -> Option<char> {
        let primary = load_font(seal_font_profile_asset(primary_key).ok()?).ok()?;
        let fallback = load_font(seal_font_profile_asset(fallback_key).ok()?).ok()?;
        fallback_test_candidates().into_iter().find(|character| {
            primary.lookup_glyph_index(*character) == 0
                && fallback.lookup_glyph_index(*character) != 0
        })
    }

    fn find_character_missing_from_all_fonts() -> Option<char> {
        let fonts = seal_font_profile_assets()
            .iter()
            .map(load_font)
            .collect::<Result<Vec<_>>>()
            .ok()?;
        unsupported_test_candidates().into_iter().find(|character| {
            fonts
                .iter()
                .all(|font| font.lookup_glyph_index(*character) == 0)
        })
    }

    fn fallback_test_candidates() -> Vec<char> {
        vec![
            '龘', '麤', '鱻', '灩', '靐', '齉', '纛', '爨', '齾', '鬱', '髙', '﨑', '㐂', '㐀',
            '㐁', '㐄',
        ]
    }

    fn unsupported_test_candidates() -> Vec<char> {
        vec!['㐀', '㐁', '㐄', '㐅', '㐆', '㐇', '㐈', '㐉', '䶵', '䶴']
    }

    fn decode_fixed_rule_image(image: &RenderedFixedRuleSealImage) -> RgbaImage {
        assert_eq!(image.content_type, SEAL_RENDER_CONTENT_TYPE);
        assert_eq!(image.width, SEAL_RENDER_CANVAS_SIZE);
        assert_eq!(image.height, SEAL_RENDER_CANVAS_SIZE);
        assert!(image.bytes.starts_with(b"\x89PNG\r\n\x1a\n"));
        image::load_from_memory_with_format(&image.bytes, ImageFormat::Png)
            .expect("fixed-rule seal png should decode")
            .to_rgba8()
    }

    fn assert_fixed_square_frame(image: &RgbaImage) {
        assert_eq!(image.width(), SEAL_RENDER_CANVAS_SIZE);
        assert_eq!(image.height(), SEAL_RENDER_CANVAS_SIZE);
        assert_eq!(*image.get_pixel(0, 0), Rgba(SEAL_RENDER_WHITE));
        assert_eq!(
            *image.get_pixel(SQUARE_FRAME_OUTER_INSET, SEAL_RENDER_CANVAS_SIZE / 2),
            Rgba(SEAL_RENDER_RED)
        );
        assert_eq!(
            *image.get_pixel(
                SEAL_RENDER_CANVAS_SIZE - SQUARE_FRAME_OUTER_INSET - 1,
                SEAL_RENDER_CANVAS_SIZE / 2
            ),
            Rgba(SEAL_RENDER_RED)
        );
        assert_eq!(
            *image.get_pixel(SEAL_RENDER_CANVAS_SIZE / 2, SQUARE_FRAME_OUTER_INSET),
            Rgba(SEAL_RENDER_RED)
        );
        assert_eq!(
            *image.get_pixel(
                SQUARE_FRAME_OUTER_INSET + FIXED_FRAME_STROKE_WIDTH + 16,
                SQUARE_FRAME_OUTER_INSET + FIXED_FRAME_STROKE_WIDTH + 16
            ),
            Rgba(SEAL_RENDER_WHITE)
        );
        assert_eq!(
            count_exact_red_segments_on_row(
                image,
                SQUARE_FRAME_OUTER_INSET + FIXED_FRAME_STROKE_WIDTH / 2
            ),
            1
        );
    }

    fn assert_fixed_round_frame(image: &RgbaImage) {
        assert_eq!(image.width(), SEAL_RENDER_CANVAS_SIZE);
        assert_eq!(image.height(), SEAL_RENDER_CANVAS_SIZE);
        assert_eq!(*image.get_pixel(0, 0), Rgba(SEAL_RENDER_WHITE));
        assert_eq!(
            *image.get_pixel(ROUND_FRAME_OUTER_INSET, ROUND_FRAME_OUTER_INSET),
            Rgba(SEAL_RENDER_WHITE)
        );
        assert_eq!(
            *image.get_pixel(SEAL_RENDER_CANVAS_SIZE / 2, ROUND_FRAME_OUTER_INSET),
            Rgba(SEAL_RENDER_RED)
        );
        assert_eq!(
            *image.get_pixel(ROUND_FRAME_OUTER_INSET, SEAL_RENDER_CANVAS_SIZE / 2),
            Rgba(SEAL_RENDER_RED)
        );
        assert_eq!(
            *image.get_pixel(
                SEAL_RENDER_CANVAS_SIZE / 2,
                ROUND_FRAME_OUTER_INSET + FIXED_FRAME_STROKE_WIDTH + 20
            ),
            Rgba(SEAL_RENDER_WHITE)
        );
        assert_eq!(
            *image.get_pixel(
                SEAL_RENDER_CANVAS_SIZE / 2,
                SEAL_RENDER_CANVAS_SIZE - ROUND_FRAME_OUTER_INSET - FIXED_FRAME_STROKE_WIDTH - 20
            ),
            Rgba(SEAL_RENDER_WHITE)
        );
    }

    fn assert_red_or_white_palette(image: &RgbaImage) {
        for pixel in image.pixels() {
            let [red, green, blue, alpha] = pixel.0;
            assert_eq!(alpha, 255);
            if pixel.0 == SEAL_RENDER_WHITE || pixel.0 == SEAL_RENDER_RED {
                continue;
            }
            assert!(
                red >= green
                    && red >= blue
                    && green >= SEAL_RENDER_RED[1]
                    && blue >= SEAL_RENDER_RED[2],
                "pixel must be red antialias or white, got rgba({red},{green},{blue},{alpha})"
            );
        }
    }

    fn count_inner_red_pixels(image: &RgbaImage) -> usize {
        let start =
            SQUARE_FRAME_OUTER_INSET + FIXED_FRAME_STROKE_WIDTH + BALANCED_GLYPH_FRAME_INNER_MARGIN;
        let end = SEAL_RENDER_CANVAS_SIZE - start;
        let mut count = 0;
        for y in start..end {
            for x in start..end {
                let [red, green, blue, alpha] = image.get_pixel(x, y).0;
                if alpha == 255
                    && red >= green
                    && red >= blue
                    && image.get_pixel(x, y).0 != SEAL_RENDER_WHITE
                {
                    count += 1;
                }
            }
        }
        count
    }

    fn assert_glyph_ink_centered_and_inside_target(
        image: &RgbaImage,
        shape: SealShape,
        spacing: SealRecipeSpacing,
        selected_text: &str,
    ) -> InkBounds {
        let target = fixed_frame_layout_bounds(shape, spacing);
        let search_bounds = fixed_frame_inner_search_bounds(shape);
        let ink_bounds = rendered_glyph_ink_bounds(image, shape, search_bounds)
            .unwrap_or_else(|| panic!("rendered glyph ink should exist for {selected_text:?}"));
        let tolerance = 6;

        assert!(
            ink_bounds.left >= target.left - tolerance,
            "{selected_text:?} {shape:?} {spacing:?} ink bounds {ink_bounds:?} should be inside target {target:?}"
        );
        assert!(
            ink_bounds.top >= target.top - tolerance,
            "{selected_text:?} {shape:?} {spacing:?} ink bounds {ink_bounds:?} should be inside target {target:?}"
        );
        assert!(
            ink_bounds.right <= target.right + tolerance,
            "{selected_text:?} {shape:?} {spacing:?} ink bounds {ink_bounds:?} should be inside target {target:?}"
        );
        assert!(
            ink_bounds.bottom <= target.bottom + tolerance,
            "{selected_text:?} {shape:?} {spacing:?} ink bounds {ink_bounds:?} should be inside target {target:?}"
        );

        let ink_center_x = (ink_bounds.left + ink_bounds.right) as f32 / 2.0;
        let ink_center_y = (ink_bounds.top + ink_bounds.bottom) as f32 / 2.0;
        let target_center_x = (target.left + target.right) as f32 / 2.0;
        let target_center_y = (target.top + target.bottom) as f32 / 2.0;
        assert!(
            (ink_center_x - target_center_x).abs() <= 2.0,
            "{selected_text:?} {shape:?} {spacing:?} ink center x {ink_center_x} should match target {target_center_x}"
        );
        assert!(
            (ink_center_y - target_center_y).abs() <= 6.0,
            "{selected_text:?} {shape:?} {spacing:?} ink center y {ink_center_y} should match target {target_center_y}"
        );

        ink_bounds
    }

    fn fixed_frame_inner_search_bounds(shape: SealShape) -> LayoutBounds {
        let frame_inner_edge = match shape {
            SealShape::Square => SQUARE_FRAME_OUTER_INSET + FIXED_FRAME_STROKE_WIDTH + 1,
            SealShape::Round => ROUND_FRAME_OUTER_INSET + FIXED_FRAME_STROKE_WIDTH + 1,
        } as i32;
        LayoutBounds {
            left: frame_inner_edge,
            top: frame_inner_edge,
            right: SEAL_RENDER_CANVAS_SIZE as i32 - frame_inner_edge,
            bottom: SEAL_RENDER_CANVAS_SIZE as i32 - frame_inner_edge,
        }
    }

    fn rendered_glyph_ink_bounds(
        image: &RgbaImage,
        shape: SealShape,
        search_bounds: LayoutBounds,
    ) -> Option<InkBounds> {
        let mut bounds: Option<InkBounds> = None;
        for y in search_bounds.top..search_bounds.bottom {
            for x in search_bounds.left..search_bounds.right {
                if !is_inside_frame_open_area(x, y, shape) {
                    continue;
                }
                if !is_rendered_ink_pixel_for_bounds(image.get_pixel(x as u32, y as u32)) {
                    continue;
                }
                let pixel_bounds = InkBounds {
                    left: x,
                    top: y,
                    right: x + 1,
                    bottom: y + 1,
                };
                bounds = Some(match bounds {
                    Some(current) => current.union(pixel_bounds),
                    None => pixel_bounds,
                });
            }
        }
        bounds
    }

    fn is_inside_frame_open_area(x: i32, y: i32, shape: SealShape) -> bool {
        match shape {
            SealShape::Square => true,
            SealShape::Round => {
                let center = (SEAL_RENDER_CANVAS_SIZE as f32 - 1.0) / 2.0;
                let inner_radius = (SEAL_RENDER_CANVAS_SIZE - ROUND_FRAME_OUTER_INSET * 2) as f32
                    / 2.0
                    - FIXED_FRAME_STROKE_WIDTH as f32
                    - 1.0;
                let dx = x as f32 - center;
                let dy = y as f32 - center;
                dx * dx + dy * dy < inner_radius * inner_radius
            }
        }
    }

    fn is_rendered_ink_pixel_for_bounds(pixel: &Rgba<u8>) -> bool {
        let [red, green, blue, alpha] = pixel.0;
        alpha == 255
            && pixel.0 != SEAL_RENDER_WHITE
            && red >= green
            && red >= blue
            && green <= 240
            && blue <= 241
    }

    fn count_exact_red_segments_on_row(image: &RgbaImage, y: u32) -> usize {
        let mut count = 0;
        let mut in_segment = false;
        for x in 0..SEAL_RENDER_CANVAS_SIZE {
            let red = *image.get_pixel(x, y) == Rgba(SEAL_RENDER_RED);
            if red && !in_segment {
                count += 1;
                in_segment = true;
            } else if !red {
                in_segment = false;
            }
        }
        count
    }
}
