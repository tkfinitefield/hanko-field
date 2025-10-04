# Integrate Firebase services (Auth, Messaging, Remote Config) and configure environment-specific options.

**Parent Section:** 1. Project Setup & Tooling
**Task ID:** 009

## Goal
Integrate Firebase for auth, messaging, remote config across flavors.

## Implementation Steps
1. Run `flutterfire configure` per flavor and commit generated `firebase_options.dart` (exclude secrets).
2. Enable sign-in providers (Apple/Google/Email) and configure redirect URIs.
3. Set up Firebase Messaging for push notifications and background handling.
4. Initialize Remote Config with defaults and fetch/activate lifecycle.
5. Provide environment setup guide for developers.
