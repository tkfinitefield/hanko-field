import 'package:app/core/domain/entities/content.dart';

GuideCategory _parseGuideCategory(String value) {
  switch (value) {
    case 'culture':
      return GuideCategory.culture;
    case 'howto':
      return GuideCategory.howto;
    case 'policy':
      return GuideCategory.policy;
    case 'faq':
      return GuideCategory.faq;
    case 'news':
      return GuideCategory.news;
    case 'other':
      return GuideCategory.other;
  }
  throw ArgumentError.value(value, 'value', 'Unknown GuideCategory');
}

String _guideCategoryToJson(GuideCategory category) {
  switch (category) {
    case GuideCategory.culture:
      return 'culture';
    case GuideCategory.howto:
      return 'howto';
    case GuideCategory.policy:
      return 'policy';
    case GuideCategory.faq:
      return 'faq';
    case GuideCategory.news:
      return 'news';
    case GuideCategory.other:
      return 'other';
  }
}

ContentPageType _parsePageType(String value) {
  switch (value) {
    case 'landing':
      return ContentPageType.landing;
    case 'legal':
      return ContentPageType.legal;
    case 'help':
      return ContentPageType.help;
    case 'faq':
      return ContentPageType.faq;
    case 'pricing':
      return ContentPageType.pricing;
    case 'system':
      return ContentPageType.system;
    case 'other':
      return ContentPageType.other;
  }
  throw ArgumentError.value(value, 'value', 'Unknown ContentPageType');
}

String _pageTypeToJson(ContentPageType type) {
  switch (type) {
    case ContentPageType.landing:
      return 'landing';
    case ContentPageType.legal:
      return 'legal';
    case ContentPageType.help:
      return 'help';
    case ContentPageType.faq:
      return 'faq';
    case ContentPageType.pricing:
      return 'pricing';
    case ContentPageType.system:
      return 'system';
    case ContentPageType.other:
      return 'other';
  }
}

class SeoMetadataDto {
  SeoMetadataDto({this.metaTitle, this.metaDescription, this.ogImage});

  factory SeoMetadataDto.fromJson(Map<String, dynamic> json) {
    return SeoMetadataDto(
      metaTitle: json['metaTitle'] as String?,
      metaDescription: json['metaDescription'] as String?,
      ogImage: json['ogImage'] as String?,
    );
  }

  factory SeoMetadataDto.fromDomain(SeoMetadata? domain) {
    if (domain == null) {
      return SeoMetadataDto();
    }
    return SeoMetadataDto(
      metaTitle: domain.metaTitle,
      metaDescription: domain.metaDescription,
      ogImage: domain.ogImage,
    );
  }

  final String? metaTitle;
  final String? metaDescription;
  final String? ogImage;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'metaTitle': metaTitle,
      'metaDescription': metaDescription,
      'ogImage': ogImage,
    };
  }

  SeoMetadata toDomain() {
    return SeoMetadata(
      metaTitle: metaTitle,
      metaDescription: metaDescription,
      ogImage: ogImage,
    );
  }
}

class GuideTranslationDto {
  GuideTranslationDto({
    required this.locale,
    required this.title,
    required this.body,
    this.summary,
    this.seo,
  });

  factory GuideTranslationDto.fromJson(Map<String, dynamic> json) {
    return GuideTranslationDto(
      locale: json['locale'] as String,
      title: json['title'] as String,
      body: json['body'] as String,
      summary: json['summary'] as String?,
      seo: json['seo'] == null
          ? null
          : SeoMetadataDto.fromJson(json['seo'] as Map<String, dynamic>),
    );
  }

  factory GuideTranslationDto.fromDomain(GuideTranslation domain) {
    return GuideTranslationDto(
      locale: domain.locale,
      title: domain.title,
      body: domain.body,
      summary: domain.summary,
      seo: domain.seo == null ? null : SeoMetadataDto.fromDomain(domain.seo),
    );
  }

  final String locale;
  final String title;
  final String body;
  final String? summary;
  final SeoMetadataDto? seo;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'locale': locale,
      'title': title,
      'summary': summary,
      'body': body,
      'seo': seo?.toJson(),
    };
  }

  GuideTranslation toDomain() {
    return GuideTranslation(
      locale: locale,
      title: title,
      body: body,
      summary: summary,
      seo: seo?.toDomain(),
    );
  }
}

class GuideAuthorDto {
  GuideAuthorDto({this.name, this.profileUrl});

  factory GuideAuthorDto.fromJson(Map<String, dynamic> json) {
    return GuideAuthorDto(
      name: json['name'] as String?,
      profileUrl: json['profileUrl'] as String?,
    );
  }

  factory GuideAuthorDto.fromDomain(GuideAuthor? domain) {
    if (domain == null) {
      return GuideAuthorDto();
    }
    return GuideAuthorDto(name: domain.name, profileUrl: domain.profileUrl);
  }

  final String? name;
  final String? profileUrl;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{'name': name, 'profileUrl': profileUrl};
  }

  GuideAuthor toDomain() {
    return GuideAuthor(name: name, profileUrl: profileUrl);
  }
}

class GuideArticleDto {
  GuideArticleDto({
    required this.id,
    required this.slug,
    required this.category,
    required this.isPublic,
    required this.createdAt,
    required this.updatedAt,
    this.tags,
    this.heroImageUrl,
    this.readingTimeMinutes,
    this.author,
    this.sources,
    this.translations,
    this.publishAt,
    this.version,
    this.isDeprecated = false,
  });

  factory GuideArticleDto.fromJson(Map<String, dynamic> json) {
    return GuideArticleDto(
      id: json['id'] as String,
      slug: json['slug'] as String,
      category: json['category'] as String,
      tags: (json['tags'] as List<dynamic>?)?.cast<String>(),
      heroImageUrl: json['heroImageUrl'] as String?,
      readingTimeMinutes: json['readingTimeMinutes'] as int?,
      author: json['author'] == null
          ? null
          : GuideAuthorDto.fromJson(json['author'] as Map<String, dynamic>),
      sources: (json['sources'] as List<dynamic>?)?.cast<String>(),
      translations: json['translations'] == null
          ? null
          : (json['translations'] as Map<String, dynamic>).entries
                .map(
                  (MapEntry<String, dynamic> entry) =>
                      GuideTranslationDto.fromJson(<String, dynamic>{
                        'locale': entry.key,
                        ...entry.value as Map<String, dynamic>,
                      }),
                )
                .toList(),
      isPublic: json['isPublic'] as bool? ?? true,
      publishAt: json['publishAt'] as String?,
      version: json['version'] as String?,
      isDeprecated: json['isDeprecated'] as bool? ?? false,
      createdAt: json['createdAt'] as String,
      updatedAt: json['updatedAt'] as String,
    );
  }

  factory GuideArticleDto.fromDomain(GuideArticle domain) {
    return GuideArticleDto(
      id: domain.id,
      slug: domain.slug,
      category: _guideCategoryToJson(domain.category),
      tags: domain.tags,
      heroImageUrl: domain.heroImageUrl,
      readingTimeMinutes: domain.readingTimeMinutes,
      author: domain.author == null
          ? null
          : GuideAuthorDto.fromDomain(domain.author),
      sources: domain.sources,
      translations: domain.translations
          .map(GuideTranslationDto.fromDomain)
          .toList(),
      isPublic: domain.isPublic,
      publishAt: domain.publishAt?.toIso8601String(),
      version: domain.version,
      isDeprecated: domain.isDeprecated,
      createdAt: domain.createdAt.toIso8601String(),
      updatedAt: domain.updatedAt.toIso8601String(),
    );
  }

  final String id;
  final String slug;
  final String category;
  final List<String>? tags;
  final String? heroImageUrl;
  final int? readingTimeMinutes;
  final GuideAuthorDto? author;
  final List<String>? sources;
  final List<GuideTranslationDto>? translations;
  final bool isPublic;
  final String? publishAt;
  final String? version;
  final bool isDeprecated;
  final String createdAt;
  final String updatedAt;

  Map<String, dynamic> toJson() {
    final Map<String, dynamic>? translationsJson = translations == null
        ? null
        : <String, dynamic>{
            for (final GuideTranslationDto translation in translations!)
              translation.locale: <String, dynamic>{
                'title': translation.title,
                'summary': translation.summary,
                'body': translation.body,
                'seo': translation.seo?.toJson(),
              },
          };
    return <String, dynamic>{
      'id': id,
      'slug': slug,
      'category': category,
      'tags': tags,
      'heroImageUrl': heroImageUrl,
      'readingTimeMinutes': readingTimeMinutes,
      'author': author?.toJson(),
      'sources': sources,
      'translations': translationsJson,
      'isPublic': isPublic,
      'publishAt': publishAt,
      'version': version,
      'isDeprecated': isDeprecated,
      'createdAt': createdAt,
      'updatedAt': updatedAt,
    };
  }

  GuideArticle toDomain() {
    return GuideArticle(
      id: id,
      slug: slug,
      category: _parseGuideCategory(category),
      tags: tags ?? const [],
      heroImageUrl: heroImageUrl,
      readingTimeMinutes: readingTimeMinutes,
      author: author?.toDomain(),
      sources: sources ?? const [],
      translations:
          translations?.map((GuideTranslationDto e) => e.toDomain()).toList() ??
          const [],
      isPublic: isPublic,
      publishAt: publishAt == null ? null : DateTime.parse(publishAt!),
      version: version,
      isDeprecated: isDeprecated,
      createdAt: DateTime.parse(createdAt),
      updatedAt: DateTime.parse(updatedAt),
    );
  }
}

class ContentBlockDto {
  ContentBlockDto({required this.type, required this.data});

  factory ContentBlockDto.fromJson(Map<String, dynamic> json) {
    return ContentBlockDto(
      type: json['type'] as String,
      data: Map<String, dynamic>.from(json['data'] as Map),
    );
  }

  factory ContentBlockDto.fromDomain(ContentBlock domain) {
    return ContentBlockDto(
      type: domain.type,
      data: Map<String, dynamic>.from(domain.data),
    );
  }

  final String type;
  final Map<String, dynamic> data;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{'type': type, 'data': data};
  }

  ContentBlock toDomain() {
    return ContentBlock(type: type, data: Map<String, dynamic>.from(data));
  }
}

class ContentPageTranslationDto {
  ContentPageTranslationDto({
    required this.locale,
    required this.title,
    this.body,
    this.blocks,
    this.seo,
  });

  factory ContentPageTranslationDto.fromJson(Map<String, dynamic> json) {
    return ContentPageTranslationDto(
      locale: json['locale'] as String,
      title: json['title'] as String,
      body: json['body'] as String?,
      blocks: (json['blocks'] as List<dynamic>?)
          ?.map(
            (dynamic e) => ContentBlockDto.fromJson(e as Map<String, dynamic>),
          )
          .toList(),
      seo: json['seo'] == null
          ? null
          : SeoMetadataDto.fromJson(json['seo'] as Map<String, dynamic>),
    );
  }

  factory ContentPageTranslationDto.fromDomain(ContentPageTranslation domain) {
    return ContentPageTranslationDto(
      locale: domain.locale,
      title: domain.title,
      body: domain.body,
      blocks: domain.blocks.map(ContentBlockDto.fromDomain).toList(),
      seo: domain.seo == null ? null : SeoMetadataDto.fromDomain(domain.seo),
    );
  }

  final String locale;
  final String title;
  final String? body;
  final List<ContentBlockDto>? blocks;
  final SeoMetadataDto? seo;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'locale': locale,
      'title': title,
      'body': body,
      'blocks': blocks?.map((ContentBlockDto e) => e.toJson()).toList(),
      'seo': seo?.toJson(),
    };
  }

  ContentPageTranslation toDomain() {
    return ContentPageTranslation(
      locale: locale,
      title: title,
      body: body,
      blocks:
          blocks?.map((ContentBlockDto e) => e.toDomain()).toList() ?? const [],
      seo: seo?.toDomain(),
    );
  }
}

class ContentPageDto {
  ContentPageDto({
    required this.id,
    required this.slug,
    required this.type,
    required this.isPublic,
    required this.createdAt,
    required this.updatedAt,
    this.tags,
    this.translations,
    this.navOrder,
    this.publishAt,
    this.version,
    this.isDeprecated = false,
  });

  factory ContentPageDto.fromJson(Map<String, dynamic> json) {
    return ContentPageDto(
      id: json['id'] as String,
      slug: json['slug'] as String,
      type: json['type'] as String,
      tags: (json['tags'] as List<dynamic>?)?.cast<String>(),
      translations: json['translations'] == null
          ? null
          : (json['translations'] as Map<String, dynamic>).entries
                .map(
                  (MapEntry<String, dynamic> entry) =>
                      ContentPageTranslationDto.fromJson(<String, dynamic>{
                        'locale': entry.key,
                        ...entry.value as Map<String, dynamic>,
                      }),
                )
                .toList(),
      navOrder: json['navOrder'] as int?,
      isPublic: json['isPublic'] as bool? ?? true,
      publishAt: json['publishAt'] as String?,
      version: json['version'] as String?,
      isDeprecated: json['isDeprecated'] as bool? ?? false,
      createdAt: json['createdAt'] as String,
      updatedAt: json['updatedAt'] as String,
    );
  }

  factory ContentPageDto.fromDomain(ContentPage domain) {
    return ContentPageDto(
      id: domain.id,
      slug: domain.slug,
      type: _pageTypeToJson(domain.type),
      tags: domain.tags,
      translations: domain.translations
          .map(ContentPageTranslationDto.fromDomain)
          .toList(),
      navOrder: domain.navOrder,
      isPublic: domain.isPublic,
      publishAt: domain.publishAt?.toIso8601String(),
      version: domain.version,
      isDeprecated: domain.isDeprecated,
      createdAt: domain.createdAt.toIso8601String(),
      updatedAt: domain.updatedAt.toIso8601String(),
    );
  }

  final String id;
  final String slug;
  final String type;
  final List<String>? tags;
  final List<ContentPageTranslationDto>? translations;
  final int? navOrder;
  final bool isPublic;
  final String? publishAt;
  final String? version;
  final bool isDeprecated;
  final String createdAt;
  final String updatedAt;

  Map<String, dynamic> toJson() {
    final Map<String, dynamic>? translationsJson = translations == null
        ? null
        : <String, dynamic>{
            for (final ContentPageTranslationDto translation in translations!)
              translation.locale: <String, dynamic>{
                'title': translation.title,
                'body': translation.body,
                'blocks': translation.blocks
                    ?.map((ContentBlockDto e) => e.toJson())
                    .toList(),
                'seo': translation.seo?.toJson(),
              },
          };
    return <String, dynamic>{
      'id': id,
      'slug': slug,
      'type': type,
      'tags': tags,
      'translations': translationsJson,
      'navOrder': navOrder,
      'isPublic': isPublic,
      'publishAt': publishAt,
      'version': version,
      'isDeprecated': isDeprecated,
      'createdAt': createdAt,
      'updatedAt': updatedAt,
    };
  }

  ContentPage toDomain() {
    return ContentPage(
      id: id,
      slug: slug,
      type: _parsePageType(type),
      tags: tags ?? const [],
      translations:
          translations
              ?.map((ContentPageTranslationDto e) => e.toDomain())
              .toList() ??
          const [],
      navOrder: navOrder,
      isPublic: isPublic,
      publishAt: publishAt == null ? null : DateTime.parse(publishAt!),
      version: version,
      isDeprecated: isDeprecated,
      createdAt: DateTime.parse(createdAt),
      updatedAt: DateTime.parse(updatedAt),
    );
  }
}
