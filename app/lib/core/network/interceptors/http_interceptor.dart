import 'package:app/core/network/network_exception.dart';
import 'package:http/http.dart' as http;

class HttpRequestContext {
  HttpRequestContext({required this.request, required this.attempt});

  final http.Request request;
  final int attempt;
}

class HttpResponseContext {
  HttpResponseContext({
    required this.request,
    required this.response,
    required this.attempt,
    required this.elapsed,
  });

  final http.Request request;
  final http.Response response;
  final int attempt;
  final Duration elapsed;
}

class HttpErrorContext {
  HttpErrorContext({
    required this.request,
    required this.error,
    required this.attempt,
    required this.elapsed,
    required this.willRetry,
    required this.stackTrace,
  });

  final http.Request request;
  final NetworkException error;
  final int attempt;
  final Duration elapsed;
  final bool willRetry;
  final StackTrace stackTrace;
}

abstract class HttpInterceptor {
  Future<void> onRequest(HttpRequestContext context) async {}

  Future<void> onResponse(HttpResponseContext context) async {}

  Future<void> onError(HttpErrorContext context) async {}
}
