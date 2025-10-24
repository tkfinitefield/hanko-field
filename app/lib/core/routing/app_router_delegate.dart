import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'app_route_configuration.dart';
import 'app_state_notifier.dart';
import 'app_tab.dart';

/// アプリのルーティングデリゲート
class AppRouterDelegate extends RouterDelegate<AppRoute>
    with ChangeNotifier, PopNavigatorRouterDelegateMixin<AppRoute> {
  AppRouterDelegate(this.ref) {
    ref.listenManual(appStateProvider, (previous, next) {
      notifyListeners();
    });
  }

  @override
  final GlobalKey<NavigatorState> navigatorKey = GlobalKey();

  final WidgetRef ref;

  @override
  AppRoute? get currentConfiguration => ref.read(appStateProvider).currentRoute;

  @override
  Widget build(BuildContext context) {
    return Navigator(
      key: navigatorKey,
      pages: const [
        MaterialPage(key: ValueKey('RootPage'), child: SizedBox.shrink()),
      ],
      onDidRemovePage: (page) {},
    );
  }

  @override
  Future<void> setNewRoutePath(AppRoute configuration) async {
    final notifier = ref.read(appStateProvider.notifier);
    notifier.setRouteAndTab(configuration, AppTab.home);
  }
}
