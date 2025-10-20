import 'app_route_configuration.dart';
import 'app_tab.dart';

/// アプリのルーティングスタック
class AppStack {
  AppStack({required Map<AppTab, List<IndependentRoute>> stack})
    : _stack = stack;

  AppStack.empty()
    : this(stack: {for (final tab in AppTab.values) tab: <IndependentRoute>[]});

  final Map<AppTab, List<IndependentRoute>> _stack;

  AppStack copyWith(AppTab tab, List<IndependentRoute>? routes) {
    final newStack = <AppTab, List<IndependentRoute>>{};
    for (final t in AppTab.values) {
      newStack[t] = [
        if (t == tab) ...?(routes ?? _stack[tab]) else ...?_stack[t],
      ];
    }
    return AppStack(stack: newStack);
  }

  AppStack push(AppTab t, IndependentRoute route) {
    final newStack = <AppTab, List<IndependentRoute>>{};
    for (final tab in AppTab.values) {
      newStack[tab] = [...?_stack[tab], if (tab == t) route];
    }
    return AppStack(stack: newStack);
  }

  @override
  String toString() {
    return 'AppStack(stack: $_stack)';
  }

  List<IndependentRoute> getStack(AppTab tab) {
    return _stack[tab] ?? [];
  }

  AppStack popStack(AppTab tab) {
    return copyWith(tab, [
      if (_stack[tab] != null && _stack[tab]!.isNotEmpty)
        for (final (index, route) in _stack[tab]!.indexed)
          if (index != _stack[tab]!.length - 1) route,
    ]);
  }
}
