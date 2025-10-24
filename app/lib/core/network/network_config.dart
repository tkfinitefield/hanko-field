class NetworkConfig {
  const NetworkConfig({
    required this.baseUrl,
    required this.userAgent,
    required this.localeTag,
    this.connectTimeout = const Duration(seconds: 10),
    this.receiveTimeout = const Duration(seconds: 20),
    this.sendTimeout = const Duration(seconds: 10),
    this.defaultHeaders = const <String, String>{},
  });

  final String baseUrl;
  final String userAgent;
  final String localeTag;
  final Duration connectTimeout;
  final Duration receiveTimeout;
  final Duration sendTimeout;
  final Map<String, String> defaultHeaders;

  Map<String, String> buildHeaders() {
    return {
      'User-Agent': userAgent,
      'Accept-Language': localeTag,
      'Accept': 'application/json',
      ...defaultHeaders,
    };
  }
}
