import 'package:flutter/material.dart';

import 'package:app/core/theme/tokens.dart';

enum AppButtonVariant { primary, secondary, ghost }

enum AppButtonSize { small, medium, large }

/// Shared button component that follows the design tokens.
class AppButton extends StatelessWidget {
  const AppButton({
    super.key,
    required this.label,
    this.onPressed,
    this.variant = AppButtonVariant.primary,
    this.size = AppButtonSize.medium,
    this.leadingIcon,
    this.trailingIcon,
    this.loading = false,
    this.fullWidth = false,
    this.alignLabel = TextAlign.center,
  });

  final String label;
  final VoidCallback? onPressed;
  final AppButtonVariant variant;
  final AppButtonSize size;
  final Widget? leadingIcon;
  final Widget? trailingIcon;
  final bool loading;
  final bool fullWidth;
  final TextAlign alignLabel;

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    final buttonStyle = _styleForVariant(context, colorScheme);
    final button = switch (variant) {
      AppButtonVariant.primary => FilledButton(
        onPressed: loading ? null : onPressed,
        style: buttonStyle,
        child: _ButtonContent(
          label: label,
          size: size,
          leadingIcon: leadingIcon,
          trailingIcon: trailingIcon,
          loading: loading,
          progressColor: colorScheme.onPrimary,
          alignLabel: alignLabel,
        ),
      ),
      AppButtonVariant.secondary => OutlinedButton(
        onPressed: loading ? null : onPressed,
        style: buttonStyle,
        child: _ButtonContent(
          label: label,
          size: size,
          leadingIcon: leadingIcon,
          trailingIcon: trailingIcon,
          loading: loading,
          progressColor: colorScheme.primary,
          alignLabel: alignLabel,
        ),
      ),
      AppButtonVariant.ghost => TextButton(
        onPressed: loading ? null : onPressed,
        style: buttonStyle,
        child: _ButtonContent(
          label: label,
          size: size,
          leadingIcon: leadingIcon,
          trailingIcon: trailingIcon,
          loading: loading,
          progressColor: colorScheme.primary,
          alignLabel: alignLabel,
        ),
      ),
    };

    if (!fullWidth) {
      return button;
    }
    return SizedBox(width: double.infinity, child: button);
  }

  ButtonStyle _styleForVariant(BuildContext context, ColorScheme scheme) {
    final padding = switch (size) {
      AppButtonSize.small => const EdgeInsets.symmetric(
        horizontal: AppTokens.spaceM,
        vertical: AppTokens.spaceS,
      ),
      AppButtonSize.medium => const EdgeInsets.symmetric(
        horizontal: AppTokens.spaceL,
        vertical: AppTokens.spaceS + 2,
      ),
      AppButtonSize.large => const EdgeInsets.symmetric(
        horizontal: AppTokens.spaceXL,
        vertical: AppTokens.spaceM,
      ),
    };

    final textStyle = Theme.of(context).textTheme.labelLarge;
    final minimumHeight = switch (size) {
      AppButtonSize.small => 36.0,
      AppButtonSize.medium => 44.0,
      AppButtonSize.large => 52.0,
    };

    final shape = RoundedRectangleBorder(borderRadius: AppTokens.radiusM);

    return switch (variant) {
      AppButtonVariant.primary => FilledButton.styleFrom(
        minimumSize: Size(0, minimumHeight),
        padding: padding,
        textStyle: textStyle,
        shape: shape,
      ),
      AppButtonVariant.secondary => OutlinedButton.styleFrom(
        minimumSize: Size(0, minimumHeight),
        padding: padding,
        textStyle: textStyle,
        shape: shape,
        side: BorderSide(color: scheme.outline),
      ),
      AppButtonVariant.ghost => TextButton.styleFrom(
        minimumSize: Size(0, minimumHeight),
        padding: padding,
        textStyle: textStyle,
        shape: shape,
        foregroundColor: scheme.primary,
      ),
    };
  }
}

class _ButtonContent extends StatelessWidget {
  const _ButtonContent({
    required this.label,
    required this.size,
    required this.loading,
    required this.progressColor,
    required this.alignLabel,
    this.leadingIcon,
    this.trailingIcon,
  });

  final String label;
  final AppButtonSize size;
  final bool loading;
  final Color progressColor;
  final Widget? leadingIcon;
  final Widget? trailingIcon;
  final TextAlign alignLabel;

  @override
  Widget build(BuildContext context) {
    final spacing = size == AppButtonSize.small
        ? AppTokens.spaceS
        : AppTokens.spaceM;

    final child = Row(
      mainAxisSize: MainAxisSize.min,
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        if (leadingIcon != null)
          Padding(
            padding: EdgeInsets.only(right: spacing),
            child: SizedBox.square(dimension: 20, child: leadingIcon!),
          ),
        Flexible(
          child: Text(
            label,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            textAlign: alignLabel,
          ),
        ),
        if (trailingIcon != null)
          Padding(
            padding: EdgeInsets.only(left: spacing),
            child: SizedBox.square(dimension: 20, child: trailingIcon!),
          ),
      ],
    );

    if (!loading) {
      return child;
    }

    return Stack(
      alignment: Alignment.center,
      children: [
        Opacity(opacity: 0, child: child),
        SizedBox(
          width: 18,
          height: 18,
          child: CircularProgressIndicator(
            strokeWidth: 2,
            valueColor: AlwaysStoppedAnimation<Color>(progressColor),
          ),
        ),
      ],
    );
  }
}
