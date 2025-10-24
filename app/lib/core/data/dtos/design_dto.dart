import 'package:app/core/domain/entities/design.dart';

DesignStatus _parseDesignStatus(String value) {
  switch (value) {
    case 'draft':
      return DesignStatus.draft;
    case 'ready':
      return DesignStatus.ready;
    case 'ordered':
      return DesignStatus.ordered;
    case 'locked':
      return DesignStatus.locked;
  }
  throw ArgumentError.value(value, 'value', 'Unknown DesignStatus');
}

String _designStatusToJson(DesignStatus status) {
  switch (status) {
    case DesignStatus.draft:
      return 'draft';
    case DesignStatus.ready:
      return 'ready';
    case DesignStatus.ordered:
      return 'ordered';
    case DesignStatus.locked:
      return 'locked';
  }
}

DesignSourceType _parseDesignSourceType(String value) {
  switch (value) {
    case 'typed':
      return DesignSourceType.typed;
    case 'uploaded':
      return DesignSourceType.uploaded;
    case 'logo':
      return DesignSourceType.logo;
  }
  throw ArgumentError.value(value, 'value', 'Unknown DesignSourceType');
}

String _designSourceTypeToJson(DesignSourceType type) {
  switch (type) {
    case DesignSourceType.typed:
      return 'typed';
    case DesignSourceType.uploaded:
      return 'uploaded';
    case DesignSourceType.logo:
      return 'logo';
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

DesignWritingStyle _parseDesignWritingStyle(String value) {
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

String _designWritingStyleToJson(DesignWritingStyle style) {
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

class DesignKanjiMappingDto {
  DesignKanjiMappingDto({required this.value, this.mappingRef});

  factory DesignKanjiMappingDto.fromJson(Map<String, dynamic> json) {
    return DesignKanjiMappingDto(
      value: json['value'] as String,
      mappingRef: json['mappingRef'] as String?,
    );
  }

  factory DesignKanjiMappingDto.fromDomain(DesignKanjiMapping domain) {
    return DesignKanjiMappingDto(
      value: domain.value,
      mappingRef: domain.mappingRef,
    );
  }

  final String value;
  final String? mappingRef;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{'value': value, 'mappingRef': mappingRef};
  }

  DesignKanjiMapping toDomain() {
    return DesignKanjiMapping(value: value, mappingRef: mappingRef);
  }
}

class DesignInputDto {
  DesignInputDto({required this.sourceType, required this.rawName, this.kanji});

  factory DesignInputDto.fromJson(Map<String, dynamic> json) {
    return DesignInputDto(
      sourceType: json['sourceType'] as String,
      rawName: json['rawName'] as String,
      kanji: json['kanji'] == null
          ? null
          : DesignKanjiMappingDto.fromJson(
              json['kanji'] as Map<String, dynamic>,
            ),
    );
  }

  factory DesignInputDto.fromDomain(DesignInput domain) {
    return DesignInputDto(
      sourceType: _designSourceTypeToJson(domain.sourceType),
      rawName: domain.rawName,
      kanji: domain.kanji == null
          ? null
          : DesignKanjiMappingDto.fromDomain(domain.kanji!),
    );
  }

  final String sourceType;
  final String rawName;
  final DesignKanjiMappingDto? kanji;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'sourceType': sourceType,
      'rawName': rawName,
      'kanji': kanji?.toJson(),
    };
  }

  DesignInput toDomain() {
    return DesignInput(
      sourceType: _parseDesignSourceType(sourceType),
      rawName: rawName,
      kanji: kanji?.toDomain(),
    );
  }
}

class DesignSizeDto {
  DesignSizeDto({required this.mm});

  factory DesignSizeDto.fromJson(Map<String, dynamic> json) {
    return DesignSizeDto(mm: (json['mm'] as num).toDouble());
  }

  factory DesignSizeDto.fromDomain(DesignSize domain) {
    return DesignSizeDto(mm: domain.mm);
  }

  final double mm;

  Map<String, dynamic> toJson() => <String, dynamic>{'mm': mm};

  DesignSize toDomain() => DesignSize(mm: mm);
}

class DesignStrokeDto {
  DesignStrokeDto({this.weight, this.contrast});

  factory DesignStrokeDto.fromJson(Map<String, dynamic> json) {
    return DesignStrokeDto(
      weight: (json['weight'] as num?)?.toDouble(),
      contrast: (json['contrast'] as num?)?.toDouble(),
    );
  }

  factory DesignStrokeDto.fromDomain(DesignStroke? domain) {
    if (domain == null) {
      return DesignStrokeDto();
    }
    return DesignStrokeDto(weight: domain.weight, contrast: domain.contrast);
  }

  final double? weight;
  final double? contrast;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{'weight': weight, 'contrast': contrast};
  }

  DesignStroke toDomain() {
    return DesignStroke(weight: weight, contrast: contrast);
  }
}

class DesignLayoutDto {
  DesignLayoutDto({this.grid, this.margin});

  factory DesignLayoutDto.fromJson(Map<String, dynamic> json) {
    return DesignLayoutDto(
      grid: json['grid'] as String?,
      margin: (json['margin'] as num?)?.toDouble(),
    );
  }

  factory DesignLayoutDto.fromDomain(DesignLayout? domain) {
    if (domain == null) {
      return DesignLayoutDto();
    }
    return DesignLayoutDto(grid: domain.grid, margin: domain.margin);
  }

  final String? grid;
  final double? margin;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{'grid': grid, 'margin': margin};
  }

  DesignLayout toDomain() {
    return DesignLayout(grid: grid, margin: margin);
  }
}

class DesignStyleDto {
  DesignStyleDto({
    required this.writing,
    this.fontRef,
    this.templateRef,
    this.stroke,
    this.layout,
  });

  factory DesignStyleDto.fromJson(Map<String, dynamic> json) {
    return DesignStyleDto(
      writing: json['writing'] as String,
      fontRef: json['fontRef'] as String?,
      templateRef: json['templateRef'] as String?,
      stroke: json['stroke'] == null
          ? null
          : DesignStrokeDto.fromJson(json['stroke'] as Map<String, dynamic>),
      layout: json['layout'] == null
          ? null
          : DesignLayoutDto.fromJson(json['layout'] as Map<String, dynamic>),
    );
  }

  factory DesignStyleDto.fromDomain(DesignStyle domain) {
    return DesignStyleDto(
      writing: _designWritingStyleToJson(domain.writing),
      fontRef: domain.fontRef,
      templateRef: domain.templateRef,
      stroke: domain.stroke == null
          ? null
          : DesignStrokeDto.fromDomain(domain.stroke),
      layout: domain.layout == null
          ? null
          : DesignLayoutDto.fromDomain(domain.layout),
    );
  }

  final String writing;
  final String? fontRef;
  final String? templateRef;
  final DesignStrokeDto? stroke;
  final DesignLayoutDto? layout;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'writing': writing,
      'fontRef': fontRef,
      'templateRef': templateRef,
      'stroke': stroke?.toJson(),
      'layout': layout?.toJson(),
    };
  }

  DesignStyle toDomain() {
    return DesignStyle(
      writing: _parseDesignWritingStyle(writing),
      fontRef: fontRef,
      templateRef: templateRef,
      stroke: stroke?.toDomain(),
      layout: layout?.toDomain(),
    );
  }
}

class DesignAiMetadataDto {
  DesignAiMetadataDto({
    this.enabled,
    this.lastJobRef,
    this.qualityScore,
    this.registrable,
    this.diagnostics,
  });

  factory DesignAiMetadataDto.fromJson(Map<String, dynamic> json) {
    return DesignAiMetadataDto(
      enabled: json['enabled'] as bool?,
      lastJobRef: json['lastJobRef'] as String?,
      qualityScore: (json['qualityScore'] as num?)?.toDouble(),
      registrable: json['registrable'] as bool?,
      diagnostics: (json['diagnostics'] as List<dynamic>?)?.cast<String>(),
    );
  }

  factory DesignAiMetadataDto.fromDomain(DesignAiMetadata? domain) {
    if (domain == null) {
      return DesignAiMetadataDto();
    }
    return DesignAiMetadataDto(
      enabled: domain.enabled,
      lastJobRef: domain.lastJobRef,
      qualityScore: domain.qualityScore,
      registrable: domain.registrable,
      diagnostics: domain.diagnostics,
    );
  }

  final bool? enabled;
  final String? lastJobRef;
  final double? qualityScore;
  final bool? registrable;
  final List<String>? diagnostics;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'enabled': enabled,
      'lastJobRef': lastJobRef,
      'qualityScore': qualityScore,
      'registrable': registrable,
      'diagnostics': diagnostics,
    };
  }

  DesignAiMetadata toDomain() {
    return DesignAiMetadata(
      enabled: enabled,
      lastJobRef: lastJobRef,
      qualityScore: qualityScore,
      registrable: registrable,
      diagnostics: diagnostics ?? const [],
    );
  }
}

class DesignAssetsDto {
  DesignAssetsDto({
    this.vectorSvg,
    this.previewPng,
    this.previewPngUrl,
    this.stampMockUrl,
  });

  factory DesignAssetsDto.fromJson(Map<String, dynamic> json) {
    return DesignAssetsDto(
      vectorSvg: json['vectorSvg'] as String?,
      previewPng: json['previewPng'] as String?,
      previewPngUrl: json['previewPngUrl'] as String?,
      stampMockUrl: json['stampMockUrl'] as String?,
    );
  }

  factory DesignAssetsDto.fromDomain(DesignAssets? domain) {
    if (domain == null) {
      return DesignAssetsDto();
    }
    return DesignAssetsDto(
      vectorSvg: domain.vectorSvg,
      previewPng: domain.previewPng,
      previewPngUrl: domain.previewPngUrl,
      stampMockUrl: domain.stampMockUrl,
    );
  }

  final String? vectorSvg;
  final String? previewPng;
  final String? previewPngUrl;
  final String? stampMockUrl;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'vectorSvg': vectorSvg,
      'previewPng': previewPng,
      'previewPngUrl': previewPngUrl,
      'stampMockUrl': stampMockUrl,
    };
  }

  DesignAssets toDomain() {
    return DesignAssets(
      vectorSvg: vectorSvg,
      previewPng: previewPng,
      previewPngUrl: previewPngUrl,
      stampMockUrl: stampMockUrl,
    );
  }
}

class DesignDto {
  DesignDto({
    required this.id,
    required this.ownerRef,
    required this.status,
    required this.shape,
    required this.size,
    required this.style,
    required this.version,
    required this.createdAt,
    required this.updatedAt,
    this.input,
    this.ai,
    this.assets,
    this.hash,
    this.lastOrderedAt,
  });

  factory DesignDto.fromJson(Map<String, dynamic> json) {
    return DesignDto(
      id: json['id'] as String,
      ownerRef: json['ownerRef'] as String,
      status: json['status'] as String,
      shape: json['shape'] as String,
      size: DesignSizeDto.fromJson(json['size'] as Map<String, dynamic>),
      style: DesignStyleDto.fromJson(json['style'] as Map<String, dynamic>),
      version: json['version'] as int,
      createdAt: json['createdAt'] as String,
      updatedAt: json['updatedAt'] as String,
      input: json['input'] == null
          ? null
          : DesignInputDto.fromJson(json['input'] as Map<String, dynamic>),
      ai: json['ai'] == null
          ? null
          : DesignAiMetadataDto.fromJson(json['ai'] as Map<String, dynamic>),
      assets: json['assets'] == null
          ? null
          : DesignAssetsDto.fromJson(json['assets'] as Map<String, dynamic>),
      hash: json['hash'] as String?,
      lastOrderedAt: json['lastOrderedAt'] as String?,
    );
  }

  factory DesignDto.fromDomain(Design domain) {
    return DesignDto(
      id: domain.id,
      ownerRef: domain.ownerRef,
      status: _designStatusToJson(domain.status),
      shape: _designShapeToJson(domain.shape),
      size: DesignSizeDto.fromDomain(domain.size),
      style: DesignStyleDto.fromDomain(domain.style),
      version: domain.version,
      createdAt: domain.createdAt.toIso8601String(),
      updatedAt: domain.updatedAt.toIso8601String(),
      input: domain.input == null
          ? null
          : DesignInputDto.fromDomain(domain.input!),
      ai: domain.ai == null ? null : DesignAiMetadataDto.fromDomain(domain.ai),
      assets: domain.assets == null
          ? null
          : DesignAssetsDto.fromDomain(domain.assets),
      hash: domain.hash,
      lastOrderedAt: domain.lastOrderedAt?.toIso8601String(),
    );
  }

  final String id;
  final String ownerRef;
  final String status;
  final String shape;
  final DesignSizeDto size;
  final DesignStyleDto style;
  final int version;
  final String createdAt;
  final String updatedAt;
  final DesignInputDto? input;
  final DesignAiMetadataDto? ai;
  final DesignAssetsDto? assets;
  final String? hash;
  final String? lastOrderedAt;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'id': id,
      'ownerRef': ownerRef,
      'status': status,
      'shape': shape,
      'size': size.toJson(),
      'style': style.toJson(),
      'version': version,
      'createdAt': createdAt,
      'updatedAt': updatedAt,
      'input': input?.toJson(),
      'ai': ai?.toJson(),
      'assets': assets?.toJson(),
      'hash': hash,
      'lastOrderedAt': lastOrderedAt,
    };
  }

  Design toDomain() {
    return Design(
      id: id,
      ownerRef: ownerRef,
      status: _parseDesignStatus(status),
      shape: _parseDesignShape(shape),
      size: size.toDomain(),
      style: style.toDomain(),
      version: version,
      createdAt: DateTime.parse(createdAt),
      updatedAt: DateTime.parse(updatedAt),
      input: input?.toDomain(),
      ai: ai?.toDomain(),
      assets: assets?.toDomain(),
      hash: hash,
      lastOrderedAt: lastOrderedAt == null
          ? null
          : DateTime.parse(lastOrderedAt!),
    );
  }
}
