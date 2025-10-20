// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for English (`en`).
class AppLocalizationsEn extends AppLocalizations {
  AppLocalizationsEn([String locale = 'en']) : super(locale);

  @override
  String get appTitle => 'Hanko Field';

  @override
  String get counterScreenTitle => 'Sample Counter';

  @override
  String get increment => 'Increment';

  @override
  String countLabel(int value) {
    return 'Count: $value';
  }
}
