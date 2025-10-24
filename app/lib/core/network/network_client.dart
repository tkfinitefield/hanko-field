import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:app/core/network/interceptors/http_interceptor.dart';
import 'package:app/core/network/network_config.dart';
import 'package:app/core/network/network_exception.dart';
import 'package:app/core/network/retry_policy.dart';
import 'package:http/http.dart' as http;

typedef ResponseParser<T> = T Function(dynamic data);

class NetworkClient {
  NetworkClient({
    required http.Client client,
    required NetworkConfig config,
    required List<HttpInterceptor> interceptors,
    RetryPolicy? retryPolicy,
  }) : _client = client,
       _config = config,
       _interceptors = List.unmodifiable(interceptors),
       _retryPolicy = retryPolicy ?? const RetryPolicy();

  final http.Client _client;
  final NetworkConfig _config;
  final List<HttpInterceptor> _interceptors;
  final RetryPolicy _retryPolicy;

  Future<T> get<T>(
    String path, {
    Map<String, dynamic>? queryParameters,
    Map<String, String>? headers,
    ResponseParser<T>? parser,
  }) {
    return _send(
      method: 'GET',
      path: path,
      queryParameters: queryParameters,
      headers: headers,
      parser: parser,
    );
  }

  Future<T> post<T>(
    String path, {
    Object? data,
    Map<String, dynamic>? queryParameters,
    Map<String, String>? headers,
    ResponseParser<T>? parser,
  }) {
    return _send(
      method: 'POST',
      path: path,
      data: data,
      queryParameters: queryParameters,
      headers: headers,
      parser: parser,
    );
  }

  Future<T> put<T>(
    String path, {
    Object? data,
    Map<String, dynamic>? queryParameters,
    Map<String, String>? headers,
    ResponseParser<T>? parser,
  }) {
    return _send(
      method: 'PUT',
      path: path,
      data: data,
      queryParameters: queryParameters,
      headers: headers,
      parser: parser,
    );
  }

  Future<T> patch<T>(
    String path, {
    Object? data,
    Map<String, dynamic>? queryParameters,
    Map<String, String>? headers,
    ResponseParser<T>? parser,
  }) {
    return _send(
      method: 'PATCH',
      path: path,
      data: data,
      queryParameters: queryParameters,
      headers: headers,
      parser: parser,
    );
  }

  Future<T> delete<T>(
    String path, {
    Object? data,
    Map<String, dynamic>? queryParameters,
    Map<String, String>? headers,
    ResponseParser<T>? parser,
  }) {
    return _send(
      method: 'DELETE',
      path: path,
      data: data,
      queryParameters: queryParameters,
      headers: headers,
      parser: parser,
    );
  }

  Future<T> _send<T>({
    required String method,
    required String path,
    Object? data,
    Map<String, dynamic>? queryParameters,
    Map<String, String>? headers,
    ResponseParser<T>? parser,
  }) async {
    final uri = _resolveUri(path, queryParameters);
    final maxAttempts = _retryPolicy.maxAttempts;
    NetworkException? lastError;

    for (var attempt = 1; attempt <= maxAttempts; attempt++) {
      final request = http.Request(method.toUpperCase(), uri);
      request.headers.addAll(_config.buildHeaders());
      if (headers != null) {
        request.headers.addAll(headers);
      }
      _applyBody(request, data);

      final requestContext = HttpRequestContext(
        request: request,
        attempt: attempt,
      );
      try {
        for (final interceptor in _interceptors) {
          await interceptor.onRequest(requestContext);
        }
      } on NetworkException catch (error, stackTrace) {
        final willRetry = _canRetry(method, error, attempt);
        final context = HttpErrorContext(
          request: request,
          error: error,
          attempt: attempt,
          elapsed: Duration.zero,
          willRetry: willRetry,
          stackTrace: stackTrace,
        );
        await _notifyError(context);

        if (!willRetry) {
          rethrow;
        }
        lastError = error;
        await Future<void>.delayed(_retryPolicy.delay(attempt - 1));
        continue;
      } catch (error, stackTrace) {
        final wrapped = NetworkUnknownException(error, stackTrace: stackTrace);
        final willRetry = _canRetry(method, wrapped, attempt);
        final context = HttpErrorContext(
          request: request,
          error: wrapped,
          attempt: attempt,
          elapsed: Duration.zero,
          willRetry: willRetry,
          stackTrace: stackTrace,
        );
        await _notifyError(context);

        if (!willRetry) {
          throw wrapped;
        }
        lastError = wrapped;
        await Future<void>.delayed(_retryPolicy.delay(attempt - 1));
        continue;
      }

      final stopwatch = Stopwatch()..start();
      try {
        final streamed = await _client
            .send(request)
            .timeout(_config.connectTimeout);
        final response = await http.Response.fromStream(
          streamed,
        ).timeout(_config.receiveTimeout);
        stopwatch.stop();

        final responseContext = HttpResponseContext(
          request: request,
          response: response,
          attempt: attempt,
          elapsed: stopwatch.elapsed,
        );

        for (final interceptor in _interceptors.reversed) {
          await interceptor.onResponse(responseContext);
        }

        if (response.statusCode >= 400) {
          throw _mapResponseError(response);
        }

        return _parseResponse<T>(response, parser);
      } on TimeoutException catch (error, stackTrace) {
        stopwatch.stop();
        final wrapped = NetworkTimeoutException(
          error.message ?? 'Request timed out.',
        );
        final willRetry = _canRetry(method, wrapped, attempt);
        final context = HttpErrorContext(
          request: request,
          error: wrapped,
          attempt: attempt,
          elapsed: stopwatch.elapsed,
          willRetry: willRetry,
          stackTrace: stackTrace,
        );
        await _notifyError(context);

        if (willRetry) {
          lastError = wrapped;
          await Future<void>.delayed(_retryPolicy.delay(attempt - 1));
          continue;
        }
        throw wrapped;
      } on SocketException catch (_, stackTrace) {
        stopwatch.stop();
        const wrapped = NetworkConnectionException();
        final willRetry = _canRetry(method, wrapped, attempt);
        final context = HttpErrorContext(
          request: request,
          error: wrapped,
          attempt: attempt,
          elapsed: stopwatch.elapsed,
          willRetry: willRetry,
          stackTrace: stackTrace,
        );
        await _notifyError(context);

        if (willRetry) {
          lastError = wrapped;
          await Future<void>.delayed(_retryPolicy.delay(attempt - 1));
          continue;
        }
        throw wrapped;
      } on HandshakeException catch (_, stackTrace) {
        stopwatch.stop();
        const wrapped = NetworkSecurityException();
        final context = HttpErrorContext(
          request: request,
          error: wrapped,
          attempt: attempt,
          elapsed: stopwatch.elapsed,
          willRetry: false,
          stackTrace: stackTrace,
        );
        await _notifyError(context);
        throw wrapped;
      } on TlsException catch (_, stackTrace) {
        stopwatch.stop();
        const wrapped = NetworkSecurityException();
        final context = HttpErrorContext(
          request: request,
          error: wrapped,
          attempt: attempt,
          elapsed: stopwatch.elapsed,
          willRetry: false,
          stackTrace: stackTrace,
        );
        await _notifyError(context);
        throw wrapped;
      } on NetworkException catch (error, stackTrace) {
        stopwatch.stop();
        final willRetry = _canRetry(method, error, attempt);
        final context = HttpErrorContext(
          request: request,
          error: error,
          attempt: attempt,
          elapsed: stopwatch.elapsed,
          willRetry: willRetry,
          stackTrace: stackTrace,
        );
        await _notifyError(context);

        if (willRetry) {
          lastError = error;
          await Future<void>.delayed(_retryPolicy.delay(attempt - 1));
          continue;
        }
        rethrow;
      } catch (error, stackTrace) {
        stopwatch.stop();
        final wrapped = NetworkUnknownException(error, stackTrace: stackTrace);
        final willRetry = _canRetry(method, wrapped, attempt);
        final context = HttpErrorContext(
          request: request,
          error: wrapped,
          attempt: attempt,
          elapsed: stopwatch.elapsed,
          willRetry: willRetry,
          stackTrace: stackTrace,
        );
        await _notifyError(context);

        if (willRetry) {
          lastError = wrapped;
          await Future<void>.delayed(_retryPolicy.delay(attempt - 1));
          continue;
        }
        throw wrapped;
      }
    }

    throw lastError ?? const NetworkUnknownException('Request failed.');
  }

  Future<void> _notifyError(HttpErrorContext context) async {
    for (final interceptor in _interceptors.reversed) {
      await interceptor.onError(context);
    }
  }

  bool _canRetry(String method, NetworkException error, int attempt) {
    if (attempt >= _retryPolicy.maxAttempts) {
      return false;
    }
    return _retryPolicy.shouldRetry(
      method: method,
      error: error,
      attempt: attempt,
    );
  }

  Uri _resolveUri(String path, Map<String, dynamic>? queryParameters) {
    final baseUri = Uri.parse(_config.baseUrl);
    final requestUri = Uri.parse(path);

    if (requestUri.hasScheme || requestUri.hasAuthority) {
      final mergedQuery = <String, String>{...requestUri.queryParameters};
      if (queryParameters != null) {
        queryParameters.forEach((key, value) {
          final dynamicValue = value;
          if (dynamicValue != null) {
            mergedQuery[key] = dynamicValue.toString();
          }
        });
      }
      return requestUri.replace(
        queryParameters: mergedQuery.isEmpty ? null : mergedQuery,
      );
    }

    final combinedSegments = <String>[];

    void pushSegment(String segment) {
      if (segment.isEmpty || segment == '.') {
        return;
      }
      if (segment == '..') {
        if (combinedSegments.isNotEmpty) {
          combinedSegments.removeLast();
        }
        return;
      }
      combinedSegments.add(segment);
    }

    for (final segment in baseUri.pathSegments) {
      pushSegment(segment);
    }
    for (final segment in requestUri.pathSegments) {
      pushSegment(segment);
    }

    final mergedQuery = <String, String>{...baseUri.queryParameters};
    mergedQuery.addAll(requestUri.queryParameters);
    if (queryParameters != null) {
      queryParameters.forEach((key, value) {
        final dynamicValue = value;
        if (dynamicValue != null) {
          mergedQuery[key] = dynamicValue.toString();
        }
      });
    }

    return baseUri.replace(
      pathSegments: combinedSegments,
      queryParameters: mergedQuery.isEmpty ? null : mergedQuery,
      fragment: requestUri.fragment.isEmpty ? null : requestUri.fragment,
    );
  }

  void _applyBody(http.Request request, Object? data) {
    if (data == null) {
      return;
    }
    if (data is String) {
      request.body = data;
      return;
    }
    if (data is List<int>) {
      request.bodyBytes = data;
      return;
    }
    if (data is Map || data is Iterable) {
      request.body = jsonEncode(data);
      request.headers.putIfAbsent('Content-Type', () => 'application/json');
      return;
    }
    request.body = data.toString();
  }

  T _parseResponse<T>(http.Response response, ResponseParser<T>? parser) {
    dynamic payload;
    try {
      payload = _decodeBody(response);
    } on NetworkSerializationException {
      rethrow;
    } catch (error, stackTrace) {
      throw NetworkSerializationException(
        'Failed to decode response from ${response.request?.url}: $error',
        stackTrace: stackTrace,
      );
    }

    if (parser != null) {
      try {
        return parser(payload);
      } catch (error, stackTrace) {
        throw NetworkSerializationException(
          'Failed to parse response for ${response.request?.url}: $error',
          stackTrace: stackTrace,
        );
      }
    }

    if (payload is T || payload == null) {
      return payload as T;
    }

    if (T == http.Response) {
      return response as T;
    }

    throw NetworkSerializationException(
      'Unexpected response type ${payload.runtimeType} for ${response.request?.url}.',
    );
  }

  dynamic _decodeBody(http.Response response) {
    if (response.body.isEmpty) {
      return null;
    }
    final contentType = response.headers['content-type'] ?? '';
    if (contentType.contains('application/json')) {
      try {
        return jsonDecode(response.body);
      } on Object catch (error, stackTrace) {
        throw NetworkSerializationException(
          'Invalid JSON response: $error',
          stackTrace: stackTrace,
        );
      }
    }
    return response.body;
  }

  NetworkException _mapResponseError(http.Response response) {
    final status = response.statusCode;
    final body = _safeDecodeBody(response);

    if (status == 401) {
      return const NetworkUnauthorizedException();
    }
    if (status == 403) {
      return const NetworkForbiddenException();
    }
    if (status == 404) {
      return const NetworkNotFoundException();
    }
    if (status == 409) {
      return const NetworkConflictException();
    }
    if (status >= 500) {
      return NetworkServerException(statusCode: status, body: body);
    }
    if (status == 408 || status == 429 || status == 425) {
      return NetworkResponseException(statusCode: status, body: body);
    }
    return NetworkResponseException(statusCode: status, body: body);
  }

  dynamic _safeDecodeBody(http.Response response) {
    try {
      return _decodeBody(response);
    } catch (_) {
      return response.body;
    }
  }
}
