import 'package:flutter/material.dart';

import 'package:app/core/theme/tokens.dart';
import 'package:app/core/ui/widgets/app_button.dart';

class AppModalAction {
  const AppModalAction({
    required this.label,
    required this.onPressed,
    this.variant = AppButtonVariant.primary,
  });

  final String label;
  final VoidCallback onPressed;
  final AppButtonVariant variant;
}

class AppModal extends StatelessWidget {
  const AppModal({
    super.key,
    this.icon,
    required this.title,
    this.body,
    this.actions = const [],
  });

  final Widget? icon;
  final Widget title;
  final Widget? body;
  final List<AppModalAction> actions;

  @override
  Widget build(BuildContext context) {
    return Dialog(
      insetPadding: const EdgeInsets.symmetric(horizontal: AppTokens.spaceXL),
      shape: RoundedRectangleBorder(borderRadius: AppTokens.radiusL),
      child: Padding(
        padding: const EdgeInsets.all(AppTokens.spaceXL),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            if (icon != null)
              Padding(
                padding: const EdgeInsets.only(bottom: AppTokens.spaceL),
                child: IconTheme(
                  data: IconTheme.of(context).copyWith(size: 48),
                  child: icon!,
                ),
              ),
            DefaultTextStyle.merge(
              style: Theme.of(
                context,
              ).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w600),
              textAlign: TextAlign.center,
              child: title,
            ),
            if (body != null)
              Padding(
                padding: const EdgeInsets.only(top: AppTokens.spaceM),
                child: DefaultTextStyle.merge(
                  style: Theme.of(
                    context,
                  ).textTheme.bodyMedium?.copyWith(height: 1.4),
                  textAlign: TextAlign.center,
                  child: body!,
                ),
              ),
            if (actions.isNotEmpty)
              Padding(
                padding: const EdgeInsets.only(top: AppTokens.spaceXL),
                child: Column(
                  children: [
                    for (final action in actions)
                      Padding(
                        padding: const EdgeInsets.only(
                          bottom: AppTokens.spaceS,
                        ),
                        child: AppButton(
                          label: action.label,
                          variant: action.variant,
                          fullWidth: true,
                          onPressed: () {
                            action.onPressed();
                            if (context.mounted) {
                              Navigator.of(context).pop();
                            }
                          },
                        ),
                      ),
                  ],
                ),
              ),
          ],
        ),
      ),
    );
  }
}

Future<T?> showAppModal<T>({
  required BuildContext context,
  required AppModal modal,
  bool barrierDismissible = true,
}) {
  return showDialog<T>(
    context: context,
    barrierDismissible: barrierDismissible,
    builder: (_) => modal,
  );
}

Future<T?> showAppBottomSheet<T>({
  required BuildContext context,
  required WidgetBuilder builder,
  bool isScrollControlled = false,
}) {
  return showModalBottomSheet<T>(
    context: context,
    isScrollControlled: isScrollControlled,
    useSafeArea: true,
    showDragHandle: true,
    backgroundColor: Theme.of(context).colorScheme.surface,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(24)),
    ),
    builder: (sheetContext) {
      return Padding(
        padding: EdgeInsets.only(
          left: AppTokens.spaceXL,
          right: AppTokens.spaceXL,
          top: AppTokens.spaceXL,
          bottom:
              MediaQuery.of(sheetContext).viewInsets.bottom + AppTokens.spaceXL,
        ),
        child: builder(sheetContext),
      );
    },
  );
}
