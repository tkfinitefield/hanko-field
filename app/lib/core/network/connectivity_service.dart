import 'dart:async';

import 'package:connectivity_plus/connectivity_plus.dart';

class ConnectivityService {
  ConnectivityService(Connectivity? connectivity)
    : _connectivity = connectivity ?? Connectivity();

  final Connectivity _connectivity;

  Future<bool> isOnline() async {
    final statuses = await _connectivity.checkConnectivity();
    return _hasConnection(statuses);
  }

  Stream<bool> onlineStream() {
    return _connectivity.onConnectivityChanged.map(_hasConnection).distinct();
  }

  bool _hasConnection(List<ConnectivityResult> statuses) {
    return statuses.any((status) => status != ConnectivityResult.none);
  }
}
