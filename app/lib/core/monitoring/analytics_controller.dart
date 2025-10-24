import 'dart:async';

import 'package:app/core/app/app_flavor.dart';
import 'package:app/core/firebase/firebase_providers.dart';
import 'package:app/core/monitoring/analytics_events.dart';
import 'package:app/core/privacy/privacy_settings_repository.dart';
import 'package:firebase_analytics/firebase_analytics.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class AnalyticsState {
  const AnalyticsState({required this.consentGranted});

  final bool consentGranted;

  AnalyticsState copyWith({bool? consentGranted}) {
    return AnalyticsState(
      consentGranted: consentGranted ?? this.consentGranted,
    );
  }
}

class AnalyticsController extends AsyncNotifier<AnalyticsState> {
  bool _consentGranted = false;

  FirebaseAnalytics get _analytics => FirebaseAnalytics.instance;

  PrivacySettingsRepository get _privacyRepository =>
      ref.read(privacySettingsRepositoryProvider);

  AppFlavor get _flavor => ref.read(appFlavorProvider);

  @override
  Future<AnalyticsState> build() async {
    await ref.read(firebaseInitializedProvider.future);

    final allowed = await _privacyRepository.isAnalyticsAllowed();
    await _applyConsent(allowed);
    _consentGranted = allowed;

    return AnalyticsState(consentGranted: allowed);
  }

  Future<void> updateConsent(bool allowCollection) async {
    final current = await future;
    try {
      await _privacyRepository.setAnalyticsAllowed(allowCollection);
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

  Future<void> logEvent(AnalyticsEvent event) async {
    if (!_consentGranted) {
      return;
    }
    event.validate();
    await _analytics.logEvent(
      name: event.name,
      parameters: event.toParameters(),
    );
  }

  Future<void> logScreenView(ScreenViewAnalyticsEvent event) async {
    if (!_consentGranted) {
      return;
    }
    event.validate();
    await _analytics.logScreenView(
      screenName: event.screenName,
      screenClass: event.screenClass,
    );
  }

  Future<void> setUserId(String? userId) async {
    if (!_consentGranted) {
      return;
    }
    await _analytics.setUserId(id: userId);
  }

  Future<void> _applyConsent(bool allowed) async {
    await _analytics.setAnalyticsCollectionEnabled(allowed);
    await _analytics.setUserProperty(name: 'app_env', value: _flavor.name);
  }
}

final analyticsControllerProvider =
    AsyncNotifierProvider<AnalyticsController, AnalyticsState>(
      AnalyticsController.new,
    );
