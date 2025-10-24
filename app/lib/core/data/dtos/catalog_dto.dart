import 'package:app/core/domain/entities/catalog.dart';
import 'package:app/core/domain/entities/design.dart';

CatalogMaterialType _parseMaterialType(String value) {
  switch (value) {
    case 'horn':
      return CatalogMaterialType.horn;
    case 'wood':
      return CatalogMaterialType.wood;
    case 'titanium':
      return CatalogMaterialType.titanium;
    case 'acrylic':
      return CatalogMaterialType.acrylic;
  }
  throw ArgumentError.value(value, 'value', 'Unknown CatalogMaterialType');
}

String _materialTypeToJson(CatalogMaterialType type) {
  switch (type) {
    case CatalogMaterialType.horn:
      return 'horn';
    case CatalogMaterialType.wood:
      return 'wood';
    case CatalogMaterialType.titanium:
      return 'titanium';
    case CatalogMaterialType.acrylic:
      return 'acrylic';
  }
}

CatalogMaterialFinish? _parseMaterialFinish(String? value) {
  switch (value) {
    case null:
      return null;
    case 'matte':
      return CatalogMaterialFinish.matte;
    case 'gloss':
      return CatalogMaterialFinish.gloss;
    case 'hairline':
      return CatalogMaterialFinish.hairline;
  }
  throw ArgumentError.value(value, 'value', 'Unknown CatalogMaterialFinish');
}

String? _materialFinishToJson(CatalogMaterialFinish? finish) {
  switch (finish) {
    case null:
      return null;
    case CatalogMaterialFinish.matte:
      return 'matte';
    case CatalogMaterialFinish.gloss:
      return 'gloss';
    case CatalogMaterialFinish.hairline:
      return 'hairline';
  }
}

CatalogProductShape _parseProductShape(String value) {
  switch (value) {
    case 'round':
      return CatalogProductShape.round;
    case 'square':
      return CatalogProductShape.square;
  }
  throw ArgumentError.value(value, 'value', 'Unknown CatalogProductShape');
}

String _productShapeToJson(CatalogProductShape shape) {
  switch (shape) {
    case CatalogProductShape.round:
      return 'round';
    case CatalogProductShape.square:
      return 'square';
  }
}

CatalogStockPolicy _parseStockPolicy(String value) {
  switch (value) {
    case 'madeToOrder':
      return CatalogStockPolicy.madeToOrder;
    case 'inventory':
      return CatalogStockPolicy.inventory;
  }
  throw ArgumentError.value(value, 'value', 'Unknown CatalogStockPolicy');
}

String _stockPolicyToJson(CatalogStockPolicy policy) {
  switch (policy) {
    case CatalogStockPolicy.madeToOrder:
      return 'madeToOrder';
    case CatalogStockPolicy.inventory:
      return 'inventory';
  }
}

DesignShape _parseDesignShape(String value) {
  switch (value) {
    case 'round':
      return DesignShape.round;
    case 'square':
      return DesignShape.square;
  }
  throw ArgumentError.value(value, 'value', 'Unknown DesignShape');
}

String _designShapeToJson(DesignShape shape) {
  switch (shape) {
    case DesignShape.round:
      return 'round';
    case DesignShape.square:
      return 'square';
  }
}

CatalogLicenseType _parseLicenseType(String value) {
  switch (value) {
    case 'commercial':
      return CatalogLicenseType.commercial;
    case 'open':
      return CatalogLicenseType.open;
    case 'custom':
      return CatalogLicenseType.custom;
  }
  throw ArgumentError.value(value, 'value', 'Unknown CatalogLicenseType');
}

String _licenseTypeToJson(CatalogLicenseType type) {
  switch (type) {
    case CatalogLicenseType.commercial:
      return 'commercial';
    case CatalogLicenseType.open:
      return 'open';
    case CatalogLicenseType.custom:
      return 'custom';
  }
}

CatalogExportPermission? _parseExportPermission(String? value) {
  switch (value) {
    case null:
      return null;
    case 'none':
      return CatalogExportPermission.none;
    case 'render_png':
    case 'renderPng':
      return CatalogExportPermission.renderPng;
    case 'export_svg':
    case 'exportSvg':
      return CatalogExportPermission.exportSvg;
    case 'full':
      return CatalogExportPermission.full;
  }
  throw ArgumentError.value(value, 'value', 'Unknown CatalogExportPermission');
}

String? _exportPermissionToJson(CatalogExportPermission? permission) {
  switch (permission) {
    case null:
      return null;
    case CatalogExportPermission.none:
      return 'none';
    case CatalogExportPermission.renderPng:
      return 'render_png';
    case CatalogExportPermission.exportSvg:
      return 'export_svg';
    case CatalogExportPermission.full:
      return 'full';
  }
}

DesignWritingStyle _parseWritingStyle(String value) {
  switch (value) {
    case 'tensho':
      return DesignWritingStyle.tensho;
    case 'reisho':
      return DesignWritingStyle.reisho;
    case 'kaisho':
      return DesignWritingStyle.kaisho;
    case 'gyosho':
      return DesignWritingStyle.gyosho;
    case 'koentai':
      return DesignWritingStyle.koentai;
    case 'custom':
      return DesignWritingStyle.custom;
  }
  throw ArgumentError.value(value, 'value', 'Unknown DesignWritingStyle');
}

String _writingStyleToJson(DesignWritingStyle style) {
  switch (style) {
    case DesignWritingStyle.tensho:
      return 'tensho';
    case DesignWritingStyle.reisho:
      return 'reisho';
    case DesignWritingStyle.kaisho:
      return 'kaisho';
    case DesignWritingStyle.gyosho:
      return 'gyosho';
    case DesignWritingStyle.koentai:
      return 'koentai';
    case DesignWritingStyle.custom:
      return 'custom';
  }
}

class CatalogMaterialSustainabilityDto {
  CatalogMaterialSustainabilityDto({this.certifications, this.notes});

  factory CatalogMaterialSustainabilityDto.fromJson(Map<String, dynamic> json) {
    return CatalogMaterialSustainabilityDto(
      certifications: (json['certifications'] as List<dynamic>?)
          ?.cast<String>(),
      notes: json['notes'] as String?,
    );
  }

  factory CatalogMaterialSustainabilityDto.fromDomain(
    CatalogMaterialSustainability? domain,
  ) {
    if (domain == null) {
      return CatalogMaterialSustainabilityDto();
    }
    return CatalogMaterialSustainabilityDto(
      certifications: domain.certifications,
      notes: domain.notes,
    );
  }

  final List<String>? certifications;
  final String? notes;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{'certifications': certifications, 'notes': notes};
  }

  CatalogMaterialSustainability toDomain() {
    return CatalogMaterialSustainability(
      certifications: certifications ?? const [],
      notes: notes,
    );
  }
}

class CatalogMaterialDto {
  CatalogMaterialDto({
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
    this.photos,
    this.updatedAt,
  });

  factory CatalogMaterialDto.fromJson(Map<String, dynamic> json) {
    return CatalogMaterialDto(
      id: json['id'] as String,
      name: json['name'] as String,
      type: json['type'] as String,
      finish: json['finish'] as String?,
      color: json['color'] as String?,
      hardness: (json['hardness'] as num?)?.toDouble(),
      density: (json['density'] as num?)?.toDouble(),
      careNotes: json['careNotes'] as String?,
      sustainability: json['sustainability'] == null
          ? null
          : CatalogMaterialSustainabilityDto.fromJson(
              json['sustainability'] as Map<String, dynamic>,
            ),
      photos: (json['photos'] as List<dynamic>?)?.cast<String>(),
      isActive: json['isActive'] as bool? ?? true,
      createdAt: json['createdAt'] as String,
      updatedAt: json['updatedAt'] as String?,
    );
  }

  factory CatalogMaterialDto.fromDomain(CatalogMaterial domain) {
    return CatalogMaterialDto(
      id: domain.id,
      name: domain.name,
      type: _materialTypeToJson(domain.type),
      finish: _materialFinishToJson(domain.finish),
      color: domain.color,
      hardness: domain.hardness,
      density: domain.density,
      careNotes: domain.careNotes,
      sustainability: domain.sustainability == null
          ? null
          : CatalogMaterialSustainabilityDto.fromDomain(domain.sustainability),
      photos: domain.photos,
      isActive: domain.isActive,
      createdAt: domain.createdAt.toIso8601String(),
      updatedAt: domain.updatedAt?.toIso8601String(),
    );
  }

  final String id;
  final String name;
  final String type;
  final String? finish;
  final String? color;
  final double? hardness;
  final double? density;
  final String? careNotes;
  final CatalogMaterialSustainabilityDto? sustainability;
  final List<String>? photos;
  final bool isActive;
  final String createdAt;
  final String? updatedAt;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'id': id,
      'name': name,
      'type': type,
      'finish': finish,
      'color': color,
      'hardness': hardness,
      'density': density,
      'careNotes': careNotes,
      'sustainability': sustainability?.toJson(),
      'photos': photos,
      'isActive': isActive,
      'createdAt': createdAt,
      'updatedAt': updatedAt,
    };
  }

  CatalogMaterial toDomain() {
    return CatalogMaterial(
      id: id,
      name: name,
      type: _parseMaterialType(type),
      finish: _parseMaterialFinish(finish),
      color: color,
      hardness: hardness,
      density: density,
      careNotes: careNotes,
      sustainability: sustainability?.toDomain(),
      photos: photos ?? const [],
      isActive: isActive,
      createdAt: DateTime.parse(createdAt),
      updatedAt: updatedAt == null ? null : DateTime.parse(updatedAt!),
    );
  }
}

class CatalogMoneyDto {
  CatalogMoneyDto({required this.amount, required this.currency});

  factory CatalogMoneyDto.fromJson(Map<String, dynamic> json) {
    return CatalogMoneyDto(
      amount: json['amount'] as int,
      currency: json['currency'] as String,
    );
  }

  factory CatalogMoneyDto.fromDomain(CatalogMoney domain) {
    return CatalogMoneyDto(amount: domain.amount, currency: domain.currency);
  }

  final int amount;
  final String currency;

  Map<String, dynamic> toJson() => <String, dynamic>{
    'amount': amount,
    'currency': currency,
  };

  CatalogMoney toDomain() => CatalogMoney(amount: amount, currency: currency);
}

class CatalogSalePriceDto {
  CatalogSalePriceDto({
    required this.amount,
    required this.currency,
    this.startsAt,
    this.endsAt,
    this.active,
  });

  factory CatalogSalePriceDto.fromJson(Map<String, dynamic> json) {
    return CatalogSalePriceDto(
      amount: json['amount'] as int,
      currency: json['currency'] as String,
      startsAt: json['startsAt'] as String?,
      endsAt: json['endsAt'] as String?,
      active: json['active'] as bool?,
    );
  }

  factory CatalogSalePriceDto.fromDomain(CatalogSalePrice domain) {
    return CatalogSalePriceDto(
      amount: domain.amount,
      currency: domain.currency,
      startsAt: domain.startsAt?.toIso8601String(),
      endsAt: domain.endsAt?.toIso8601String(),
      active: domain.active,
    );
  }

  final int amount;
  final String currency;
  final String? startsAt;
  final String? endsAt;
  final bool? active;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'amount': amount,
      'currency': currency,
      'startsAt': startsAt,
      'endsAt': endsAt,
      'active': active,
    };
  }

  CatalogSalePrice toDomain() {
    return CatalogSalePrice(
      amount: amount,
      currency: currency,
      startsAt: startsAt == null ? null : DateTime.parse(startsAt!),
      endsAt: endsAt == null ? null : DateTime.parse(endsAt!),
      active: active,
    );
  }
}

class CatalogProductSizeDto {
  CatalogProductSizeDto({required this.mm});

  factory CatalogProductSizeDto.fromJson(Map<String, dynamic> json) {
    return CatalogProductSizeDto(mm: (json['mm'] as num).toDouble());
  }

  factory CatalogProductSizeDto.fromDomain(CatalogProductSize domain) {
    return CatalogProductSizeDto(mm: domain.mm);
  }

  final double mm;

  Map<String, dynamic> toJson() => <String, dynamic>{'mm': mm};

  CatalogProductSize toDomain() => CatalogProductSize(mm: mm);
}

class CatalogShippingInfoDto {
  CatalogShippingInfoDto({this.weightGr, this.boxSize});

  factory CatalogShippingInfoDto.fromJson(Map<String, dynamic> json) {
    return CatalogShippingInfoDto(
      weightGr: json['weightGr'] as int?,
      boxSize: json['boxSize'] as String?,
    );
  }

  factory CatalogShippingInfoDto.fromDomain(CatalogShippingInfo? domain) {
    if (domain == null) {
      return CatalogShippingInfoDto();
    }
    return CatalogShippingInfoDto(
      weightGr: domain.weightGr,
      boxSize: domain.boxSize,
    );
  }

  final int? weightGr;
  final String? boxSize;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{'weightGr': weightGr, 'boxSize': boxSize};
  }

  CatalogShippingInfo toDomain() {
    return CatalogShippingInfo(weightGr: weightGr, boxSize: boxSize);
  }
}

class CatalogProductDto {
  CatalogProductDto({
    required this.id,
    required this.sku,
    required this.materialRef,
    required this.shape,
    required this.size,
    required this.basePrice,
    required this.stockPolicy,
    required this.isActive,
    required this.createdAt,
    this.salePrice,
    this.engraveDepthMm,
    this.stockQuantity,
    this.stockSafety,
    this.photos,
    this.shipping,
    this.attributes,
    this.updatedAt,
  });

  factory CatalogProductDto.fromJson(Map<String, dynamic> json) {
    return CatalogProductDto(
      id: json['id'] as String,
      sku: json['sku'] as String,
      materialRef: json['materialRef'] as String,
      shape: json['shape'] as String,
      size: CatalogProductSizeDto.fromJson(
        json['size'] as Map<String, dynamic>,
      ),
      basePrice: CatalogMoneyDto.fromJson(
        json['basePrice'] as Map<String, dynamic>,
      ),
      salePrice: json['salePrice'] == null
          ? null
          : CatalogSalePriceDto.fromJson(
              json['salePrice'] as Map<String, dynamic>,
            ),
      engraveDepthMm: (json['engraveDepthMm'] as num?)?.toDouble(),
      stockPolicy: json['stockPolicy'] as String,
      stockQuantity: json['stockQuantity'] as int?,
      stockSafety: json['stockSafety'] as int?,
      photos: (json['photos'] as List<dynamic>?)?.cast<String>(),
      shipping: json['shipping'] == null
          ? null
          : CatalogShippingInfoDto.fromJson(
              json['shipping'] as Map<String, dynamic>,
            ),
      attributes: json['attributes'] == null
          ? null
          : Map<String, dynamic>.from(json['attributes'] as Map),
      isActive: json['isActive'] as bool? ?? true,
      createdAt: json['createdAt'] as String,
      updatedAt: json['updatedAt'] as String?,
    );
  }

  factory CatalogProductDto.fromDomain(CatalogProduct domain) {
    return CatalogProductDto(
      id: domain.id,
      sku: domain.sku,
      materialRef: domain.materialRef,
      shape: _productShapeToJson(domain.shape),
      size: CatalogProductSizeDto.fromDomain(domain.size),
      basePrice: CatalogMoneyDto.fromDomain(domain.basePrice),
      salePrice: domain.salePrice == null
          ? null
          : CatalogSalePriceDto.fromDomain(domain.salePrice!),
      engraveDepthMm: domain.engraveDepthMm,
      stockPolicy: _stockPolicyToJson(domain.stockPolicy),
      stockQuantity: domain.stockQuantity,
      stockSafety: domain.stockSafety,
      photos: domain.photos,
      shipping: domain.shipping == null
          ? null
          : CatalogShippingInfoDto.fromDomain(domain.shipping),
      attributes: domain.attributes == null
          ? null
          : Map<String, dynamic>.from(domain.attributes!),
      isActive: domain.isActive,
      createdAt: domain.createdAt.toIso8601String(),
      updatedAt: domain.updatedAt?.toIso8601String(),
    );
  }

  final String id;
  final String sku;
  final String materialRef;
  final String shape;
  final CatalogProductSizeDto size;
  final CatalogMoneyDto basePrice;
  final CatalogSalePriceDto? salePrice;
  final double? engraveDepthMm;
  final String stockPolicy;
  final int? stockQuantity;
  final int? stockSafety;
  final List<String>? photos;
  final CatalogShippingInfoDto? shipping;
  final Map<String, dynamic>? attributes;
  final bool isActive;
  final String createdAt;
  final String? updatedAt;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'id': id,
      'sku': sku,
      'materialRef': materialRef,
      'shape': shape,
      'size': size.toJson(),
      'basePrice': basePrice.toJson(),
      'salePrice': salePrice?.toJson(),
      'engraveDepthMm': engraveDepthMm,
      'stockPolicy': stockPolicy,
      'stockQuantity': stockQuantity,
      'stockSafety': stockSafety,
      'photos': photos,
      'shipping': shipping?.toJson(),
      'attributes': attributes,
      'isActive': isActive,
      'createdAt': createdAt,
      'updatedAt': updatedAt,
    };
  }

  CatalogProduct toDomain() {
    return CatalogProduct(
      id: id,
      sku: sku,
      materialRef: materialRef,
      shape: _parseProductShape(shape),
      size: size.toDomain(),
      basePrice: basePrice.toDomain(),
      salePrice: salePrice?.toDomain(),
      engraveDepthMm: engraveDepthMm,
      stockPolicy: _parseStockPolicy(stockPolicy),
      stockQuantity: stockQuantity,
      stockSafety: stockSafety,
      photos: photos ?? const [],
      shipping: shipping?.toDomain(),
      attributes: attributes == null
          ? null
          : Map<String, dynamic>.from(attributes!),
      isActive: isActive,
      createdAt: DateTime.parse(createdAt),
      updatedAt: updatedAt == null ? null : DateTime.parse(updatedAt!),
    );
  }
}

class CatalogFontLicenseDto {
  CatalogFontLicenseDto({
    required this.type,
    this.uri,
    this.text,
    this.restrictions,
    this.embeddable,
    this.exportPermission,
  });

  factory CatalogFontLicenseDto.fromJson(Map<String, dynamic> json) {
    return CatalogFontLicenseDto(
      type: json['type'] as String,
      uri: json['uri'] as String?,
      text: json['text'] as String?,
      restrictions: (json['restrictions'] as List<dynamic>?)?.cast<String>(),
      embeddable: json['embeddable'] as bool?,
      exportPermission: json['exportPermission'] as String?,
    );
  }

  factory CatalogFontLicenseDto.fromDomain(CatalogFontLicense domain) {
    return CatalogFontLicenseDto(
      type: _licenseTypeToJson(domain.type),
      uri: domain.uri,
      text: domain.text,
      restrictions: domain.restrictions,
      embeddable: domain.embeddable,
      exportPermission: _exportPermissionToJson(domain.exportPermission),
    );
  }

  final String type;
  final String? uri;
  final String? text;
  final List<String>? restrictions;
  final bool? embeddable;
  final String? exportPermission;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'type': type,
      'uri': uri,
      'text': text,
      'restrictions': restrictions,
      'embeddable': embeddable,
      'exportPermission': exportPermission,
    };
  }

  CatalogFontLicense toDomain() {
    return CatalogFontLicense(
      type: _parseLicenseType(type),
      uri: uri,
      text: text,
      restrictions: restrictions ?? const [],
      embeddable: embeddable,
      exportPermission: _parseExportPermission(exportPermission),
    );
  }
}

class CatalogUnicodeRangeDto {
  CatalogUnicodeRangeDto({required this.start, required this.end, this.label});

  factory CatalogUnicodeRangeDto.fromJson(Map<String, dynamic> json) {
    return CatalogUnicodeRangeDto(
      start: json['start'] as String,
      end: json['end'] as String,
      label: json['label'] as String?,
    );
  }

  factory CatalogUnicodeRangeDto.fromDomain(CatalogUnicodeRange domain) {
    return CatalogUnicodeRangeDto(
      start: domain.start,
      end: domain.end,
      label: domain.label,
    );
  }

  final String start;
  final String end;
  final String? label;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{'start': start, 'end': end, 'label': label};
  }

  CatalogUnicodeRange toDomain() {
    return CatalogUnicodeRange(start: start, end: end, label: label);
  }
}

class CatalogFontMetricsDto {
  CatalogFontMetricsDto({
    this.unitsPerEm,
    this.ascent,
    this.descent,
    this.capHeight,
    this.xHeight,
    this.weightRangeMin,
    this.weightRangeMax,
  });

  factory CatalogFontMetricsDto.fromJson(Map<String, dynamic> json) {
    return CatalogFontMetricsDto(
      unitsPerEm: json['unitsPerEm'] as int?,
      ascent: (json['ascent'] as num?)?.toDouble(),
      descent: (json['descent'] as num?)?.toDouble(),
      capHeight: (json['capHeight'] as num?)?.toDouble(),
      xHeight: (json['xHeight'] as num?)?.toDouble(),
      weightRangeMin: json['weightRange'] == null
          ? null
          : (json['weightRange'] as Map<String, dynamic>)['min'] as int?,
      weightRangeMax: json['weightRange'] == null
          ? null
          : (json['weightRange'] as Map<String, dynamic>)['max'] as int?,
    );
  }

  factory CatalogFontMetricsDto.fromDomain(CatalogFontMetrics? domain) {
    if (domain == null) {
      return CatalogFontMetricsDto();
    }
    return CatalogFontMetricsDto(
      unitsPerEm: domain.unitsPerEm,
      ascent: domain.ascent,
      descent: domain.descent,
      capHeight: domain.capHeight,
      xHeight: domain.xHeight,
      weightRangeMin: domain.weightRangeMin,
      weightRangeMax: domain.weightRangeMax,
    );
  }

  final int? unitsPerEm;
  final double? ascent;
  final double? descent;
  final double? capHeight;
  final double? xHeight;
  final int? weightRangeMin;
  final int? weightRangeMax;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'unitsPerEm': unitsPerEm,
      'ascent': ascent,
      'descent': descent,
      'capHeight': capHeight,
      'xHeight': xHeight,
      'weightRange': weightRangeMin == null && weightRangeMax == null
          ? null
          : <String, dynamic>{'min': weightRangeMin, 'max': weightRangeMax},
    };
  }

  CatalogFontMetrics toDomain() {
    return CatalogFontMetrics(
      unitsPerEm: unitsPerEm,
      ascent: ascent,
      descent: descent,
      capHeight: capHeight,
      xHeight: xHeight,
      weightRangeMin: weightRangeMin,
      weightRangeMax: weightRangeMax,
    );
  }
}

class CatalogFontDto {
  CatalogFontDto({
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
    this.glyphCoverage,
    this.unicodeRanges,
    this.metrics,
    this.opentypeFeatures,
    this.files,
    this.previewUrl,
    this.sampleText,
    this.sort,
    this.isDeprecated = false,
    this.replacedBy,
    this.updatedAt,
  });

  factory CatalogFontDto.fromJson(Map<String, dynamic> json) {
    return CatalogFontDto(
      id: json['id'] as String,
      family: json['family'] as String,
      writing: json['writing'] as String,
      license: CatalogFontLicenseDto.fromJson(
        json['license'] as Map<String, dynamic>,
      ),
      isPublic: json['isPublic'] as bool? ?? true,
      createdAt: json['createdAt'] as String,
      subfamily: json['subfamily'] as String?,
      vendor: json['vendor'] as String?,
      version: json['version'] as String?,
      designClass: json['designClass'] as String?,
      glyphCoverage: (json['glyphCoverage'] as List<dynamic>?)?.cast<String>(),
      unicodeRanges: (json['unicodeRanges'] as List<dynamic>?)
          ?.map(
            (dynamic e) =>
                CatalogUnicodeRangeDto.fromJson(e as Map<String, dynamic>),
          )
          .toList(),
      metrics: json['metrics'] == null
          ? null
          : CatalogFontMetricsDto.fromJson(
              json['metrics'] as Map<String, dynamic>,
            ),
      opentypeFeatures:
          (json['opentype'] as Map<String, dynamic>?)?['features'] == null
          ? null
          : ((json['opentype'] as Map<String, dynamic>)['features']
                    as List<dynamic>)
                .cast<String>(),
      files: json['files'] == null
          ? null
          : Map<String, String>.from(json['files'] as Map),
      previewUrl: json['previewUrl'] as String?,
      sampleText: json['sampleText'] as String?,
      sort: json['sort'] as int?,
      isDeprecated: json['isDeprecated'] as bool? ?? false,
      replacedBy: json['replacedBy'] as String?,
      updatedAt: json['updatedAt'] as String?,
    );
  }

  factory CatalogFontDto.fromDomain(CatalogFont domain) {
    return CatalogFontDto(
      id: domain.id,
      family: domain.family,
      writing: _writingStyleToJson(domain.writing),
      license: CatalogFontLicenseDto.fromDomain(domain.license),
      isPublic: domain.isPublic,
      createdAt: domain.createdAt.toIso8601String(),
      subfamily: domain.subfamily,
      vendor: domain.vendor,
      version: domain.version,
      designClass: domain.designClass,
      glyphCoverage: domain.glyphCoverage,
      unicodeRanges: domain.unicodeRanges
          .map(CatalogUnicodeRangeDto.fromDomain)
          .toList(),
      metrics: domain.metrics == null
          ? null
          : CatalogFontMetricsDto.fromDomain(domain.metrics),
      opentypeFeatures: domain.opentypeFeatures,
      files: domain.files,
      previewUrl: domain.previewUrl,
      sampleText: domain.sampleText,
      sort: domain.sort,
      isDeprecated: domain.isDeprecated,
      replacedBy: domain.replacedBy,
      updatedAt: domain.updatedAt?.toIso8601String(),
    );
  }

  final String id;
  final String family;
  final String writing;
  final CatalogFontLicenseDto license;
  final bool isPublic;
  final String createdAt;
  final String? subfamily;
  final String? vendor;
  final String? version;
  final String? designClass;
  final List<String>? glyphCoverage;
  final List<CatalogUnicodeRangeDto>? unicodeRanges;
  final CatalogFontMetricsDto? metrics;
  final List<String>? opentypeFeatures;
  final Map<String, String>? files;
  final String? previewUrl;
  final String? sampleText;
  final int? sort;
  final bool isDeprecated;
  final String? replacedBy;
  final String? updatedAt;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'id': id,
      'family': family,
      'writing': writing,
      'license': license.toJson(),
      'isPublic': isPublic,
      'createdAt': createdAt,
      'subfamily': subfamily,
      'vendor': vendor,
      'version': version,
      'designClass': designClass,
      'glyphCoverage': glyphCoverage,
      'unicodeRanges': unicodeRanges
          ?.map((CatalogUnicodeRangeDto e) => e.toJson())
          .toList(),
      'metrics': metrics?.toJson(),
      'opentype': opentypeFeatures == null
          ? null
          : <String, dynamic>{'features': opentypeFeatures},
      'files': files,
      'previewUrl': previewUrl,
      'sampleText': sampleText,
      'sort': sort,
      'isDeprecated': isDeprecated,
      'replacedBy': replacedBy,
      'updatedAt': updatedAt,
    };
  }

  CatalogFont toDomain() {
    return CatalogFont(
      id: id,
      family: family,
      writing: _parseWritingStyle(writing),
      license: license.toDomain(),
      isPublic: isPublic,
      createdAt: DateTime.parse(createdAt),
      subfamily: subfamily,
      vendor: vendor,
      version: version,
      designClass: designClass,
      glyphCoverage: glyphCoverage ?? const [],
      unicodeRanges:
          unicodeRanges
              ?.map((CatalogUnicodeRangeDto e) => e.toDomain())
              .toList() ??
          const [],
      metrics: metrics?.toDomain(),
      opentypeFeatures: opentypeFeatures ?? const [],
      files: files == null ? null : Map<String, String>.from(files!),
      previewUrl: previewUrl,
      sampleText: sampleText,
      sort: sort,
      isDeprecated: isDeprecated,
      replacedBy: replacedBy,
      updatedAt: updatedAt == null ? null : DateTime.parse(updatedAt!),
    );
  }
}

class CatalogTemplateDefaultsDto {
  CatalogTemplateDefaultsDto({
    this.sizeMm,
    this.layout,
    this.stroke,
    this.fontRef,
  });

  factory CatalogTemplateDefaultsDto.fromJson(Map<String, dynamic> json) {
    return CatalogTemplateDefaultsDto(
      sizeMm: (json['sizeMm'] as num?)?.toDouble(),
      layout: json['layout'] == null
          ? null
          : CatalogTemplateLayoutDefaultsDto.fromJson(
              json['layout'] as Map<String, dynamic>,
            ),
      stroke: json['stroke'] == null
          ? null
          : CatalogTemplateStrokeDefaultsDto.fromJson(
              json['stroke'] as Map<String, dynamic>,
            ),
      fontRef: json['fontRef'] as String?,
    );
  }

  factory CatalogTemplateDefaultsDto.fromDomain(
    CatalogTemplateDefaults? domain,
  ) {
    if (domain == null) {
      return CatalogTemplateDefaultsDto();
    }
    return CatalogTemplateDefaultsDto(
      sizeMm: domain.sizeMm,
      layout: domain.layout == null
          ? null
          : CatalogTemplateLayoutDefaultsDto.fromDomain(domain.layout),
      stroke: domain.stroke == null
          ? null
          : CatalogTemplateStrokeDefaultsDto.fromDomain(domain.stroke),
      fontRef: domain.fontRef,
    );
  }

  final double? sizeMm;
  final CatalogTemplateLayoutDefaultsDto? layout;
  final CatalogTemplateStrokeDefaultsDto? stroke;
  final String? fontRef;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'sizeMm': sizeMm,
      'layout': layout?.toJson(),
      'stroke': stroke?.toJson(),
      'fontRef': fontRef,
    };
  }

  CatalogTemplateDefaults toDomain() {
    return CatalogTemplateDefaults(
      sizeMm: sizeMm,
      layout: layout?.toDomain(),
      stroke: stroke?.toDomain(),
      fontRef: fontRef,
    );
  }
}

class CatalogTemplateLayoutDefaultsDto {
  CatalogTemplateLayoutDefaultsDto({
    this.grid,
    this.margin,
    this.autoKern,
    this.centerBias,
  });

  factory CatalogTemplateLayoutDefaultsDto.fromJson(Map<String, dynamic> json) {
    return CatalogTemplateLayoutDefaultsDto(
      grid: json['grid'] as String?,
      margin: (json['margin'] as num?)?.toDouble(),
      autoKern: json['autoKern'] as bool?,
      centerBias: (json['centerBias'] as num?)?.toDouble(),
    );
  }

  factory CatalogTemplateLayoutDefaultsDto.fromDomain(
    CatalogTemplateLayoutDefaults? domain,
  ) {
    if (domain == null) {
      return CatalogTemplateLayoutDefaultsDto();
    }
    return CatalogTemplateLayoutDefaultsDto(
      grid: domain.grid,
      margin: domain.margin,
      autoKern: domain.autoKern,
      centerBias: domain.centerBias,
    );
  }

  final String? grid;
  final double? margin;
  final bool? autoKern;
  final double? centerBias;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'grid': grid,
      'margin': margin,
      'autoKern': autoKern,
      'centerBias': centerBias,
    };
  }

  CatalogTemplateLayoutDefaults toDomain() {
    return CatalogTemplateLayoutDefaults(
      grid: grid,
      margin: margin,
      autoKern: autoKern,
      centerBias: centerBias,
    );
  }
}

class CatalogTemplateStrokeDefaultsDto {
  CatalogTemplateStrokeDefaultsDto({this.weight, this.contrast});

  factory CatalogTemplateStrokeDefaultsDto.fromJson(Map<String, dynamic> json) {
    return CatalogTemplateStrokeDefaultsDto(
      weight: (json['weight'] as num?)?.toDouble(),
      contrast: (json['contrast'] as num?)?.toDouble(),
    );
  }

  factory CatalogTemplateStrokeDefaultsDto.fromDomain(
    CatalogTemplateStrokeDefaults? domain,
  ) {
    if (domain == null) {
      return CatalogTemplateStrokeDefaultsDto();
    }
    return CatalogTemplateStrokeDefaultsDto(
      weight: domain.weight,
      contrast: domain.contrast,
    );
  }

  final double? weight;
  final double? contrast;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{'weight': weight, 'contrast': contrast};
  }

  CatalogTemplateStrokeDefaults toDomain() {
    return CatalogTemplateStrokeDefaults(weight: weight, contrast: contrast);
  }
}

class CatalogTemplateSizeConstraintDto {
  CatalogTemplateSizeConstraintDto({
    required this.min,
    required this.max,
    this.step,
  });

  factory CatalogTemplateSizeConstraintDto.fromJson(Map<String, dynamic> json) {
    return CatalogTemplateSizeConstraintDto(
      min: (json['min'] as num).toDouble(),
      max: (json['max'] as num).toDouble(),
      step: (json['step'] as num?)?.toDouble(),
    );
  }

  factory CatalogTemplateSizeConstraintDto.fromDomain(
    CatalogTemplateSizeConstraint domain,
  ) {
    return CatalogTemplateSizeConstraintDto(
      min: domain.min,
      max: domain.max,
      step: domain.step,
    );
  }

  final double min;
  final double max;
  final double? step;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{'min': min, 'max': max, 'step': step};
  }

  CatalogTemplateSizeConstraint toDomain() =>
      CatalogTemplateSizeConstraint(min: min, max: max, step: step);
}

class CatalogTemplateRangeConstraintDto {
  CatalogTemplateRangeConstraintDto({this.min, this.max});

  factory CatalogTemplateRangeConstraintDto.fromJson(
    Map<String, dynamic> json,
  ) {
    return CatalogTemplateRangeConstraintDto(
      min: (json['min'] as num?)?.toDouble(),
      max: (json['max'] as num?)?.toDouble(),
    );
  }

  factory CatalogTemplateRangeConstraintDto.fromDomain(
    CatalogTemplateRangeConstraint? domain,
  ) {
    if (domain == null) {
      return CatalogTemplateRangeConstraintDto();
    }
    return CatalogTemplateRangeConstraintDto(min: domain.min, max: domain.max);
  }

  final double? min;
  final double? max;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{'min': min, 'max': max};
  }

  CatalogTemplateRangeConstraint toDomain() =>
      CatalogTemplateRangeConstraint(min: min, max: max);
}

class CatalogTemplateGlyphConstraintDto {
  CatalogTemplateGlyphConstraintDto({
    this.maxChars,
    this.allowRepeat,
    this.allowedScripts,
    this.prohibitedChars,
  });

  factory CatalogTemplateGlyphConstraintDto.fromJson(
    Map<String, dynamic> json,
  ) {
    return CatalogTemplateGlyphConstraintDto(
      maxChars: json['maxChars'] as int?,
      allowRepeat: json['allowRepeat'] as bool?,
      allowedScripts: (json['allowedScripts'] as List<dynamic>?)
          ?.cast<String>(),
      prohibitedChars: (json['prohibitedChars'] as List<dynamic>?)
          ?.cast<String>(),
    );
  }

  factory CatalogTemplateGlyphConstraintDto.fromDomain(
    CatalogTemplateGlyphConstraint? domain,
  ) {
    if (domain == null) {
      return CatalogTemplateGlyphConstraintDto();
    }
    return CatalogTemplateGlyphConstraintDto(
      maxChars: domain.maxChars,
      allowRepeat: domain.allowRepeat,
      allowedScripts: domain.allowedScripts,
      prohibitedChars: domain.prohibitedChars,
    );
  }

  final int? maxChars;
  final bool? allowRepeat;
  final List<String>? allowedScripts;
  final List<String>? prohibitedChars;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'maxChars': maxChars,
      'allowRepeat': allowRepeat,
      'allowedScripts': allowedScripts,
      'prohibitedChars': prohibitedChars,
    };
  }

  CatalogTemplateGlyphConstraint toDomain() {
    return CatalogTemplateGlyphConstraint(
      maxChars: maxChars,
      allowRepeat: allowRepeat,
      allowedScripts: allowedScripts ?? const [],
      prohibitedChars: prohibitedChars ?? const [],
    );
  }
}

class CatalogTemplateRegistrabilityHintDto {
  CatalogTemplateRegistrabilityHintDto({
    this.jpJitsuinAllowed,
    this.bankInAllowed,
    this.notes,
  });

  factory CatalogTemplateRegistrabilityHintDto.fromJson(
    Map<String, dynamic> json,
  ) {
    return CatalogTemplateRegistrabilityHintDto(
      jpJitsuinAllowed: json['jpJitsuinAllowed'] as bool?,
      bankInAllowed: json['bankInAllowed'] as bool?,
      notes: json['notes'] as String?,
    );
  }

  factory CatalogTemplateRegistrabilityHintDto.fromDomain(
    CatalogTemplateRegistrabilityHint? domain,
  ) {
    if (domain == null) {
      return CatalogTemplateRegistrabilityHintDto();
    }
    return CatalogTemplateRegistrabilityHintDto(
      jpJitsuinAllowed: domain.jpJitsuinAllowed,
      bankInAllowed: domain.bankInAllowed,
      notes: domain.notes,
    );
  }

  final bool? jpJitsuinAllowed;
  final bool? bankInAllowed;
  final String? notes;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'jpJitsuinAllowed': jpJitsuinAllowed,
      'bankInAllowed': bankInAllowed,
      'notes': notes,
    };
  }

  CatalogTemplateRegistrabilityHint toDomain() {
    return CatalogTemplateRegistrabilityHint(
      jpJitsuinAllowed: jpJitsuinAllowed,
      bankInAllowed: bankInAllowed,
      notes: notes,
    );
  }
}

class CatalogTemplateConstraintsDto {
  CatalogTemplateConstraintsDto({
    required this.sizeMm,
    required this.strokeWeight,
    this.margin,
    this.glyph,
    this.registrability,
  });

  factory CatalogTemplateConstraintsDto.fromJson(Map<String, dynamic> json) {
    return CatalogTemplateConstraintsDto(
      sizeMm: CatalogTemplateSizeConstraintDto.fromJson(
        json['sizeMm'] as Map<String, dynamic>,
      ),
      strokeWeight: CatalogTemplateRangeConstraintDto.fromJson(
        json['strokeWeight'] as Map<String, dynamic>,
      ),
      margin: json['margin'] == null
          ? null
          : CatalogTemplateRangeConstraintDto.fromJson(
              json['margin'] as Map<String, dynamic>,
            ),
      glyph: json['glyph'] == null
          ? null
          : CatalogTemplateGlyphConstraintDto.fromJson(
              json['glyph'] as Map<String, dynamic>,
            ),
      registrability: json['registrability'] == null
          ? null
          : CatalogTemplateRegistrabilityHintDto.fromJson(
              json['registrability'] as Map<String, dynamic>,
            ),
    );
  }

  factory CatalogTemplateConstraintsDto.fromDomain(
    CatalogTemplateConstraints domain,
  ) {
    return CatalogTemplateConstraintsDto(
      sizeMm: CatalogTemplateSizeConstraintDto.fromDomain(domain.sizeMm),
      strokeWeight: CatalogTemplateRangeConstraintDto.fromDomain(
        domain.strokeWeight,
      ),
      margin: domain.margin == null
          ? null
          : CatalogTemplateRangeConstraintDto.fromDomain(domain.margin),
      glyph: domain.glyph == null
          ? null
          : CatalogTemplateGlyphConstraintDto.fromDomain(domain.glyph),
      registrability: domain.registrability == null
          ? null
          : CatalogTemplateRegistrabilityHintDto.fromDomain(
              domain.registrability,
            ),
    );
  }

  final CatalogTemplateSizeConstraintDto sizeMm;
  final CatalogTemplateRangeConstraintDto strokeWeight;
  final CatalogTemplateRangeConstraintDto? margin;
  final CatalogTemplateGlyphConstraintDto? glyph;
  final CatalogTemplateRegistrabilityHintDto? registrability;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'sizeMm': sizeMm.toJson(),
      'strokeWeight': strokeWeight.toJson(),
      'margin': margin?.toJson(),
      'glyph': glyph?.toJson(),
      'registrability': registrability?.toJson(),
    };
  }

  CatalogTemplateConstraints toDomain() {
    return CatalogTemplateConstraints(
      sizeMm: sizeMm.toDomain(),
      strokeWeight: strokeWeight.toDomain(),
      margin: margin?.toDomain(),
      glyph: glyph?.toDomain(),
      registrability: registrability?.toDomain(),
    );
  }
}

class CatalogTemplateRecommendationsDto {
  CatalogTemplateRecommendationsDto({
    this.defaultSizeMm,
    this.materialRefs,
    this.productRefs,
  });

  factory CatalogTemplateRecommendationsDto.fromJson(
    Map<String, dynamic> json,
  ) {
    return CatalogTemplateRecommendationsDto(
      defaultSizeMm: (json['defaultSizeMm'] as num?)?.toDouble(),
      materialRefs: (json['materials'] as List<dynamic>?)?.cast<String>(),
      productRefs: (json['products'] as List<dynamic>?)?.cast<String>(),
    );
  }

  factory CatalogTemplateRecommendationsDto.fromDomain(
    CatalogTemplateRecommendations? domain,
  ) {
    if (domain == null) {
      return CatalogTemplateRecommendationsDto();
    }
    return CatalogTemplateRecommendationsDto(
      defaultSizeMm: domain.defaultSizeMm,
      materialRefs: domain.materialRefs,
      productRefs: domain.productRefs,
    );
  }

  final double? defaultSizeMm;
  final List<String>? materialRefs;
  final List<String>? productRefs;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'defaultSizeMm': defaultSizeMm,
      'materials': materialRefs,
      'products': productRefs,
    };
  }

  CatalogTemplateRecommendations toDomain() {
    return CatalogTemplateRecommendations(
      defaultSizeMm: defaultSizeMm,
      materialRefs: materialRefs ?? const [],
      productRefs: productRefs ?? const [],
    );
  }
}

class CatalogTemplateDto {
  CatalogTemplateDto({
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
    this.tags,
    this.defaults,
    this.previewUrl,
    this.exampleImages,
    this.recommendations,
    this.version,
    this.isDeprecated = false,
    this.replacedBy,
  });

  factory CatalogTemplateDto.fromJson(Map<String, dynamic> json) {
    return CatalogTemplateDto(
      id: json['id'] as String,
      name: json['name'] as String,
      shape: json['shape'] as String,
      writing: json['writing'] as String,
      constraints: CatalogTemplateConstraintsDto.fromJson(
        json['constraints'] as Map<String, dynamic>,
      ),
      isPublic: json['isPublic'] as bool? ?? true,
      sort: json['sort'] as int? ?? 0,
      createdAt: json['createdAt'] as String,
      updatedAt: json['updatedAt'] as String,
      slug: json['slug'] as String?,
      description: json['description'] as String?,
      tags: (json['tags'] as List<dynamic>?)?.cast<String>(),
      defaults: json['defaults'] == null
          ? null
          : CatalogTemplateDefaultsDto.fromJson(
              json['defaults'] as Map<String, dynamic>,
            ),
      previewUrl: json['previewUrl'] as String?,
      exampleImages: (json['exampleImages'] as List<dynamic>?)?.cast<String>(),
      recommendations: json['recommendations'] == null
          ? null
          : CatalogTemplateRecommendationsDto.fromJson(
              json['recommendations'] as Map<String, dynamic>,
            ),
      version: json['version'] as String?,
      isDeprecated: json['isDeprecated'] as bool? ?? false,
      replacedBy: json['replacedBy'] as String?,
    );
  }

  factory CatalogTemplateDto.fromDomain(CatalogTemplate domain) {
    return CatalogTemplateDto(
      id: domain.id,
      name: domain.name,
      shape: _designShapeToJson(domain.shape),
      writing: _writingStyleToJson(domain.writing),
      constraints: CatalogTemplateConstraintsDto.fromDomain(domain.constraints),
      isPublic: domain.isPublic,
      sort: domain.sort,
      createdAt: domain.createdAt.toIso8601String(),
      updatedAt: domain.updatedAt.toIso8601String(),
      slug: domain.slug,
      description: domain.description,
      tags: domain.tags,
      defaults: domain.defaults == null
          ? null
          : CatalogTemplateDefaultsDto.fromDomain(domain.defaults),
      previewUrl: domain.previewUrl,
      exampleImages: domain.exampleImages,
      recommendations: domain.recommendations == null
          ? null
          : CatalogTemplateRecommendationsDto.fromDomain(
              domain.recommendations,
            ),
      version: domain.version,
      isDeprecated: domain.isDeprecated,
      replacedBy: domain.replacedBy,
    );
  }

  final String id;
  final String name;
  final String shape;
  final String writing;
  final CatalogTemplateConstraintsDto constraints;
  final bool isPublic;
  final int sort;
  final String createdAt;
  final String updatedAt;
  final String? slug;
  final String? description;
  final List<String>? tags;
  final CatalogTemplateDefaultsDto? defaults;
  final String? previewUrl;
  final List<String>? exampleImages;
  final CatalogTemplateRecommendationsDto? recommendations;
  final String? version;
  final bool isDeprecated;
  final String? replacedBy;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'id': id,
      'name': name,
      'shape': shape,
      'writing': writing,
      'constraints': constraints.toJson(),
      'isPublic': isPublic,
      'sort': sort,
      'createdAt': createdAt,
      'updatedAt': updatedAt,
      'slug': slug,
      'description': description,
      'tags': tags,
      'defaults': defaults?.toJson(),
      'previewUrl': previewUrl,
      'exampleImages': exampleImages,
      'recommendations': recommendations?.toJson(),
      'version': version,
      'isDeprecated': isDeprecated,
      'replacedBy': replacedBy,
    };
  }

  CatalogTemplate toDomain() {
    return CatalogTemplate(
      id: id,
      name: name,
      shape: _parseDesignShape(shape),
      writing: _parseWritingStyle(writing),
      constraints: constraints.toDomain(),
      isPublic: isPublic,
      sort: sort,
      createdAt: DateTime.parse(createdAt),
      updatedAt: DateTime.parse(updatedAt),
      slug: slug,
      description: description,
      tags: tags ?? const [],
      defaults: defaults?.toDomain(),
      previewUrl: previewUrl,
      exampleImages: exampleImages ?? const [],
      recommendations: recommendations?.toDomain(),
      version: version,
      isDeprecated: isDeprecated,
      replacedBy: replacedBy,
    );
  }
}
