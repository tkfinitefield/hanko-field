import 'package:app/core/routing/app_route_configuration.dart';
import 'package:app/core/routing/app_tab.dart';
import 'package:flutter/material.dart';

/// RouteInformationParser to handle deep links.
class AppRouteInformationParser extends RouteInformationParser<AppRoute> {
  const AppRouteInformationParser();

  @override
  Future<AppRoute> parseRouteInformation(
    RouteInformation routeInformation,
  ) async {
    final uri = routeInformation.uri;
    if (uri.pathSegments.isEmpty) {
      return const TabRoute(currentTab: kDefaultAppTab);
    }
    final firstSegment = uri.pathSegments.first;
    final remaining = uri.pathSegments.skip(1).toList();
    final tab = _tabFromSegment(firstSegment);
    switch (tab) {
      case AppTab.creation:
        return TabRoute(
          currentTab: AppTab.creation,
          stack: remaining.isEmpty ? const [] : [CreationStageRoute(remaining)],
        );
      case AppTab.shop:
        return TabRoute(
          currentTab: AppTab.shop,
          stack: _buildShopStack(remaining),
        );
      case AppTab.orders:
        return TabRoute(
          currentTab: AppTab.orders,
          stack: _buildOrderStack(remaining),
        );
      case AppTab.library:
        return TabRoute(
          currentTab: AppTab.library,
          stack: _buildLibraryStack(remaining),
        );
      case AppTab.profile:
        return TabRoute(
          currentTab: AppTab.profile,
          stack: remaining.isEmpty
              ? const []
              : [ProfileSectionRoute(remaining)],
        );
    }
  }

  @override
  RouteInformation restoreRouteInformation(AppRoute configuration) {
    return RouteInformation(uri: Uri.parse(configuration.location));
  }

  AppTab _tabFromSegment(String segment) {
    for (final tab in AppTab.values) {
      if (tab.pathSegment == segment) {
        return tab;
      }
    }
    return kDefaultAppTab;
  }

  List<IndependentRoute> _buildShopStack(List<String> segments) {
    if (segments.length < 2) {
      return const [];
    }
    final entity = segments.first;
    final identifier = segments[1];
    final trailingSegments = segments.length > 2
        ? segments.sublist(2)
        : const <String>[];
    return [
      ShopDetailRoute(
        entity: entity,
        identifier: identifier,
        trailingSegments: trailingSegments,
      ),
    ];
  }

  List<IndependentRoute> _buildOrderStack(List<String> segments) {
    if (segments.isEmpty) {
      return const [];
    }
    final orderId = segments.first;
    final trailingSegments = segments.length > 1
        ? segments.sublist(1)
        : const <String>[];
    return [OrderDetailsRoute(orderId: orderId, trailing: trailingSegments)];
  }

  List<IndependentRoute> _buildLibraryStack(List<String> segments) {
    if (segments.isEmpty) {
      return const [];
    }
    final designId = segments.first;
    final trailingSegments = segments.length > 1
        ? segments.sublist(1)
        : const <String>[];
    return [LibraryEntryRoute(designId: designId, trailing: trailingSegments)];
  }
}
