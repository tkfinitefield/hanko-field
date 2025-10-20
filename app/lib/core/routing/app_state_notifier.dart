import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'app_route_configuration.dart';
import 'app_stack.dart';
import 'app_state.dart';
import 'app_tab.dart';

/// アプリのルーティング状態を管理するプロバイダー
final appStateProvider = NotifierProvider<AppStateNotifier, AppState>(
  AppStateNotifier.new,
);

/// アプリのルーティング状態を管理する
class AppStateNotifier extends Notifier<AppState> {
  @override
  AppState build() {
    return AppState(
      currentRoute: const HomeRoute(),
      currentTab: AppTab.home,
      stack: AppStack.empty(),
    );
  }

  void setRoute(AppRoute route) {
    Future.microtask(() {
      state = state.copyWith(currentRoute: route);
    });
  }

  void push(IndependentRoute route) {
    Future.microtask(() {
      state = state.push(route);
    });
  }

  void pop() {
    Future.microtask(() {
      if (state.stack.getStack(state.currentTab).isEmpty) {
        state = state.copyWith(currentRoute: state.currentRoute.parent);
      } else {
        state = state.copyWith(stack: state.stack.popStack(state.currentTab));
      }
    });
  }

  void selectTab(AppTab tab) {
    Future.microtask(() {
      state = state.copyWith(currentTab: tab);
    });
  }

  void setRouteAndTab(AppRoute route, AppTab tab) {
    Future.microtask(() {
      state = state.copyWith(currentRoute: route, currentTab: tab);
    });
  }

  void removeFromStack(AppTab tab, List<int> removeStackIndexes) {
    Future.microtask(() {
      state = state.copyWith(
        stack: state.stack.copyWith(tab, [
          for (final (index, route) in state.stack.getStack(tab).indexed)
            if (!removeStackIndexes.contains(index)) route,
        ]),
      );
    });
  }

  @override
  String toString() {
    return 'AppStateNotifier(state: $state)';
  }
}
