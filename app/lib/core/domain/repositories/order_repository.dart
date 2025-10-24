import 'package:app/core/data/dtos/order_dto.dart';
import 'package:app/core/domain/entities/order.dart';

mixin OrderDtoMapper {
  Order mapOrder(OrderDto dto) => dto.toDomain();
  OrderDto mapOrderToDto(Order domain) => OrderDto.fromDomain(domain);

  OrderShipment mapShipment(OrderShipmentDto dto) => dto.toDomain();
  OrderShipmentDto mapShipmentToDto(OrderShipment domain) =>
      OrderShipmentDto.fromDomain(domain);

  OrderPayment mapPayment(OrderPaymentDto dto) => dto.toDomain();
  OrderPaymentDto mapPaymentToDto(OrderPayment domain) =>
      OrderPaymentDto.fromDomain(domain);

  ProductionEvent mapProductionEvent(ProductionEventDto dto) => dto.toDomain();
  ProductionEventDto mapProductionEventToDto(ProductionEvent domain) =>
      ProductionEventDto.fromDomain(domain);
}

abstract class OrderRepository with OrderDtoMapper {
  Future<List<Order>> fetchOrders({
    int? pageSize,
    String? pageToken,
    Map<String, dynamic>? filters,
  });
  Future<Order> fetchOrder(String orderId);

  Future<List<OrderPayment>> fetchPayments(String orderId);
  Future<List<OrderShipment>> fetchShipments(String orderId);
  Future<List<ProductionEvent>> fetchProductionEvents(String orderId);

  Future<Order> cancelOrder(String orderId, {String? reason});
  Future<Order> requestInvoice(String orderId);
  Future<Order> reorder(String orderId);
}
