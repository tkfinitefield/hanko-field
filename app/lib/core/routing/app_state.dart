import 'package:app/core/routing/app_route_configuration.dart';
import 'package:app/core/routing/app_stack.dart';
import 'package:app/core/routing/app_tab.dart';

/// アプリのルーティング状態
class AppState {
  AppState({
    required this.currentRoute,
    required this.currentTab,
    required this.stack,
  });

  final AppRoute currentRoute;
  final AppTab currentTab;
  final AppStack stack;

  AppState copyWith({
    AppRoute? currentRoute,
    AppTab? currentTab,
    AppStack? stack,
  }) {
    return AppState(
      currentRoute: currentRoute ?? this.currentRoute,
      currentTab: currentTab ?? this.currentTab,
      stack: stack ?? this.stack,
    );
  }

  AppState push(IndependentRoute route) {
    final updatedStack = stack.push(currentTab, route);
    return copyWith(
      stack: updatedStack,
      currentRoute: TabRoute(
        currentTab: currentTab,
        stack: updatedStack.getStack(currentTab),
      ),
    );
  }

  @override
  String toString() {
    return 'AppState(currentRoute: $currentRoute, tab: $currentTab, stack: $stack)';
  }
}
