import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:app/core/theme/tokens.dart';
import 'package:app/core/ui/ui.dart';
import 'package:app/l10n/gen/app_localizations.dart';

import 'package:app/features/sample_counter/application/providers.dart';

class CounterScreen extends ConsumerWidget {
  const CounterScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final counter = ref.watch(counterProvider);
    final l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(l10n.counterScreenTitle)),
      body: ResponsivePagePadding(
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 420),
            child: switch (counter) {
              AsyncData(:final value) => AppCard(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      l10n.countLabel(value),
                      style: Theme.of(context).textTheme.headlineSmall
                          ?.copyWith(fontWeight: FontWeight.bold),
                    ),
                    const SizedBox(height: AppTokens.spaceL),
                    const _CounterNoteField(),
                    const SizedBox(height: AppTokens.spaceL),
                    Row(
                      children: [
                        Expanded(
                          child: AppButton(
                            label: l10n.increment,
                            fullWidth: true,
                            onPressed: () =>
                                ref.read(counterProvider.notifier).increment(),
                            leadingIcon: const Icon(
                              Icons.add_outlined,
                              size: 18,
                            ),
                          ),
                        ),
                        const SizedBox(width: AppTokens.spaceM),
                        Expanded(
                          child: AppButton(
                            label: 'Show Sheet',
                            variant: AppButtonVariant.ghost,
                            fullWidth: true,
                            onPressed: () => showAppBottomSheet(
                              context: context,
                              builder: (ctx) => Column(
                                mainAxisSize: MainAxisSize.min,
                                crossAxisAlignment: CrossAxisAlignment.start,
                                children: [
                                  Text(
                                    'Bottom sheet',
                                    style: Theme.of(ctx).textTheme.titleMedium
                                        ?.copyWith(fontWeight: FontWeight.w600),
                                  ),
                                  const SizedBox(height: AppTokens.spaceM),
                                  Text(
                                    'This sheet uses shared spacing and '
                                    'rounded corners to match the design system.',
                                    style: Theme.of(ctx).textTheme.bodyMedium,
                                  ),
                                ],
                              ),
                            ),
                          ),
                        ),
                      ],
                    ),
                  ],
                ),
              ),
              AsyncLoading() => const AppListSkeleton(items: 1),
              AsyncError(:final error) => AppEmptyState(
                title: 'Something went wrong',
                message: '$error',
                icon: Icon(
                  Icons.sentiment_dissatisfied_outlined,
                  color: Theme.of(context).colorScheme.error,
                ),
                primaryAction: AppButton(
                  label: 'Retry',
                  onPressed: () => ref.invalidate(counterProvider),
                  fullWidth: true,
                ),
              ),
            },
          ),
        ),
      ),
    );
  }
}

class _CounterNoteField extends StatefulWidget {
  const _CounterNoteField();

  @override
  State<_CounterNoteField> createState() => _CounterNoteFieldState();
}

class _CounterNoteFieldState extends State<_CounterNoteField> {
  late final TextEditingController _controller;
  AppValidationState? _state;
  String? _message;

  @override
  void initState() {
    super.initState();
    _controller = TextEditingController();
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  void _handleChanged(String value) {
    setState(() {
      if (value.length < 3) {
        _state = AppValidationState.warning;
        _message = 'Enter at least 3 characters';
      } else {
        _state = AppValidationState.success;
        _message = 'Looks good';
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    return AppTextField(
      controller: _controller,
      label: 'Notes',
      hint: 'Optional memo for the count',
      helper: 'Shared form field component preview',
      onChanged: _handleChanged,
      validationMessage: _message,
      validationState: _state,
    );
  }
}
