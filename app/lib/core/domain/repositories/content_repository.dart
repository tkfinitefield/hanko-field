import 'package:app/core/data/dtos/content_dto.dart';
import 'package:app/core/domain/entities/content.dart';

mixin ContentDtoMapper {
  GuideArticle mapGuide(GuideArticleDto dto) => dto.toDomain();
  GuideArticleDto mapGuideToDto(GuideArticle domain) =>
      GuideArticleDto.fromDomain(domain);

  ContentPage mapPage(ContentPageDto dto) => dto.toDomain();
  ContentPageDto mapPageToDto(ContentPage domain) =>
      ContentPageDto.fromDomain(domain);
}

abstract class ContentRepository with ContentDtoMapper {
  Future<List<GuideArticle>> fetchGuides({
    GuideCategory? category,
    String? locale,
    String? pageToken,
  });
  Future<GuideArticle> fetchGuideBySlug(String slug, {String? locale});

  Future<List<ContentPage>> fetchPages({ContentPageType? type, String? locale});
  Future<ContentPage> fetchPageBySlug(String slug, {String? locale});
}
