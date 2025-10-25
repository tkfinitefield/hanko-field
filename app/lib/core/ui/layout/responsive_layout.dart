import 'package:flutter/material.dart';

import 'package:app/core/theme/tokens.dart';

class AppBreakpoints {
  static const double mobile = 0;
  static const double tablet = 600;
  static const double desktop = 1024;
}

typedef ResponsiveWidgetBuilder = Widget Function(BuildContext context);

class ResponsiveLayout extends StatelessWidget {
  const ResponsiveLayout({
    super.key,
    required this.mobile,
    this.tablet,
    this.desktop,
  });

  final ResponsiveWidgetBuilder mobile;
  final ResponsiveWidgetBuilder? tablet;
  final ResponsiveWidgetBuilder? desktop;

  static bool isTablet(BuildContext context) =>
      MediaQuery.sizeOf(context).width >= AppBreakpoints.tablet;

  static bool isDesktop(BuildContext context) =>
      MediaQuery.sizeOf(context).width >= AppBreakpoints.desktop;

  @override
  Widget build(BuildContext context) {
    final width = MediaQuery.sizeOf(context).width;
    if (width >= AppBreakpoints.desktop && desktop != null) {
      return desktop!(context);
    }
    if (width >= AppBreakpoints.tablet && tablet != null) {
      return tablet!(context);
    }
    return mobile(context);
  }
}

/// Provides adaptive page padding that scales with breakpoints.
class ResponsivePagePadding extends StatelessWidget {
  const ResponsivePagePadding({
    super.key,
    required this.child,
    this.desktopPadding = const EdgeInsets.symmetric(
      horizontal: AppTokens.spaceXXL,
      vertical: AppTokens.spaceXL,
    ),
    this.tabletPadding = const EdgeInsets.symmetric(
      horizontal: AppTokens.spaceXL,
      vertical: AppTokens.spaceL,
    ),
    this.mobilePadding = const EdgeInsets.symmetric(
      horizontal: AppTokens.spaceL,
      vertical: AppTokens.spaceM,
    ),
  });

  final Widget child;
  final EdgeInsets desktopPadding;
  final EdgeInsets tabletPadding;
  final EdgeInsets mobilePadding;

  @override
  Widget build(BuildContext context) {
    final padding = ResponsiveLayout.isDesktop(context)
        ? desktopPadding
        : ResponsiveLayout.isTablet(context)
        ? tabletPadding
        : mobilePadding;
    return Padding(padding: padding, child: child);
  }
}
