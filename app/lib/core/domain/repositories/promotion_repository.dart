import 'package:app/core/data/dtos/promotion_dto.dart';
import 'package:app/core/domain/entities/promotion.dart';

mixin PromotionDtoMapper {
  Promotion mapPromotion(PromotionDto dto) => dto.toDomain();
  PromotionDto mapPromotionToDto(Promotion domain) =>
      PromotionDto.fromDomain(domain);
}

abstract class PromotionRepository with PromotionDtoMapper {
  Future<List<Promotion>> fetchPromotions({
    int? pageSize,
    String? pageToken,
    bool includeInactive = false,
  });
  Future<Promotion> fetchPromotion(String promotionId);
  Future<Promotion> validateCode(
    String code, {
    String? currency,
    int? subtotal,
  });
}
