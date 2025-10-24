import 'package:app/core/domain/entities/user.dart';

UserPersona _parseUserPersona(String value) {
  switch (value) {
    case 'foreigner':
      return UserPersona.foreigner;
    case 'japanese':
      return UserPersona.japanese;
  }
  throw ArgumentError.value(value, 'value', 'Unknown UserPersona');
}

String _userPersonaToJson(UserPersona persona) {
  switch (persona) {
    case UserPersona.foreigner:
      return 'foreigner';
    case UserPersona.japanese:
      return 'japanese';
  }
}

UserLanguage _parseUserLanguage(String value) {
  switch (value) {
    case 'ja':
      return UserLanguage.ja;
    case 'en':
      return UserLanguage.en;
  }
  throw ArgumentError.value(value, 'value', 'Unknown UserLanguage');
}

String _userLanguageToJson(UserLanguage language) {
  switch (language) {
    case UserLanguage.ja:
      return 'ja';
    case UserLanguage.en:
      return 'en';
  }
}

UserRole _parseUserRole(String? value) {
  switch (value) {
    case 'staff':
      return UserRole.staff;
    case 'admin':
      return UserRole.admin;
    case 'user':
    case null:
      return UserRole.user;
  }
  throw ArgumentError.value(value, 'value', 'Unknown UserRole');
}

String _userRoleToJson(UserRole role) {
  switch (role) {
    case UserRole.user:
      return 'user';
    case UserRole.staff:
      return 'staff';
    case UserRole.admin:
      return 'admin';
  }
}

PaymentProvider _parsePaymentProvider(String value) {
  switch (value) {
    case 'stripe':
      return PaymentProvider.stripe;
    case 'paypal':
      return PaymentProvider.paypal;
    case 'other':
      return PaymentProvider.other;
  }
  throw ArgumentError.value(value, 'value', 'Unknown PaymentProvider');
}

String _paymentProviderToJson(PaymentProvider provider) {
  switch (provider) {
    case PaymentProvider.stripe:
      return 'stripe';
    case PaymentProvider.paypal:
      return 'paypal';
    case PaymentProvider.other:
      return 'other';
  }
}

PaymentMethodType _parsePaymentMethodType(String value) {
  switch (value) {
    case 'card':
      return PaymentMethodType.card;
    case 'wallet':
      return PaymentMethodType.wallet;
    case 'bank':
      return PaymentMethodType.bank;
    case 'other':
      return PaymentMethodType.other;
  }
  throw ArgumentError.value(value, 'value', 'Unknown PaymentMethodType');
}

String _paymentMethodTypeToJson(PaymentMethodType type) {
  switch (type) {
    case PaymentMethodType.card:
      return 'card';
    case PaymentMethodType.wallet:
      return 'wallet';
    case PaymentMethodType.bank:
      return 'bank';
    case PaymentMethodType.other:
      return 'other';
  }
}

class UserProfileDto {
  UserProfileDto({
    required this.id,
    required this.persona,
    required this.preferredLang,
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
    this.role,
    this.deletedAt,
  });

  factory UserProfileDto.fromJson(Map<String, dynamic> json) {
    return UserProfileDto(
      id: json['id'] as String,
      displayName: json['displayName'] as String?,
      email: json['email'] as String?,
      phone: json['phone'] as String?,
      avatarUrl: json['avatarUrl'] as String?,
      persona: json['persona'] as String,
      preferredLang: json['preferredLang'] as String,
      country: json['country'] as String?,
      onboarding: json['onboarding'] == null
          ? null
          : Map<String, dynamic>.from(json['onboarding'] as Map),
      marketingOptIn: json['marketingOptIn'] as bool?,
      role: json['role'] as String?,
      isActive: json['isActive'] as bool? ?? true,
      piiMasked: json['piiMasked'] as bool? ?? false,
      createdAt: json['createdAt'] as String,
      updatedAt: json['updatedAt'] as String,
      deletedAt: json['deletedAt'] as String?,
    );
  }

  factory UserProfileDto.fromDomain(UserProfile domain) {
    return UserProfileDto(
      id: domain.id,
      displayName: domain.displayName,
      email: domain.email,
      phone: domain.phone,
      avatarUrl: domain.avatarUrl,
      persona: _userPersonaToJson(domain.persona),
      preferredLang: _userLanguageToJson(domain.preferredLanguage),
      country: domain.country,
      onboarding: domain.onboarding == null
          ? null
          : Map<String, dynamic>.from(domain.onboarding!),
      marketingOptIn: domain.marketingOptIn,
      role: _userRoleToJson(domain.role),
      isActive: domain.isActive,
      piiMasked: domain.piiMasked,
      createdAt: domain.createdAt.toIso8601String(),
      updatedAt: domain.updatedAt.toIso8601String(),
      deletedAt: domain.deletedAt?.toIso8601String(),
    );
  }

  final String id;
  final String? displayName;
  final String? email;
  final String? phone;
  final String? avatarUrl;
  final String persona;
  final String preferredLang;
  final String? country;
  final Map<String, dynamic>? onboarding;
  final bool? marketingOptIn;
  final String? role;
  final bool isActive;
  final bool piiMasked;
  final String createdAt;
  final String updatedAt;
  final String? deletedAt;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'id': id,
      'displayName': displayName,
      'email': email,
      'phone': phone,
      'avatarUrl': avatarUrl,
      'persona': persona,
      'preferredLang': preferredLang,
      'country': country,
      'onboarding': onboarding,
      'marketingOptIn': marketingOptIn,
      'role': role,
      'isActive': isActive,
      'piiMasked': piiMasked,
      'createdAt': createdAt,
      'updatedAt': updatedAt,
      'deletedAt': deletedAt,
    };
  }

  UserProfile toDomain() {
    return UserProfile(
      id: id,
      displayName: displayName,
      email: email,
      phone: phone,
      avatarUrl: avatarUrl,
      persona: _parseUserPersona(persona),
      preferredLanguage: _parseUserLanguage(preferredLang),
      country: country,
      onboarding: onboarding == null
          ? null
          : Map<String, dynamic>.from(onboarding!),
      marketingOptIn: marketingOptIn,
      role: _parseUserRole(role),
      isActive: isActive,
      piiMasked: piiMasked,
      createdAt: DateTime.parse(createdAt),
      updatedAt: DateTime.parse(updatedAt),
      deletedAt: deletedAt == null ? null : DateTime.parse(deletedAt!),
    );
  }
}

class UserAddressDto {
  UserAddressDto({
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

  factory UserAddressDto.fromJson(Map<String, dynamic> json) {
    return UserAddressDto(
      id: json['id'] as String,
      label: json['label'] as String?,
      recipient: json['recipient'] as String,
      company: json['company'] as String?,
      line1: json['line1'] as String,
      line2: json['line2'] as String?,
      city: json['city'] as String,
      state: json['state'] as String?,
      postalCode: json['postalCode'] as String,
      country: json['country'] as String,
      phone: json['phone'] as String?,
      isDefault: json['isDefault'] as bool? ?? false,
      createdAt: json['createdAt'] as String,
      updatedAt: json['updatedAt'] as String?,
    );
  }

  factory UserAddressDto.fromDomain(UserAddress domain) {
    return UserAddressDto(
      id: domain.id,
      label: domain.label,
      recipient: domain.recipient,
      company: domain.company,
      line1: domain.line1,
      line2: domain.line2,
      city: domain.city,
      state: domain.state,
      postalCode: domain.postalCode,
      country: domain.country,
      phone: domain.phone,
      isDefault: domain.isDefault,
      createdAt: domain.createdAt.toIso8601String(),
      updatedAt: domain.updatedAt?.toIso8601String(),
    );
  }

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
  final String createdAt;
  final String? updatedAt;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'id': id,
      'label': label,
      'recipient': recipient,
      'company': company,
      'line1': line1,
      'line2': line2,
      'city': city,
      'state': state,
      'postalCode': postalCode,
      'country': country,
      'phone': phone,
      'isDefault': isDefault,
      'createdAt': createdAt,
      'updatedAt': updatedAt,
    };
  }

  UserAddress toDomain() {
    return UserAddress(
      id: id,
      label: label,
      recipient: recipient,
      company: company,
      line1: line1,
      line2: line2,
      city: city,
      state: state,
      postalCode: postalCode,
      country: country,
      phone: phone,
      isDefault: isDefault,
      createdAt: DateTime.parse(createdAt),
      updatedAt: updatedAt == null ? null : DateTime.parse(updatedAt!),
    );
  }
}

class UserPaymentMethodDto {
  UserPaymentMethodDto({
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

  factory UserPaymentMethodDto.fromJson(Map<String, dynamic> json) {
    return UserPaymentMethodDto(
      id: json['id'] as String,
      provider: json['provider'] as String,
      methodType: json['methodType'] as String,
      brand: json['brand'] as String?,
      last4: json['last4'] as String?,
      expMonth: json['expMonth'] as int?,
      expYear: json['expYear'] as int?,
      fingerprint: json['fingerprint'] as String?,
      billingName: json['billingName'] as String?,
      providerRef: json['providerRef'] as String,
      createdAt: json['createdAt'] as String,
      updatedAt: json['updatedAt'] as String?,
    );
  }

  factory UserPaymentMethodDto.fromDomain(UserPaymentMethod domain) {
    return UserPaymentMethodDto(
      id: domain.id,
      provider: _paymentProviderToJson(domain.provider),
      methodType: _paymentMethodTypeToJson(domain.methodType),
      brand: domain.brand,
      last4: domain.last4,
      expMonth: domain.expMonth,
      expYear: domain.expYear,
      fingerprint: domain.fingerprint,
      billingName: domain.billingName,
      providerRef: domain.providerRef,
      createdAt: domain.createdAt.toIso8601String(),
      updatedAt: domain.updatedAt?.toIso8601String(),
    );
  }

  final String id;
  final String provider;
  final String methodType;
  final String? brand;
  final String? last4;
  final int? expMonth;
  final int? expYear;
  final String? fingerprint;
  final String? billingName;
  final String providerRef;
  final String createdAt;
  final String? updatedAt;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'id': id,
      'provider': provider,
      'methodType': methodType,
      'brand': brand,
      'last4': last4,
      'expMonth': expMonth,
      'expYear': expYear,
      'fingerprint': fingerprint,
      'billingName': billingName,
      'providerRef': providerRef,
      'createdAt': createdAt,
      'updatedAt': updatedAt,
    };
  }

  UserPaymentMethod toDomain() {
    return UserPaymentMethod(
      id: id,
      provider: _parsePaymentProvider(provider),
      methodType: _parsePaymentMethodType(methodType),
      brand: brand,
      last4: last4,
      expMonth: expMonth,
      expYear: expYear,
      fingerprint: fingerprint,
      billingName: billingName,
      providerRef: providerRef,
      createdAt: DateTime.parse(createdAt),
      updatedAt: updatedAt == null ? null : DateTime.parse(updatedAt!),
    );
  }
}

class UserFavoriteDesignDto {
  UserFavoriteDesignDto({
    required this.id,
    required this.designRef,
    required this.addedAt,
    this.note,
    this.tags,
  });

  factory UserFavoriteDesignDto.fromJson(Map<String, dynamic> json) {
    return UserFavoriteDesignDto(
      id: json['id'] as String,
      designRef: json['designRef'] as String,
      note: json['note'] as String?,
      tags: (json['tags'] as List<dynamic>?)?.cast<String>(),
      addedAt: json['addedAt'] as String,
    );
  }

  factory UserFavoriteDesignDto.fromDomain(UserFavoriteDesign domain) {
    return UserFavoriteDesignDto(
      id: domain.id,
      designRef: domain.designRef,
      note: domain.note,
      tags: domain.tags,
      addedAt: domain.addedAt.toIso8601String(),
    );
  }

  final String id;
  final String designRef;
  final String? note;
  final List<String>? tags;
  final String addedAt;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'id': id,
      'designRef': designRef,
      'note': note,
      'tags': tags,
      'addedAt': addedAt,
    };
  }

  UserFavoriteDesign toDomain() {
    return UserFavoriteDesign(
      id: id,
      designRef: designRef,
      note: note,
      tags: tags ?? const [],
      addedAt: DateTime.parse(addedAt),
    );
  }
}
