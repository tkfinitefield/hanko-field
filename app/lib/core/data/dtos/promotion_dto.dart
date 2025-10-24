import 'package:app/core/domain/entities/promotion.dart';

PromotionKind _parsePromotionKind(String value) {
  switch (value) {
    case 'percent':
      return PromotionKind.percent;
    case 'fixed':
      return PromotionKind.fixed;
  }
  throw ArgumentError.value(value, 'value', 'Unknown PromotionKind');
}

String _promotionKindToJson(PromotionKind kind) {
  switch (kind) {
    case PromotionKind.percent:
      return 'percent';
    case PromotionKind.fixed:
      return 'fixed';
  }
}

class PromotionStackingRulesDto {
  PromotionStackingRulesDto({
    this.combinable,
    this.withSalePrice,
    this.maxStack,
  });

  factory PromotionStackingRulesDto.fromJson(Map<String, dynamic> json) {
    return PromotionStackingRulesDto(
      combinable: json['combinable'] as bool?,
      withSalePrice: json['withSalePrice'] as bool?,
      maxStack: json['maxStack'] as int?,
    );
  }

  factory PromotionStackingRulesDto.fromDomain(PromotionStackingRules? domain) {
    if (domain == null) {
      return PromotionStackingRulesDto();
    }
    return PromotionStackingRulesDto(
      combinable: domain.combinable,
      withSalePrice: domain.withSalePrice,
      maxStack: domain.maxStack,
    );
  }

  final bool? combinable;
  final bool? withSalePrice;
  final int? maxStack;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'combinable': combinable,
      'withSalePrice': withSalePrice,
      'maxStack': maxStack,
    };
  }

  PromotionStackingRules toDomain() {
    return PromotionStackingRules(
      combinable: combinable,
      withSalePrice: withSalePrice,
      maxStack: maxStack,
    );
  }
}

class PromotionConditionsDto {
  PromotionConditionsDto({
    this.minSubtotal,
    this.countryIn,
    this.currencyIn,
    this.shapeIn,
    this.sizeMmBetween,
    this.productRefsIn,
    this.materialRefsIn,
    this.newCustomerOnly,
  });

  factory PromotionConditionsDto.fromJson(Map<String, dynamic> json) {
    return PromotionConditionsDto(
      minSubtotal: json['minSubtotal'] as int?,
      countryIn: (json['countryIn'] as List<dynamic>?)?.cast<String>(),
      currencyIn: (json['currencyIn'] as List<dynamic>?)?.cast<String>(),
      shapeIn: (json['shapeIn'] as List<dynamic>?)?.cast<String>(),
      sizeMmBetween: json['sizeMmBetween'] == null
          ? null
          : Map<String, dynamic>.from(json['sizeMmBetween'] as Map),
      productRefsIn: (json['productRefsIn'] as List<dynamic>?)?.cast<String>(),
      materialRefsIn: (json['materialRefsIn'] as List<dynamic>?)
          ?.cast<String>(),
      newCustomerOnly: json['newCustomerOnly'] as bool?,
    );
  }

  factory PromotionConditionsDto.fromDomain(PromotionConditions? domain) {
    if (domain == null) {
      return PromotionConditionsDto();
    }
    return PromotionConditionsDto(
      minSubtotal: domain.minSubtotal,
      countryIn: domain.countryIn,
      currencyIn: domain.currencyIn,
      shapeIn: domain.shapeIn,
      sizeMmBetween:
          domain.sizeMmBetweenMin == null && domain.sizeMmBetweenMax == null
          ? null
          : <String, dynamic>{
              'min': domain.sizeMmBetweenMin,
              'max': domain.sizeMmBetweenMax,
            },
      productRefsIn: domain.productRefsIn,
      materialRefsIn: domain.materialRefsIn,
      newCustomerOnly: domain.newCustomerOnly,
    );
  }

  final int? minSubtotal;
  final List<String>? countryIn;
  final List<String>? currencyIn;
  final List<String>? shapeIn;
  final Map<String, dynamic>? sizeMmBetween;
  final List<String>? productRefsIn;
  final List<String>? materialRefsIn;
  final bool? newCustomerOnly;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'minSubtotal': minSubtotal,
      'countryIn': countryIn,
      'currencyIn': currencyIn,
      'shapeIn': shapeIn,
      'sizeMmBetween': sizeMmBetween,
      'productRefsIn': productRefsIn,
      'materialRefsIn': materialRefsIn,
      'newCustomerOnly': newCustomerOnly,
    };
  }

  PromotionConditions toDomain() {
    return PromotionConditions(
      minSubtotal: minSubtotal,
      countryIn: countryIn ?? const [],
      currencyIn: currencyIn ?? const [],
      shapeIn: shapeIn ?? const [],
      sizeMmBetweenMin: sizeMmBetween?['min'] == null
          ? null
          : (sizeMmBetween!['min'] as num).toDouble(),
      sizeMmBetweenMax: sizeMmBetween?['max'] == null
          ? null
          : (sizeMmBetween!['max'] as num).toDouble(),
      productRefsIn: productRefsIn ?? const [],
      materialRefsIn: materialRefsIn ?? const [],
      newCustomerOnly: newCustomerOnly,
    );
  }
}

class PromotionDto {
  PromotionDto({
    required this.id,
    required this.code,
    required this.kind,
    required this.value,
    required this.isActive,
    required this.startsAt,
    required this.endsAt,
    required this.usageLimit,
    required this.usageCount,
    required this.limitPerUser,
    required this.createdAt,
    this.name,
    this.currency,
    this.stacking,
    this.conditions,
    this.notes,
    this.updatedAt,
  });

  factory PromotionDto.fromJson(Map<String, dynamic> json) {
    return PromotionDto(
      id: json['id'] as String,
      code: json['code'] as String,
      name: json['name'] as String?,
      kind: json['kind'] as String,
      value: (json['value'] as num).toDouble(),
      currency: json['currency'] as String?,
      isActive: json['isActive'] as bool? ?? true,
      startsAt: json['startsAt'] as String,
      endsAt: json['endsAt'] as String,
      stacking: json['stacking'] == null
          ? null
          : PromotionStackingRulesDto.fromJson(
              json['stacking'] as Map<String, dynamic>,
            ),
      conditions: json['conditions'] == null
          ? null
          : PromotionConditionsDto.fromJson(
              json['conditions'] as Map<String, dynamic>,
            ),
      usageLimit: json['usageLimit'] as int,
      usageCount: json['usageCount'] as int,
      limitPerUser: json['limitPerUser'] as int,
      notes: json['notes'] as String?,
      createdAt: json['createdAt'] as String,
      updatedAt: json['updatedAt'] as String?,
    );
  }

  factory PromotionDto.fromDomain(Promotion domain) {
    return PromotionDto(
      id: domain.id,
      code: domain.code,
      name: domain.name,
      kind: _promotionKindToJson(domain.kind),
      value: domain.value,
      currency: domain.currency,
      isActive: domain.isActive,
      startsAt: domain.startsAt.toIso8601String(),
      endsAt: domain.endsAt.toIso8601String(),
      stacking: domain.stacking == null
          ? null
          : PromotionStackingRulesDto.fromDomain(domain.stacking),
      conditions: domain.conditions == null
          ? null
          : PromotionConditionsDto.fromDomain(domain.conditions),
      usageLimit: domain.usageLimit,
      usageCount: domain.usageCount,
      limitPerUser: domain.limitPerUser,
      notes: domain.notes,
      createdAt: domain.createdAt.toIso8601String(),
      updatedAt: domain.updatedAt?.toIso8601String(),
    );
  }

  final String id;
  final String code;
  final String? name;
  final String kind;
  final double value;
  final String? currency;
  final bool isActive;
  final String startsAt;
  final String endsAt;
  final PromotionStackingRulesDto? stacking;
  final PromotionConditionsDto? conditions;
  final int usageLimit;
  final int usageCount;
  final int limitPerUser;
  final String? notes;
  final String createdAt;
  final String? updatedAt;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'id': id,
      'code': code,
      'name': name,
      'kind': kind,
      'value': value,
      'currency': currency,
      'isActive': isActive,
      'startsAt': startsAt,
      'endsAt': endsAt,
      'stacking': stacking?.toJson(),
      'conditions': conditions?.toJson(),
      'usageLimit': usageLimit,
      'usageCount': usageCount,
      'limitPerUser': limitPerUser,
      'notes': notes,
      'createdAt': createdAt,
      'updatedAt': updatedAt,
    };
  }

  Promotion toDomain() {
    return Promotion(
      id: id,
      code: code,
      name: name,
      kind: _parsePromotionKind(kind),
      value: value,
      currency: currency,
      isActive: isActive,
      startsAt: DateTime.parse(startsAt),
      endsAt: DateTime.parse(endsAt),
      stacking: stacking?.toDomain(),
      conditions: conditions?.toDomain(),
      usageLimit: usageLimit,
      usageCount: usageCount,
      limitPerUser: limitPerUser,
      notes: notes,
      createdAt: DateTime.parse(createdAt),
      updatedAt: updatedAt == null ? null : DateTime.parse(updatedAt!),
    );
  }
}
