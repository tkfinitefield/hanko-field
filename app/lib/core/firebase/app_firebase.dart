import 'package:firebase_auth/firebase_auth.dart';
import 'package:firebase_core/firebase_core.dart';
import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:firebase_remote_config/firebase_remote_config.dart';
 

import 'package:app/core/app/app_flavor.dart';
import 'package:app/firebase_options.dart';

@pragma('vm:entry-point')
Future<void> firebaseMessagingBackgroundHandler(RemoteMessage message) async {
  // Ensure Firebase is initialized in background isolate (Android).
  // Values here are placeholders; a proper init requires flavor context which is
  // not available in background. For background tasks triggered after the main
  // isolate initialized, Firebase should already be ready.
}

class AppFirebase {
  static bool _initialized = false;

  static Future<void> initialize(AppFlavor flavor) async {
    if (_initialized) return;

    final options = DefaultFirebaseOptions.byFlavorAndPlatform(flavor);
    await Firebase.initializeApp(options: options);

    // Messaging background handler
    FirebaseMessaging.onBackgroundMessage(firebaseMessagingBackgroundHandler);

    // iOS/macOS notification permission
    try {
      await FirebaseMessaging.instance.requestPermission();
    } catch (_) {}

    // Remote Config defaults and config settings (fast dev, slower prod)
    final rc = FirebaseRemoteConfig.instance;
    final minInterval = switch (flavor) {
      AppFlavor.dev => const Duration(seconds: 0),
      AppFlavor.stg => const Duration(seconds: 30),
      AppFlavor.prod => const Duration(hours: 12),
    };
    await rc.setConfigSettings(RemoteConfigSettings(
      fetchTimeout: const Duration(seconds: 10),
      minimumFetchInterval: minInterval,
    ));
    await rc.setDefaults(const {
      'feature_sample_counter': true,
      'welcome_title': 'Hanko Field',
    });

    _initialized = true;
  }

  static FirebaseAuth get auth => FirebaseAuth.instance;
  static FirebaseMessaging get messaging => FirebaseMessaging.instance;
  static FirebaseRemoteConfig get remoteConfig => FirebaseRemoteConfig.instance;
}
