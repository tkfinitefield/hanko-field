import 'package:app/core/app/app_flavor.dart';
import 'package:app/core/firebase/app_firebase.dart';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:firebase_remote_config/firebase_remote_config.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final firebaseAuthProvider = Provider<FirebaseAuth>((_) => AppFirebase.auth);
final firebaseMessagingProvider = Provider<FirebaseMessaging>(
  (_) => AppFirebase.messaging,
);
final firebaseRemoteConfigProvider = Provider<FirebaseRemoteConfig>(
  (_) => AppFirebase.remoteConfig,
);

final firebaseInitializedProvider = FutureProvider<void>((ref) async {
  final flavor = ref.read(appFlavorProvider);
  await AppFirebase.initialize(flavor);
});

final fcmTokenProvider = FutureProvider<String?>((ref) async {
  // Ensure Firebase is initialized first.
  await ref.watch(firebaseInitializedProvider.future);
  return ref.read(firebaseMessagingProvider).getToken();
});

final welcomeTitleProvider = Provider<String>((ref) {
  // After initialization, read remote config value (defaults defined in init).
  final rc = ref.read(firebaseRemoteConfigProvider);
  return rc.getString('welcome_title');
});
