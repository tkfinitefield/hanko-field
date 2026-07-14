import 'package:flutter/material.dart';
import 'package:miniriverpod/miniriverpod.dart';

import 'app/app.dart';
import 'app/localization/language_registry.dart';
import 'features/my_seals/my_seals.dart';
import 'features/order/order.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  var supportedLocales = HankoApp.defaultSupportedLocales;
  var automaticLocales = HankoApp.defaultSupportedLocales;
  try {
    final registry = await AppLanguageRegistry.load();
    supportedLocales = registry.enabledLocales;
    automaticLocales = registry.selectableLocales;
  } catch (error) {
    debugPrint('failed to load app language registry: $error');
  }
  runApp(
    ProviderScope(
      child: HankoApp(
        supportedLocales: supportedLocales,
        automaticLocales: automaticLocales,
        localSealDesignRepository: SqfliteLocalSealDesignRepository(),
        localOrderDraftRepository: SqfliteLocalOrderDraftRepository(),
      ),
    ),
  );
}
