import 'dart:collection';

import 'package:app/core/routing/app_tab.dart';
import 'package:app/features/navigation/presentation/deep_link_pages.dart';
import 'package:flutter/material.dart';

/// タブ横断で積み上げるルート
sealed class IndependentRoute {
  List<String> get pathSegments;
  Widget get page;
  Object stackKey(AppTab tab, int index);
}

/// ルーターが復元するルート情報
sealed class AppRoute {
  AppRoute? get parent;
  Object get key;
  AppTab get tab;
  List<IndependentRoute> get stack;
  String get location;
}

/// 指定タブのルート/スタック
class TabRoute implements AppRoute {
  const TabRoute({
    required this.currentTab,
    List<IndependentRoute> stack = const [],
  }) : _stack = stack;

  final AppTab currentTab;
  final List<IndependentRoute> _stack;

  @override
  AppRoute? get parent => null;

  @override
  Object get key =>
      'TabRoute(${currentTab.name}-${_stack.map((route) => route.hashCode).join('-')})';

  @override
  AppTab get tab => currentTab;

  @override
  List<IndependentRoute> get stack => UnmodifiableListView(_stack);

  @override
  String get location {
    final segments = <String>[currentTab.pathSegment];
    for (final route in _stack) {
      segments.addAll(route.pathSegments);
    }
    return '/${segments.join('/')}';
  }

  TabRoute copyWith({List<IndependentRoute>? stack}) {
    return TabRoute(currentTab: currentTab, stack: stack ?? _stack);
  }

  @override
  String toString() {
    return 'TabRoute(tab: ${currentTab.name}, stack: $_stack)';
  }
}

/// 作成フロー内のステージ
class CreationStageRoute implements IndependentRoute {
  CreationStageRoute(List<String> segments)
    : stageSegments = List.unmodifiable(segments);

  final List<String> stageSegments;

  @override
  Widget get page => CreationStagePage(stageSegments: stageSegments);

  @override
  Object stackKey(AppTab tab, int index) =>
      'creation-${stageSegments.join('-')}-$index';

  @override
  List<String> get pathSegments => stageSegments;

  @override
  String toString() => 'CreationStageRoute(${stageSegments.join('/')})';
}

/// ショップ詳細（素材/商品など）
class ShopDetailRoute implements IndependentRoute {
  ShopDetailRoute({
    required this.entity,
    required this.identifier,
    List<String> trailingSegments = const [],
  }) : trailingSegments = List.unmodifiable(trailingSegments);

  final String entity;
  final String identifier;
  final List<String> trailingSegments;

  @override
  Widget get page => ShopDetailPage(
    entity: entity,
    identifier: identifier,
    subPage: trailingSegments.join('/'),
  );

  @override
  Object stackKey(AppTab tab, int index) =>
      'shop-$entity-$identifier-${trailingSegments.isEmpty ? 'root' : trailingSegments.join('-')}-$index';

  @override
  List<String> get pathSegments => [entity, identifier, ...trailingSegments];

  @override
  String toString() =>
      'ShopDetailRoute(entity: $entity, id: $identifier, sub: $trailingSegments)';
}

/// 注文詳細
class OrderDetailsRoute implements IndependentRoute {
  OrderDetailsRoute({required this.orderId, List<String> trailing = const []})
    : trailingSegments = List.unmodifiable(trailing);

  final String orderId;
  final List<String> trailingSegments;

  @override
  Widget get page =>
      OrderDetailsPage(orderId: orderId, subPage: trailingSegments.join('/'));

  @override
  Object stackKey(AppTab tab, int index) =>
      'orders-$orderId-${trailingSegments.isEmpty ? 'root' : trailingSegments.join('-')}-$index';

  @override
  List<String> get pathSegments => [orderId, ...trailingSegments];

  @override
  String toString() =>
      'OrderDetailsRoute(orderId: $orderId, sub: $trailingSegments)';
}

/// マイ印鑑詳細
class LibraryEntryRoute implements IndependentRoute {
  LibraryEntryRoute({required this.designId, List<String> trailing = const []})
    : trailingSegments = List.unmodifiable(trailing);

  final String designId;
  final List<String> trailingSegments;

  @override
  Widget get page =>
      LibraryEntryPage(designId: designId, subPage: trailingSegments.join('/'));

  @override
  Object stackKey(AppTab tab, int index) =>
      'library-$designId-${trailingSegments.isEmpty ? 'root' : trailingSegments.join('-')}-$index';

  @override
  List<String> get pathSegments => [designId, ...trailingSegments];

  @override
  String toString() =>
      'LibraryEntryRoute(designId: $designId, sub: $trailingSegments)';
}

/// プロフィール配下セクション
class ProfileSectionRoute implements IndependentRoute {
  ProfileSectionRoute(List<String> segments)
    : sectionSegments = List.unmodifiable(segments);

  final List<String> sectionSegments;

  @override
  Widget get page => ProfileSectionPage(sectionSegments: sectionSegments);

  @override
  Object stackKey(AppTab tab, int index) =>
      'profile-${sectionSegments.join('-')}-$index';

  @override
  List<String> get pathSegments => sectionSegments;

  @override
  String toString() => 'ProfileSectionRoute(${sectionSegments.join('/')})';
}
