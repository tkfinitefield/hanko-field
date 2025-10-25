import 'package:flutter/material.dart';

import 'package:app/core/theme/tokens.dart';

enum AppValidationState { info, success, warning, error }

class AppValidationMessage extends StatelessWidget {
  const AppValidationMessage({
    super.key,
    required this.message,
    this.state = AppValidationState.info,
  });

  final String message;
  final AppValidationState state;

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    final color = switch (state) {
      AppValidationState.info => colorScheme.primary,
      AppValidationState.success => colorScheme.tertiary,
      AppValidationState.warning => colorScheme.secondary,
      AppValidationState.error => colorScheme.error,
    };

    return Padding(
      padding: const EdgeInsets.only(top: AppTokens.spaceS),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(
            switch (state) {
              AppValidationState.info => Icons.info_outline,
              AppValidationState.success => Icons.check_circle_outline,
              AppValidationState.warning => Icons.warning_amber_outlined,
              AppValidationState.error => Icons.error_outline,
            },
            size: 18,
            color: color,
          ),
          const SizedBox(width: AppTokens.spaceS),
          Expanded(
            child: Text(
              message,
              style: Theme.of(
                context,
              ).textTheme.bodySmall?.copyWith(color: color, height: 1.35),
            ),
          ),
        ],
      ),
    );
  }
}

class AppTextField extends StatelessWidget {
  const AppTextField({
    super.key,
    this.controller,
    this.initialValue,
    this.label,
    this.hint,
    this.helper,
    this.prefix,
    this.suffix,
    this.keyboardType,
    this.textInputAction,
    this.obscureText = false,
    this.enabled = true,
    this.maxLines = 1,
    this.onChanged,
    this.validator,
    this.autovalidateMode,
    this.validationMessage,
    this.validationState,
    this.required = false,
  }) : assert(
         controller == null || initialValue == null,
         'Cannot provide both controller and initialValue',
       );

  final TextEditingController? controller;
  final String? initialValue;
  final String? label;
  final String? hint;
  final String? helper;
  final Widget? prefix;
  final Widget? suffix;
  final TextInputType? keyboardType;
  final TextInputAction? textInputAction;
  final bool obscureText;
  final bool enabled;
  final int? maxLines;
  final ValueChanged<String>? onChanged;
  final FormFieldValidator<String>? validator;
  final AutovalidateMode? autovalidateMode;
  final String? validationMessage;
  final AppValidationState? validationState;
  final bool required;

  @override
  Widget build(BuildContext context) {
    final labelText = switch ((required, label)) {
      (true, final text?) => '$text *',
      (_, final text?) => text,
      _ => null,
    };

    final field = TextFormField(
      controller: controller,
      initialValue: initialValue,
      keyboardType: keyboardType,
      textInputAction: textInputAction,
      obscureText: obscureText,
      enabled: enabled,
      maxLines: obscureText ? 1 : maxLines,
      onChanged: onChanged,
      validator: validator,
      autovalidateMode: autovalidateMode,
      decoration: InputDecoration(
        labelText: labelText,
        hintText: hint,
        helperText: helper,
        prefixIcon: prefix,
        suffixIcon: suffix,
        filled: true,
        fillColor: Theme.of(context).colorScheme.surface,
        contentPadding: const EdgeInsets.symmetric(
          horizontal: AppTokens.spaceL,
          vertical: AppTokens.spaceM,
        ),
        border: OutlineInputBorder(borderRadius: AppTokens.radiusM),
      ),
    );

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        field,
        if (validationMessage != null)
          AppValidationMessage(
            message: validationMessage!,
            state: validationState ?? AppValidationState.info,
          ),
      ],
    );
  }
}
