import 'dart:convert';
import 'dart:io';

/// Simple LCOV coverage checker.
///
/// Usage:
///   dart run tool/check_coverage.dart --lcov=coverage/lcov.info --min=60
///
/// Exits with a non-zero code if coverage is below the threshold.
Future<void> main(List<String> args) async {
  final params = _parseArgs(args);
  final lcovPath = params['lcov'] ?? 'coverage/lcov.info';
  final minStr = params['min'] ?? '60';
  final min = double.tryParse(minStr) ?? 60;

  final file = File(lcovPath);
  if (!await file.exists()) {
    stderr.writeln('LCOV file not found: $lcovPath');
    exitCode = 2;
    return;
  }

  final lines = const LineSplitter().convert(await file.readAsString());

  int total = 0;
  int hit = 0;

  for (final line in lines) {
    if (line.startsWith('DA:')) {
      final parts = line.substring(3).split(',');
      if (parts.length == 2) {
        final count = int.tryParse(parts[1]) ?? 0;
        total += 1;
        if (count > 0) hit += 1;
      }
    }
  }

  if (total == 0) {
    stderr.writeln('No coverage data found in $lcovPath');
    exitCode = 3;
    return;
  }

  final pct = (hit / total) * 100.0;
  final pctStr = pct.toStringAsFixed(2);
  stdout.writeln('Coverage: $pctStr% (hits: $hit / total: $total). Minimum: $min%');

  if (pct + 1e-9 < min) {
    stderr.writeln('Coverage below threshold.');
    exitCode = 1;
  }
}

Map<String, String> _parseArgs(List<String> args) {
  final map = <String, String>{};
  for (final arg in args) {
    if (arg.startsWith('--') && arg.contains('=')) {
      final idx = arg.indexOf('=');
      final key = arg.substring(2, idx);
      final value = arg.substring(idx + 1);
      map[key] = value;
    }
  }
  return map;
}

