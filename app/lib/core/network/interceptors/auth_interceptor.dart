import 'package:app/core/network/interceptors/http_interceptor.dart';
import 'package:app/core/storage/secure_storage_service.dart';

/// Injects the bearer access token into outgoing requests when present.
class AuthInterceptor extends HttpInterceptor {
  AuthInterceptor(this._tokenStorage);

  final AuthTokenStorage _tokenStorage;

  @override
  Future<void> onRequest(HttpRequestContext context) async {
    final headers = context.request.headers;
    if (headers.containsKey('Authorization')) {
      return;
    }
    final token = await _tokenStorage.readAccessToken();
    if (token != null && token.isNotEmpty) {
      headers['Authorization'] = 'Bearer $token';
    }
  }
}
