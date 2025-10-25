import 'package:app/core/routing/app_route_configuration.dart';
import 'package:app/core/routing/app_state_notifier.dart';
import 'package:app/core/routing/app_tab.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// ビューモデル/サービスから使うナビゲーションヘルパー
final appNavigationControllerProvider = Provider<AppNavigationController>((
  ref,
) {
  final notifier = ref.read(appStateProvider.notifier);
  return AppNavigationController(notifier);
});

class AppNavigationController {
  AppNavigationController(this._notifier);

  final AppStateNotifier _notifier;

  void switchTab(AppTab tab) => _notifier.selectTab(tab);

  void push(IndependentRoute route) => _notifier.push(route);

  void pop() => _notifier.pop();

  void clearTab(AppTab tab) => _notifier.clearTabStack(tab);

  void handleDeepLink(AppRoute route) => _notifier.applyDeepLink(route);
}
