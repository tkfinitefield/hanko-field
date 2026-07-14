import 'package:flutter/widgets.dart';

import '../../l10n/generated/generated_hanko_localizations.dart';

extension GeneratedHankoLocalizationsBuildContext on BuildContext {
  GeneratedHankoLocalizations get l10n => GeneratedHankoLocalizations.of(this);
}

extension GeneratedHankoLocalizationsLocale on GeneratedHankoLocalizations {
  Locale get locale {
    final parts = localeName.split('_');
    return Locale.fromSubtags(
      languageCode: parts.isNotEmpty ? parts[0] : 'en',
      scriptCode: parts.length > 1 && parts[1].isNotEmpty ? parts[1] : null,
      countryCode: parts.length > 2 && parts[2].isNotEmpty ? parts[2] : null,
    );
  }
}
