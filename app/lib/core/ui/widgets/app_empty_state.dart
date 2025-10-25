import 'package:flutter/material.dart';

import 'package:app/core/theme/tokens.dart';
import 'package:app/core/ui/widgets/app_button.dart';

class AppEmptyState extends StatelessWidget {
  const AppEmptyState({
    super.key,
    required this.title,
    this.message,
    this.icon,
    this.primaryAction,
    this.secondaryAction,
  });

  final String title;
  final String? message;
  final Widget? icon;
  final AppButton? primaryAction;
  final AppButton? secondaryAction;

  @override
  Widget build(BuildContext context) {
    final children = <Widget>[
      if (icon != null)
        Padding(
          padding: const EdgeInsets.only(bottom: AppTokens.spaceL),
          child: IconTheme(
            data: IconTheme.of(context).copyWith(size: 56),
            child: icon!,
          ),
        ),
      Text(
        title,
        textAlign: TextAlign.center,
        style: Theme.of(
          context,
        ).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w600),
      ),
      if (message != null)
        Padding(
          padding: const EdgeInsets.only(top: AppTokens.spaceM),
          child: Text(
            message!,
            textAlign: TextAlign.center,
            style: Theme.of(context).textTheme.bodyMedium?.copyWith(
              color: Theme.of(context).colorScheme.onSurfaceVariant,
            ),
          ),
        ),
      if (primaryAction != null || secondaryAction != null)
        Padding(
          padding: const EdgeInsets.only(top: AppTokens.spaceXL),
          child: Column(
            children: [
              if (primaryAction != null)
                SizedBox(width: double.infinity, child: primaryAction!),
              if (secondaryAction != null)
                Padding(
                  padding: const EdgeInsets.only(top: AppTokens.spaceS),
                  child: SizedBox(
                    width: double.infinity,
                    child: secondaryAction!,
                  ),
                ),
            ],
          ),
        ),
    ];

    return Padding(
      padding: const EdgeInsets.all(AppTokens.spaceXL),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        crossAxisAlignment: CrossAxisAlignment.center,
        mainAxisSize: MainAxisSize.min,
        children: children,
      ),
    );
  }
}
