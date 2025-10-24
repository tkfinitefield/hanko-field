import 'package:flutter/foundation.dart';

enum PromotionKind { percent, fixed }

@immutable
class PromotionStackingRules {
  const PromotionStackingRules({
    this.combinable,
    this.withSalePrice,
    this.maxStack,
  });

  final bool? combinable;
  final bool? withSalePrice;
  final int? maxStack;

  PromotionStackingRules copyWith({
    bool? combinable,
    bool? withSalePrice,
    int? maxStack,
  }) {
    return PromotionStackingRules(
      combinable: combinable ?? this.combinable,
      withSalePrice: withSalePrice ?? this.withSalePrice,
      maxStack: maxStack ?? this.maxStack,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is PromotionStackingRules &&
            other.combinable == combinable &&
            other.withSalePrice == withSalePrice &&
            other.maxStack == maxStack);
  }

  @override
  int get hashCode => Object.hash(combinable, withSalePrice, maxStack);
}

@immutable
class PromotionConditions {
  const PromotionConditions({
    this.minSubtotal,
    this.countryIn = const [],
    this.currencyIn = const [],
    this.shapeIn = const [],
    this.sizeMmBetweenMin,
    this.sizeMmBetweenMax,
    this.productRefsIn = const [],
    this.materialRefsIn = const [],
    this.newCustomerOnly,
  });

  final int? minSubtotal;
  final List<String> countryIn;
  final List<String> currencyIn;
  final List<String> shapeIn;
  final double? sizeMmBetweenMin;
  final double? sizeMmBetweenMax;
  final List<String> productRefsIn;
  final List<String> materialRefsIn;
  final bool? newCustomerOnly;

  PromotionConditions copyWith({
    int? minSubtotal,
    List<String>? countryIn,
    List<String>? currencyIn,
    List<String>? shapeIn,
    double? sizeMmBetweenMin,
    double? sizeMmBetweenMax,
    List<String>? productRefsIn,
    List<String>? materialRefsIn,
    bool? newCustomerOnly,
  }) {
    return PromotionConditions(
      minSubtotal: minSubtotal ?? this.minSubtotal,
      countryIn: countryIn ?? this.countryIn,
      currencyIn: currencyIn ?? this.currencyIn,
      shapeIn: shapeIn ?? this.shapeIn,
      sizeMmBetweenMin: sizeMmBetweenMin ?? this.sizeMmBetweenMin,
      sizeMmBetweenMax: sizeMmBetweenMax ?? this.sizeMmBetweenMax,
      productRefsIn: productRefsIn ?? this.productRefsIn,
      materialRefsIn: materialRefsIn ?? this.materialRefsIn,
      newCustomerOnly: newCustomerOnly ?? this.newCustomerOnly,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is PromotionConditions &&
            other.minSubtotal == minSubtotal &&
            listEquals(other.countryIn, countryIn) &&
            listEquals(other.currencyIn, currencyIn) &&
            listEquals(other.shapeIn, shapeIn) &&
            other.sizeMmBetweenMin == sizeMmBetweenMin &&
            other.sizeMmBetweenMax == sizeMmBetweenMax &&
            listEquals(other.productRefsIn, productRefsIn) &&
            listEquals(other.materialRefsIn, materialRefsIn) &&
            other.newCustomerOnly == newCustomerOnly);
  }

  @override
  int get hashCode {
    return Object.hashAll([
      minSubtotal,
      Object.hashAll(countryIn),
      Object.hashAll(currencyIn),
      Object.hashAll(shapeIn),
      sizeMmBetweenMin,
      sizeMmBetweenMax,
      Object.hashAll(productRefsIn),
      Object.hashAll(materialRefsIn),
      newCustomerOnly,
    ]);
  }
}

@immutable
class Promotion {
  const Promotion({
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

  final String id;
  final String code;
  final String? name;
  final PromotionKind kind;
  final double value;
  final String? currency;
  final bool isActive;
  final DateTime startsAt;
  final DateTime endsAt;
  final PromotionStackingRules? stacking;
  final PromotionConditions? conditions;
  final int usageLimit;
  final int usageCount;
  final int limitPerUser;
  final String? notes;
  final DateTime createdAt;
  final DateTime? updatedAt;

  Promotion copyWith({
    String? id,
    String? code,
    String? name,
    PromotionKind? kind,
    double? value,
    String? currency,
    bool? isActive,
    DateTime? startsAt,
    DateTime? endsAt,
    PromotionStackingRules? stacking,
    PromotionConditions? conditions,
    int? usageLimit,
    int? usageCount,
    int? limitPerUser,
    String? notes,
    DateTime? createdAt,
    DateTime? updatedAt,
  }) {
    return Promotion(
      id: id ?? this.id,
      code: code ?? this.code,
      name: name ?? this.name,
      kind: kind ?? this.kind,
      value: value ?? this.value,
      currency: currency ?? this.currency,
      isActive: isActive ?? this.isActive,
      startsAt: startsAt ?? this.startsAt,
      endsAt: endsAt ?? this.endsAt,
      stacking: stacking ?? this.stacking,
      conditions: conditions ?? this.conditions,
      usageLimit: usageLimit ?? this.usageLimit,
      usageCount: usageCount ?? this.usageCount,
      limitPerUser: limitPerUser ?? this.limitPerUser,
      notes: notes ?? this.notes,
      createdAt: createdAt ?? this.createdAt,
      updatedAt: updatedAt ?? this.updatedAt,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is Promotion &&
            other.id == id &&
            other.code == code &&
            other.name == name &&
            other.kind == kind &&
            other.value == value &&
            other.currency == currency &&
            other.isActive == isActive &&
            other.startsAt == startsAt &&
            other.endsAt == endsAt &&
            other.stacking == stacking &&
            other.conditions == conditions &&
            other.usageLimit == usageLimit &&
            other.usageCount == usageCount &&
            other.limitPerUser == limitPerUser &&
            other.notes == notes &&
            other.createdAt == createdAt &&
            other.updatedAt == updatedAt);
  }

  @override
  int get hashCode {
    return Object.hashAll([
      id,
      code,
      name,
      kind,
      value,
      currency,
      isActive,
      startsAt,
      endsAt,
      stacking,
      conditions,
      usageLimit,
      usageCount,
      limitPerUser,
      notes,
      createdAt,
      updatedAt,
    ]);
  }
}
