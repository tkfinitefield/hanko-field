import 'package:flutter/material.dart';

import 'package:app/core/theme/tokens.dart';

enum AppCardVariant { elevated, outlined, filled }

class AppCard extends StatelessWidget {
  const AppCard({
    super.key,
    required this.child,
    this.variant = AppCardVariant.elevated,
    this.padding = const EdgeInsets.all(AppTokens.spaceL),
    this.onTap,
    this.backgroundColor,
    this.borderColor,
    this.margin,
  });

  final Widget child;
  final AppCardVariant variant;
  final EdgeInsetsGeometry padding;
  final VoidCallback? onTap;
  final Color? backgroundColor;
  final Color? borderColor;
  final EdgeInsetsGeometry? margin;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final cardColor =
        backgroundColor ??
        switch (variant) {
          AppCardVariant.elevated => scheme.surface,
          AppCardVariant.outlined => scheme.surface,
          AppCardVariant.filled => scheme.surfaceVariant,
        };

    final radius = AppTokens.radiusL;
    final border = switch (variant) {
      AppCardVariant.outlined => Border.all(
        color: borderColor ?? scheme.outlineVariant,
      ),
      _ => null,
    };

    final shadows = variant == AppCardVariant.elevated
        ? [
            BoxShadow(
              color: scheme.shadow.withOpacity(0.08),
              blurRadius: 12,
              offset: const Offset(0, 6),
            ),
          ]
        : null;

    final content = Padding(padding: padding, child: child);

    final body = DecoratedBox(
      decoration: BoxDecoration(
        color: cardColor,
        borderRadius: radius,
        border: border,
        boxShadow: shadows,
      ),
      child: Material(
        type: MaterialType.transparency,
        child: InkWell(borderRadius: radius, onTap: onTap, child: content),
      ),
    );

    if (margin == null) {
      return body;
    }
    return Padding(padding: margin!, child: body);
  }
}

class AppListTile extends StatelessWidget {
  const AppListTile({
    super.key,
    required this.title,
    this.subtitle,
    this.leading,
    this.trailing,
    this.onTap,
    this.showDivider = false,
    this.padding,
    this.dividerSpacing = AppTokens.spaceS,
  });

  final String title;
  final String? subtitle;
  final Widget? leading;
  final Widget? trailing;
  final VoidCallback? onTap;
  final bool showDivider;
  final EdgeInsetsGeometry? padding;
  final double dividerSpacing;

  @override
  Widget build(BuildContext context) {
    final tile = AppCard(
      onTap: onTap,
      padding: padding ?? const EdgeInsets.all(AppTokens.spaceL),
      variant: AppCardVariant.outlined,
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.center,
        children: [
          if (leading != null)
            Padding(
              padding: const EdgeInsets.only(right: AppTokens.spaceM),
              child: leading!,
            ),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  title,
                  style: Theme.of(context).textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.w600,
                  ),
                ),
                if (subtitle != null)
                  Padding(
                    padding: const EdgeInsets.only(top: AppTokens.spaceXS),
                    child: Text(
                      subtitle!,
                      style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                        color: Theme.of(context).colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ),
              ],
            ),
          ),
          if (trailing != null)
            Padding(
              padding: const EdgeInsets.only(left: AppTokens.spaceM),
              child: trailing!,
            ),
        ],
      ),
    );

    if (!showDivider) {
      return tile;
    }

    return Column(
      children: [
        tile,
        Padding(
          padding: EdgeInsets.only(top: dividerSpacing),
          child: const Divider(height: 1),
        ),
      ],
    );
  }
}
