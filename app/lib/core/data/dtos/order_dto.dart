import 'package:app/core/domain/entities/order.dart';

OrderStatus _parseOrderStatus(String value) {
  switch (value) {
    case 'draft':
      return OrderStatus.draft;
    case 'pending_payment':
      return OrderStatus.pendingPayment;
    case 'paid':
      return OrderStatus.paid;
    case 'in_production':
      return OrderStatus.inProduction;
    case 'ready_to_ship':
      return OrderStatus.readyToShip;
    case 'shipped':
      return OrderStatus.shipped;
    case 'delivered':
      return OrderStatus.delivered;
    case 'canceled':
      return OrderStatus.canceled;
  }
  throw ArgumentError.value(value, 'value', 'Unknown OrderStatus');
}

String _orderStatusToJson(OrderStatus status) {
  switch (status) {
    case OrderStatus.draft:
      return 'draft';
    case OrderStatus.pendingPayment:
      return 'pending_payment';
    case OrderStatus.paid:
      return 'paid';
    case OrderStatus.inProduction:
      return 'in_production';
    case OrderStatus.readyToShip:
      return 'ready_to_ship';
    case OrderStatus.shipped:
      return 'shipped';
    case OrderStatus.delivered:
      return 'delivered';
    case OrderStatus.canceled:
      return 'canceled';
  }
}

OrderShipmentCarrier _parseShipmentCarrier(String value) {
  switch (value.toUpperCase()) {
    case 'JPPOST':
      return OrderShipmentCarrier.jppost;
    case 'YAMATO':
      return OrderShipmentCarrier.yamato;
    case 'SAGAWA':
      return OrderShipmentCarrier.sagawa;
    case 'DHL':
      return OrderShipmentCarrier.dhl;
    case 'UPS':
      return OrderShipmentCarrier.ups;
    case 'FEDEX':
      return OrderShipmentCarrier.fedex;
    case 'OTHER':
      return OrderShipmentCarrier.other;
  }
  throw ArgumentError.value(value, 'value', 'Unknown OrderShipmentCarrier');
}

String _shipmentCarrierToJson(OrderShipmentCarrier carrier) {
  switch (carrier) {
    case OrderShipmentCarrier.jppost:
      return 'JPPOST';
    case OrderShipmentCarrier.yamato:
      return 'YAMATO';
    case OrderShipmentCarrier.sagawa:
      return 'SAGAWA';
    case OrderShipmentCarrier.dhl:
      return 'DHL';
    case OrderShipmentCarrier.ups:
      return 'UPS';
    case OrderShipmentCarrier.fedex:
      return 'FEDEX';
    case OrderShipmentCarrier.other:
      return 'OTHER';
  }
}

OrderShipmentStatus _parseShipmentStatus(String value) {
  switch (value) {
    case 'label_created':
      return OrderShipmentStatus.labelCreated;
    case 'in_transit':
      return OrderShipmentStatus.inTransit;
    case 'out_for_delivery':
      return OrderShipmentStatus.outForDelivery;
    case 'delivered':
      return OrderShipmentStatus.delivered;
    case 'exception':
      return OrderShipmentStatus.exception;
    case 'cancelled':
      return OrderShipmentStatus.cancelled;
  }
  throw ArgumentError.value(value, 'value', 'Unknown OrderShipmentStatus');
}

String _shipmentStatusToJson(OrderShipmentStatus status) {
  switch (status) {
    case OrderShipmentStatus.labelCreated:
      return 'label_created';
    case OrderShipmentStatus.inTransit:
      return 'in_transit';
    case OrderShipmentStatus.outForDelivery:
      return 'out_for_delivery';
    case OrderShipmentStatus.delivered:
      return 'delivered';
    case OrderShipmentStatus.exception:
      return 'exception';
    case OrderShipmentStatus.cancelled:
      return 'cancelled';
  }
}

OrderShipmentEventCode _parseShipmentEventCode(String value) {
  switch (value) {
    case 'label_created':
      return OrderShipmentEventCode.labelCreated;
    case 'picked_up':
      return OrderShipmentEventCode.pickedUp;
    case 'in_transit':
      return OrderShipmentEventCode.inTransit;
    case 'arrived_hub':
      return OrderShipmentEventCode.arrivedHub;
    case 'customs_clearance':
      return OrderShipmentEventCode.customsClearance;
    case 'out_for_delivery':
      return OrderShipmentEventCode.outForDelivery;
    case 'delivered':
      return OrderShipmentEventCode.delivered;
    case 'exception':
      return OrderShipmentEventCode.exception;
    case 'return_to_sender':
      return OrderShipmentEventCode.returnToSender;
  }
  throw ArgumentError.value(value, 'value', 'Unknown OrderShipmentEventCode');
}

String _shipmentEventCodeToJson(OrderShipmentEventCode code) {
  switch (code) {
    case OrderShipmentEventCode.labelCreated:
      return 'label_created';
    case OrderShipmentEventCode.pickedUp:
      return 'picked_up';
    case OrderShipmentEventCode.inTransit:
      return 'in_transit';
    case OrderShipmentEventCode.arrivedHub:
      return 'arrived_hub';
    case OrderShipmentEventCode.customsClearance:
      return 'customs_clearance';
    case OrderShipmentEventCode.outForDelivery:
      return 'out_for_delivery';
    case OrderShipmentEventCode.delivered:
      return 'delivered';
    case OrderShipmentEventCode.exception:
      return 'exception';
    case OrderShipmentEventCode.returnToSender:
      return 'return_to_sender';
  }
}

OrderPaymentProvider _parsePaymentProvider(String value) {
  switch (value) {
    case 'stripe':
      return OrderPaymentProvider.stripe;
    case 'paypal':
      return OrderPaymentProvider.paypal;
    case 'other':
      return OrderPaymentProvider.other;
  }
  throw ArgumentError.value(value, 'value', 'Unknown OrderPaymentProvider');
}

String _paymentProviderToJson(OrderPaymentProvider provider) {
  switch (provider) {
    case OrderPaymentProvider.stripe:
      return 'stripe';
    case OrderPaymentProvider.paypal:
      return 'paypal';
    case OrderPaymentProvider.other:
      return 'other';
  }
}

OrderPaymentStatus _parsePaymentStatus(String value) {
  switch (value) {
    case 'requires_action':
      return OrderPaymentStatus.requiresAction;
    case 'authorized':
      return OrderPaymentStatus.authorized;
    case 'succeeded':
      return OrderPaymentStatus.succeeded;
    case 'failed':
      return OrderPaymentStatus.failed;
    case 'refunded':
      return OrderPaymentStatus.refunded;
    case 'partially_refunded':
      return OrderPaymentStatus.partiallyRefunded;
    case 'canceled':
      return OrderPaymentStatus.canceled;
  }
  throw ArgumentError.value(value, 'value', 'Unknown OrderPaymentStatus');
}

String _paymentStatusToJson(OrderPaymentStatus status) {
  switch (status) {
    case OrderPaymentStatus.requiresAction:
      return 'requires_action';
    case OrderPaymentStatus.authorized:
      return 'authorized';
    case OrderPaymentStatus.succeeded:
      return 'succeeded';
    case OrderPaymentStatus.failed:
      return 'failed';
    case OrderPaymentStatus.refunded:
      return 'refunded';
    case OrderPaymentStatus.partiallyRefunded:
      return 'partially_refunded';
    case OrderPaymentStatus.canceled:
      return 'canceled';
  }
}

OrderPaymentMethodType? _parsePaymentMethodType(String? value) {
  switch (value) {
    case null:
      return null;
    case 'card':
      return OrderPaymentMethodType.card;
    case 'wallet':
      return OrderPaymentMethodType.wallet;
    case 'bank':
      return OrderPaymentMethodType.bank;
    case 'other':
      return OrderPaymentMethodType.other;
  }
  throw ArgumentError.value(value, 'value', 'Unknown OrderPaymentMethodType');
}

String? _paymentMethodTypeToJson(OrderPaymentMethodType? type) {
  switch (type) {
    case null:
      return null;
    case OrderPaymentMethodType.card:
      return 'card';
    case OrderPaymentMethodType.wallet:
      return 'wallet';
    case OrderPaymentMethodType.bank:
      return 'bank';
    case OrderPaymentMethodType.other:
      return 'other';
  }
}

ProductionEventType _parseProductionEventType(String value) {
  switch (value) {
    case 'queued':
      return ProductionEventType.queued;
    case 'engraving':
      return ProductionEventType.engraving;
    case 'polishing':
      return ProductionEventType.polishing;
    case 'qc':
      return ProductionEventType.qc;
    case 'packed':
      return ProductionEventType.packed;
    case 'on_hold':
      return ProductionEventType.onHold;
    case 'rework':
      return ProductionEventType.rework;
    case 'canceled':
      return ProductionEventType.canceled;
  }
  throw ArgumentError.value(value, 'value', 'Unknown ProductionEventType');
}

String _productionEventTypeToJson(ProductionEventType type) {
  switch (type) {
    case ProductionEventType.queued:
      return 'queued';
    case ProductionEventType.engraving:
      return 'engraving';
    case ProductionEventType.polishing:
      return 'polishing';
    case ProductionEventType.qc:
      return 'qc';
    case ProductionEventType.packed:
      return 'packed';
    case ProductionEventType.onHold:
      return 'on_hold';
    case ProductionEventType.rework:
      return 'rework';
    case ProductionEventType.canceled:
      return 'canceled';
  }
}

class OrderTotalsDto {
  OrderTotalsDto({
    required this.subtotal,
    required this.discount,
    required this.shipping,
    required this.tax,
    required this.total,
    this.fees,
  });

  factory OrderTotalsDto.fromJson(Map<String, dynamic> json) {
    return OrderTotalsDto(
      subtotal: json['subtotal'] as int,
      discount: json['discount'] as int,
      shipping: json['shipping'] as int,
      tax: json['tax'] as int,
      total: json['total'] as int,
      fees: json['fees'] as int?,
    );
  }

  factory OrderTotalsDto.fromDomain(OrderTotals domain) {
    return OrderTotalsDto(
      subtotal: domain.subtotal,
      discount: domain.discount,
      shipping: domain.shipping,
      tax: domain.tax,
      total: domain.total,
      fees: domain.fees,
    );
  }

  final int subtotal;
  final int discount;
  final int shipping;
  final int tax;
  final int? fees;
  final int total;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'subtotal': subtotal,
      'discount': discount,
      'shipping': shipping,
      'tax': tax,
      'fees': fees,
      'total': total,
    };
  }

  OrderTotals toDomain() {
    return OrderTotals(
      subtotal: subtotal,
      discount: discount,
      shipping: shipping,
      tax: tax,
      fees: fees ?? 0,
      total: total,
    );
  }
}

class OrderPromotionSnapshotDto {
  OrderPromotionSnapshotDto({
    required this.code,
    required this.applied,
    this.discountAmount,
  });

  factory OrderPromotionSnapshotDto.fromJson(Map<String, dynamic> json) {
    return OrderPromotionSnapshotDto(
      code: json['code'] as String,
      applied: json['applied'] as bool? ?? false,
      discountAmount: json['discountAmount'] as int?,
    );
  }

  factory OrderPromotionSnapshotDto.fromDomain(OrderPromotionSnapshot domain) {
    return OrderPromotionSnapshotDto(
      code: domain.code,
      applied: domain.applied,
      discountAmount: domain.discountAmount,
    );
  }

  final String code;
  final bool applied;
  final int? discountAmount;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'code': code,
      'applied': applied,
      'discountAmount': discountAmount,
    };
  }

  OrderPromotionSnapshot toDomain() {
    return OrderPromotionSnapshot(
      code: code,
      applied: applied,
      discountAmount: discountAmount,
    );
  }
}

class OrderLineItemDto {
  OrderLineItemDto({
    this.id,
    required this.productRef,
    this.designRef,
    this.designSnapshot,
    required this.sku,
    this.name,
    this.options,
    required this.quantity,
    required this.unitPrice,
    required this.total,
  });

  factory OrderLineItemDto.fromJson(Map<String, dynamic> json) {
    return OrderLineItemDto(
      id: json['id'] as String?,
      productRef: json['productRef'] as String,
      designRef: json['designRef'] as String?,
      designSnapshot: json['designSnapshot'] == null
          ? null
          : Map<String, dynamic>.from(json['designSnapshot'] as Map),
      sku: json['sku'] as String,
      name: json['name'] as String?,
      options: json['options'] == null
          ? null
          : Map<String, dynamic>.from(json['options'] as Map),
      quantity: json['quantity'] as int,
      unitPrice: json['unitPrice'] as int,
      total: json['total'] as int,
    );
  }

  factory OrderLineItemDto.fromDomain(OrderLineItem domain) {
    return OrderLineItemDto(
      id: domain.id,
      productRef: domain.productRef,
      designRef: domain.designRef,
      designSnapshot: domain.designSnapshot == null
          ? null
          : Map<String, dynamic>.from(domain.designSnapshot!),
      sku: domain.sku,
      name: domain.name,
      options: domain.options == null
          ? null
          : Map<String, dynamic>.from(domain.options!),
      quantity: domain.quantity,
      unitPrice: domain.unitPrice,
      total: domain.total,
    );
  }

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

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'id': id,
      'productRef': productRef,
      'designRef': designRef,
      'designSnapshot': designSnapshot,
      'sku': sku,
      'name': name,
      'options': options,
      'quantity': quantity,
      'unitPrice': unitPrice,
      'total': total,
    };
  }

  OrderLineItem toDomain() {
    return OrderLineItem(
      id: id,
      productRef: productRef,
      designRef: designRef,
      designSnapshot: designSnapshot == null
          ? null
          : Map<String, dynamic>.from(designSnapshot!),
      sku: sku,
      name: name,
      options: options == null ? null : Map<String, dynamic>.from(options!),
      quantity: quantity,
      unitPrice: unitPrice,
      total: total,
    );
  }
}

class OrderAddressDto {
  OrderAddressDto({
    required this.recipient,
    required this.line1,
    this.line2,
    required this.city,
    this.state,
    required this.postalCode,
    required this.country,
    this.phone,
  });

  factory OrderAddressDto.fromJson(Map<String, dynamic> json) {
    return OrderAddressDto(
      recipient: json['recipient'] as String,
      line1: json['line1'] as String,
      line2: json['line2'] as String?,
      city: json['city'] as String,
      state: json['state'] as String?,
      postalCode: json['postalCode'] as String,
      country: json['country'] as String,
      phone: json['phone'] as String?,
    );
  }

  factory OrderAddressDto.fromDomain(OrderAddress domain) {
    return OrderAddressDto(
      recipient: domain.recipient,
      line1: domain.line1,
      line2: domain.line2,
      city: domain.city,
      state: domain.state,
      postalCode: domain.postalCode,
      country: domain.country,
      phone: domain.phone,
    );
  }

  final String recipient;
  final String line1;
  final String? line2;
  final String city;
  final String? state;
  final String postalCode;
  final String country;
  final String? phone;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'recipient': recipient,
      'line1': line1,
      'line2': line2,
      'city': city,
      'state': state,
      'postalCode': postalCode,
      'country': country,
      'phone': phone,
    };
  }

  OrderAddress toDomain() {
    return OrderAddress(
      recipient: recipient,
      line1: line1,
      line2: line2,
      city: city,
      state: state,
      postalCode: postalCode,
      country: country,
      phone: phone,
    );
  }
}

class OrderContactDto {
  OrderContactDto({this.email, this.phone});

  factory OrderContactDto.fromJson(Map<String, dynamic> json) {
    return OrderContactDto(
      email: json['email'] as String?,
      phone: json['phone'] as String?,
    );
  }

  factory OrderContactDto.fromDomain(OrderContact? domain) {
    if (domain == null) {
      return OrderContactDto();
    }
    return OrderContactDto(email: domain.email, phone: domain.phone);
  }

  final String? email;
  final String? phone;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{'email': email, 'phone': phone};
  }

  OrderContact toDomain() {
    return OrderContact(email: email, phone: phone);
  }
}

class OrderFulfillmentInfoDto {
  OrderFulfillmentInfoDto({
    this.requestedAt,
    this.estimatedShipDate,
    this.estimatedDeliveryDate,
  });

  factory OrderFulfillmentInfoDto.fromJson(Map<String, dynamic> json) {
    return OrderFulfillmentInfoDto(
      requestedAt: json['requestedAt'] as String?,
      estimatedShipDate: json['estimatedShipDate'] as String?,
      estimatedDeliveryDate: json['estimatedDeliveryDate'] as String?,
    );
  }

  factory OrderFulfillmentInfoDto.fromDomain(OrderFulfillmentInfo? domain) {
    if (domain == null) {
      return OrderFulfillmentInfoDto();
    }
    return OrderFulfillmentInfoDto(
      requestedAt: domain.requestedAt?.toIso8601String(),
      estimatedShipDate: domain.estimatedShipDate?.toIso8601String(),
      estimatedDeliveryDate: domain.estimatedDeliveryDate?.toIso8601String(),
    );
  }

  final String? requestedAt;
  final String? estimatedShipDate;
  final String? estimatedDeliveryDate;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'requestedAt': requestedAt,
      'estimatedShipDate': estimatedShipDate,
      'estimatedDeliveryDate': estimatedDeliveryDate,
    };
  }

  OrderFulfillmentInfo toDomain() {
    return OrderFulfillmentInfo(
      requestedAt: requestedAt == null ? null : DateTime.parse(requestedAt!),
      estimatedShipDate: estimatedShipDate == null
          ? null
          : DateTime.parse(estimatedShipDate!),
      estimatedDeliveryDate: estimatedDeliveryDate == null
          ? null
          : DateTime.parse(estimatedDeliveryDate!),
    );
  }
}

class OrderProductionInfoDto {
  OrderProductionInfoDto({
    this.queueRef,
    this.assignedStation,
    this.operatorRef,
  });

  factory OrderProductionInfoDto.fromJson(Map<String, dynamic> json) {
    return OrderProductionInfoDto(
      queueRef: json['queueRef'] as String?,
      assignedStation: json['assignedStation'] as String?,
      operatorRef: json['operatorRef'] as String?,
    );
  }

  factory OrderProductionInfoDto.fromDomain(OrderProductionInfo? domain) {
    if (domain == null) {
      return OrderProductionInfoDto();
    }
    return OrderProductionInfoDto(
      queueRef: domain.queueRef,
      assignedStation: domain.assignedStation,
      operatorRef: domain.operatorRef,
    );
  }

  final String? queueRef;
  final String? assignedStation;
  final String? operatorRef;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'queueRef': queueRef,
      'assignedStation': assignedStation,
      'operatorRef': operatorRef,
    };
  }

  OrderProductionInfo toDomain() {
    return OrderProductionInfo(
      queueRef: queueRef,
      assignedStation: assignedStation,
      operatorRef: operatorRef,
    );
  }
}

class OrderFlagsDto {
  OrderFlagsDto({this.manualReview, this.gift});

  factory OrderFlagsDto.fromJson(Map<String, dynamic> json) {
    return OrderFlagsDto(
      manualReview: json['manualReview'] as bool?,
      gift: json['gift'] as bool?,
    );
  }

  factory OrderFlagsDto.fromDomain(OrderFlags? domain) {
    if (domain == null) {
      return OrderFlagsDto();
    }
    return OrderFlagsDto(manualReview: domain.manualReview, gift: domain.gift);
  }

  final bool? manualReview;
  final bool? gift;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{'manualReview': manualReview, 'gift': gift};
  }

  OrderFlags toDomain() {
    return OrderFlags(manualReview: manualReview, gift: gift);
  }
}

class OrderAuditInfoDto {
  OrderAuditInfoDto({this.createdBy, this.updatedBy});

  factory OrderAuditInfoDto.fromJson(Map<String, dynamic> json) {
    return OrderAuditInfoDto(
      createdBy: json['createdBy'] as String?,
      updatedBy: json['updatedBy'] as String?,
    );
  }

  factory OrderAuditInfoDto.fromDomain(OrderAuditInfo? domain) {
    if (domain == null) {
      return OrderAuditInfoDto();
    }
    return OrderAuditInfoDto(
      createdBy: domain.createdBy,
      updatedBy: domain.updatedBy,
    );
  }

  final String? createdBy;
  final String? updatedBy;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{'createdBy': createdBy, 'updatedBy': updatedBy};
  }

  OrderAuditInfo toDomain() {
    return OrderAuditInfo(createdBy: createdBy, updatedBy: updatedBy);
  }
}

class OrderShipmentEventDto {
  OrderShipmentEventDto({
    required this.ts,
    required this.code,
    this.location,
    this.note,
  });

  factory OrderShipmentEventDto.fromJson(Map<String, dynamic> json) {
    return OrderShipmentEventDto(
      ts: json['ts'] as String,
      code: json['code'] as String,
      location: json['location'] as String?,
      note: json['note'] as String?,
    );
  }

  factory OrderShipmentEventDto.fromDomain(OrderShipmentEvent domain) {
    return OrderShipmentEventDto(
      ts: domain.timestamp.toIso8601String(),
      code: _shipmentEventCodeToJson(domain.code),
      location: domain.location,
      note: domain.note,
    );
  }

  final String ts;
  final String code;
  final String? location;
  final String? note;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'ts': ts,
      'code': code,
      'location': location,
      'note': note,
    };
  }

  OrderShipmentEvent toDomain() {
    return OrderShipmentEvent(
      timestamp: DateTime.parse(ts),
      code: _parseShipmentEventCode(code),
      location: location,
      note: note,
    );
  }
}

class OrderShipmentDto {
  OrderShipmentDto({
    required this.id,
    required this.carrier,
    required this.status,
    required this.createdAt,
    this.service,
    this.trackingNumber,
    this.eta,
    this.labelUrl,
    this.documents,
    this.events,
    this.updatedAt,
  });

  factory OrderShipmentDto.fromJson(Map<String, dynamic> json) {
    return OrderShipmentDto(
      id: json['id'] as String,
      carrier: json['carrier'] as String,
      status: json['status'] as String,
      createdAt: json['createdAt'] as String,
      service: json['service'] as String?,
      trackingNumber: json['trackingNumber'] as String?,
      eta: json['eta'] as String?,
      labelUrl: json['labelUrl'] as String?,
      documents: (json['documents'] as List<dynamic>?)?.cast<String>(),
      events: (json['events'] as List<dynamic>?)
          ?.map(
            (dynamic e) =>
                OrderShipmentEventDto.fromJson(e as Map<String, dynamic>),
          )
          .toList(),
      updatedAt: json['updatedAt'] as String?,
    );
  }

  factory OrderShipmentDto.fromDomain(OrderShipment domain) {
    return OrderShipmentDto(
      id: domain.id,
      carrier: _shipmentCarrierToJson(domain.carrier),
      status: _shipmentStatusToJson(domain.status),
      createdAt: domain.createdAt.toIso8601String(),
      service: domain.service,
      trackingNumber: domain.trackingNumber,
      eta: domain.eta?.toIso8601String(),
      labelUrl: domain.labelUrl,
      documents: domain.documents,
      events: domain.events.map(OrderShipmentEventDto.fromDomain).toList(),
      updatedAt: domain.updatedAt?.toIso8601String(),
    );
  }

  final String id;
  final String carrier;
  final String status;
  final String createdAt;
  final String? service;
  final String? trackingNumber;
  final String? eta;
  final String? labelUrl;
  final List<String>? documents;
  final List<OrderShipmentEventDto>? events;
  final String? updatedAt;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'id': id,
      'carrier': carrier,
      'status': status,
      'createdAt': createdAt,
      'service': service,
      'trackingNumber': trackingNumber,
      'eta': eta,
      'labelUrl': labelUrl,
      'documents': documents,
      'events': events?.map((OrderShipmentEventDto e) => e.toJson()).toList(),
      'updatedAt': updatedAt,
    };
  }

  OrderShipment toDomain() {
    return OrderShipment(
      id: id,
      carrier: _parseShipmentCarrier(carrier),
      status: _parseShipmentStatus(status),
      createdAt: DateTime.parse(createdAt),
      service: service,
      trackingNumber: trackingNumber,
      eta: eta == null ? null : DateTime.parse(eta!),
      labelUrl: labelUrl,
      documents: documents ?? const [],
      events:
          events?.map((OrderShipmentEventDto e) => e.toDomain()).toList() ??
          const <OrderShipmentEvent>[],
      updatedAt: updatedAt == null ? null : DateTime.parse(updatedAt!),
    );
  }
}

class OrderPaymentCaptureDto {
  OrderPaymentCaptureDto({this.captured, this.capturedAt});

  factory OrderPaymentCaptureDto.fromJson(Map<String, dynamic> json) {
    return OrderPaymentCaptureDto(
      captured: json['captured'] as bool?,
      capturedAt: json['capturedAt'] as String?,
    );
  }

  factory OrderPaymentCaptureDto.fromDomain(OrderPaymentCapture? domain) {
    if (domain == null) {
      return OrderPaymentCaptureDto();
    }
    return OrderPaymentCaptureDto(
      captured: domain.captured,
      capturedAt: domain.capturedAt?.toIso8601String(),
    );
  }

  final bool? captured;
  final String? capturedAt;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{'captured': captured, 'capturedAt': capturedAt};
  }

  OrderPaymentCapture toDomain() {
    return OrderPaymentCapture(
      captured: captured,
      capturedAt: capturedAt == null ? null : DateTime.parse(capturedAt!),
    );
  }
}

class OrderPaymentMethodSnapshotDto {
  OrderPaymentMethodSnapshotDto({
    this.type,
    this.brand,
    this.last4,
    this.expMonth,
    this.expYear,
  });

  factory OrderPaymentMethodSnapshotDto.fromJson(Map<String, dynamic> json) {
    return OrderPaymentMethodSnapshotDto(
      type: json['type'] as String?,
      brand: json['brand'] as String?,
      last4: json['last4'] as String?,
      expMonth: json['expMonth'] as int?,
      expYear: json['expYear'] as int?,
    );
  }

  factory OrderPaymentMethodSnapshotDto.fromDomain(
    OrderPaymentMethodSnapshot? domain,
  ) {
    if (domain == null) {
      return OrderPaymentMethodSnapshotDto();
    }
    return OrderPaymentMethodSnapshotDto(
      type: _paymentMethodTypeToJson(domain.type),
      brand: domain.brand,
      last4: domain.last4,
      expMonth: domain.expMonth,
      expYear: domain.expYear,
    );
  }

  final String? type;
  final String? brand;
  final String? last4;
  final int? expMonth;
  final int? expYear;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'type': type,
      'brand': brand,
      'last4': last4,
      'expMonth': expMonth,
      'expYear': expYear,
    };
  }

  OrderPaymentMethodSnapshot toDomain() {
    return OrderPaymentMethodSnapshot(
      type: _parsePaymentMethodType(type),
      brand: brand,
      last4: last4,
      expMonth: expMonth,
      expYear: expYear,
    );
  }
}

class OrderPaymentErrorDto {
  OrderPaymentErrorDto({this.code, this.message});

  factory OrderPaymentErrorDto.fromJson(Map<String, dynamic> json) {
    return OrderPaymentErrorDto(
      code: json['code'] as String?,
      message: json['message'] as String?,
    );
  }

  factory OrderPaymentErrorDto.fromDomain(OrderPaymentError? domain) {
    if (domain == null) {
      return OrderPaymentErrorDto();
    }
    return OrderPaymentErrorDto(code: domain.code, message: domain.message);
  }

  final String? code;
  final String? message;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{'code': code, 'message': message};
  }

  OrderPaymentError toDomain() {
    return OrderPaymentError(code: code, message: message);
  }
}

class OrderPaymentDto {
  OrderPaymentDto({
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

  factory OrderPaymentDto.fromJson(Map<String, dynamic> json) {
    return OrderPaymentDto(
      id: json['id'] as String,
      provider: json['provider'] as String,
      status: json['status'] as String,
      amount: json['amount'] as int,
      currency: json['currency'] as String,
      createdAt: json['createdAt'] as String,
      intentId: json['intentId'] as String?,
      chargeId: json['chargeId'] as String?,
      capture: json['capture'] == null
          ? null
          : OrderPaymentCaptureDto.fromJson(
              json['capture'] as Map<String, dynamic>,
            ),
      method: json['method'] == null
          ? null
          : OrderPaymentMethodSnapshotDto.fromJson(
              json['method'] as Map<String, dynamic>,
            ),
      billingAddress: json['billingAddress'] == null
          ? null
          : OrderAddressDto.fromJson(
              json['billingAddress'] as Map<String, dynamic>,
            ),
      error: json['error'] == null
          ? null
          : OrderPaymentErrorDto.fromJson(
              json['error'] as Map<String, dynamic>,
            ),
      raw: json['raw'] == null
          ? null
          : Map<String, dynamic>.from(json['raw'] as Map),
      idempotencyKey: json['idempotencyKey'] as String?,
      updatedAt: json['updatedAt'] as String?,
      settledAt: json['settledAt'] as String?,
      refundedAt: json['refundedAt'] as String?,
    );
  }

  factory OrderPaymentDto.fromDomain(OrderPayment domain) {
    return OrderPaymentDto(
      id: domain.id,
      provider: _paymentProviderToJson(domain.provider),
      status: _paymentStatusToJson(domain.status),
      amount: domain.amount,
      currency: domain.currency,
      createdAt: domain.createdAt.toIso8601String(),
      intentId: domain.intentId,
      chargeId: domain.chargeId,
      capture: domain.capture == null
          ? null
          : OrderPaymentCaptureDto.fromDomain(domain.capture),
      method: domain.method == null
          ? null
          : OrderPaymentMethodSnapshotDto.fromDomain(domain.method),
      billingAddress: domain.billingAddress == null
          ? null
          : OrderAddressDto.fromDomain(domain.billingAddress!),
      error: domain.error == null
          ? null
          : OrderPaymentErrorDto.fromDomain(domain.error),
      raw: domain.raw == null ? null : Map<String, dynamic>.from(domain.raw!),
      idempotencyKey: domain.idempotencyKey,
      updatedAt: domain.updatedAt?.toIso8601String(),
      settledAt: domain.settledAt?.toIso8601String(),
      refundedAt: domain.refundedAt?.toIso8601String(),
    );
  }

  final String id;
  final String provider;
  final String status;
  final int amount;
  final String currency;
  final String createdAt;
  final String? intentId;
  final String? chargeId;
  final OrderPaymentCaptureDto? capture;
  final OrderPaymentMethodSnapshotDto? method;
  final OrderAddressDto? billingAddress;
  final OrderPaymentErrorDto? error;
  final Map<String, dynamic>? raw;
  final String? idempotencyKey;
  final String? updatedAt;
  final String? settledAt;
  final String? refundedAt;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'id': id,
      'provider': provider,
      'status': status,
      'amount': amount,
      'currency': currency,
      'createdAt': createdAt,
      'intentId': intentId,
      'chargeId': chargeId,
      'capture': capture?.toJson(),
      'method': method?.toJson(),
      'billingAddress': billingAddress?.toJson(),
      'error': error?.toJson(),
      'raw': raw,
      'idempotencyKey': idempotencyKey,
      'updatedAt': updatedAt,
      'settledAt': settledAt,
      'refundedAt': refundedAt,
    };
  }

  OrderPayment toDomain() {
    return OrderPayment(
      id: id,
      provider: _parsePaymentProvider(provider),
      status: _parsePaymentStatus(status),
      amount: amount,
      currency: currency,
      createdAt: DateTime.parse(createdAt),
      intentId: intentId,
      chargeId: chargeId,
      capture: capture?.toDomain(),
      method: method?.toDomain(),
      billingAddress: billingAddress?.toDomain(),
      error: error?.toDomain(),
      raw: raw == null ? null : Map<String, dynamic>.from(raw!),
      idempotencyKey: idempotencyKey,
      updatedAt: updatedAt == null ? null : DateTime.parse(updatedAt!),
      settledAt: settledAt == null ? null : DateTime.parse(settledAt!),
      refundedAt: refundedAt == null ? null : DateTime.parse(refundedAt!),
    );
  }
}

class ProductionEventDto {
  ProductionEventDto({
    required this.id,
    required this.type,
    required this.createdAt,
    this.station,
    this.operatorRef,
    this.durationSec,
    this.note,
    this.photoUrl,
    this.qc,
  });

  factory ProductionEventDto.fromJson(Map<String, dynamic> json) {
    return ProductionEventDto(
      id: json['id'] as String,
      type: json['type'] as String,
      createdAt: json['createdAt'] as String,
      station: json['station'] as String?,
      operatorRef: json['operatorRef'] as String?,
      durationSec: json['durationSec'] as int?,
      note: json['note'] as String?,
      photoUrl: json['photoUrl'] as String?,
      qc: json['qc'] == null
          ? null
          : Map<String, dynamic>.from(json['qc'] as Map),
    );
  }

  factory ProductionEventDto.fromDomain(ProductionEvent domain) {
    return ProductionEventDto(
      id: domain.id,
      type: _productionEventTypeToJson(domain.type),
      createdAt: domain.createdAt.toIso8601String(),
      station: domain.station,
      operatorRef: domain.operatorRef,
      durationSec: domain.durationSec,
      note: domain.note,
      photoUrl: domain.photoUrl,
      qc: domain.qcResult == null && domain.qcDefects.isEmpty
          ? null
          : <String, dynamic>{
              'result': domain.qcResult,
              'defects': domain.qcDefects,
            },
    );
  }

  final String id;
  final String type;
  final String createdAt;
  final String? station;
  final String? operatorRef;
  final int? durationSec;
  final String? note;
  final String? photoUrl;
  final Map<String, dynamic>? qc;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'id': id,
      'type': type,
      'createdAt': createdAt,
      'station': station,
      'operatorRef': operatorRef,
      'durationSec': durationSec,
      'note': note,
      'photoUrl': photoUrl,
      'qc': qc,
    };
  }

  ProductionEvent toDomain() {
    final String? qcResult = qc == null ? null : qc!['result'] as String?;
    final List<String> qcDefects = qc == null
        ? const <String>[]
        : (qc!['defects'] as List<dynamic>? ?? const <dynamic>[])
              .cast<String>();
    return ProductionEvent(
      id: id,
      type: _parseProductionEventType(type),
      createdAt: DateTime.parse(createdAt),
      station: station,
      operatorRef: operatorRef,
      durationSec: durationSec,
      note: note,
      photoUrl: photoUrl,
      qcResult: qcResult,
      qcDefects: qcDefects,
    );
  }
}

class OrderDto {
  OrderDto({
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

  factory OrderDto.fromJson(Map<String, dynamic> json) {
    return OrderDto(
      id: json['id'] as String,
      orderNumber: json['orderNumber'] as String,
      userRef: json['userRef'] as String,
      cartRef: json['cartRef'] as String?,
      status: json['status'] as String,
      currency: json['currency'] as String,
      totals: OrderTotalsDto.fromJson(json['totals'] as Map<String, dynamic>),
      promotion: json['promotion'] == null
          ? null
          : OrderPromotionSnapshotDto.fromJson(
              json['promotion'] as Map<String, dynamic>,
            ),
      lineItems: (json['lineItems'] as List<dynamic>)
          .map(
            (dynamic e) => OrderLineItemDto.fromJson(e as Map<String, dynamic>),
          )
          .toList(),
      shippingAddress: json['shippingAddress'] == null
          ? null
          : OrderAddressDto.fromJson(
              json['shippingAddress'] as Map<String, dynamic>,
            ),
      billingAddress: json['billingAddress'] == null
          ? null
          : OrderAddressDto.fromJson(
              json['billingAddress'] as Map<String, dynamic>,
            ),
      contact: json['contact'] == null
          ? null
          : OrderContactDto.fromJson(json['contact'] as Map<String, dynamic>),
      fulfillment: json['fulfillment'] == null
          ? null
          : OrderFulfillmentInfoDto.fromJson(
              json['fulfillment'] as Map<String, dynamic>,
            ),
      production: json['production'] == null
          ? null
          : OrderProductionInfoDto.fromJson(
              json['production'] as Map<String, dynamic>,
            ),
      notes: json['notes'] == null
          ? null
          : Map<String, dynamic>.from(json['notes'] as Map),
      flags: json['flags'] == null
          ? null
          : OrderFlagsDto.fromJson(json['flags'] as Map<String, dynamic>),
      audit: json['audit'] == null
          ? null
          : OrderAuditInfoDto.fromJson(json['audit'] as Map<String, dynamic>),
      createdAt: json['createdAt'] as String,
      updatedAt: json['updatedAt'] as String,
      placedAt: json['placedAt'] as String?,
      paidAt: json['paidAt'] as String?,
      shippedAt: json['shippedAt'] as String?,
      deliveredAt: json['deliveredAt'] as String?,
      canceledAt: json['canceledAt'] as String?,
      cancelReason: json['cancelReason'] as String?,
      metadata: json['metadata'] == null
          ? null
          : Map<String, dynamic>.from(json['metadata'] as Map),
    );
  }

  factory OrderDto.fromDomain(Order domain) {
    return OrderDto(
      id: domain.id,
      orderNumber: domain.orderNumber,
      userRef: domain.userRef,
      cartRef: domain.cartRef,
      status: _orderStatusToJson(domain.status),
      currency: domain.currency,
      totals: OrderTotalsDto.fromDomain(domain.totals),
      promotion: domain.promotion == null
          ? null
          : OrderPromotionSnapshotDto.fromDomain(domain.promotion!),
      lineItems: domain.lineItems.map(OrderLineItemDto.fromDomain).toList(),
      shippingAddress: domain.shippingAddress == null
          ? null
          : OrderAddressDto.fromDomain(domain.shippingAddress!),
      billingAddress: domain.billingAddress == null
          ? null
          : OrderAddressDto.fromDomain(domain.billingAddress!),
      contact: domain.contact == null
          ? null
          : OrderContactDto.fromDomain(domain.contact),
      fulfillment: domain.fulfillment == null
          ? null
          : OrderFulfillmentInfoDto.fromDomain(domain.fulfillment),
      production: domain.production == null
          ? null
          : OrderProductionInfoDto.fromDomain(domain.production),
      notes: domain.notes == null
          ? null
          : Map<String, dynamic>.from(domain.notes!),
      flags: domain.flags == null
          ? null
          : OrderFlagsDto.fromDomain(domain.flags),
      audit: domain.audit == null
          ? null
          : OrderAuditInfoDto.fromDomain(domain.audit),
      createdAt: domain.createdAt.toIso8601String(),
      updatedAt: domain.updatedAt.toIso8601String(),
      placedAt: domain.placedAt?.toIso8601String(),
      paidAt: domain.paidAt?.toIso8601String(),
      shippedAt: domain.shippedAt?.toIso8601String(),
      deliveredAt: domain.deliveredAt?.toIso8601String(),
      canceledAt: domain.canceledAt?.toIso8601String(),
      cancelReason: domain.cancelReason,
      metadata: domain.metadata == null
          ? null
          : Map<String, dynamic>.from(domain.metadata!),
    );
  }

  final String id;
  final String orderNumber;
  final String userRef;
  final String? cartRef;
  final String status;
  final String currency;
  final OrderTotalsDto totals;
  final OrderPromotionSnapshotDto? promotion;
  final List<OrderLineItemDto> lineItems;
  final OrderAddressDto? shippingAddress;
  final OrderAddressDto? billingAddress;
  final OrderContactDto? contact;
  final OrderFulfillmentInfoDto? fulfillment;
  final OrderProductionInfoDto? production;
  final Map<String, dynamic>? notes;
  final OrderFlagsDto? flags;
  final OrderAuditInfoDto? audit;
  final String createdAt;
  final String updatedAt;
  final String? placedAt;
  final String? paidAt;
  final String? shippedAt;
  final String? deliveredAt;
  final String? canceledAt;
  final String? cancelReason;
  final Map<String, dynamic>? metadata;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'id': id,
      'orderNumber': orderNumber,
      'userRef': userRef,
      'cartRef': cartRef,
      'status': status,
      'currency': currency,
      'totals': totals.toJson(),
      'promotion': promotion?.toJson(),
      'lineItems': lineItems.map((OrderLineItemDto e) => e.toJson()).toList(),
      'shippingAddress': shippingAddress?.toJson(),
      'billingAddress': billingAddress?.toJson(),
      'contact': contact?.toJson(),
      'fulfillment': fulfillment?.toJson(),
      'production': production?.toJson(),
      'notes': notes,
      'flags': flags?.toJson(),
      'audit': audit?.toJson(),
      'createdAt': createdAt,
      'updatedAt': updatedAt,
      'placedAt': placedAt,
      'paidAt': paidAt,
      'shippedAt': shippedAt,
      'deliveredAt': deliveredAt,
      'canceledAt': canceledAt,
      'cancelReason': cancelReason,
      'metadata': metadata,
    };
  }

  Order toDomain() {
    return Order(
      id: id,
      orderNumber: orderNumber,
      userRef: userRef,
      cartRef: cartRef,
      status: _parseOrderStatus(status),
      currency: currency,
      totals: totals.toDomain(),
      promotion: promotion?.toDomain(),
      lineItems: lineItems.map((OrderLineItemDto e) => e.toDomain()).toList(),
      shippingAddress: shippingAddress?.toDomain(),
      billingAddress: billingAddress?.toDomain(),
      contact: contact?.toDomain(),
      fulfillment: fulfillment?.toDomain(),
      production: production?.toDomain(),
      notes: notes == null ? null : Map<String, dynamic>.from(notes!),
      flags: flags?.toDomain(),
      audit: audit?.toDomain(),
      createdAt: DateTime.parse(createdAt),
      updatedAt: DateTime.parse(updatedAt),
      placedAt: placedAt == null ? null : DateTime.parse(placedAt!),
      paidAt: paidAt == null ? null : DateTime.parse(paidAt!),
      shippedAt: shippedAt == null ? null : DateTime.parse(shippedAt!),
      deliveredAt: deliveredAt == null ? null : DateTime.parse(deliveredAt!),
      canceledAt: canceledAt == null ? null : DateTime.parse(canceledAt!),
      cancelReason: cancelReason,
      metadata: metadata == null ? null : Map<String, dynamic>.from(metadata!),
    );
  }
}
