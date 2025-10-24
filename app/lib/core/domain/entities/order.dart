import 'package:flutter/foundation.dart';

enum OrderStatus {
  draft,
  pendingPayment,
  paid,
  inProduction,
  readyToShip,
  shipped,
  delivered,
  canceled,
}

enum OrderShipmentCarrier { jppost, yamato, sagawa, dhl, ups, fedex, other }

enum OrderShipmentStatus {
  labelCreated,
  inTransit,
  outForDelivery,
  delivered,
  exception,
  cancelled,
}

enum OrderShipmentEventCode {
  labelCreated,
  pickedUp,
  inTransit,
  arrivedHub,
  customsClearance,
  outForDelivery,
  delivered,
  exception,
  returnToSender,
}

enum OrderPaymentProvider { stripe, paypal, other }

enum OrderPaymentStatus {
  requiresAction,
  authorized,
  succeeded,
  failed,
  refunded,
  partiallyRefunded,
  canceled,
}

enum OrderPaymentMethodType { card, wallet, bank, other }

enum ProductionEventType {
  queued,
  engraving,
  polishing,
  qc,
  packed,
  onHold,
  rework,
  canceled,
}

@immutable
class OrderTotals {
  const OrderTotals({
    required this.subtotal,
    required this.discount,
    required this.shipping,
    required this.tax,
    required this.total,
    this.fees = 0,
  });

  final int subtotal;
  final int discount;
  final int shipping;
  final int tax;
  final int fees;
  final int total;

  OrderTotals copyWith({
    int? subtotal,
    int? discount,
    int? shipping,
    int? tax,
    int? fees,
    int? total,
  }) {
    return OrderTotals(
      subtotal: subtotal ?? this.subtotal,
      discount: discount ?? this.discount,
      shipping: shipping ?? this.shipping,
      tax: tax ?? this.tax,
      fees: fees ?? this.fees,
      total: total ?? this.total,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is OrderTotals &&
            other.subtotal == subtotal &&
            other.discount == discount &&
            other.shipping == shipping &&
            other.tax == tax &&
            other.fees == fees &&
            other.total == total);
  }

  @override
  int get hashCode =>
      Object.hash(subtotal, discount, shipping, tax, fees, total);
}

@immutable
class OrderPromotionSnapshot {
  const OrderPromotionSnapshot({
    required this.code,
    required this.applied,
    this.discountAmount,
  });

  final String code;
  final bool applied;
  final int? discountAmount;

  OrderPromotionSnapshot copyWith({
    String? code,
    bool? applied,
    int? discountAmount,
  }) {
    return OrderPromotionSnapshot(
      code: code ?? this.code,
      applied: applied ?? this.applied,
      discountAmount: discountAmount ?? this.discountAmount,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is OrderPromotionSnapshot &&
            other.code == code &&
            other.applied == applied &&
            other.discountAmount == discountAmount);
  }

  @override
  int get hashCode => Object.hash(code, applied, discountAmount);
}

@immutable
class OrderLineItem {
  const OrderLineItem({
    required this.productRef,
    required this.sku,
    required this.quantity,
    required this.unitPrice,
    required this.total,
    this.id,
    this.designRef,
    this.designSnapshot,
    this.name,
    this.options,
  });

  final String? id;
  final String productRef;
  final String? designRef;
  final Map<String, dynamic>? designSnapshot;
  final String sku;
  final String? name;
  final Map<String, dynamic>? options;
  final int quantity;
  final int unitPrice;
  final int total;

  OrderLineItem copyWith({
    String? id,
    String? productRef,
    String? designRef,
    Map<String, dynamic>? designSnapshot,
    String? sku,
    String? name,
    Map<String, dynamic>? options,
    int? quantity,
    int? unitPrice,
    int? total,
  }) {
    return OrderLineItem(
      id: id ?? this.id,
      productRef: productRef ?? this.productRef,
      designRef: designRef ?? this.designRef,
      designSnapshot: designSnapshot ?? this.designSnapshot,
      sku: sku ?? this.sku,
      name: name ?? this.name,
      options: options ?? this.options,
      quantity: quantity ?? this.quantity,
      unitPrice: unitPrice ?? this.unitPrice,
      total: total ?? this.total,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is OrderLineItem &&
            other.id == id &&
            other.productRef == productRef &&
            other.designRef == designRef &&
            mapEquals(other.designSnapshot, designSnapshot) &&
            other.sku == sku &&
            other.name == name &&
            mapEquals(other.options, options) &&
            other.quantity == quantity &&
            other.unitPrice == unitPrice &&
            other.total == total);
  }

  @override
  int get hashCode {
    return Object.hashAll([
      id,
      productRef,
      designRef,
      designSnapshot == null ? null : Object.hashAll(designSnapshot!.entries),
      sku,
      name,
      options == null ? null : Object.hashAll(options!.entries),
      quantity,
      unitPrice,
      total,
    ]);
  }
}

@immutable
class OrderAddress {
  const OrderAddress({
    required this.recipient,
    required this.line1,
    required this.city,
    required this.postalCode,
    required this.country,
    this.line2,
    this.state,
    this.phone,
  });

  final String recipient;
  final String line1;
  final String? line2;
  final String city;
  final String? state;
  final String postalCode;
  final String country;
  final String? phone;

  OrderAddress copyWith({
    String? recipient,
    String? line1,
    String? line2,
    String? city,
    String? state,
    String? postalCode,
    String? country,
    String? phone,
  }) {
    return OrderAddress(
      recipient: recipient ?? this.recipient,
      line1: line1 ?? this.line1,
      line2: line2 ?? this.line2,
      city: city ?? this.city,
      state: state ?? this.state,
      postalCode: postalCode ?? this.postalCode,
      country: country ?? this.country,
      phone: phone ?? this.phone,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is OrderAddress &&
            other.recipient == recipient &&
            other.line1 == line1 &&
            other.line2 == line2 &&
            other.city == city &&
            other.state == state &&
            other.postalCode == postalCode &&
            other.country == country &&
            other.phone == phone);
  }

  @override
  int get hashCode => Object.hash(
    recipient,
    line1,
    line2,
    city,
    state,
    postalCode,
    country,
    phone,
  );
}

@immutable
class OrderContact {
  const OrderContact({this.email, this.phone});

  final String? email;
  final String? phone;

  OrderContact copyWith({String? email, String? phone}) {
    return OrderContact(email: email ?? this.email, phone: phone ?? this.phone);
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is OrderContact && other.email == email && other.phone == phone);
  }

  @override
  int get hashCode => Object.hash(email, phone);
}

@immutable
class OrderFulfillmentInfo {
  const OrderFulfillmentInfo({
    this.requestedAt,
    this.estimatedShipDate,
    this.estimatedDeliveryDate,
  });

  final DateTime? requestedAt;
  final DateTime? estimatedShipDate;
  final DateTime? estimatedDeliveryDate;

  OrderFulfillmentInfo copyWith({
    DateTime? requestedAt,
    DateTime? estimatedShipDate,
    DateTime? estimatedDeliveryDate,
  }) {
    return OrderFulfillmentInfo(
      requestedAt: requestedAt ?? this.requestedAt,
      estimatedShipDate: estimatedShipDate ?? this.estimatedShipDate,
      estimatedDeliveryDate:
          estimatedDeliveryDate ?? this.estimatedDeliveryDate,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is OrderFulfillmentInfo &&
            other.requestedAt == requestedAt &&
            other.estimatedShipDate == estimatedShipDate &&
            other.estimatedDeliveryDate == estimatedDeliveryDate);
  }

  @override
  int get hashCode =>
      Object.hash(requestedAt, estimatedShipDate, estimatedDeliveryDate);
}

@immutable
class OrderProductionInfo {
  const OrderProductionInfo({
    this.queueRef,
    this.assignedStation,
    this.operatorRef,
  });

  final String? queueRef;
  final String? assignedStation;
  final String? operatorRef;

  OrderProductionInfo copyWith({
    String? queueRef,
    String? assignedStation,
    String? operatorRef,
  }) {
    return OrderProductionInfo(
      queueRef: queueRef ?? this.queueRef,
      assignedStation: assignedStation ?? this.assignedStation,
      operatorRef: operatorRef ?? this.operatorRef,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is OrderProductionInfo &&
            other.queueRef == queueRef &&
            other.assignedStation == assignedStation &&
            other.operatorRef == operatorRef);
  }

  @override
  int get hashCode => Object.hash(queueRef, assignedStation, operatorRef);
}

@immutable
class OrderFlags {
  const OrderFlags({this.manualReview, this.gift});

  final bool? manualReview;
  final bool? gift;

  OrderFlags copyWith({bool? manualReview, bool? gift}) {
    return OrderFlags(
      manualReview: manualReview ?? this.manualReview,
      gift: gift ?? this.gift,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is OrderFlags &&
            other.manualReview == manualReview &&
            other.gift == gift);
  }

  @override
  int get hashCode => Object.hash(manualReview, gift);
}

@immutable
class OrderAuditInfo {
  const OrderAuditInfo({this.createdBy, this.updatedBy});

  final String? createdBy;
  final String? updatedBy;

  OrderAuditInfo copyWith({String? createdBy, String? updatedBy}) {
    return OrderAuditInfo(
      createdBy: createdBy ?? this.createdBy,
      updatedBy: updatedBy ?? this.updatedBy,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is OrderAuditInfo &&
            other.createdBy == createdBy &&
            other.updatedBy == updatedBy);
  }

  @override
  int get hashCode => Object.hash(createdBy, updatedBy);
}

@immutable
class OrderShipmentEvent {
  const OrderShipmentEvent({
    required this.timestamp,
    required this.code,
    this.location,
    this.note,
  });

  final DateTime timestamp;
  final OrderShipmentEventCode code;
  final String? location;
  final String? note;

  OrderShipmentEvent copyWith({
    DateTime? timestamp,
    OrderShipmentEventCode? code,
    String? location,
    String? note,
  }) {
    return OrderShipmentEvent(
      timestamp: timestamp ?? this.timestamp,
      code: code ?? this.code,
      location: location ?? this.location,
      note: note ?? this.note,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is OrderShipmentEvent &&
            other.timestamp == timestamp &&
            other.code == code &&
            other.location == location &&
            other.note == note);
  }

  @override
  int get hashCode => Object.hash(timestamp, code, location, note);
}

@immutable
class OrderShipment {
  const OrderShipment({
    required this.id,
    required this.carrier,
    required this.status,
    required this.createdAt,
    this.service,
    this.trackingNumber,
    this.eta,
    this.labelUrl,
    this.documents = const [],
    this.events = const [],
    this.updatedAt,
  });

  final String id;
  final OrderShipmentCarrier carrier;
  final String? service;
  final String? trackingNumber;
  final OrderShipmentStatus status;
  final DateTime? eta;
  final String? labelUrl;
  final List<String> documents;
  final List<OrderShipmentEvent> events;
  final DateTime createdAt;
  final DateTime? updatedAt;

  OrderShipment copyWith({
    String? id,
    OrderShipmentCarrier? carrier,
    String? service,
    String? trackingNumber,
    OrderShipmentStatus? status,
    DateTime? eta,
    String? labelUrl,
    List<String>? documents,
    List<OrderShipmentEvent>? events,
    DateTime? createdAt,
    DateTime? updatedAt,
  }) {
    return OrderShipment(
      id: id ?? this.id,
      carrier: carrier ?? this.carrier,
      service: service ?? this.service,
      trackingNumber: trackingNumber ?? this.trackingNumber,
      status: status ?? this.status,
      eta: eta ?? this.eta,
      labelUrl: labelUrl ?? this.labelUrl,
      documents: documents ?? this.documents,
      events: events ?? this.events,
      createdAt: createdAt ?? this.createdAt,
      updatedAt: updatedAt ?? this.updatedAt,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is OrderShipment &&
            other.id == id &&
            other.carrier == carrier &&
            other.service == service &&
            other.trackingNumber == trackingNumber &&
            other.status == status &&
            other.eta == eta &&
            other.labelUrl == labelUrl &&
            listEquals(other.documents, documents) &&
            listEquals(other.events, events) &&
            other.createdAt == createdAt &&
            other.updatedAt == updatedAt);
  }

  @override
  int get hashCode {
    return Object.hashAll([
      id,
      carrier,
      service,
      trackingNumber,
      status,
      eta,
      labelUrl,
      Object.hashAll(documents),
      Object.hashAll(events),
      createdAt,
      updatedAt,
    ]);
  }
}

@immutable
class OrderPaymentCapture {
  const OrderPaymentCapture({this.captured, this.capturedAt});

  final bool? captured;
  final DateTime? capturedAt;

  OrderPaymentCapture copyWith({bool? captured, DateTime? capturedAt}) {
    return OrderPaymentCapture(
      captured: captured ?? this.captured,
      capturedAt: capturedAt ?? this.capturedAt,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is OrderPaymentCapture &&
            other.captured == captured &&
            other.capturedAt == capturedAt);
  }

  @override
  int get hashCode => Object.hash(captured, capturedAt);
}

@immutable
class OrderPaymentMethodSnapshot {
  const OrderPaymentMethodSnapshot({
    this.type,
    this.brand,
    this.last4,
    this.expMonth,
    this.expYear,
  });

  final OrderPaymentMethodType? type;
  final String? brand;
  final String? last4;
  final int? expMonth;
  final int? expYear;

  OrderPaymentMethodSnapshot copyWith({
    OrderPaymentMethodType? type,
    String? brand,
    String? last4,
    int? expMonth,
    int? expYear,
  }) {
    return OrderPaymentMethodSnapshot(
      type: type ?? this.type,
      brand: brand ?? this.brand,
      last4: last4 ?? this.last4,
      expMonth: expMonth ?? this.expMonth,
      expYear: expYear ?? this.expYear,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is OrderPaymentMethodSnapshot &&
            other.type == type &&
            other.brand == brand &&
            other.last4 == last4 &&
            other.expMonth == expMonth &&
            other.expYear == expYear);
  }

  @override
  int get hashCode => Object.hash(type, brand, last4, expMonth, expYear);
}

@immutable
class OrderPaymentError {
  const OrderPaymentError({this.code, this.message});

  final String? code;
  final String? message;

  OrderPaymentError copyWith({String? code, String? message}) {
    return OrderPaymentError(
      code: code ?? this.code,
      message: message ?? this.message,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is OrderPaymentError &&
            other.code == code &&
            other.message == message);
  }

  @override
  int get hashCode => Object.hash(code, message);
}

@immutable
class OrderPayment {
  const OrderPayment({
    required this.id,
    required this.provider,
    required this.status,
    required this.amount,
    required this.currency,
    required this.createdAt,
    this.intentId,
    this.chargeId,
    this.capture,
    this.method,
    this.billingAddress,
    this.error,
    this.raw,
    this.idempotencyKey,
    this.updatedAt,
    this.settledAt,
    this.refundedAt,
  });

  final String id;
  final OrderPaymentProvider provider;
  final OrderPaymentStatus status;
  final int amount;
  final String currency;
  final String? intentId;
  final String? chargeId;
  final OrderPaymentCapture? capture;
  final OrderPaymentMethodSnapshot? method;
  final OrderAddress? billingAddress;
  final OrderPaymentError? error;
  final Map<String, dynamic>? raw;
  final String? idempotencyKey;
  final DateTime createdAt;
  final DateTime? updatedAt;
  final DateTime? settledAt;
  final DateTime? refundedAt;

  OrderPayment copyWith({
    String? id,
    OrderPaymentProvider? provider,
    OrderPaymentStatus? status,
    int? amount,
    String? currency,
    String? intentId,
    String? chargeId,
    OrderPaymentCapture? capture,
    OrderPaymentMethodSnapshot? method,
    OrderAddress? billingAddress,
    OrderPaymentError? error,
    Map<String, dynamic>? raw,
    String? idempotencyKey,
    DateTime? createdAt,
    DateTime? updatedAt,
    DateTime? settledAt,
    DateTime? refundedAt,
  }) {
    return OrderPayment(
      id: id ?? this.id,
      provider: provider ?? this.provider,
      status: status ?? this.status,
      amount: amount ?? this.amount,
      currency: currency ?? this.currency,
      intentId: intentId ?? this.intentId,
      chargeId: chargeId ?? this.chargeId,
      capture: capture ?? this.capture,
      method: method ?? this.method,
      billingAddress: billingAddress ?? this.billingAddress,
      error: error ?? this.error,
      raw: raw ?? this.raw,
      idempotencyKey: idempotencyKey ?? this.idempotencyKey,
      createdAt: createdAt ?? this.createdAt,
      updatedAt: updatedAt ?? this.updatedAt,
      settledAt: settledAt ?? this.settledAt,
      refundedAt: refundedAt ?? this.refundedAt,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is OrderPayment &&
            other.id == id &&
            other.provider == provider &&
            other.status == status &&
            other.amount == amount &&
            other.currency == currency &&
            other.intentId == intentId &&
            other.chargeId == chargeId &&
            other.capture == capture &&
            other.method == method &&
            other.billingAddress == billingAddress &&
            other.error == error &&
            mapEquals(other.raw, raw) &&
            other.idempotencyKey == idempotencyKey &&
            other.createdAt == createdAt &&
            other.updatedAt == updatedAt &&
            other.settledAt == settledAt &&
            other.refundedAt == refundedAt);
  }

  @override
  int get hashCode {
    return Object.hashAll([
      id,
      provider,
      status,
      amount,
      currency,
      intentId,
      chargeId,
      capture,
      method,
      billingAddress,
      error,
      raw == null ? null : Object.hashAll(raw!.entries),
      idempotencyKey,
      createdAt,
      updatedAt,
      settledAt,
      refundedAt,
    ]);
  }
}

@immutable
class ProductionEvent {
  const ProductionEvent({
    required this.id,
    required this.type,
    required this.createdAt,
    this.station,
    this.operatorRef,
    this.durationSec,
    this.note,
    this.photoUrl,
    this.qcResult,
    this.qcDefects = const [],
  });

  final String id;
  final ProductionEventType type;
  final String? station;
  final String? operatorRef;
  final int? durationSec;
  final String? note;
  final String? photoUrl;
  final String? qcResult;
  final List<String> qcDefects;
  final DateTime createdAt;

  ProductionEvent copyWith({
    String? id,
    ProductionEventType? type,
    String? station,
    String? operatorRef,
    int? durationSec,
    String? note,
    String? photoUrl,
    String? qcResult,
    List<String>? qcDefects,
    DateTime? createdAt,
  }) {
    return ProductionEvent(
      id: id ?? this.id,
      type: type ?? this.type,
      station: station ?? this.station,
      operatorRef: operatorRef ?? this.operatorRef,
      durationSec: durationSec ?? this.durationSec,
      note: note ?? this.note,
      photoUrl: photoUrl ?? this.photoUrl,
      qcResult: qcResult ?? this.qcResult,
      qcDefects: qcDefects ?? this.qcDefects,
      createdAt: createdAt ?? this.createdAt,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is ProductionEvent &&
            other.id == id &&
            other.type == type &&
            other.station == station &&
            other.operatorRef == operatorRef &&
            other.durationSec == durationSec &&
            other.note == note &&
            other.photoUrl == photoUrl &&
            other.qcResult == qcResult &&
            listEquals(other.qcDefects, qcDefects) &&
            other.createdAt == createdAt);
  }

  @override
  int get hashCode {
    return Object.hashAll([
      id,
      type,
      station,
      operatorRef,
      durationSec,
      note,
      photoUrl,
      qcResult,
      Object.hashAll(qcDefects),
      createdAt,
    ]);
  }
}

@immutable
class Order {
  const Order({
    required this.id,
    required this.orderNumber,
    required this.userRef,
    required this.status,
    required this.currency,
    required this.totals,
    required this.lineItems,
    required this.createdAt,
    required this.updatedAt,
    this.cartRef,
    this.promotion,
    this.shippingAddress,
    this.billingAddress,
    this.contact,
    this.fulfillment,
    this.production,
    this.notes,
    this.flags,
    this.audit,
    this.placedAt,
    this.paidAt,
    this.shippedAt,
    this.deliveredAt,
    this.canceledAt,
    this.cancelReason,
    this.metadata,
  });

  final String id;
  final String orderNumber;
  final String userRef;
  final String? cartRef;
  final OrderStatus status;
  final String currency;
  final OrderTotals totals;
  final OrderPromotionSnapshot? promotion;
  final List<OrderLineItem> lineItems;
  final OrderAddress? shippingAddress;
  final OrderAddress? billingAddress;
  final OrderContact? contact;
  final OrderFulfillmentInfo? fulfillment;
  final OrderProductionInfo? production;
  final Map<String, dynamic>? notes;
  final OrderFlags? flags;
  final OrderAuditInfo? audit;
  final DateTime createdAt;
  final DateTime updatedAt;
  final DateTime? placedAt;
  final DateTime? paidAt;
  final DateTime? shippedAt;
  final DateTime? deliveredAt;
  final DateTime? canceledAt;
  final String? cancelReason;
  final Map<String, dynamic>? metadata;

  Order copyWith({
    String? id,
    String? orderNumber,
    String? userRef,
    String? cartRef,
    OrderStatus? status,
    String? currency,
    OrderTotals? totals,
    OrderPromotionSnapshot? promotion,
    List<OrderLineItem>? lineItems,
    OrderAddress? shippingAddress,
    OrderAddress? billingAddress,
    OrderContact? contact,
    OrderFulfillmentInfo? fulfillment,
    OrderProductionInfo? production,
    Map<String, dynamic>? notes,
    OrderFlags? flags,
    OrderAuditInfo? audit,
    DateTime? createdAt,
    DateTime? updatedAt,
    DateTime? placedAt,
    DateTime? paidAt,
    DateTime? shippedAt,
    DateTime? deliveredAt,
    DateTime? canceledAt,
    String? cancelReason,
    Map<String, dynamic>? metadata,
  }) {
    return Order(
      id: id ?? this.id,
      orderNumber: orderNumber ?? this.orderNumber,
      userRef: userRef ?? this.userRef,
      cartRef: cartRef ?? this.cartRef,
      status: status ?? this.status,
      currency: currency ?? this.currency,
      totals: totals ?? this.totals,
      promotion: promotion ?? this.promotion,
      lineItems: lineItems ?? this.lineItems,
      shippingAddress: shippingAddress ?? this.shippingAddress,
      billingAddress: billingAddress ?? this.billingAddress,
      contact: contact ?? this.contact,
      fulfillment: fulfillment ?? this.fulfillment,
      production: production ?? this.production,
      notes: notes ?? this.notes,
      flags: flags ?? this.flags,
      audit: audit ?? this.audit,
      createdAt: createdAt ?? this.createdAt,
      updatedAt: updatedAt ?? this.updatedAt,
      placedAt: placedAt ?? this.placedAt,
      paidAt: paidAt ?? this.paidAt,
      shippedAt: shippedAt ?? this.shippedAt,
      deliveredAt: deliveredAt ?? this.deliveredAt,
      canceledAt: canceledAt ?? this.canceledAt,
      cancelReason: cancelReason ?? this.cancelReason,
      metadata: metadata ?? this.metadata,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is Order &&
            other.id == id &&
            other.orderNumber == orderNumber &&
            other.userRef == userRef &&
            other.cartRef == cartRef &&
            other.status == status &&
            other.currency == currency &&
            other.totals == totals &&
            other.promotion == promotion &&
            listEquals(other.lineItems, lineItems) &&
            other.shippingAddress == shippingAddress &&
            other.billingAddress == billingAddress &&
            other.contact == contact &&
            other.fulfillment == fulfillment &&
            other.production == production &&
            mapEquals(other.notes, notes) &&
            other.flags == flags &&
            other.audit == audit &&
            other.createdAt == createdAt &&
            other.updatedAt == updatedAt &&
            other.placedAt == placedAt &&
            other.paidAt == paidAt &&
            other.shippedAt == shippedAt &&
            other.deliveredAt == deliveredAt &&
            other.canceledAt == canceledAt &&
            other.cancelReason == cancelReason &&
            mapEquals(other.metadata, metadata));
  }

  @override
  int get hashCode {
    return Object.hashAll([
      id,
      orderNumber,
      userRef,
      cartRef,
      status,
      currency,
      totals,
      promotion,
      Object.hashAll(lineItems),
      shippingAddress,
      billingAddress,
      contact,
      fulfillment,
      production,
      notes == null ? null : Object.hashAll(notes!.entries),
      flags,
      audit,
      createdAt,
      updatedAt,
      placedAt,
      paidAt,
      shippedAt,
      deliveredAt,
      canceledAt,
      cancelReason,
      metadata == null ? null : Object.hashAll(metadata!.entries),
    ]);
  }
}
