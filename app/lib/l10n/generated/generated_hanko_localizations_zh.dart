// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'generated_hanko_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Chinese (`zh`).
class GeneratedHankoLocalizationsZh extends GeneratedHankoLocalizations {
  GeneratedHankoLocalizationsZh([String locale = 'zh']) : super(locale);

  @override
  String get appTitle => 'STONE SIGNATURE';

  @override
  String get designKanjiStyleChinese => '中国风格';

  @override
  String get designKanjiStyleTaiwanese => '台湾风格';
}

/// The translations for Chinese, using the Han script (`zh_Hant`).
class GeneratedHankoLocalizationsZhHant extends GeneratedHankoLocalizationsZh {
  GeneratedHankoLocalizationsZhHant() : super('zh_Hant');

  @override
  String get appTitle => 'STONE SIGNATURE';

  @override
  String get designKanjiStyleChinese => '中國風格';

  @override
  String get designKanjiStyleTaiwanese => '台灣風格';
}
