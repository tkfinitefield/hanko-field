import 'dart:async';

import 'package:app/core/app/app_flavor.dart';
import 'package:app/core/firebase/firebase_providers.dart';
import 'package:app/core/privacy/privacy_settings_repository.dart';
import 'package:firebase_crashlytics/firebase_crashlytics.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class CrashReportingState {
  const CrashReportingState({
    required this.consentGranted,
    required this.environment,
  });

  final bool consentGranted;
  final AppFlavor environment;

  CrashReportingState copyWith({bool? consentGranted, AppFlavor? environment}) {
    return CrashReportingState(
      consentGranted: consentGranted ?? this.consentGranted,
      environment: environment ?? this.environment,
    );
  }
}

class CrashReportingController extends AsyncNotifier<CrashReportingState> {
  bool _consentGranted = false;

  FirebaseCrashlytics get _crashlytics => FirebaseCrashlytics.instance;

  PrivacySettingsRepository get _privacyRepository =>
      ref.read(privacySettingsRepositoryProvider);

  AppFlavor get _flavor => ref.read(appFlavorProvider);

  @override
  Future<CrashReportingState> build() async {
    await ref.read(firebaseInitializedProvider.future);

    final allowed = await _privacyRepository.isCrashReportingAllowed();
    await _applyConsent(allowed);
    _consentGranted = allowed;

    return CrashReportingState(consentGranted: allowed, environment: _flavor);
  }

  Future<void> updateConsent(bool allowCollection) async {
    final current = await future;
    try {
      await _privacyRepository.setCrashReportingAllowed(allowCollection);
      await _applyConsent(allowCollection);
      _consentGranted = allowCollection;
      state = AsyncValue.data(
        current.copyWith(consentGranted: allowCollection),
      );
    } catch (error, stack) {
      state = AsyncValue.error(error, stack);
      rethrow;
    }
  }

  void recordError(Object error, StackTrace stack, {bool fatal = false}) {
    if (!_consentGranted) {
      return;
    }
    unawaited(_crashlytics.recordError(error, stack, fatal: fatal));
  }

  void recordFlutterError(FlutterErrorDetails details) {
    if (!_consentGranted) {
      return;
    }
    unawaited(_crashlytics.recordFlutterFatalError(details));
  }

  Future<void> _applyConsent(bool allowed) async {
    await _crashlytics.setCrashlyticsCollectionEnabled(allowed);
    await _crashlytics.setCustomKey('environment', _flavor.name);
    await _crashlytics.setCustomKey(
      'privacy_consent_crash_reporting',
      allowed ? 'granted' : 'denied',
    );
  }
}

final crashReportingControllerProvider =
    AsyncNotifierProvider<CrashReportingController, CrashReportingState>(
      CrashReportingController.new,
    );
