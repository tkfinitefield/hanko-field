import 'package:app/core/network/connectivity_service.dart';
import 'package:app/core/network/interceptors/http_interceptor.dart';
import 'package:app/core/network/network_exception.dart';

/// Prevents network calls when the device is offline.
class ConnectivityInterceptor extends HttpInterceptor {
  ConnectivityInterceptor(this._connectivity);

  final ConnectivityService _connectivity;

  @override
  Future<void> onRequest(HttpRequestContext context) async {
    final online = await _connectivity.isOnline();
    if (!online) {
      throw const NetworkOfflineException();
    }
  }
}
