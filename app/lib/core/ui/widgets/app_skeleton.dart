import 'package:flutter/material.dart';

import 'package:app/core/theme/tokens.dart';
import 'package:app/core/ui/widgets/app_card.dart';

class AppShimmer extends StatefulWidget {
  const AppShimmer({super.key, required this.child, this.duration});

  final Widget child;
  final Duration? duration;

  @override
  State<AppShimmer> createState() => _AppShimmerState();
}

class _AppShimmerState extends State<AppShimmer>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      vsync: this,
      duration: widget.duration ?? const Duration(milliseconds: 1200),
    )..repeat();
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final baseColor = Theme.of(context).colorScheme.surfaceVariant;
    final highlight = Theme.of(context).colorScheme.surface;

    return AnimatedBuilder(
      animation: _controller,
      builder: (context, child) {
        return ShaderMask(
          shaderCallback: (rect) {
            final gradient = LinearGradient(
              begin: const Alignment(-1, 0),
              end: const Alignment(1, 0),
              colors: [
                baseColor.withOpacity(0.25),
                highlight.withOpacity(0.6),
                baseColor.withOpacity(0.25),
              ],
              stops: const [0.1, 0.5, 0.9],
              transform: _SlidingGradientTransform(
                slidePercent: _controller.value,
              ),
            );
            return gradient.createShader(rect);
          },
          blendMode: BlendMode.srcATop,
          child: child,
        );
      },
      child: widget.child,
    );
  }
}

class _SlidingGradientTransform extends GradientTransform {
  const _SlidingGradientTransform({required this.slidePercent});

  final double slidePercent;

  @override
  Matrix4 transform(Rect bounds, {TextDirection? textDirection}) {
    final dx = bounds.width * (slidePercent * 2 - 1);
    return Matrix4.translationValues(dx, 0, 0);
  }
}

class AppSkeletonBlock extends StatelessWidget {
  const AppSkeletonBlock({
    super.key,
    required this.height,
    this.width,
    this.borderRadius,
  });

  final double height;
  final double? width;
  final BorderRadius? borderRadius;

  @override
  Widget build(BuildContext context) {
    final radius = borderRadius ?? AppTokens.radiusM;
    final color = Theme.of(context).colorScheme.surfaceVariant;
    return AppShimmer(
      child: Container(
        width: width ?? double.infinity,
        height: height,
        decoration: BoxDecoration(
          color: color.withOpacity(0.5),
          borderRadius: radius,
        ),
      ),
    );
  }
}

class AppListSkeleton extends StatelessWidget {
  const AppListSkeleton({
    super.key,
    this.items = 3,
    this.spacing = AppTokens.spaceM,
  });

  final int items;
  final double spacing;

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        for (int index = 0; index < items; index++)
          Padding(
            padding: EdgeInsets.only(bottom: index == items - 1 ? 0 : spacing),
            child: AppCard(
              variant: AppCardVariant.filled,
              padding: const EdgeInsets.all(AppTokens.spaceL),
              child: Row(
                children: [
                  const AppSkeletonBlock(
                    height: 48,
                    width: 48,
                    borderRadius: AppTokens.radiusL,
                  ),
                  const SizedBox(width: AppTokens.spaceL),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: const [
                        FractionallySizedBox(
                          widthFactor: 0.7,
                          child: AppSkeletonBlock(height: 12),
                        ),
                        SizedBox(height: AppTokens.spaceS),
                        FractionallySizedBox(
                          widthFactor: 0.45,
                          child: AppSkeletonBlock(height: 12),
                        ),
                      ],
                    ),
                  ),
                ],
              ),
            ),
          ),
      ],
    );
  }
}
