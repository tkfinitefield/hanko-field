import 'package:flutter/material.dart';

import 'package:app/core/theme/tokens.dart';

class AppTheme {
  static ThemeData light() {
    final scheme = ColorScheme.fromSeed(seedColor: AppTokens.brandSeed, brightness: Brightness.light);
    return ThemeData(
      colorScheme: scheme,
      useMaterial3: true,
      visualDensity: VisualDensity.adaptivePlatformDensity,
      appBarTheme: AppBarTheme(
        backgroundColor: scheme.surface,
        foregroundColor: scheme.onSurface,
      ),
      textTheme: AppTypography.textTheme(Brightness.light),
      inputDecorationTheme: const InputDecorationTheme(border: OutlineInputBorder()),
      elevatedButtonTheme: ElevatedButtonThemeData(
        style: ElevatedButton.styleFrom(
          shape: RoundedRectangleBorder(borderRadius: AppTokens.radiusM),
          padding: const EdgeInsets.symmetric(horizontal: AppTokens.spaceL, vertical: AppTokens.spaceS),
        ),
      ),
    );
  }

  static ThemeData dark() {
    final scheme = ColorScheme.fromSeed(seedColor: AppTokens.brandSeed, brightness: Brightness.dark);
    return ThemeData(
      colorScheme: scheme,
      useMaterial3: true,
      visualDensity: VisualDensity.adaptivePlatformDensity,
      appBarTheme: AppBarTheme(
        backgroundColor: scheme.surface,
        foregroundColor: scheme.onSurface,
      ),
      textTheme: AppTypography.textTheme(Brightness.dark),
      inputDecorationTheme: const InputDecorationTheme(border: OutlineInputBorder()),
      elevatedButtonTheme: ElevatedButtonThemeData(
        style: ElevatedButton.styleFrom(
          shape: RoundedRectangleBorder(borderRadius: AppTokens.radiusM),
          padding: const EdgeInsets.symmetric(horizontal: AppTokens.spaceL, vertical: AppTokens.spaceS),
        ),
      ),
    );
  }
}
