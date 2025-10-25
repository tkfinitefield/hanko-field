import 'package:app/core/routing/app_route_configuration.dart';
import 'package:app/core/routing/app_stack.dart';
import 'package:app/core/routing/app_state.dart';
import 'package:app/core/routing/app_tab.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// アプリのルーティング状態を管理するプロバイダー
final appStateProvider = NotifierProvider<AppStateNotifier, AppState>(
  AppStateNotifier.new,
);

/// アプリのルーティング状態を管理する
class AppStateNotifier extends Notifier<AppState> {
  @override
  AppState build() {
    return AppState(
      currentRoute: const TabRoute(currentTab: kDefaultAppTab),
      currentTab: kDefaultAppTab,
      stack: AppStack.empty(),
    );
  }

  void setRoute(AppRoute route) {
    applyDeepLink(route);
  }

  void push(IndependentRoute route) {
    Future.microtask(() {
      state = state.push(route);
    });
  }

  void pop() {
    Future.microtask(() {
      final tab = state.currentTab;
      final stackForTab = state.stack.getStack(tab);
      if (stackForTab.isEmpty) {
        state = state.copyWith(currentRoute: TabRoute(currentTab: tab));
        return;
      }
      final nextStack = state.stack.popStack(tab);
      state = state.copyWith(
        stack: nextStack,
        currentRoute: _routeFor(tab, nextStack),
      );
    });
  }

  void selectTab(AppTab tab) {
    Future.microtask(() {
      state = state.copyWith(
        currentTab: tab,
        currentRoute: _routeFor(tab, state.stack),
      );
    });
  }

  void setRouteAndTab(AppRoute route, AppTab tab) {
    Future.microtask(() {
      final updatedStack = state.stack.copyWith(tab, route.stack);
      state = state.copyWith(
        currentRoute: route,
        currentTab: tab,
        stack: updatedStack,
      );
    });
  }

  void removeFromStack(AppTab tab, List<int> removeStackIndexes) {
    Future.microtask(() {
      final remaining = [
        for (final (index, route) in state.stack.getStack(tab).indexed)
          if (!removeStackIndexes.contains(index)) route,
      ];
      final updatedStack = state.stack.copyWith(tab, remaining);
      state = state.copyWith(
        stack: updatedStack,
        currentRoute: tab == state.currentTab
            ? _routeFor(state.currentTab, updatedStack)
            : state.currentRoute,
      );
    });
  }

  void clearTabStack(AppTab tab) {
    Future.microtask(() {
      if (state.stack.getStack(tab).isEmpty) {
        return;
      }
      final updatedStack = state.stack.copyWith(tab, const []);
      state = state.copyWith(
        stack: updatedStack,
        currentRoute: tab == state.currentTab
            ? _routeFor(state.currentTab, updatedStack)
            : state.currentRoute,
      );
    });
  }

  void applyDeepLink(AppRoute route) {
    Future.microtask(() {
      final updatedStack = state.stack.copyWith(route.tab, route.stack);
      state = state.copyWith(
        currentRoute: route,
        currentTab: route.tab,
        stack: updatedStack,
      );
    });
  }

  TabRoute _routeFor(AppTab tab, AppStack stack) {
    return TabRoute(currentTab: tab, stack: stack.getStack(tab));
  }

  @override
  String toString() {
    return 'AppStateNotifier(state: $state)';
  }
}
