# Implement user profile service mapping Firebase Auth users to Firestore documents.

**Parent Section:** 3. Shared Domain Services
**Task ID:** 017

## Goal
Maintain a projection of Firebase Auth users inside Firestore with profile metadata required by `/me` endpoints and administrative views.

## Responsibilities
- Sync Firebase user records (display name, email, phone, provider data) into `users` collection.
- Manage profile updates (displayName, language, notification settings) with validation and audit logging.
- Expose methods for address management service to link addresses, and for admin deactivation/masking flows.

## Data Model
- Collection `users` with fields: `uid`, `displayName`, `email`, `phoneNumber`, `photoURL`, `locale`, `isActive`, `roles[]`, `createdAt`, `updatedAt`, `piiMaskedAt`.
- Sub-collections: `addresses`, `paymentMethods` (token references).

## Steps
1. Implement `UserRepository` for Firestore CRUD with optimistic locking (updateTime precondition).
2. Implement `UserService` with methods `GetByUID`, `UpdateProfile`, `ListAddresses`, etc., enforcing editable fields.
3. Integrate with Firebase Admin SDK to fetch canonical email/phone when needed.
4. Emit audit logs on profile changes via audit service.
5. Write unit tests covering update validation, masking logic, and error handling.
