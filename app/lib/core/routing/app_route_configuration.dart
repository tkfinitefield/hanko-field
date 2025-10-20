import 'package:flutter/material.dart';

import 'app_tab.dart';

/// どのタブでも表示できるルート
sealed class IndependentRoute {
  Object stackKey(AppTab tab, int index);
  Widget get page;
}

/// 固定の親子関係を持つルート
sealed class AppRoute {
  AppRoute? get parent;
  Object get key;
}

/// ホームルート
class HomeRoute implements AppRoute {
  const HomeRoute();

  @override
  AppRoute? get parent => null;

  @override
  Object get key => getRouteKey();

  static Object getRouteKey() => 'HomeRoute';

  @override
  String toString() {
    return 'HomeRoute';
  }
}
