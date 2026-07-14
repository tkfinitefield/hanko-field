import 'dart:convert';
import 'dart:io';

import 'package:flutter/foundation.dart';

import 'api_json.dart';

const _hankoApiBaseUrlOverride = String.fromEnvironment('HANKO_API_BASE_URL');
const _hankoWebApiBaseUrlOverride = String.fromEnvironment(
  'HANKO_WEB_API_BASE_URL',
);
const _legacyHankoAppProdApiBaseUrlOverride = String.fromEnvironment(
  'HANKO_APP_PROD_API_BASE_URL',
);
const productionHankoApiBaseUrl = String.fromEnvironment(
  'HANKO_PRODUCTION_API_BASE_URL',
  defaultValue: 'https://hanko-field-api-26orkkye6a-an.a.run.app',
);

final defaultHankoApiBaseUrl = resolveDefaultHankoApiBaseUrl();

String resolveDefaultHankoApiBaseUrl({bool? isAndroid, bool? isReleaseMode}) {
  final configured = _hankoApiBaseUrlOverride.trim();
  if (configured.isNotEmpty) {
    return configured;
  }

  final webConfigured = _hankoWebApiBaseUrlOverride.trim();
  if (webConfigured.isNotEmpty) {
    return webConfigured;
  }

  final legacyConfigured = _legacyHankoAppProdApiBaseUrlOverride.trim();
  if (legacyConfigured.isNotEmpty) {
    return legacyConfigured;
  }

  if (isReleaseMode ?? kReleaseMode) {
    return productionHankoApiBaseUrl;
  }

  if (isAndroid ?? Platform.isAndroid) {
    return 'http://10.0.2.2:3050';
  }
  return 'http://127.0.0.1:3050';
}

class HankoApiClient {
  HankoApiClient({required this.baseUri, HankoApiTransport? transport})
    : _transport = transport ?? HankoHttpApiTransport();

  final Uri baseUri;
  final HankoApiTransport _transport;

  Future<JsonMap> getJson(
    String path, {
    Map<String, String?> queryParameters = const {},
  }) {
    return _send('GET', path, queryParameters: queryParameters);
  }

  Future<JsonMap> postJson(String path, JsonMap body) {
    return _send('POST', path, body: body);
  }

  Future<JsonMap> _send(
    String method,
    String path, {
    JsonMap? body,
    Map<String, String?> queryParameters = const {},
  }) async {
    final response = await _transport.send(
      HankoApiRequest(
        method: method,
        uri: _buildUri(path, queryParameters),
        body: body,
        headers: const {
          HttpHeaders.acceptHeader: 'application/json',
          HttpHeaders.contentTypeHeader: 'application/json; charset=utf-8',
        },
      ),
    );

    late final JsonMap decoded;
    try {
      decoded = _decodeJsonObject(response.body, 'API response');
    } on FormatException catch (error) {
      if (!response.isSuccess) {
        throw HankoApiException(
          statusCode: response.statusCode,
          code: 'http_${response.statusCode}',
          message: 'API request failed with a non-JSON response',
          payload: {'parse_error': error.message, 'body': response.body},
        );
      }
      rethrow;
    }

    if (response.isSuccess) {
      return decoded;
    }

    final error = decoded['error'];
    if (error is Map) {
      final errorJson = asJsonMap(error, 'API error');
      throw HankoApiException(
        statusCode: response.statusCode,
        code: readString(errorJson, 'code', defaultValue: 'api_error'),
        message: readString(errorJson, 'message', defaultValue: 'API error'),
        payload: decoded,
      );
    }

    throw HankoApiException(
      statusCode: response.statusCode,
      code: 'http_${response.statusCode}',
      message: 'API request failed with status ${response.statusCode}',
      payload: decoded,
    );
  }

  Uri _buildUri(String path, Map<String, String?> queryParameters) {
    final base = baseUri.toString().replaceFirst(RegExp(r'/$'), '');
    final normalizedPath = path.startsWith('/') ? path : '/$path';
    final query = <String, String>{};
    for (final entry in queryParameters.entries) {
      final value = entry.value;
      if (value != null && value.isNotEmpty) {
        query[entry.key] = value;
      }
    }
    return Uri.parse(
      '$base$normalizedPath',
    ).replace(queryParameters: query.isEmpty ? null : query);
  }

  JsonMap _decodeJsonObject(String body, String context) {
    final trimmed = body.trim();
    if (trimmed.isEmpty) {
      return const {};
    }
    try {
      return asJsonMap(jsonDecode(trimmed), context);
    } on FormatException {
      rethrow;
    } catch (error) {
      throw FormatException('$context is not valid JSON: $error');
    }
  }
}

class HankoApiRequest {
  const HankoApiRequest({
    required this.method,
    required this.uri,
    this.body,
    this.headers = const {},
  });

  final String method;
  final Uri uri;
  final JsonMap? body;
  final Map<String, String> headers;
}

class HankoApiResponse {
  const HankoApiResponse({required this.statusCode, required this.body});

  final int statusCode;
  final String body;

  bool get isSuccess => statusCode >= 200 && statusCode < 300;
}

abstract interface class HankoApiTransport {
  Future<HankoApiResponse> send(HankoApiRequest request);
}

class HankoHttpApiTransport implements HankoApiTransport {
  HankoHttpApiTransport({HttpClient? httpClient})
    : _httpClient = httpClient ?? (HttpClient()..connectionTimeout = _timeout);

  static const _timeout = Duration(seconds: 120);
  final HttpClient _httpClient;

  @override
  Future<HankoApiResponse> send(HankoApiRequest request) async {
    final httpRequest = await _httpClient.openUrl(request.method, request.uri);
    for (final header in request.headers.entries) {
      httpRequest.headers.set(header.key, header.value);
    }

    final body = request.body;
    if (body != null) {
      httpRequest.add(utf8.encode(jsonEncode(body)));
    }

    final response = await httpRequest.close().timeout(_timeout);
    return HankoApiResponse(
      statusCode: response.statusCode,
      body: await response.transform(utf8.decoder).join().timeout(_timeout),
    );
  }
}

class HankoApiException implements Exception {
  const HankoApiException({
    required this.statusCode,
    required this.code,
    required this.message,
    required this.payload,
  });

  final int statusCode;
  final String code;
  final String message;
  final JsonMap payload;

  @override
  String toString() {
    return 'HankoApiException($statusCode, $code, $message)';
  }
}
