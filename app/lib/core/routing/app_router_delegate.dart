import 'package:app/core/routing/app_route_configuration.dart';
import 'package:app/core/routing/app_state.dart';
import 'package:app/core/routing/app_state_notifier.dart';
import 'package:app/core/routing/presentation/app_navigation_shell.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// アプリのルーティングデリゲート
final appRouterDelegateProvider = Provider<AppRouterDelegate>(
  AppRouterDelegate.new,
);

class AppRouterDelegate extends RouterDelegate<AppRoute>
    with ChangeNotifier, PopNavigatorRouterDelegateMixin<AppRoute> {
  AppRouterDelegate(this.ref) {
    ref.listen<AppState>(
      appStateProvider,
      (previous, next) => notifyListeners(),
    );
  }

  @override
  final GlobalKey<NavigatorState> navigatorKey = GlobalKey();

  final Ref ref;

  @override
  AppRoute? get currentConfiguration => ref.read(appStateProvider).currentRoute;

  @override
  Widget build(BuildContext context) {
    return Navigator(
      key: navigatorKey,
      pages: const [
        MaterialPage(
          key: ValueKey('AppNavigationShell'),
          child: AppNavigationShell(),
        ),
      ],
    );
  }

  @override
  Future<void> setNewRoutePath(AppRoute configuration) async {
    final notifier = ref.read(appStateProvider.notifier);
    notifier.applyDeepLink(configuration);
  }
}
