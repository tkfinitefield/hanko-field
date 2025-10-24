import 'dart:convert';

import 'package:app/core/network/interceptors/http_interceptor.dart';
import 'package:logging/logging.dart';

/// Logs outbound requests and inbound responses with light redaction.
class LoggingInterceptor extends HttpInterceptor {
  LoggingInterceptor({
    Logger? logger,
    this.logRequestHeaders = true,
    this.logRequestBody = false,
    this.logResponseBody = false,
    this.maxBodyCharacters = 512,
  }) : _logger = logger ?? Logger('network');

  final Logger _logger;
  final bool logRequestHeaders;
  final bool logRequestBody;
  final bool logResponseBody;
  final int maxBodyCharacters;

  static const _redactedHeaders = {'authorization', 'cookie', 'set-cookie'};

  @override
  Future<void> onRequest(HttpRequestContext context) async {
    final request = context.request;
    final buffer = StringBuffer(
      '--> ${request.method.toUpperCase()} ${request.url}',
    );
    if (logRequestHeaders && request.headers.isNotEmpty) {
      buffer.write('\nheaders: ${jsonEncode(_redactHeaders(request.headers))}');
    }
    if (logRequestBody && request.body.isNotEmpty) {
      buffer.write('\nbody: ${_truncate(request.body)}');
    }
    _logger.fine(buffer.toString());
  }

  @override
  Future<void> onResponse(HttpResponseContext context) async {
    final response = context.response;
    final buffer = StringBuffer(
      '<-- ${response.statusCode} '
      '${context.request.method.toUpperCase()} '
      '${context.request.url} (${context.elapsed.inMilliseconds}ms)',
    );
    if (logResponseBody && response.body.isNotEmpty) {
      buffer.write('\n${_truncate(response.body)}');
    }
    _logger.fine(buffer.toString());
  }

  @override
  Future<void> onError(HttpErrorContext context) async {
    final buffer = StringBuffer(
      'xx> ${context.request.method.toUpperCase()} '
      '${context.request.url} failed (${context.elapsed.inMilliseconds}ms): '
      '${context.error.message}',
    );
    if (context.willRetry) {
      buffer.write(' (retrying)');
    }
    _logger.warning(buffer.toString(), context.error, context.stackTrace);
  }

  Map<String, String> _redactHeaders(Map<String, String> headers) {
    return headers.map(
      (key, value) => _redactedHeaders.contains(key.toLowerCase())
          ? MapEntry(key, 'REDACTED')
          : MapEntry(key, value),
    );
  }

  String _truncate(String value) {
    if (value.length <= maxBodyCharacters) {
      return value;
    }
    return '${value.substring(0, maxBodyCharacters)}...';
  }
}
