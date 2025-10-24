import 'package:flutter/foundation.dart';

enum GuideCategory { culture, howto, policy, faq, news, other }

@immutable
class GuideTranslation {
  const GuideTranslation({
    required this.locale,
    required this.title,
    required this.body,
    this.summary,
    this.seo,
  });

  final String locale;
  final String title;
  final String body;
  final String? summary;
  final SeoMetadata? seo;

  GuideTranslation copyWith({
    String? locale,
    String? title,
    String? body,
    String? summary,
    SeoMetadata? seo,
  }) {
    return GuideTranslation(
      locale: locale ?? this.locale,
      title: title ?? this.title,
      body: body ?? this.body,
      summary: summary ?? this.summary,
      seo: seo ?? this.seo,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is GuideTranslation &&
            other.locale == locale &&
            other.title == title &&
            other.body == body &&
            other.summary == summary &&
            other.seo == seo);
  }

  @override
  int get hashCode => Object.hash(locale, title, body, summary, seo);
}

@immutable
class SeoMetadata {
  const SeoMetadata({this.metaTitle, this.metaDescription, this.ogImage});

  final String? metaTitle;
  final String? metaDescription;
  final String? ogImage;

  SeoMetadata copyWith({
    String? metaTitle,
    String? metaDescription,
    String? ogImage,
  }) {
    return SeoMetadata(
      metaTitle: metaTitle ?? this.metaTitle,
      metaDescription: metaDescription ?? this.metaDescription,
      ogImage: ogImage ?? this.ogImage,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is SeoMetadata &&
            other.metaTitle == metaTitle &&
            other.metaDescription == metaDescription &&
            other.ogImage == ogImage);
  }

  @override
  int get hashCode => Object.hash(metaTitle, metaDescription, ogImage);
}

@immutable
class GuideAuthor {
  const GuideAuthor({this.name, this.profileUrl});

  final String? name;
  final String? profileUrl;

  GuideAuthor copyWith({String? name, String? profileUrl}) {
    return GuideAuthor(
      name: name ?? this.name,
      profileUrl: profileUrl ?? this.profileUrl,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is GuideAuthor &&
            other.name == name &&
            other.profileUrl == profileUrl);
  }

  @override
  int get hashCode => Object.hash(name, profileUrl);
}

@immutable
class GuideArticle {
  const GuideArticle({
    required this.id,
    required this.slug,
    required this.category,
    required this.isPublic,
    required this.createdAt,
    required this.updatedAt,
    this.tags = const [],
    this.heroImageUrl,
    this.readingTimeMinutes,
    this.author,
    this.sources = const [],
    this.translations = const [],
    this.publishAt,
    this.version,
    this.isDeprecated = false,
  });

  final String id;
  final String slug;
  final GuideCategory category;
  final List<String> tags;
  final String? heroImageUrl;
  final int? readingTimeMinutes;
  final GuideAuthor? author;
  final List<String> sources;
  final List<GuideTranslation> translations;
  final bool isPublic;
  final DateTime? publishAt;
  final String? version;
  final bool isDeprecated;
  final DateTime createdAt;
  final DateTime updatedAt;

  GuideArticle copyWith({
    String? id,
    String? slug,
    GuideCategory? category,
    List<String>? tags,
    String? heroImageUrl,
    int? readingTimeMinutes,
    GuideAuthor? author,
    List<String>? sources,
    List<GuideTranslation>? translations,
    bool? isPublic,
    DateTime? publishAt,
    String? version,
    bool? isDeprecated,
    DateTime? createdAt,
    DateTime? updatedAt,
  }) {
    return GuideArticle(
      id: id ?? this.id,
      slug: slug ?? this.slug,
      category: category ?? this.category,
      tags: tags ?? this.tags,
      heroImageUrl: heroImageUrl ?? this.heroImageUrl,
      readingTimeMinutes: readingTimeMinutes ?? this.readingTimeMinutes,
      author: author ?? this.author,
      sources: sources ?? this.sources,
      translations: translations ?? this.translations,
      isPublic: isPublic ?? this.isPublic,
      publishAt: publishAt ?? this.publishAt,
      version: version ?? this.version,
      isDeprecated: isDeprecated ?? this.isDeprecated,
      createdAt: createdAt ?? this.createdAt,
      updatedAt: updatedAt ?? this.updatedAt,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is GuideArticle &&
            other.id == id &&
            other.slug == slug &&
            other.category == category &&
            listEquals(other.tags, tags) &&
            other.heroImageUrl == heroImageUrl &&
            other.readingTimeMinutes == readingTimeMinutes &&
            other.author == author &&
            listEquals(other.sources, sources) &&
            listEquals(other.translations, translations) &&
            other.isPublic == isPublic &&
            other.publishAt == publishAt &&
            other.version == version &&
            other.isDeprecated == isDeprecated &&
            other.createdAt == createdAt &&
            other.updatedAt == updatedAt);
  }

  @override
  int get hashCode {
    return Object.hashAll([
      id,
      slug,
      category,
      Object.hashAll(tags),
      heroImageUrl,
      readingTimeMinutes,
      author,
      Object.hashAll(sources),
      Object.hashAll(translations),
      isPublic,
      publishAt,
      version,
      isDeprecated,
      createdAt,
      updatedAt,
    ]);
  }
}

enum ContentPageType { landing, legal, help, faq, pricing, system, other }

@immutable
class ContentBlock {
  const ContentBlock({required this.type, required this.data});

  final String type;
  final Map<String, dynamic> data;

  ContentBlock copyWith({String? type, Map<String, dynamic>? data}) {
    return ContentBlock(type: type ?? this.type, data: data ?? this.data);
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is ContentBlock &&
            other.type == type &&
            mapEquals(other.data, data));
  }

  @override
  int get hashCode => Object.hash(type, Object.hashAll(data.entries));
}

@immutable
class ContentPageTranslation {
  const ContentPageTranslation({
    required this.locale,
    required this.title,
    this.body,
    this.blocks = const [],
    this.seo,
  });

  final String locale;
  final String title;
  final String? body;
  final List<ContentBlock> blocks;
  final SeoMetadata? seo;

  ContentPageTranslation copyWith({
    String? locale,
    String? title,
    String? body,
    List<ContentBlock>? blocks,
    SeoMetadata? seo,
  }) {
    return ContentPageTranslation(
      locale: locale ?? this.locale,
      title: title ?? this.title,
      body: body ?? this.body,
      blocks: blocks ?? this.blocks,
      seo: seo ?? this.seo,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is ContentPageTranslation &&
            other.locale == locale &&
            other.title == title &&
            other.body == body &&
            listEquals(other.blocks, blocks) &&
            other.seo == seo);
  }

  @override
  int get hashCode =>
      Object.hash(locale, title, body, Object.hashAll(blocks), seo);
}

@immutable
class ContentPage {
  const ContentPage({
    required this.id,
    required this.slug,
    required this.type,
    required this.isPublic,
    required this.createdAt,
    required this.updatedAt,
    this.tags = const [],
    this.translations = const [],
    this.navOrder,
    this.publishAt,
    this.version,
    this.isDeprecated = false,
  });

  final String id;
  final String slug;
  final ContentPageType type;
  final List<String> tags;
  final List<ContentPageTranslation> translations;
  final int? navOrder;
  final bool isPublic;
  final DateTime? publishAt;
  final String? version;
  final bool isDeprecated;
  final DateTime createdAt;
  final DateTime updatedAt;

  ContentPage copyWith({
    String? id,
    String? slug,
    ContentPageType? type,
    List<String>? tags,
    List<ContentPageTranslation>? translations,
    int? navOrder,
    bool? isPublic,
    DateTime? publishAt,
    String? version,
    bool? isDeprecated,
    DateTime? createdAt,
    DateTime? updatedAt,
  }) {
    return ContentPage(
      id: id ?? this.id,
      slug: slug ?? this.slug,
      type: type ?? this.type,
      tags: tags ?? this.tags,
      translations: translations ?? this.translations,
      navOrder: navOrder ?? this.navOrder,
      isPublic: isPublic ?? this.isPublic,
      publishAt: publishAt ?? this.publishAt,
      version: version ?? this.version,
      isDeprecated: isDeprecated ?? this.isDeprecated,
      createdAt: createdAt ?? this.createdAt,
      updatedAt: updatedAt ?? this.updatedAt,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other is ContentPage &&
            other.id == id &&
            other.slug == slug &&
            other.type == type &&
            listEquals(other.tags, tags) &&
            listEquals(other.translations, translations) &&
            other.navOrder == navOrder &&
            other.isPublic == isPublic &&
            other.publishAt == publishAt &&
            other.version == version &&
            other.isDeprecated == isDeprecated &&
            other.createdAt == createdAt &&
            other.updatedAt == updatedAt);
  }

  @override
  int get hashCode {
    return Object.hashAll([
      id,
      slug,
      type,
      Object.hashAll(tags),
      Object.hashAll(translations),
      navOrder,
      isPublic,
      publishAt,
      version,
      isDeprecated,
      createdAt,
      updatedAt,
    ]);
  }
}
