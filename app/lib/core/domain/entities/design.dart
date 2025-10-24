import 'package:flutter/foundation.dart';

enum DesignStatus { draft, ready, ordered, locked }

enum DesignSourceType { typed, uploaded, logo }

enum DesignShape { round, square }

enum DesignWritingStyle { tensho, reisho, kaisho, gyosho, koentai, custom }

@immutable
class DesignKanjiMapping {
  const DesignKanjiMapping({required this.value, this.mappingRef});

  final String value;
  final String? mappingRef;

  DesignKanjiMapping copyWith({String? value, String? mappingRef}) {
    return DesignKanjiMapping(
      value: value ?? this.value,
      mappingRef: mappingRef ?? this.mappingRef,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is DesignKanjiMapping &&
            other.value == value &&
            other.mappingRef == mappingRef);
  }

  @override
  int get hashCode => Object.hash(value, mappingRef);
}

@immutable
class DesignInput {
  const DesignInput({
    required this.sourceType,
    required this.rawName,
    this.kanji,
  });

  final DesignSourceType sourceType;
  final String rawName;
  final DesignKanjiMapping? kanji;

  DesignInput copyWith({
    DesignSourceType? sourceType,
    String? rawName,
    DesignKanjiMapping? kanji,
  }) {
    return DesignInput(
      sourceType: sourceType ?? this.sourceType,
      rawName: rawName ?? this.rawName,
      kanji: kanji ?? this.kanji,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is DesignInput &&
            other.sourceType == sourceType &&
            other.rawName == rawName &&
            other.kanji == kanji);
  }

  @override
  int get hashCode => Object.hash(sourceType, rawName, kanji);
}

@immutable
class DesignSize {
  const DesignSize({required this.mm});

  final double mm;

  DesignSize copyWith({double? mm}) {
    return DesignSize(mm: mm ?? this.mm);
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) || (other is DesignSize && other.mm == mm);
  }

  @override
  int get hashCode => mm.hashCode;
}

@immutable
class DesignStroke {
  const DesignStroke({this.weight, this.contrast});

  final double? weight;
  final double? contrast;

  DesignStroke copyWith({double? weight, double? contrast}) {
    return DesignStroke(
      weight: weight ?? this.weight,
      contrast: contrast ?? this.contrast,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is DesignStroke &&
            other.weight == weight &&
            other.contrast == contrast);
  }

  @override
  int get hashCode => Object.hash(weight, contrast);
}

@immutable
class DesignLayout {
  const DesignLayout({this.grid, this.margin});

  final String? grid;
  final double? margin;

  DesignLayout copyWith({String? grid, double? margin}) {
    return DesignLayout(grid: grid ?? this.grid, margin: margin ?? this.margin);
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is DesignLayout && other.grid == grid && other.margin == margin);
  }

  @override
  int get hashCode => Object.hash(grid, margin);
}

@immutable
class DesignStyle {
  const DesignStyle({
    required this.writing,
    this.fontRef,
    this.templateRef,
    this.stroke,
    this.layout,
  });

  final DesignWritingStyle writing;
  final String? fontRef;
  final String? templateRef;
  final DesignStroke? stroke;
  final DesignLayout? layout;

  DesignStyle copyWith({
    DesignWritingStyle? writing,
    String? fontRef,
    String? templateRef,
    DesignStroke? stroke,
    DesignLayout? layout,
  }) {
    return DesignStyle(
      writing: writing ?? this.writing,
      fontRef: fontRef ?? this.fontRef,
      templateRef: templateRef ?? this.templateRef,
      stroke: stroke ?? this.stroke,
      layout: layout ?? this.layout,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is DesignStyle &&
            other.writing == writing &&
            other.fontRef == fontRef &&
            other.templateRef == templateRef &&
            other.stroke == stroke &&
            other.layout == layout);
  }

  @override
  int get hashCode =>
      Object.hash(writing, fontRef, templateRef, stroke, layout);
}

@immutable
class DesignAiMetadata {
  const DesignAiMetadata({
    this.enabled,
    this.lastJobRef,
    this.qualityScore,
    this.registrable,
    this.diagnostics = const [],
  });

  final bool? enabled;
  final String? lastJobRef;
  final double? qualityScore;
  final bool? registrable;
  final List<String> diagnostics;

  DesignAiMetadata copyWith({
    bool? enabled,
    String? lastJobRef,
    double? qualityScore,
    bool? registrable,
    List<String>? diagnostics,
  }) {
    return DesignAiMetadata(
      enabled: enabled ?? this.enabled,
      lastJobRef: lastJobRef ?? this.lastJobRef,
      qualityScore: qualityScore ?? this.qualityScore,
      registrable: registrable ?? this.registrable,
      diagnostics: diagnostics ?? this.diagnostics,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is DesignAiMetadata &&
            other.enabled == enabled &&
            other.lastJobRef == lastJobRef &&
            other.qualityScore == qualityScore &&
            other.registrable == registrable &&
            listEquals(other.diagnostics, diagnostics));
  }

  @override
  int get hashCode => Object.hash(
    enabled,
    lastJobRef,
    qualityScore,
    registrable,
    Object.hashAll(diagnostics),
  );
}

@immutable
class DesignAssets {
  const DesignAssets({
    this.vectorSvg,
    this.previewPng,
    this.previewPngUrl,
    this.stampMockUrl,
  });

  final String? vectorSvg;
  final String? previewPng;
  final String? previewPngUrl;
  final String? stampMockUrl;

  DesignAssets copyWith({
    String? vectorSvg,
    String? previewPng,
    String? previewPngUrl,
    String? stampMockUrl,
  }) {
    return DesignAssets(
      vectorSvg: vectorSvg ?? this.vectorSvg,
      previewPng: previewPng ?? this.previewPng,
      previewPngUrl: previewPngUrl ?? this.previewPngUrl,
      stampMockUrl: stampMockUrl ?? this.stampMockUrl,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is DesignAssets &&
            other.vectorSvg == vectorSvg &&
            other.previewPng == previewPng &&
            other.previewPngUrl == previewPngUrl &&
            other.stampMockUrl == stampMockUrl);
  }

  @override
  int get hashCode =>
      Object.hash(vectorSvg, previewPng, previewPngUrl, stampMockUrl);
}

@immutable
class Design {
  const Design({
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

  final String id;
  final String ownerRef;
  final DesignStatus status;
  final DesignShape shape;
  final DesignSize size;
  final DesignStyle style;
  final int version;
  final DateTime createdAt;
  final DateTime updatedAt;
  final DesignInput? input;
  final DesignAiMetadata? ai;
  final DesignAssets? assets;
  final String? hash;
  final DateTime? lastOrderedAt;

  Design copyWith({
    String? id,
    String? ownerRef,
    DesignStatus? status,
    DesignShape? shape,
    DesignSize? size,
    DesignStyle? style,
    int? version,
    DateTime? createdAt,
    DateTime? updatedAt,
    DesignInput? input,
    DesignAiMetadata? ai,
    DesignAssets? assets,
    String? hash,
    DateTime? lastOrderedAt,
  }) {
    return Design(
      id: id ?? this.id,
      ownerRef: ownerRef ?? this.ownerRef,
      status: status ?? this.status,
      shape: shape ?? this.shape,
      size: size ?? this.size,
      style: style ?? this.style,
      version: version ?? this.version,
      createdAt: createdAt ?? this.createdAt,
      updatedAt: updatedAt ?? this.updatedAt,
      input: input ?? this.input,
      ai: ai ?? this.ai,
      assets: assets ?? this.assets,
      hash: hash ?? this.hash,
      lastOrderedAt: lastOrderedAt ?? this.lastOrderedAt,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is Design &&
            other.id == id &&
            other.ownerRef == ownerRef &&
            other.status == status &&
            other.shape == shape &&
            other.size == size &&
            other.style == style &&
            other.version == version &&
            other.createdAt == createdAt &&
            other.updatedAt == updatedAt &&
            other.input == input &&
            other.ai == ai &&
            other.assets == assets &&
            other.hash == hash &&
            other.lastOrderedAt == lastOrderedAt);
  }

  @override
  int get hashCode {
    return Object.hashAll([
      id,
      ownerRef,
      status,
      shape,
      size,
      style,
      version,
      createdAt,
      updatedAt,
      input,
      ai,
      assets,
      hash,
      lastOrderedAt,
    ]);
  }
}
