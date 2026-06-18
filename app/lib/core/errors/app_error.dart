import 'dart:async';
import 'dart:io';

import '../../app/localization/app_localization.dart';
import '../api/core_api.dart';

enum HankoAppErrorKind { network, server, storage, deepLink, generic }

class HankoStorageException implements Exception {
  const HankoStorageException([this.cause]);

  final Object? cause;

  @override
  String toString() =>
      cause == null ? 'HankoStorageException' : 'HankoStorageException: $cause';
}

class HankoDeepLinkException implements Exception {
  const HankoDeepLinkException(this.route, [this.cause]);

  final String route;
  final Object? cause;

  @override
  String toString() => cause == null
      ? 'HankoDeepLinkException: $route'
      : 'HankoDeepLinkException: $route ($cause)';
}

class HankoAppError {
  const HankoAppError({
    required this.kind,
    this.statusCode,
    this.code,
    this.cause,
  });

  const HankoAppError.storage({Object? cause})
    : this(kind: HankoAppErrorKind.storage, cause: cause);

  const HankoAppError.deepLink({Object? cause})
    : this(kind: HankoAppErrorKind.deepLink, cause: cause);

  factory HankoAppError.fromObject(Object? error) {
    if (error is HankoAppError) {
      return error;
    }

    if (error is HankoStorageException) {
      return HankoAppError(kind: HankoAppErrorKind.storage, cause: error);
    }

    if (error is HankoDeepLinkException) {
      return HankoAppError(kind: HankoAppErrorKind.deepLink, cause: error);
    }

    if (error is HankoApiException) {
      final code = error.code.trim().toLowerCase();
      if (_isNetworkApiError(error.statusCode, code)) {
        return HankoAppError(
          kind: HankoAppErrorKind.network,
          statusCode: error.statusCode,
          code: error.code,
          cause: error,
        );
      }
      if (_isServerApiError(error.statusCode, code)) {
        return HankoAppError(
          kind: HankoAppErrorKind.server,
          statusCode: error.statusCode,
          code: error.code,
          cause: error,
        );
      }
      return HankoAppError(
        kind: HankoAppErrorKind.generic,
        statusCode: error.statusCode,
        code: error.code,
        cause: error,
      );
    }

    if (error is SocketException ||
        error is TimeoutException ||
        error is HandshakeException) {
      return HankoAppError(kind: HankoAppErrorKind.network, cause: error);
    }

    if (error is FileSystemException) {
      return HankoAppError(kind: HankoAppErrorKind.storage, cause: error);
    }

    return HankoAppError(kind: HankoAppErrorKind.generic, cause: error);
  }

  final HankoAppErrorKind kind;
  final int? statusCode;
  final String? code;
  final Object? cause;

  String title(GeneratedHankoLocalizations l10n) {
    return switch (kind) {
      HankoAppErrorKind.network => l10n.commonNetworkErrorTitle,
      HankoAppErrorKind.server => l10n.commonServerErrorTitle,
      HankoAppErrorKind.storage => l10n.storageErrorTitle,
      HankoAppErrorKind.deepLink => l10n.deepLinkErrorTitle,
      HankoAppErrorKind.generic => l10n.commonGenericErrorTitle,
    };
  }

  String message(GeneratedHankoLocalizations l10n) {
    return switch (kind) {
      HankoAppErrorKind.network => l10n.commonNetworkErrorMessage,
      HankoAppErrorKind.server => l10n.commonServerErrorMessage,
      HankoAppErrorKind.storage => l10n.storageErrorMessage,
      HankoAppErrorKind.deepLink => l10n.deepLinkErrorMessage,
      HankoAppErrorKind.generic => l10n.commonGenericErrorMessage,
    };
  }
}

bool _isNetworkApiError(int statusCode, String code) {
  return statusCode <= 0 ||
      code == 'network_error' ||
      code == 'connection_error' ||
      code == 'timeout';
}

bool _isServerApiError(int statusCode, String code) {
  return (statusCode >= 500 && statusCode < 600) ||
      code == 'internal' ||
      code == 'server_error' ||
      code == 'service_unavailable';
}
