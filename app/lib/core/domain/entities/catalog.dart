import 'package:flutter/foundation.dart';

import 'design.dart';

enum CatalogMaterialType { horn, wood, titanium, acrylic }

enum CatalogMaterialFinish { matte, gloss, hairline }

@immutable
class CatalogMaterialSustainability {
  const CatalogMaterialSustainability({
    this.certifications = const [],
    this.notes,
  });

  final List<String> certifications;
  final String? notes;

  CatalogMaterialSustainability copyWith({
    List<String>? certifications,
    String? notes,
  }) {
    return CatalogMaterialSustainability(
      certifications: certifications ?? this.certifications,
      notes: notes ?? this.notes,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is CatalogMaterialSustainability &&
            listEquals(other.certifications, certifications) &&
            other.notes == notes);
  }

  @override
  int get hashCode => Object.hash(Object.hashAll(certifications), notes);
}

@immutable
class CatalogMaterial {
  const CatalogMaterial({
    required this.id,
    required this.name,
    required this.type,
    required this.isActive,
    required this.createdAt,
    this.finish,
    this.color,
    this.hardness,
    this.density,
    this.careNotes,
    this.sustainability,
    this.photos = const [],
    this.updatedAt,
  });

  final String id;
  final String name;
  final CatalogMaterialType type;
  final CatalogMaterialFinish? finish;
  final String? color;
  final double? hardness;
  final double? density;
  final String? careNotes;
  final CatalogMaterialSustainability? sustainability;
  final List<String> photos;
  final bool isActive;
  final DateTime createdAt;
  final DateTime? updatedAt;

  CatalogMaterial copyWith({
    String? id,
    String? name,
    CatalogMaterialType? type,
    CatalogMaterialFinish? finish,
    String? color,
    double? hardness,
    double? density,
    String? careNotes,
    CatalogMaterialSustainability? sustainability,
    List<String>? photos,
    bool? isActive,
    DateTime? createdAt,
    DateTime? updatedAt,
  }) {
    return CatalogMaterial(
      id: id ?? this.id,
      name: name ?? this.name,
      type: type ?? this.type,
      finish: finish ?? this.finish,
      color: color ?? this.color,
      hardness: hardness ?? this.hardness,
      density: density ?? this.density,
      careNotes: careNotes ?? this.careNotes,
      sustainability: sustainability ?? this.sustainability,
      photos: photos ?? this.photos,
      isActive: isActive ?? this.isActive,
      createdAt: createdAt ?? this.createdAt,
      updatedAt: updatedAt ?? this.updatedAt,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is CatalogMaterial &&
            other.id == id &&
            other.name == name &&
            other.type == type &&
            other.finish == finish &&
            other.color == color &&
            other.hardness == hardness &&
            other.density == density &&
            other.careNotes == careNotes &&
            other.sustainability == sustainability &&
            listEquals(other.photos, photos) &&
            other.isActive == isActive &&
            other.createdAt == createdAt &&
            other.updatedAt == updatedAt);
  }

  @override
  int get hashCode {
    return Object.hashAll([
      id,
      name,
      type,
      finish,
      color,
      hardness,
      density,
      careNotes,
      sustainability,
      Object.hashAll(photos),
      isActive,
      createdAt,
      updatedAt,
    ]);
  }
}

enum CatalogProductShape { round, square }

enum CatalogStockPolicy { madeToOrder, inventory }

@immutable
class CatalogMoney {
  const CatalogMoney({required this.amount, required this.currency});

  final int amount;
  final String currency;

  CatalogMoney copyWith({int? amount, String? currency}) {
    return CatalogMoney(
      amount: amount ?? this.amount,
      currency: currency ?? this.currency,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is CatalogMoney &&
            other.amount == amount &&
            other.currency == currency);
  }

  @override
  int get hashCode => Object.hash(amount, currency);
}

@immutable
class CatalogSalePrice {
  const CatalogSalePrice({
    required this.amount,
    required this.currency,
    this.startsAt,
    this.endsAt,
    this.active,
  });

  final int amount;
  final String currency;
  final DateTime? startsAt;
  final DateTime? endsAt;
  final bool? active;

  CatalogSalePrice copyWith({
    int? amount,
    String? currency,
    DateTime? startsAt,
    DateTime? endsAt,
    bool? active,
  }) {
    return CatalogSalePrice(
      amount: amount ?? this.amount,
      currency: currency ?? this.currency,
      startsAt: startsAt ?? this.startsAt,
      endsAt: endsAt ?? this.endsAt,
      active: active ?? this.active,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is CatalogSalePrice &&
            other.amount == amount &&
            other.currency == currency &&
            other.startsAt == startsAt &&
            other.endsAt == endsAt &&
            other.active == active);
  }

  @override
  int get hashCode => Object.hash(amount, currency, startsAt, endsAt, active);
}

@immutable
class CatalogProductSize {
  const CatalogProductSize({required this.mm});

  final double mm;

  CatalogProductSize copyWith({double? mm}) {
    return CatalogProductSize(mm: mm ?? this.mm);
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is CatalogProductSize && other.mm == mm);
  }

  @override
  int get hashCode => mm.hashCode;
}

@immutable
class CatalogShippingInfo {
  const CatalogShippingInfo({this.weightGr, this.boxSize});

  final int? weightGr;
  final String? boxSize;

  CatalogShippingInfo copyWith({int? weightGr, String? boxSize}) {
    return CatalogShippingInfo(
      weightGr: weightGr ?? this.weightGr,
      boxSize: boxSize ?? this.boxSize,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is CatalogShippingInfo &&
            other.weightGr == weightGr &&
            other.boxSize == boxSize);
  }

  @override
  int get hashCode => Object.hash(weightGr, boxSize);
}

@immutable
class CatalogProduct {
  const CatalogProduct({
    required this.id,
    required this.sku,
    required this.materialRef,
    required this.shape,
    required this.size,
    required this.basePrice,
    required this.stockPolicy,
    required this.isActive,
    required this.createdAt,
    this.engraveDepthMm,
    this.salePrice,
    this.stockQuantity,
    this.stockSafety,
    this.photos = const [],
    this.shipping,
    this.attributes,
    this.updatedAt,
  });

  final String id;
  final String sku;
  final String materialRef;
  final CatalogProductShape shape;
  final CatalogProductSize size;
  final CatalogMoney basePrice;
  final CatalogSalePrice? salePrice;
  final double? engraveDepthMm;
  final CatalogStockPolicy stockPolicy;
  final int? stockQuantity;
  final int? stockSafety;
  final List<String> photos;
  final CatalogShippingInfo? shipping;
  final Map<String, dynamic>? attributes;
  final bool isActive;
  final DateTime createdAt;
  final DateTime? updatedAt;

  CatalogProduct copyWith({
    String? id,
    String? sku,
    String? materialRef,
    CatalogProductShape? shape,
    CatalogProductSize? size,
    CatalogMoney? basePrice,
    CatalogSalePrice? salePrice,
    double? engraveDepthMm,
    CatalogStockPolicy? stockPolicy,
    int? stockQuantity,
    int? stockSafety,
    List<String>? photos,
    CatalogShippingInfo? shipping,
    Map<String, dynamic>? attributes,
    bool? isActive,
    DateTime? createdAt,
    DateTime? updatedAt,
  }) {
    return CatalogProduct(
      id: id ?? this.id,
      sku: sku ?? this.sku,
      materialRef: materialRef ?? this.materialRef,
      shape: shape ?? this.shape,
      size: size ?? this.size,
      basePrice: basePrice ?? this.basePrice,
      salePrice: salePrice ?? this.salePrice,
      engraveDepthMm: engraveDepthMm ?? this.engraveDepthMm,
      stockPolicy: stockPolicy ?? this.stockPolicy,
      stockQuantity: stockQuantity ?? this.stockQuantity,
      stockSafety: stockSafety ?? this.stockSafety,
      photos: photos ?? this.photos,
      shipping: shipping ?? this.shipping,
      attributes: attributes ?? this.attributes,
      isActive: isActive ?? this.isActive,
      createdAt: createdAt ?? this.createdAt,
      updatedAt: updatedAt ?? this.updatedAt,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is CatalogProduct &&
            other.id == id &&
            other.sku == sku &&
            other.materialRef == materialRef &&
            other.shape == shape &&
            other.size == size &&
            other.basePrice == basePrice &&
            other.salePrice == salePrice &&
            other.engraveDepthMm == engraveDepthMm &&
            other.stockPolicy == stockPolicy &&
            other.stockQuantity == stockQuantity &&
            other.stockSafety == stockSafety &&
            listEquals(other.photos, photos) &&
            mapEquals(other.attributes, attributes) &&
            other.shipping == shipping &&
            other.isActive == isActive &&
            other.createdAt == createdAt &&
            other.updatedAt == updatedAt);
  }

  @override
  int get hashCode {
    return Object.hashAll([
      id,
      sku,
      materialRef,
      shape,
      size,
      basePrice,
      salePrice,
      engraveDepthMm,
      stockPolicy,
      stockQuantity,
      stockSafety,
      Object.hashAll(photos),
      shipping,
      attributes == null ? null : Object.hashAll(attributes!.entries),
      isActive,
      createdAt,
      updatedAt,
    ]);
  }
}

enum CatalogLicenseType { commercial, open, custom }

enum CatalogExportPermission { none, renderPng, exportSvg, full }

@immutable
class CatalogFontLicense {
  const CatalogFontLicense({
    required this.type,
    this.uri,
    this.text,
    this.restrictions = const [],
    this.embeddable,
    this.exportPermission,
  });

  final CatalogLicenseType type;
  final String? uri;
  final String? text;
  final List<String> restrictions;
  final bool? embeddable;
  final CatalogExportPermission? exportPermission;

  CatalogFontLicense copyWith({
    CatalogLicenseType? type,
    String? uri,
    String? text,
    List<String>? restrictions,
    bool? embeddable,
    CatalogExportPermission? exportPermission,
  }) {
    return CatalogFontLicense(
      type: type ?? this.type,
      uri: uri ?? this.uri,
      text: text ?? this.text,
      restrictions: restrictions ?? this.restrictions,
      embeddable: embeddable ?? this.embeddable,
      exportPermission: exportPermission ?? this.exportPermission,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is CatalogFontLicense &&
            other.type == type &&
            other.uri == uri &&
            other.text == text &&
            listEquals(other.restrictions, restrictions) &&
            other.embeddable == embeddable &&
            other.exportPermission == exportPermission);
  }

  @override
  int get hashCode => Object.hash(
    type,
    uri,
    text,
    Object.hashAll(restrictions),
    embeddable,
    exportPermission,
  );
}

@immutable
class CatalogUnicodeRange {
  const CatalogUnicodeRange({
    required this.start,
    required this.end,
    this.label,
  });

  final String start;
  final String end;
  final String? label;

  CatalogUnicodeRange copyWith({String? start, String? end, String? label}) {
    return CatalogUnicodeRange(
      start: start ?? this.start,
      end: end ?? this.end,
      label: label ?? this.label,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is CatalogUnicodeRange &&
            other.start == start &&
            other.end == end &&
            other.label == label);
  }

  @override
  int get hashCode => Object.hash(start, end, label);
}

@immutable
class CatalogFontMetrics {
  const CatalogFontMetrics({
    this.unitsPerEm,
    this.ascent,
    this.descent,
    this.capHeight,
    this.xHeight,
    this.weightRangeMin,
    this.weightRangeMax,
  });

  final int? unitsPerEm;
  final double? ascent;
  final double? descent;
  final double? capHeight;
  final double? xHeight;
  final int? weightRangeMin;
  final int? weightRangeMax;

  CatalogFontMetrics copyWith({
    int? unitsPerEm,
    double? ascent,
    double? descent,
    double? capHeight,
    double? xHeight,
    int? weightRangeMin,
    int? weightRangeMax,
  }) {
    return CatalogFontMetrics(
      unitsPerEm: unitsPerEm ?? this.unitsPerEm,
      ascent: ascent ?? this.ascent,
      descent: descent ?? this.descent,
      capHeight: capHeight ?? this.capHeight,
      xHeight: xHeight ?? this.xHeight,
      weightRangeMin: weightRangeMin ?? this.weightRangeMin,
      weightRangeMax: weightRangeMax ?? this.weightRangeMax,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is CatalogFontMetrics &&
            other.unitsPerEm == unitsPerEm &&
            other.ascent == ascent &&
            other.descent == descent &&
            other.capHeight == capHeight &&
            other.xHeight == xHeight &&
            other.weightRangeMin == weightRangeMin &&
            other.weightRangeMax == weightRangeMax);
  }

  @override
  int get hashCode => Object.hash(
    unitsPerEm,
    ascent,
    descent,
    capHeight,
    xHeight,
    weightRangeMin,
    weightRangeMax,
  );
}

@immutable
class CatalogFont {
  const CatalogFont({
    required this.id,
    required this.family,
    required this.writing,
    required this.license,
    required this.isPublic,
    required this.createdAt,
    this.subfamily,
    this.vendor,
    this.version,
    this.designClass,
    this.glyphCoverage = const [],
    this.unicodeRanges = const [],
    this.metrics,
    this.opentypeFeatures = const [],
    this.files,
    this.previewUrl,
    this.sampleText,
    this.sort,
    this.isDeprecated = false,
    this.replacedBy,
    this.updatedAt,
  });

  final String id;
  final String family;
  final DesignWritingStyle writing;
  final CatalogFontLicense license;
  final bool isPublic;
  final DateTime createdAt;
  final String? subfamily;
  final String? vendor;
  final String? version;
  final String? designClass;
  final List<String> glyphCoverage;
  final List<CatalogUnicodeRange> unicodeRanges;
  final CatalogFontMetrics? metrics;
  final List<String> opentypeFeatures;
  final Map<String, String>? files;
  final String? previewUrl;
  final String? sampleText;
  final int? sort;
  final bool isDeprecated;
  final String? replacedBy;
  final DateTime? updatedAt;

  CatalogFont copyWith({
    String? id,
    String? family,
    DesignWritingStyle? writing,
    CatalogFontLicense? license,
    bool? isPublic,
    DateTime? createdAt,
    String? subfamily,
    String? vendor,
    String? version,
    String? designClass,
    List<String>? glyphCoverage,
    List<CatalogUnicodeRange>? unicodeRanges,
    CatalogFontMetrics? metrics,
    List<String>? opentypeFeatures,
    Map<String, String>? files,
    String? previewUrl,
    String? sampleText,
    int? sort,
    bool? isDeprecated,
    String? replacedBy,
    DateTime? updatedAt,
  }) {
    return CatalogFont(
      id: id ?? this.id,
      family: family ?? this.family,
      writing: writing ?? this.writing,
      license: license ?? this.license,
      isPublic: isPublic ?? this.isPublic,
      createdAt: createdAt ?? this.createdAt,
      subfamily: subfamily ?? this.subfamily,
      vendor: vendor ?? this.vendor,
      version: version ?? this.version,
      designClass: designClass ?? this.designClass,
      glyphCoverage: glyphCoverage ?? this.glyphCoverage,
      unicodeRanges: unicodeRanges ?? this.unicodeRanges,
      metrics: metrics ?? this.metrics,
      opentypeFeatures: opentypeFeatures ?? this.opentypeFeatures,
      files: files ?? this.files,
      previewUrl: previewUrl ?? this.previewUrl,
      sampleText: sampleText ?? this.sampleText,
      sort: sort ?? this.sort,
      isDeprecated: isDeprecated ?? this.isDeprecated,
      replacedBy: replacedBy ?? this.replacedBy,
      updatedAt: updatedAt ?? this.updatedAt,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is CatalogFont &&
            other.id == id &&
            other.family == family &&
            other.writing == writing &&
            other.license == license &&
            other.isPublic == isPublic &&
            other.createdAt == createdAt &&
            other.subfamily == subfamily &&
            other.vendor == vendor &&
            other.version == version &&
            other.designClass == designClass &&
            listEquals(other.glyphCoverage, glyphCoverage) &&
            listEquals(other.unicodeRanges, unicodeRanges) &&
            other.metrics == metrics &&
            listEquals(other.opentypeFeatures, opentypeFeatures) &&
            mapEquals(other.files, files) &&
            other.previewUrl == previewUrl &&
            other.sampleText == sampleText &&
            other.sort == sort &&
            other.isDeprecated == isDeprecated &&
            other.replacedBy == replacedBy &&
            other.updatedAt == updatedAt);
  }

  @override
  int get hashCode {
    return Object.hashAll([
      id,
      family,
      writing,
      license,
      isPublic,
      createdAt,
      subfamily,
      vendor,
      version,
      designClass,
      Object.hashAll(glyphCoverage),
      Object.hashAll(unicodeRanges),
      metrics,
      Object.hashAll(opentypeFeatures),
      files == null ? null : Object.hashAll(files!.entries),
      previewUrl,
      sampleText,
      sort,
      isDeprecated,
      replacedBy,
      updatedAt,
    ]);
  }
}

@immutable
class CatalogTemplateDefaults {
  const CatalogTemplateDefaults({
    this.sizeMm,
    this.layout,
    this.stroke,
    this.fontRef,
  });

  final double? sizeMm;
  final CatalogTemplateLayoutDefaults? layout;
  final CatalogTemplateStrokeDefaults? stroke;
  final String? fontRef;

  CatalogTemplateDefaults copyWith({
    double? sizeMm,
    CatalogTemplateLayoutDefaults? layout,
    CatalogTemplateStrokeDefaults? stroke,
    String? fontRef,
  }) {
    return CatalogTemplateDefaults(
      sizeMm: sizeMm ?? this.sizeMm,
      layout: layout ?? this.layout,
      stroke: stroke ?? this.stroke,
      fontRef: fontRef ?? this.fontRef,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is CatalogTemplateDefaults &&
            other.sizeMm == sizeMm &&
            other.layout == layout &&
            other.stroke == stroke &&
            other.fontRef == fontRef);
  }

  @override
  int get hashCode => Object.hash(sizeMm, layout, stroke, fontRef);
}

@immutable
class CatalogTemplateLayoutDefaults {
  const CatalogTemplateLayoutDefaults({
    this.grid,
    this.margin,
    this.autoKern,
    this.centerBias,
  });

  final String? grid;
  final double? margin;
  final bool? autoKern;
  final double? centerBias;

  CatalogTemplateLayoutDefaults copyWith({
    String? grid,
    double? margin,
    bool? autoKern,
    double? centerBias,
  }) {
    return CatalogTemplateLayoutDefaults(
      grid: grid ?? this.grid,
      margin: margin ?? this.margin,
      autoKern: autoKern ?? this.autoKern,
      centerBias: centerBias ?? this.centerBias,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is CatalogTemplateLayoutDefaults &&
            other.grid == grid &&
            other.margin == margin &&
            other.autoKern == autoKern &&
            other.centerBias == centerBias);
  }

  @override
  int get hashCode => Object.hash(grid, margin, autoKern, centerBias);
}

@immutable
class CatalogTemplateStrokeDefaults {
  const CatalogTemplateStrokeDefaults({this.weight, this.contrast});

  final double? weight;
  final double? contrast;

  CatalogTemplateStrokeDefaults copyWith({double? weight, double? contrast}) {
    return CatalogTemplateStrokeDefaults(
      weight: weight ?? this.weight,
      contrast: contrast ?? this.contrast,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is CatalogTemplateStrokeDefaults &&
            other.weight == weight &&
            other.contrast == contrast);
  }

  @override
  int get hashCode => Object.hash(weight, contrast);
}

@immutable
class CatalogTemplateSizeConstraint {
  const CatalogTemplateSizeConstraint({
    required this.min,
    required this.max,
    this.step,
  });

  final double min;
  final double max;
  final double? step;

  CatalogTemplateSizeConstraint copyWith({
    double? min,
    double? max,
    double? step,
  }) {
    return CatalogTemplateSizeConstraint(
      min: min ?? this.min,
      max: max ?? this.max,
      step: step ?? this.step,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is CatalogTemplateSizeConstraint &&
            other.min == min &&
            other.max == max &&
            other.step == step);
  }

  @override
  int get hashCode => Object.hash(min, max, step);
}

@immutable
class CatalogTemplateRangeConstraint {
  const CatalogTemplateRangeConstraint({this.min, this.max});

  final double? min;
  final double? max;

  CatalogTemplateRangeConstraint copyWith({double? min, double? max}) {
    return CatalogTemplateRangeConstraint(
      min: min ?? this.min,
      max: max ?? this.max,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is CatalogTemplateRangeConstraint &&
            other.min == min &&
            other.max == max);
  }

  @override
  int get hashCode => Object.hash(min, max);
}

@immutable
class CatalogTemplateGlyphConstraint {
  const CatalogTemplateGlyphConstraint({
    this.maxChars,
    this.allowRepeat,
    this.allowedScripts = const [],
    this.prohibitedChars = const [],
  });

  final int? maxChars;
  final bool? allowRepeat;
  final List<String> allowedScripts;
  final List<String> prohibitedChars;

  CatalogTemplateGlyphConstraint copyWith({
    int? maxChars,
    bool? allowRepeat,
    List<String>? allowedScripts,
    List<String>? prohibitedChars,
  }) {
    return CatalogTemplateGlyphConstraint(
      maxChars: maxChars ?? this.maxChars,
      allowRepeat: allowRepeat ?? this.allowRepeat,
      allowedScripts: allowedScripts ?? this.allowedScripts,
      prohibitedChars: prohibitedChars ?? this.prohibitedChars,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is CatalogTemplateGlyphConstraint &&
            other.maxChars == maxChars &&
            other.allowRepeat == allowRepeat &&
            listEquals(other.allowedScripts, allowedScripts) &&
            listEquals(other.prohibitedChars, prohibitedChars));
  }

  @override
  int get hashCode => Object.hash(
    maxChars,
    allowRepeat,
    Object.hashAll(allowedScripts),
    Object.hashAll(prohibitedChars),
  );
}

@immutable
class CatalogTemplateRegistrabilityHint {
  const CatalogTemplateRegistrabilityHint({
    this.jpJitsuinAllowed,
    this.bankInAllowed,
    this.notes,
  });

  final bool? jpJitsuinAllowed;
  final bool? bankInAllowed;
  final String? notes;

  CatalogTemplateRegistrabilityHint copyWith({
    bool? jpJitsuinAllowed,
    bool? bankInAllowed,
    String? notes,
  }) {
    return CatalogTemplateRegistrabilityHint(
      jpJitsuinAllowed: jpJitsuinAllowed ?? this.jpJitsuinAllowed,
      bankInAllowed: bankInAllowed ?? this.bankInAllowed,
      notes: notes ?? this.notes,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is CatalogTemplateRegistrabilityHint &&
            other.jpJitsuinAllowed == jpJitsuinAllowed &&
            other.bankInAllowed == bankInAllowed &&
            other.notes == notes);
  }

  @override
  int get hashCode => Object.hash(jpJitsuinAllowed, bankInAllowed, notes);
}

@immutable
class CatalogTemplateConstraints {
  const CatalogTemplateConstraints({
    required this.sizeMm,
    required this.strokeWeight,
    this.margin,
    this.glyph,
    this.registrability,
  });

  final CatalogTemplateSizeConstraint sizeMm;
  final CatalogTemplateRangeConstraint strokeWeight;
  final CatalogTemplateRangeConstraint? margin;
  final CatalogTemplateGlyphConstraint? glyph;
  final CatalogTemplateRegistrabilityHint? registrability;

  CatalogTemplateConstraints copyWith({
    CatalogTemplateSizeConstraint? sizeMm,
    CatalogTemplateRangeConstraint? strokeWeight,
    CatalogTemplateRangeConstraint? margin,
    CatalogTemplateGlyphConstraint? glyph,
    CatalogTemplateRegistrabilityHint? registrability,
  }) {
    return CatalogTemplateConstraints(
      sizeMm: sizeMm ?? this.sizeMm,
      strokeWeight: strokeWeight ?? this.strokeWeight,
      margin: margin ?? this.margin,
      glyph: glyph ?? this.glyph,
      registrability: registrability ?? this.registrability,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is CatalogTemplateConstraints &&
            other.sizeMm == sizeMm &&
            other.strokeWeight == strokeWeight &&
            other.margin == margin &&
            other.glyph == glyph &&
            other.registrability == registrability);
  }

  @override
  int get hashCode =>
      Object.hash(sizeMm, strokeWeight, margin, glyph, registrability);
}

@immutable
class CatalogTemplateRecommendations {
  const CatalogTemplateRecommendations({
    this.defaultSizeMm,
    this.materialRefs = const [],
    this.productRefs = const [],
  });

  final double? defaultSizeMm;
  final List<String> materialRefs;
  final List<String> productRefs;

  CatalogTemplateRecommendations copyWith({
    double? defaultSizeMm,
    List<String>? materialRefs,
    List<String>? productRefs,
  }) {
    return CatalogTemplateRecommendations(
      defaultSizeMm: defaultSizeMm ?? this.defaultSizeMm,
      materialRefs: materialRefs ?? this.materialRefs,
      productRefs: productRefs ?? this.productRefs,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is CatalogTemplateRecommendations &&
            other.defaultSizeMm == defaultSizeMm &&
            listEquals(other.materialRefs, materialRefs) &&
            listEquals(other.productRefs, productRefs));
  }

  @override
  int get hashCode => Object.hash(
    defaultSizeMm,
    Object.hashAll(materialRefs),
    Object.hashAll(productRefs),
  );
}

@immutable
class CatalogTemplate {
  const CatalogTemplate({
    required this.id,
    required this.name,
    required this.shape,
    required this.writing,
    required this.constraints,
    required this.isPublic,
    required this.sort,
    required this.createdAt,
    required this.updatedAt,
    this.slug,
    this.description,
    this.tags = const [],
    this.defaults,
    this.previewUrl,
    this.exampleImages = const [],
    this.recommendations,
    this.version,
    this.isDeprecated = false,
    this.replacedBy,
  });

  final String id;
  final String name;
  final DesignShape shape;
  final DesignWritingStyle writing;
  final CatalogTemplateConstraints constraints;
  final bool isPublic;
  final int sort;
  final DateTime createdAt;
  final DateTime updatedAt;
  final String? slug;
  final String? description;
  final List<String> tags;
  final CatalogTemplateDefaults? defaults;
  final String? previewUrl;
  final List<String> exampleImages;
  final CatalogTemplateRecommendations? recommendations;
  final String? version;
  final bool isDeprecated;
  final String? replacedBy;

  CatalogTemplate copyWith({
    String? id,
    String? name,
    DesignShape? shape,
    DesignWritingStyle? writing,
    CatalogTemplateConstraints? constraints,
    bool? isPublic,
    int? sort,
    DateTime? createdAt,
    DateTime? updatedAt,
    String? slug,
    String? description,
    List<String>? tags,
    CatalogTemplateDefaults? defaults,
    String? previewUrl,
    List<String>? exampleImages,
    CatalogTemplateRecommendations? recommendations,
    String? version,
    bool? isDeprecated,
    String? replacedBy,
  }) {
    return CatalogTemplate(
      id: id ?? this.id,
      name: name ?? this.name,
      shape: shape ?? this.shape,
      writing: writing ?? this.writing,
      constraints: constraints ?? this.constraints,
      isPublic: isPublic ?? this.isPublic,
      sort: sort ?? this.sort,
      createdAt: createdAt ?? this.createdAt,
      updatedAt: updatedAt ?? this.updatedAt,
      slug: slug ?? this.slug,
      description: description ?? this.description,
      tags: tags ?? this.tags,
      defaults: defaults ?? this.defaults,
      previewUrl: previewUrl ?? this.previewUrl,
      exampleImages: exampleImages ?? this.exampleImages,
      recommendations: recommendations ?? this.recommendations,
      version: version ?? this.version,
      isDeprecated: isDeprecated ?? this.isDeprecated,
      replacedBy: replacedBy ?? this.replacedBy,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is CatalogTemplate &&
            other.id == id &&
            other.name == name &&
            other.shape == shape &&
            other.writing == writing &&
            other.constraints == constraints &&
            other.isPublic == isPublic &&
            other.sort == sort &&
            other.createdAt == createdAt &&
            other.updatedAt == updatedAt &&
            other.slug == slug &&
            other.description == description &&
            listEquals(other.tags, tags) &&
            other.defaults == defaults &&
            other.previewUrl == previewUrl &&
            listEquals(other.exampleImages, exampleImages) &&
            other.recommendations == recommendations &&
            other.version == version &&
            other.isDeprecated == isDeprecated &&
            other.replacedBy == replacedBy);
  }

  @override
  int get hashCode {
    return Object.hashAll([
      id,
      name,
      shape,
      writing,
      constraints,
      isPublic,
      sort,
      createdAt,
      updatedAt,
      slug,
      description,
      Object.hashAll(tags),
      defaults,
      previewUrl,
      Object.hashAll(exampleImages),
      recommendations,
      version,
      isDeprecated,
      replacedBy,
    ]);
  }
}
