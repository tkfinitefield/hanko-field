import 'package:app/core/data/dtos/catalog_dto.dart';
import 'package:app/core/domain/entities/catalog.dart';
import 'package:app/core/domain/entities/design.dart';

mixin CatalogDtoMapper {
  CatalogMaterial mapMaterial(CatalogMaterialDto dto) => dto.toDomain();
  CatalogMaterialDto mapMaterialToDto(CatalogMaterial domain) =>
      CatalogMaterialDto.fromDomain(domain);

  CatalogProduct mapProduct(CatalogProductDto dto) => dto.toDomain();
  CatalogProductDto mapProductToDto(CatalogProduct domain) =>
      CatalogProductDto.fromDomain(domain);

  CatalogFont mapFont(CatalogFontDto dto) => dto.toDomain();
  CatalogFontDto mapFontToDto(CatalogFont domain) =>
      CatalogFontDto.fromDomain(domain);

  CatalogTemplate mapTemplate(CatalogTemplateDto dto) => dto.toDomain();
  CatalogTemplateDto mapTemplateToDto(CatalogTemplate domain) =>
      CatalogTemplateDto.fromDomain(domain);
}

abstract class CatalogRepository with CatalogDtoMapper {
  Future<List<CatalogMaterial>> fetchMaterials({String? cursor});
  Future<List<CatalogProduct>> fetchProducts({
    String? cursor,
    Map<String, dynamic>? filters,
  });
  Future<List<CatalogFont>> fetchFonts({
    String? cursor,
    DesignWritingStyle? writing,
  });
  Future<List<CatalogTemplate>> fetchTemplates({
    String? cursor,
    DesignShape? shape,
  });

  Future<CatalogMaterial> fetchMaterial(String materialId);
  Future<CatalogProduct> fetchProduct(String productId);
  Future<CatalogFont> fetchFont(String fontId);
  Future<CatalogTemplate> fetchTemplate(String templateId);
}
