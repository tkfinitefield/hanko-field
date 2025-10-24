import 'package:app/core/network/network_exception.dart';

class RetryPolicy {
  const RetryPolicy({
    this.maxAttempts = 3,
    this.baseDelay = const Duration(milliseconds: 300),
    this.maxDelay = const Duration(seconds: 5),
    Set<int>? retryableStatusCodes,
    Set<String>? retryableMethods,
  }) : retryableStatusCodes =
           retryableStatusCodes ??
           const <int>{408, 425, 429, 500, 502, 503, 504},
       retryableMethods =
           retryableMethods ??
           const <String>{'GET', 'PUT', 'DELETE', 'HEAD', 'PATCH'};

  final int maxAttempts;
  final Duration baseDelay;
  final Duration maxDelay;
  final Set<int> retryableStatusCodes;
  final Set<String> retryableMethods;

  bool shouldRetry({
    required String method,
    required NetworkException error,
    required int attempt,
  }) {
    if (attempt >= maxAttempts) {
      return false;
    }

    final normalizedMethod = method.toUpperCase();
    if (!retryableMethods.contains(normalizedMethod)) {
      return false;
    }

    if (error is NetworkTimeoutException ||
        error is NetworkConnectionException) {
      return true;
    }

    if (error is NetworkServerException) {
      return true;
    }

    if (error is NetworkResponseException) {
      return retryableStatusCodes.contains(error.statusCode);
    }

    return false;
  }

  Duration delay(int previousAttempts) {
    final multiplier = 1 << previousAttempts;
    final computed = baseDelay * multiplier;
    return computed > maxDelay ? maxDelay : computed;
  }
}
