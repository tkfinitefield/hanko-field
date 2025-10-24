import 'package:flutter/foundation.dart';

/// ユーザーペルソナ（初期導線切り替えに使用）
enum UserPersona { foreigner, japanese }

/// アプリ表示言語
enum UserLanguage { ja, en }

/// カスタムクレームと連動するロール
enum UserRole { user, staff, admin }

@immutable
class UserProfile {
  const UserProfile({
    required this.id,
    required this.persona,
    required this.preferredLanguage,
    required this.isActive,
    required this.piiMasked,
    required this.createdAt,
    required this.updatedAt,
    this.displayName,
    this.email,
    this.phone,
    this.avatarUrl,
    this.country,
    this.onboarding,
    this.marketingOptIn,
    this.role = UserRole.user,
    this.deletedAt,
  });

  final String id;
  final String? displayName;
  final String? email;
  final String? phone;
  final String? avatarUrl;
  final UserPersona persona;
  final UserLanguage preferredLanguage;
  final String? country;
  final Map<String, dynamic>? onboarding;
  final bool? marketingOptIn;
  final UserRole role;
  final bool isActive;
  final bool piiMasked;
  final DateTime createdAt;
  final DateTime updatedAt;
  final DateTime? deletedAt;

  UserProfile copyWith({
    String? id,
    String? displayName,
    String? email,
    String? phone,
    String? avatarUrl,
    UserPersona? persona,
    UserLanguage? preferredLanguage,
    String? country,
    Map<String, dynamic>? onboarding,
    bool? marketingOptIn,
    UserRole? role,
    bool? isActive,
    bool? piiMasked,
    DateTime? createdAt,
    DateTime? updatedAt,
    DateTime? deletedAt,
  }) {
    return UserProfile(
      id: id ?? this.id,
      displayName: displayName ?? this.displayName,
      email: email ?? this.email,
      phone: phone ?? this.phone,
      avatarUrl: avatarUrl ?? this.avatarUrl,
      persona: persona ?? this.persona,
      preferredLanguage: preferredLanguage ?? this.preferredLanguage,
      country: country ?? this.country,
      onboarding: onboarding ?? this.onboarding,
      marketingOptIn: marketingOptIn ?? this.marketingOptIn,
      role: role ?? this.role,
      isActive: isActive ?? this.isActive,
      piiMasked: piiMasked ?? this.piiMasked,
      createdAt: createdAt ?? this.createdAt,
      updatedAt: updatedAt ?? this.updatedAt,
      deletedAt: deletedAt ?? this.deletedAt,
    );
  }

  @override
  bool operator ==(Object other) {
    if (identical(this, other)) {
      return true;
    }
    return other is UserProfile &&
        other.id == id &&
        other.displayName == displayName &&
        other.email == email &&
        other.phone == phone &&
        other.avatarUrl == avatarUrl &&
        other.persona == persona &&
        other.preferredLanguage == preferredLanguage &&
        other.country == country &&
        mapEquals(other.onboarding, onboarding) &&
        other.marketingOptIn == marketingOptIn &&
        other.role == role &&
        other.isActive == isActive &&
        other.piiMasked == piiMasked &&
        other.createdAt == createdAt &&
        other.updatedAt == updatedAt &&
        other.deletedAt == deletedAt;
  }

  @override
  int get hashCode {
    return Object.hashAll([
      id,
      displayName,
      email,
      phone,
      avatarUrl,
      persona,
      preferredLanguage,
      country,
      onboarding == null ? null : Object.hashAll(onboarding!.entries),
      marketingOptIn,
      role,
      isActive,
      piiMasked,
      createdAt,
      updatedAt,
      deletedAt,
    ]);
  }
}

@immutable
class UserAddress {
  const UserAddress({
    required this.id,
    required this.recipient,
    required this.line1,
    required this.city,
    required this.postalCode,
    required this.country,
    required this.createdAt,
    this.label,
    this.company,
    this.line2,
    this.state,
    this.phone,
    this.isDefault = false,
    this.updatedAt,
  });

  final String id;
  final String? label;
  final String recipient;
  final String? company;
  final String line1;
  final String? line2;
  final String city;
  final String? state;
  final String postalCode;
  final String country;
  final String? phone;
  final bool isDefault;
  final DateTime createdAt;
  final DateTime? updatedAt;

  UserAddress copyWith({
    String? id,
    String? label,
    String? recipient,
    String? company,
    String? line1,
    String? line2,
    String? city,
    String? state,
    String? postalCode,
    String? country,
    String? phone,
    bool? isDefault,
    DateTime? createdAt,
    DateTime? updatedAt,
  }) {
    return UserAddress(
      id: id ?? this.id,
      label: label ?? this.label,
      recipient: recipient ?? this.recipient,
      company: company ?? this.company,
      line1: line1 ?? this.line1,
      line2: line2 ?? this.line2,
      city: city ?? this.city,
      state: state ?? this.state,
      postalCode: postalCode ?? this.postalCode,
      country: country ?? this.country,
      phone: phone ?? this.phone,
      isDefault: isDefault ?? this.isDefault,
      createdAt: createdAt ?? this.createdAt,
      updatedAt: updatedAt ?? this.updatedAt,
    );
  }

  @override
  bool operator ==(Object other) {
    if (identical(this, other)) {
      return true;
    }
    return other is UserAddress &&
        other.id == id &&
        other.label == label &&
        other.recipient == recipient &&
        other.company == company &&
        other.line1 == line1 &&
        other.line2 == line2 &&
        other.city == city &&
        other.state == state &&
        other.postalCode == postalCode &&
        other.country == country &&
        other.phone == phone &&
        other.isDefault == isDefault &&
        other.createdAt == createdAt &&
        other.updatedAt == updatedAt;
  }

  @override
  int get hashCode {
    return Object.hashAll([
      id,
      label,
      recipient,
      company,
      line1,
      line2,
      city,
      state,
      postalCode,
      country,
      phone,
      isDefault,
      createdAt,
      updatedAt,
    ]);
  }
}

enum PaymentProvider { stripe, paypal, other }

enum PaymentMethodType { card, wallet, bank, other }

@immutable
class UserPaymentMethod {
  const UserPaymentMethod({
    required this.id,
    required this.provider,
    required this.methodType,
    required this.providerRef,
    required this.createdAt,
    this.brand,
    this.last4,
    this.expMonth,
    this.expYear,
    this.fingerprint,
    this.billingName,
    this.updatedAt,
  });

  final String id;
  final PaymentProvider provider;
  final PaymentMethodType methodType;
  final String? brand;
  final String? last4;
  final int? expMonth;
  final int? expYear;
  final String? fingerprint;
  final String? billingName;
  final String providerRef;
  final DateTime createdAt;
  final DateTime? updatedAt;

  UserPaymentMethod copyWith({
    String? id,
    PaymentProvider? provider,
    PaymentMethodType? methodType,
    String? brand,
    String? last4,
    int? expMonth,
    int? expYear,
    String? fingerprint,
    String? billingName,
    String? providerRef,
    DateTime? createdAt,
    DateTime? updatedAt,
  }) {
    return UserPaymentMethod(
      id: id ?? this.id,
      provider: provider ?? this.provider,
      methodType: methodType ?? this.methodType,
      brand: brand ?? this.brand,
      last4: last4 ?? this.last4,
      expMonth: expMonth ?? this.expMonth,
      expYear: expYear ?? this.expYear,
      fingerprint: fingerprint ?? this.fingerprint,
      billingName: billingName ?? this.billingName,
      providerRef: providerRef ?? this.providerRef,
      createdAt: createdAt ?? this.createdAt,
      updatedAt: updatedAt ?? this.updatedAt,
    );
  }

  @override
  bool operator ==(Object other) {
    if (identical(this, other)) {
      return true;
    }
    return other is UserPaymentMethod &&
        other.id == id &&
        other.provider == provider &&
        other.methodType == methodType &&
        other.brand == brand &&
        other.last4 == last4 &&
        other.expMonth == expMonth &&
        other.expYear == expYear &&
        other.fingerprint == fingerprint &&
        other.billingName == billingName &&
        other.providerRef == providerRef &&
        other.createdAt == createdAt &&
        other.updatedAt == updatedAt;
  }

  @override
  int get hashCode {
    return Object.hashAll([
      id,
      provider,
      methodType,
      brand,
      last4,
      expMonth,
      expYear,
      fingerprint,
      billingName,
      providerRef,
      createdAt,
      updatedAt,
    ]);
  }
}

@immutable
class UserFavoriteDesign {
  const UserFavoriteDesign({
    required this.id,
    required this.designRef,
    required this.addedAt,
    this.note,
    this.tags = const [],
  });

  final String id;
  final String designRef;
  final String? note;
  final List<String> tags;
  final DateTime addedAt;

  UserFavoriteDesign copyWith({
    String? id,
    String? designRef,
    String? note,
    List<String>? tags,
    DateTime? addedAt,
  }) {
    return UserFavoriteDesign(
      id: id ?? this.id,
      designRef: designRef ?? this.designRef,
      note: note ?? this.note,
      tags: tags ?? this.tags,
      addedAt: addedAt ?? this.addedAt,
    );
  }

  @override
  bool operator ==(Object other) {
    if (identical(this, other)) {
      return true;
    }
    return other is UserFavoriteDesign &&
        other.id == id &&
        other.designRef == designRef &&
        other.note == note &&
        listEquals(other.tags, tags) &&
        other.addedAt == addedAt;
  }

  @override
  int get hashCode {
    return Object.hashAll([id, designRef, note, Object.hashAll(tags), addedAt]);
  }
}
