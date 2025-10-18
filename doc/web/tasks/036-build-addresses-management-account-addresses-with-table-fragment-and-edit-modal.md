# Build addresses management (`/account/addresses`) with table fragment and edit modal.

**Parent Section:** 6. Account & Library
**Task ID:** 036

## Goal
Manage user address book.

## Implementation Steps
1. Display table fragment with saved addresses, default tags, actions.
2. Provide modal for add/edit calling `/me/addresses` API.
3. Update list via htmx on success.

## UI Components
- **Layout:** `AccountLayout` with `AccountNav` and addresses `SectionHeader`.
- **Address table:** `AddressTable` fragment listing label, address, default `Badge`, actions menu.
- **Add/Edit modal:** `AddressModal` containing form fields and validation summary.
- **Sync banner:** `InlineAlert` for shipping sync status.
- **Empty state:** `EmptyState` card prompting add address.
- **Action button:** `PrimaryButton` to add new address pinned near header.
