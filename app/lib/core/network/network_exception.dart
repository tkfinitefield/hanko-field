sealed class NetworkException implements Exception {
  const NetworkException(this.message, {this.stackTrace});

  final String message;
  final StackTrace? stackTrace;

  @override
  String toString() => '$runtimeType: $message';
}

class NetworkOfflineException extends NetworkException {
  const NetworkOfflineException() : super('No network connection available.');
}

class NetworkTimeoutException extends NetworkException {
  const NetworkTimeoutException(super.message);
}

class NetworkCancelledException extends NetworkException {
  const NetworkCancelledException() : super('Request cancelled.');
}

class NetworkUnauthorizedException extends NetworkException {
  const NetworkUnauthorizedException() : super('Authentication required.');
}

class NetworkForbiddenException extends NetworkException {
  const NetworkForbiddenException() : super('Access is forbidden.');
}

class NetworkNotFoundException extends NetworkException {
  const NetworkNotFoundException() : super('Resource not found.');
}

class NetworkConflictException extends NetworkException {
  const NetworkConflictException() : super('Resource conflict detected.');
}

class NetworkServerException extends NetworkException {
  const NetworkServerException({required this.statusCode, this.body})
    : super('Server error ($statusCode).');

  final int statusCode;
  final dynamic body;
}

class NetworkResponseException extends NetworkException {
  const NetworkResponseException({
    required this.statusCode,
    this.body,
    String? message,
  }) : super(message ?? 'Unexpected response ($statusCode).');

  final int statusCode;
  final dynamic body;
}

class NetworkSerializationException extends NetworkException {
  const NetworkSerializationException(super.message, {super.stackTrace});
}

class NetworkUnknownException extends NetworkException {
  const NetworkUnknownException(Object error, {StackTrace? stackTrace})
    : super('Unhandled network error: $error', stackTrace: stackTrace);
}

class NetworkConnectionException extends NetworkException {
  const NetworkConnectionException() : super('Failed to connect to server.');
}

class NetworkSecurityException extends NetworkException {
  const NetworkSecurityException() : super('Secure connection failed.');
}
