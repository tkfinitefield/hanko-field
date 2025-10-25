import 'package:app/core/routing/app_route_configuration.dart';
import 'package:app/core/routing/app_state.dart';
import 'package:app/core/routing/app_state_notifier.dart';
import 'package:app/core/routing/app_tab.dart';
import 'package:app/features/navigation/presentation/tab_root_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// アプリ全体のボトムタブシェル
class AppNavigationShell extends ConsumerStatefulWidget {
  const AppNavigationShell({super.key});

  @override
  ConsumerState<AppNavigationShell> createState() => _AppNavigationShellState();
}

class _AppNavigationShellState extends ConsumerState<AppNavigationShell> {
  late final Map<AppTab, GlobalKey<NavigatorState>> _navigatorKeys = {
    for (final tab in AppTab.values)
      tab: GlobalKey<NavigatorState>(debugLabel: 'TabNavigator-${tab.name}'),
  };

  @override
  Widget build(BuildContext context) {
    final appState = ref.watch(appStateProvider);
    final canSystemPop = _canSystemPop(appState);
    return PopScope(
      canPop: canSystemPop,
      onPopInvokedWithResult: (didPop, _) {
        if (didPop) {
          return;
        }
        final handled = _handleBackPress(appState);
        if (!handled && canSystemPop) {
          Navigator.of(context).maybePop();
        }
      },
      child: Scaffold(
        body: IndexedStack(
          index: appState.currentTab.index,
          children: [
            for (final tab in AppTab.values)
              _buildTabNavigator(
                tab: tab,
                routes: appState.stack.getStack(tab),
                ref: ref,
              ),
          ],
        ),
        bottomNavigationBar: NavigationBar(
          selectedIndex: appState.currentTab.index,
          onDestinationSelected: (index) {
            final nextTab = AppTab.values[index];
            if (nextTab == appState.currentTab) {
              final stackRoutes = appState.stack.getStack(nextTab);
              if (stackRoutes.isEmpty) return;
              final indexes = List.generate(stackRoutes.length, (i) => i);
              ref
                  .read(appStateProvider.notifier)
                  .removeFromStack(nextTab, indexes);
            } else {
              ref.read(appStateProvider.notifier).selectTab(nextTab);
            }
          },
          destinations: [
            for (final tab in AppTab.values)
              NavigationDestination(icon: Icon(tab.icon), label: tab.label),
          ],
        ),
      ),
    );
  }

  Widget _buildTabNavigator({
    required AppTab tab,
    required List<IndependentRoute> routes,
    required WidgetRef ref,
  }) {
    return Navigator(
      key: _navigatorKeys[tab],
      observers: [_TabNavigatorObserver(tab: tab, ref: ref)],
      pages: [
        MaterialPage(
          key: ValueKey('Root-${tab.name}'),
          child: AppTabRootPage(tab: tab),
        ),
        for (final (index, route) in routes.indexed)
          MaterialPage(
            key: ValueKey(route.stackKey(tab, index)),
            child: route.page,
          ),
      ],
    );
  }

  bool _canSystemPop(AppState appState) {
    final stackRoutes = appState.stack.getStack(appState.currentTab);
    return appState.currentTab == kDefaultAppTab && stackRoutes.isEmpty;
  }

  bool _handleBackPress(AppState appState) {
    final currentTab = appState.currentTab;
    final stackRoutes = appState.stack.getStack(currentTab);
    if (stackRoutes.isNotEmpty) {
      ref.read(appStateProvider.notifier).pop();
      return true;
    }
    if (currentTab != kDefaultAppTab) {
      ref.read(appStateProvider.notifier).selectTab(kDefaultAppTab);
      return true;
    }
    return false;
  }
}

class _TabNavigatorObserver extends NavigatorObserver {
  _TabNavigatorObserver({required this.tab, required this.ref});

  final AppTab tab;
  final WidgetRef ref;

  @override
  void didPop(Route route, Route? previousRoute) {
    final appState = ref.read(appStateProvider);
    if (appState.stack.getStack(tab).isEmpty) {
      super.didPop(route, previousRoute);
      return;
    }
    if (appState.currentTab == tab) {
      ref.read(appStateProvider.notifier).pop();
    } else {
      final stackLength = appState.stack.getStack(tab).length;
      if (stackLength > 0) {
        ref.read(appStateProvider.notifier).removeFromStack(tab, [
          stackLength - 1,
        ]);
      }
    }
    super.didPop(route, previousRoute);
  }
}
