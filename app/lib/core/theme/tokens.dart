import 'package:flutter/material.dart';

/// Design tokens shared across the app.
class AppTokens {
  // Colors
  static const Color brandSeed = Color(0xFF6C5CE7); // Violet
  static const Color neutralSeed = Color(0xFF2A2A2A);

  // Spacing (in logical pixels)
  static const double spaceXS = 4;
  static const double spaceS = 8;
  static const double spaceM = 12;
  static const double spaceL = 16;
  static const double spaceXL = 24;
  static const double spaceXXL = 32;

  // Radius
  static const BorderRadius radiusS = BorderRadius.all(Radius.circular(4));
  static const BorderRadius radiusM = BorderRadius.all(Radius.circular(8));
  static const BorderRadius radiusL = BorderRadius.all(Radius.circular(12));

  // Animation durations
  static const Duration fast = Duration(milliseconds: 150);
  static const Duration medium = Duration(milliseconds: 250);
  static const Duration slow = Duration(milliseconds: 400);

  // Elevation levels
  static const List<double> elevations = [0, 1, 3, 6, 8, 12];
}

class AppTypography {
  static TextTheme textTheme(Brightness brightness) {
    final base = brightness == Brightness.dark
        ? Typography.blackMountainView
        : Typography.whiteMountainView;
    // Start from Material defaults and fine-tune if needed.
    return base
        .copyWith(
          headlineMedium: base.headlineMedium?.copyWith(fontWeight: FontWeight.w600),
          titleLarge: base.titleLarge?.copyWith(fontWeight: FontWeight.w600),
          bodyLarge: base.bodyLarge?.copyWith(height: 1.3),
          bodyMedium: base.bodyMedium?.copyWith(height: 1.35),
        )
        .apply(fontFamily: '');
  }
}

