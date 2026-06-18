import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:intl/intl.dart' as intl;

import 'generated_hanko_localizations_en.dart';
import 'generated_hanko_localizations_ja.dart';
import 'generated_hanko_localizations_zh.dart';

// ignore_for_file: type=lint

/// Callers can lookup localized strings with an instance of GeneratedHankoLocalizations
/// returned by `GeneratedHankoLocalizations.of(context)`.
///
/// Applications need to include `GeneratedHankoLocalizations.delegate()` in their app's
/// `localizationDelegates` list, and the locales they support in the app's
/// `supportedLocales` list. For example:
///
/// ```dart
/// import 'generated/generated_hanko_localizations.dart';
///
/// return MaterialApp(
///   localizationsDelegates: GeneratedHankoLocalizations.localizationsDelegates,
///   supportedLocales: GeneratedHankoLocalizations.supportedLocales,
///   home: MyApplicationHome(),
/// );
/// ```
///
/// ## Update pubspec.yaml
///
/// Please make sure to update your pubspec.yaml to include the following
/// packages:
///
/// ```yaml
/// dependencies:
///   # Internationalization support.
///   flutter_localizations:
///     sdk: flutter
///   intl: any # Use the pinned version from flutter_localizations
///
///   # Rest of dependencies
/// ```
///
/// ## iOS Applications
///
/// iOS applications define key application metadata, including supported
/// locales, in an Info.plist file that is built into the application bundle.
/// To configure the locales supported by your app, you’ll need to edit this
/// file.
///
/// First, open your project’s ios/Runner.xcworkspace Xcode workspace file.
/// Then, in the Project Navigator, open the Info.plist file under the Runner
/// project’s Runner folder.
///
/// Next, select the Information Property List item, select Add Item from the
/// Editor menu, then select Localizations from the pop-up menu.
///
/// Select and expand the newly-created Localizations item then, for each
/// locale your application supports, add a new item and select the locale
/// you wish to add from the pop-up menu in the Value field. This list should
/// be consistent with the languages listed in the GeneratedHankoLocalizations.supportedLocales
/// property.
abstract class GeneratedHankoLocalizations {
  GeneratedHankoLocalizations(String locale)
    : localeName = intl.Intl.canonicalizedLocale(locale.toString());

  final String localeName;

  static GeneratedHankoLocalizations of(BuildContext context) {
    return Localizations.of<GeneratedHankoLocalizations>(
      context,
      GeneratedHankoLocalizations,
    )!;
  }

  static const LocalizationsDelegate<GeneratedHankoLocalizations> delegate =
      _GeneratedHankoLocalizationsDelegate();

  /// A list of this localizations delegate along with the default localizations
  /// delegates.
  ///
  /// Returns a list of localizations delegates containing this delegate along with
  /// GlobalMaterialLocalizations.delegate, GlobalCupertinoLocalizations.delegate,
  /// and GlobalWidgetsLocalizations.delegate.
  ///
  /// Additional delegates can be added by appending to this list in
  /// MaterialApp. This list does not have to be used at all if a custom list
  /// of delegates is preferred or required.
  static const List<LocalizationsDelegate<dynamic>> localizationsDelegates =
      <LocalizationsDelegate<dynamic>>[
        delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
      ];

  /// A list of this localizations delegate's supported locales.
  static const List<Locale> supportedLocales = <Locale>[
    Locale('en'),
    Locale('ja'),
    Locale('zh'),
    Locale.fromSubtags(languageCode: 'zh', scriptCode: 'Hant'),
  ];

  /// Application title shown by MaterialApp. The brand name intentionally remains in English.
  ///
  /// In en, this message translates to:
  /// **'STONE SIGNATURE'**
  String get appTitle;

  /// Localized app string for splashPreparing.
  ///
  /// In en, this message translates to:
  /// **'Preparing your design experience.'**
  String get splashPreparing;

  /// Localized app string for onboardingTitle.
  ///
  /// In en, this message translates to:
  /// **'Welcome'**
  String get onboardingTitle;

  /// Localized app string for onboardingHeroTitle.
  ///
  /// In en, this message translates to:
  /// **'Create your\nseal in minutes'**
  String get onboardingHeroTitle;

  /// Localized app string for onboardingTagline.
  ///
  /// In en, this message translates to:
  /// **'Personalized. Timeless.\nUniquely yours.'**
  String get onboardingTagline;

  /// Localized app string for onboardingKanjiTitle.
  ///
  /// In en, this message translates to:
  /// **'Choose kanji from your name'**
  String get onboardingKanjiTitle;

  /// Localized app string for onboardingKanjiMessage.
  ///
  /// In en, this message translates to:
  /// **'We suggest meaningful kanji based on your name.'**
  String get onboardingKanjiMessage;

  /// Localized app string for onboardingAiTitle.
  ///
  /// In en, this message translates to:
  /// **'Generate a seal design with AI'**
  String get onboardingAiTitle;

  /// Localized app string for onboardingAiMessage.
  ///
  /// In en, this message translates to:
  /// **'Our AI creates beautiful, balanced seal designs just for you.'**
  String get onboardingAiMessage;

  /// Localized app string for onboardingStoneTitle.
  ///
  /// In en, this message translates to:
  /// **'Select a gemstone and order'**
  String get onboardingStoneTitle;

  /// Localized app string for onboardingStoneMessage.
  ///
  /// In en, this message translates to:
  /// **'Pick your favorite gemstone and we\'ll craft your seal with care.'**
  String get onboardingStoneMessage;

  /// Localized app string for onboardingStorageTitle.
  ///
  /// In en, this message translates to:
  /// **'Saved on this device'**
  String get onboardingStorageTitle;

  /// Localized app string for onboardingStorageMessage.
  ///
  /// In en, this message translates to:
  /// **'Saved seal designs and preview images stay on this device. Payment details and checkout secrets are never saved locally.'**
  String get onboardingStorageMessage;

  /// Localized app string for onboardingGetStarted.
  ///
  /// In en, this message translates to:
  /// **'Get Started'**
  String get onboardingGetStarted;

  /// Localized app string for onboardingSaving.
  ///
  /// In en, this message translates to:
  /// **'Saving...'**
  String get onboardingSaving;

  /// Localized app string for onboardingSkip.
  ///
  /// In en, this message translates to:
  /// **'Skip'**
  String get onboardingSkip;

  /// Localized app string for onboardingSaveError.
  ///
  /// In en, this message translates to:
  /// **'Could not save onboarding status. Please try again.'**
  String get onboardingSaveError;

  /// Localized app string for design.
  ///
  /// In en, this message translates to:
  /// **'Design'**
  String get design;

  /// Localized app string for mySeals.
  ///
  /// In en, this message translates to:
  /// **'My Seals'**
  String get mySeals;

  /// Localized app string for stones.
  ///
  /// In en, this message translates to:
  /// **'Stones'**
  String get stones;

  /// Localized app string for settings.
  ///
  /// In en, this message translates to:
  /// **'Settings'**
  String get settings;

  /// Localized app string for createCustomSeal.
  ///
  /// In en, this message translates to:
  /// **'Create your\ncustom seal'**
  String get createCustomSeal;

  /// Localized app string for customSealDescription.
  ///
  /// In en, this message translates to:
  /// **'Turn your name into a\npersonalized gemstone seal.'**
  String get customSealDescription;

  /// Localized app string for startDesigning.
  ///
  /// In en, this message translates to:
  /// **'Start Designing'**
  String get startDesigning;

  /// Localized app string for designNameTitle.
  ///
  /// In en, this message translates to:
  /// **'Enter Your Name'**
  String get designNameTitle;

  /// Localized app string for designNameIntro.
  ///
  /// In en, this message translates to:
  /// **'We\'ll suggest kanji based on your preferences.'**
  String get designNameIntro;

  /// Localized app string for designNameLabel.
  ///
  /// In en, this message translates to:
  /// **'Your name'**
  String get designNameLabel;

  /// Localized app string for designNameHint.
  ///
  /// In en, this message translates to:
  /// **'Michael Smith'**
  String get designNameHint;

  /// Localized app string for designNameHelp.
  ///
  /// In en, this message translates to:
  /// **'1-2 kanji will be suggested for a small personal seal.'**
  String get designNameHelp;

  /// Localized app string for designGenderLabel.
  ///
  /// In en, this message translates to:
  /// **'Gender preference'**
  String get designGenderLabel;

  /// Localized app string for designGenderUnspecified.
  ///
  /// In en, this message translates to:
  /// **'No preference'**
  String get designGenderUnspecified;

  /// Localized app string for designGenderMale.
  ///
  /// In en, this message translates to:
  /// **'Masculine'**
  String get designGenderMale;

  /// Localized app string for designGenderFemale.
  ///
  /// In en, this message translates to:
  /// **'Feminine'**
  String get designGenderFemale;

  /// Localized app string for designKanjiStyleLabel.
  ///
  /// In en, this message translates to:
  /// **'Kanji style'**
  String get designKanjiStyleLabel;

  /// Localized app string for designKanjiStyleJapanese.
  ///
  /// In en, this message translates to:
  /// **'Japanese style'**
  String get designKanjiStyleJapanese;

  /// Localized app string for designKanjiStyleChinese.
  ///
  /// In en, this message translates to:
  /// **'Chinese style'**
  String get designKanjiStyleChinese;

  /// Localized app string for designKanjiStyleTaiwanese.
  ///
  /// In en, this message translates to:
  /// **'Taiwanese style'**
  String get designKanjiStyleTaiwanese;

  /// Localized app string for suggestKanji.
  ///
  /// In en, this message translates to:
  /// **'Suggest Kanji'**
  String get suggestKanji;

  /// Localized app string for designKanjiTipTitle.
  ///
  /// In en, this message translates to:
  /// **'Simple kanji work best for gemstone engraving.'**
  String get designKanjiTipTitle;

  /// Localized app string for designKanjiTipMessage.
  ///
  /// In en, this message translates to:
  /// **'Clear, balanced characters create the most beautiful and timeless results.'**
  String get designKanjiTipMessage;

  /// Localized app string for designCandidateReadyTitle.
  ///
  /// In en, this message translates to:
  /// **'Ready to Suggest Kanji'**
  String get designCandidateReadyTitle;

  /// Localized app string for designCandidateReadyMessage.
  ///
  /// In en, this message translates to:
  /// **'Candidate generation can now use this name and preference set.'**
  String get designCandidateReadyMessage;

  /// Localized app string for designRequestDetails.
  ///
  /// In en, this message translates to:
  /// **'Request details'**
  String get designRequestDetails;

  /// Localized app string for editName.
  ///
  /// In en, this message translates to:
  /// **'Edit Name'**
  String get editName;

  /// Localized app string for designLoadingTitle.
  ///
  /// In en, this message translates to:
  /// **'Finding Kanji'**
  String get designLoadingTitle;

  /// Localized app string for designLoadingMessage.
  ///
  /// In en, this message translates to:
  /// **'Creating kanji suggestions...'**
  String get designLoadingMessage;

  /// Localized app string for designInvalidNameSummary.
  ///
  /// In en, this message translates to:
  /// **'Enter your name to continue.'**
  String get designInvalidNameSummary;

  /// Localized app string for designInvalidNameMessage.
  ///
  /// In en, this message translates to:
  /// **'Please enter a valid first name or short name.'**
  String get designInvalidNameMessage;

  /// Localized app string for designSuggestionErrorTitle.
  ///
  /// In en, this message translates to:
  /// **'We couldn\'t suggest kanji'**
  String get designSuggestionErrorTitle;

  /// Localized app string for designSuggestionErrorMessage.
  ///
  /// In en, this message translates to:
  /// **'Something went wrong while generating kanji suggestions for your name. Please try again.'**
  String get designSuggestionErrorMessage;

  /// Localized app string for designNoKanjiTitle.
  ///
  /// In en, this message translates to:
  /// **'We couldn\'t find a suitable kanji'**
  String get designNoKanjiTitle;

  /// Localized app string for designNoKanjiMessage.
  ///
  /// In en, this message translates to:
  /// **'Your name did not return any kanji suggestions that fit our engraving rules.'**
  String get designNoKanjiMessage;

  /// Localized app string for designNoKanjiRuleCharacters.
  ///
  /// In en, this message translates to:
  /// **'1-2 characters only'**
  String get designNoKanjiRuleCharacters;

  /// Localized app string for designNoKanjiRuleCommon.
  ///
  /// In en, this message translates to:
  /// **'Simple, common kanji'**
  String get designNoKanjiRuleCommon;

  /// Localized app string for designNoKanjiRuleEngraving.
  ///
  /// In en, this message translates to:
  /// **'Suitable for seal engraving'**
  String get designNoKanjiRuleEngraving;

  /// Localized app string for designErrorTip.
  ///
  /// In en, this message translates to:
  /// **'Use a simple first name or short name.'**
  String get designErrorTip;

  /// Localized app string for designNoKanjiTip.
  ///
  /// In en, this message translates to:
  /// **'Try a shorter name or nickname.'**
  String get designNoKanjiTip;

  /// Localized app string for tryAgain.
  ///
  /// In en, this message translates to:
  /// **'Try Again'**
  String get tryAgain;

  /// Localized app string for back.
  ///
  /// In en, this message translates to:
  /// **'Back'**
  String get back;

  /// Localized app string for commonNetworkErrorTitle.
  ///
  /// In en, this message translates to:
  /// **'Network Error'**
  String get commonNetworkErrorTitle;

  /// Localized app string for commonNetworkErrorMessage.
  ///
  /// In en, this message translates to:
  /// **'We\'re unable to connect to the server. Please check your internet connection and try again.'**
  String get commonNetworkErrorMessage;

  /// Localized app string for commonServerErrorTitle.
  ///
  /// In en, this message translates to:
  /// **'Server Error'**
  String get commonServerErrorTitle;

  /// Localized app string for commonServerErrorMessage.
  ///
  /// In en, this message translates to:
  /// **'We\'re experiencing a temporary issue on our end. Please wait a moment and try again.'**
  String get commonServerErrorMessage;

  /// Localized app string for storageErrorTitle.
  ///
  /// In en, this message translates to:
  /// **'Couldn\'t Save Seal'**
  String get storageErrorTitle;

  /// Localized app string for storageErrorMessage.
  ///
  /// In en, this message translates to:
  /// **'The seal image couldn\'t be saved on this device. Check storage permissions and available space, then try again.'**
  String get storageErrorMessage;

  /// Localized app string for deepLinkErrorTitle.
  ///
  /// In en, this message translates to:
  /// **'Checkout Return Link Error'**
  String get deepLinkErrorTitle;

  /// Localized app string for deepLinkErrorMessage.
  ///
  /// In en, this message translates to:
  /// **'The Stripe Checkout return link couldn\'t be processed. Please open Checkout again or contact support if payment may have completed.'**
  String get deepLinkErrorMessage;

  /// Localized app string for maintenanceTitle.
  ///
  /// In en, this message translates to:
  /// **'Temporarily Unavailable'**
  String get maintenanceTitle;

  /// Localized app string for maintenanceMessage.
  ///
  /// In en, this message translates to:
  /// **'Stone Signature is currently undergoing maintenance. Please check back in a little while.'**
  String get maintenanceMessage;

  /// Localized app string for appUpdateRequiredTitle.
  ///
  /// In en, this message translates to:
  /// **'Update Required'**
  String get appUpdateRequiredTitle;

  /// Localized app string for appUpdateRequiredMessage.
  ///
  /// In en, this message translates to:
  /// **'A newer app version is required to continue. Please update the app, then open Stone Signature again.'**
  String get appUpdateRequiredMessage;

  /// Localized app string for appUpdateRequiredAction.
  ///
  /// In en, this message translates to:
  /// **'Update App'**
  String get appUpdateRequiredAction;

  /// Localized app string for commonGenericErrorTitle.
  ///
  /// In en, this message translates to:
  /// **'Something Went Wrong'**
  String get commonGenericErrorTitle;

  /// Localized app string for commonGenericErrorMessage.
  ///
  /// In en, this message translates to:
  /// **'An unexpected error occurred. Please try again in a few moments.'**
  String get commonGenericErrorMessage;

  /// Localized app string for kanjiSuggestionsTitle.
  ///
  /// In en, this message translates to:
  /// **'Kanji Suggestions'**
  String get kanjiSuggestionsTitle;

  /// Localized app string for kanjiSuggestionsMessage.
  ///
  /// In en, this message translates to:
  /// **'Choose the kanji that best fits your seal.'**
  String get kanjiSuggestionsMessage;

  /// Localized app string for kanjiCandidateDetailTitle.
  ///
  /// In en, this message translates to:
  /// **'Kanji Detail'**
  String get kanjiCandidateDetailTitle;

  /// Localized app string for kanjiReadingLabel.
  ///
  /// In en, this message translates to:
  /// **'Reading'**
  String get kanjiReadingLabel;

  /// Localized app string for kanjiMeaningLabel.
  ///
  /// In en, this message translates to:
  /// **'Meaning'**
  String get kanjiMeaningLabel;

  /// Localized app string for kanjiImpressionLabel.
  ///
  /// In en, this message translates to:
  /// **'Impression'**
  String get kanjiImpressionLabel;

  /// Localized app string for kanjiReasonLabel.
  ///
  /// In en, this message translates to:
  /// **'Reason'**
  String get kanjiReasonLabel;

  /// Localized app string for kanjiCharacterCountLabel.
  ///
  /// In en, this message translates to:
  /// **'Characters'**
  String get kanjiCharacterCountLabel;

  /// Localized app string for kanjiStrokeComplexityLabel.
  ///
  /// In en, this message translates to:
  /// **'Stroke complexity'**
  String get kanjiStrokeComplexityLabel;

  /// Localized app string for kanjiEngravingSuitabilityLabel.
  ///
  /// In en, this message translates to:
  /// **'Engraving suitability'**
  String get kanjiEngravingSuitabilityLabel;

  /// Localized app string for selectKanji.
  ///
  /// In en, this message translates to:
  /// **'Select Kanji'**
  String get selectKanji;

  /// Localized app string for kanjiSelectedTitle.
  ///
  /// In en, this message translates to:
  /// **'Kanji selected'**
  String get kanjiSelectedTitle;

  /// Localized app string for kanjiSelectedMessage.
  ///
  /// In en, this message translates to:
  /// **'This kanji is ready for seal style selection.'**
  String get kanjiSelectedMessage;

  /// Localized app string for sealStyleTitle.
  ///
  /// In en, this message translates to:
  /// **'Seal Style'**
  String get sealStyleTitle;

  /// Localized app string for sealStyleMessage.
  ///
  /// In en, this message translates to:
  /// **'Customize your seal style.'**
  String get sealStyleMessage;

  /// Localized app string for sealStyleSelectedKanjiLabel.
  ///
  /// In en, this message translates to:
  /// **'Selected kanji'**
  String get sealStyleSelectedKanjiLabel;

  /// Localized app string for sealShapeLabel.
  ///
  /// In en, this message translates to:
  /// **'Shape'**
  String get sealShapeLabel;

  /// Localized app string for sealShapeSquare.
  ///
  /// In en, this message translates to:
  /// **'Square'**
  String get sealShapeSquare;

  /// Localized app string for sealShapeRound.
  ///
  /// In en, this message translates to:
  /// **'Round'**
  String get sealShapeRound;

  /// Localized app string for sealStyleNameLabel.
  ///
  /// In en, this message translates to:
  /// **'Style'**
  String get sealStyleNameLabel;

  /// Localized app string for sealStyleTraditional.
  ///
  /// In en, this message translates to:
  /// **'Traditional'**
  String get sealStyleTraditional;

  /// Localized app string for sealStyleElegant.
  ///
  /// In en, this message translates to:
  /// **'Elegant'**
  String get sealStyleElegant;

  /// Localized app string for sealStyleSoft.
  ///
  /// In en, this message translates to:
  /// **'Soft'**
  String get sealStyleSoft;

  /// Localized app string for sealStyleBold.
  ///
  /// In en, this message translates to:
  /// **'Bold'**
  String get sealStyleBold;

  /// Localized app string for sealStrokeWeightLabel.
  ///
  /// In en, this message translates to:
  /// **'Stroke Weight'**
  String get sealStrokeWeightLabel;

  /// Localized app string for sealStrokeStandard.
  ///
  /// In en, this message translates to:
  /// **'Standard'**
  String get sealStrokeStandard;

  /// Localized app string for sealStrokeBold.
  ///
  /// In en, this message translates to:
  /// **'Bold'**
  String get sealStrokeBold;

  /// Localized app string for sealBalanceLabel.
  ///
  /// In en, this message translates to:
  /// **'Balance'**
  String get sealBalanceLabel;

  /// Localized app string for sealBalanceAiry.
  ///
  /// In en, this message translates to:
  /// **'Airy'**
  String get sealBalanceAiry;

  /// Localized app string for sealBalanceBalanced.
  ///
  /// In en, this message translates to:
  /// **'Balanced'**
  String get sealBalanceBalanced;

  /// Localized app string for sealBalanceDense.
  ///
  /// In en, this message translates to:
  /// **'Dense'**
  String get sealBalanceDense;

  /// Localized app string for sealStyleSummaryTitle.
  ///
  /// In en, this message translates to:
  /// **'Current style'**
  String get sealStyleSummaryTitle;

  /// Localized app string for confirmStyle.
  ///
  /// In en, this message translates to:
  /// **'Confirm Style'**
  String get confirmStyle;

  /// Localized app string for sealStyleConfirmedTitle.
  ///
  /// In en, this message translates to:
  /// **'Style selected'**
  String get sealStyleConfirmedTitle;

  /// Localized app string for sealStyleConfirmedMessage.
  ///
  /// In en, this message translates to:
  /// **'These style choices are ready for AI seal generation.'**
  String get sealStyleConfirmedMessage;

  /// Localized app string for generateSeal.
  ///
  /// In en, this message translates to:
  /// **'Generate Seal'**
  String get generateSeal;

  /// Localized app string for sealGenerationLoadingTitle.
  ///
  /// In en, this message translates to:
  /// **'Generating Seal'**
  String get sealGenerationLoadingTitle;

  /// Localized app string for sealGenerationLoadingMessage.
  ///
  /// In en, this message translates to:
  /// **'Creating three AI seal design directions...'**
  String get sealGenerationLoadingMessage;

  /// Localized app string for sealGenerationLoadingDetail.
  ///
  /// In en, this message translates to:
  /// **'We are checking the kanji and style before saving previews.'**
  String get sealGenerationLoadingDetail;

  /// Localized app string for sealGenerationErrorTitle.
  ///
  /// In en, this message translates to:
  /// **'We couldn\'t generate seal designs'**
  String get sealGenerationErrorTitle;

  /// Localized app string for sealGenerationErrorMessage.
  ///
  /// In en, this message translates to:
  /// **'Something went wrong while creating AI seal previews. Please try again.'**
  String get sealGenerationErrorMessage;

  /// Localized app string for sealGenerationLimitTitle.
  ///
  /// In en, this message translates to:
  /// **'Generation limit reached'**
  String get sealGenerationLimitTitle;

  /// Localized app string for sealGenerationLimitMessage.
  ///
  /// In en, this message translates to:
  /// **'You have used all generation attempts for this style set. Adjust the style before trying again.'**
  String get sealGenerationLimitMessage;

  /// Localized app string for sealGenerationAttemptLabel.
  ///
  /// In en, this message translates to:
  /// **'Attempts'**
  String get sealGenerationAttemptLabel;

  /// Localized app string for sealGenerationStyleDetails.
  ///
  /// In en, this message translates to:
  /// **'Generation details'**
  String get sealGenerationStyleDetails;

  /// Localized app string for sealGenerationErrorTip.
  ///
  /// In en, this message translates to:
  /// **'Try again once. If it still fails, adjust the style or choose a simpler kanji.'**
  String get sealGenerationErrorTip;

  /// Localized app string for sealGenerationLimitTip.
  ///
  /// In en, this message translates to:
  /// **'Choose a different balance, stroke weight, or kanji to start a fresh generation.'**
  String get sealGenerationLimitTip;

  /// Localized app string for adjustStyle.
  ///
  /// In en, this message translates to:
  /// **'Adjust Style'**
  String get adjustStyle;

  /// Localized app string for sealVariantSelectionTitle.
  ///
  /// In en, this message translates to:
  /// **'Seal Options'**
  String get sealVariantSelectionTitle;

  /// Localized app string for sealVariantSelectionMessage.
  ///
  /// In en, this message translates to:
  /// **'Choose one AI seal design.'**
  String get sealVariantSelectionMessage;

  /// Localized app string for sealVariantSelectedBadge.
  ///
  /// In en, this message translates to:
  /// **'Selected'**
  String get sealVariantSelectedBadge;

  /// Localized app string for sealVariantSelectedTitle.
  ///
  /// In en, this message translates to:
  /// **'Seal design selected'**
  String get sealVariantSelectedTitle;

  /// Localized app string for sealVariantSelectedMessage.
  ///
  /// In en, this message translates to:
  /// **'This AI seal design is ready for preview and saving.'**
  String get sealVariantSelectedMessage;

  /// Localized app string for regenerateSeal.
  ///
  /// In en, this message translates to:
  /// **'Regenerate Seal'**
  String get regenerateSeal;

  /// Localized app string for sealPreviewTitle.
  ///
  /// In en, this message translates to:
  /// **'Seal Preview'**
  String get sealPreviewTitle;

  /// Localized app string for sealPreviewMessage.
  ///
  /// In en, this message translates to:
  /// **'Review your selected seal design before saving.'**
  String get sealPreviewMessage;

  /// Localized app string for sealPreviewVariantLabel.
  ///
  /// In en, this message translates to:
  /// **'AI Variant'**
  String get sealPreviewVariantLabel;

  /// Localized app string for saveSeal.
  ///
  /// In en, this message translates to:
  /// **'Save Seal'**
  String get saveSeal;

  /// Localized app string for chooseStone.
  ///
  /// In en, this message translates to:
  /// **'Choose a Stone'**
  String get chooseStone;

  /// Localized app string for sealSavedTitle.
  ///
  /// In en, this message translates to:
  /// **'Seal Saved'**
  String get sealSavedTitle;

  /// Localized app string for sealSavedHeading.
  ///
  /// In en, this message translates to:
  /// **'Seal saved to My Seals'**
  String get sealSavedHeading;

  /// Localized app string for sealSavedMessage.
  ///
  /// In en, this message translates to:
  /// **'Your custom seal design is ready for comparison and ordering.'**
  String get sealSavedMessage;

  /// Localized app string for goToMySeals.
  ///
  /// In en, this message translates to:
  /// **'Go to My Seals'**
  String get goToMySeals;

  /// Localized app string for createAnotherSeal.
  ///
  /// In en, this message translates to:
  /// **'Create Another Seal'**
  String get createAnotherSeal;

  /// Localized app string for savedSeals.
  ///
  /// In en, this message translates to:
  /// **'Saved Seals'**
  String get savedSeals;

  /// Localized app string for savedSealsDescription.
  ///
  /// In en, this message translates to:
  /// **'View and manage your\nsaved seal designs.'**
  String get savedSealsDescription;

  /// Localized app string for savedOnThisDevice.
  ///
  /// In en, this message translates to:
  /// **'Saved on this device'**
  String get savedOnThisDevice;

  /// Localized app string for savedSealsLoadingTitle.
  ///
  /// In en, this message translates to:
  /// **'Loading saved seals'**
  String get savedSealsLoadingTitle;

  /// Localized app string for savedSealsLoadingMessage.
  ///
  /// In en, this message translates to:
  /// **'Checking seal designs saved on this device.'**
  String get savedSealsLoadingMessage;

  /// Localized app string for savedSealsLoadErrorTitle.
  ///
  /// In en, this message translates to:
  /// **'Couldn\'t load saved seals'**
  String get savedSealsLoadErrorTitle;

  /// Localized app string for savedSealsLoadErrorMessage.
  ///
  /// In en, this message translates to:
  /// **'Open My Seals again, or create a new seal design.'**
  String get savedSealsLoadErrorMessage;

  /// Localized app string for chooseSavedSeal.
  ///
  /// In en, this message translates to:
  /// **'Choose'**
  String get chooseSavedSeal;

  /// Localized app string for viewSealDetails.
  ///
  /// In en, this message translates to:
  /// **'View Details'**
  String get viewSealDetails;

  /// Localized app string for favoriteSeal.
  ///
  /// In en, this message translates to:
  /// **'Favorite seal'**
  String get favoriteSeal;

  /// Localized app string for removeFavoriteSeal.
  ///
  /// In en, this message translates to:
  /// **'Remove favorite'**
  String get removeFavoriteSeal;

  /// Localized app string for sealDetailTitle.
  ///
  /// In en, this message translates to:
  /// **'Seal Detail'**
  String get sealDetailTitle;

  /// Localized app string for kanjiLabel.
  ///
  /// In en, this message translates to:
  /// **'Kanji'**
  String get kanjiLabel;

  /// Localized app string for createdAtLabel.
  ///
  /// In en, this message translates to:
  /// **'Created'**
  String get createdAtLabel;

  /// Localized app string for compareSavedSeals.
  ///
  /// In en, this message translates to:
  /// **'Compare Seals'**
  String get compareSavedSeals;

  /// Localized app string for compareSavedSealsTitle.
  ///
  /// In en, this message translates to:
  /// **'Compare saved seals'**
  String get compareSavedSealsTitle;

  /// Localized app string for compareSavedSealsMessage.
  ///
  /// In en, this message translates to:
  /// **'Review saved seal previews, kanji meanings, and style choices side by side.'**
  String get compareSavedSealsMessage;

  /// Localized app string for editSavedSeal.
  ///
  /// In en, this message translates to:
  /// **'Edit / Regenerate'**
  String get editSavedSeal;

  /// Localized app string for editSavedSealTitle.
  ///
  /// In en, this message translates to:
  /// **'Create a new version from Design'**
  String get editSavedSealTitle;

  /// Localized app string for editSavedSealMessage.
  ///
  /// In en, this message translates to:
  /// **'Saved seals stay unchanged. To try different kanji or style choices, start a new design and save it.'**
  String get editSavedSealMessage;

  /// Localized app string for chooseSealForOrder.
  ///
  /// In en, this message translates to:
  /// **'Choose for Order'**
  String get chooseSealForOrder;

  /// Localized app string for sealSelectedForOrderTitle.
  ///
  /// In en, this message translates to:
  /// **'Selected for order'**
  String get sealSelectedForOrderTitle;

  /// Localized app string for sealSelectedForOrderMessage.
  ///
  /// In en, this message translates to:
  /// **'This seal is now saved in the order draft.'**
  String get sealSelectedForOrderMessage;

  /// Localized app string for sealSelectedForOrderAction.
  ///
  /// In en, this message translates to:
  /// **'Selected for Order'**
  String get sealSelectedForOrderAction;

  /// Localized app string for deleteSavedSeal.
  ///
  /// In en, this message translates to:
  /// **'Delete Seal'**
  String get deleteSavedSeal;

  /// Localized app string for deleteSealTitle.
  ///
  /// In en, this message translates to:
  /// **'Delete saved seal?'**
  String get deleteSealTitle;

  /// Localized app string for deleteSealMessage.
  ///
  /// In en, this message translates to:
  /// **'This removes the seal design from this device. This action cannot be undone.'**
  String get deleteSealMessage;

  /// Localized app string for deleteSealConfirm.
  ///
  /// In en, this message translates to:
  /// **'Delete'**
  String get deleteSealConfirm;

  /// Localized app string for cancel.
  ///
  /// In en, this message translates to:
  /// **'Cancel'**
  String get cancel;

  /// Localized app string for close.
  ///
  /// In en, this message translates to:
  /// **'Close'**
  String get close;

  /// Localized app string for browseStones.
  ///
  /// In en, this message translates to:
  /// **'Browse Stones'**
  String get browseStones;

  /// Localized app string for browseStonesDescription.
  ///
  /// In en, this message translates to:
  /// **'Explore our collection of\nnatural gemstones.'**
  String get browseStonesDescription;

  /// Localized app string for noSavedSeals.
  ///
  /// In en, this message translates to:
  /// **'No saved seals'**
  String get noSavedSeals;

  /// Localized app string for noSavedSealsMessage.
  ///
  /// In en, this message translates to:
  /// **'Saved seal designs will appear here after you create one.'**
  String get noSavedSealsMessage;

  /// Localized app string for stonesLoadingTitle.
  ///
  /// In en, this message translates to:
  /// **'Loading stones'**
  String get stonesLoadingTitle;

  /// Localized app string for stonesLoadingMessage.
  ///
  /// In en, this message translates to:
  /// **'Checking available one-of-a-kind seal stones.'**
  String get stonesLoadingMessage;

  /// Localized app string for stonesLoadErrorTitle.
  ///
  /// In en, this message translates to:
  /// **'Couldn\'t load stones'**
  String get stonesLoadErrorTitle;

  /// Localized app string for stonesLoadErrorMessage.
  ///
  /// In en, this message translates to:
  /// **'Try again to refresh the available stone listings.'**
  String get stonesLoadErrorMessage;

  /// Localized app string for noStonesLoaded.
  ///
  /// In en, this message translates to:
  /// **'No stones loaded'**
  String get noStonesLoaded;

  /// Localized app string for noStonesLoadedMessage.
  ///
  /// In en, this message translates to:
  /// **'Available one-of-a-kind stones will be shown here.'**
  String get noStonesLoadedMessage;

  /// Localized app string for stoneFiltersTitle.
  ///
  /// In en, this message translates to:
  /// **'Filters'**
  String get stoneFiltersTitle;

  /// Localized app string for stoneFilterAll.
  ///
  /// In en, this message translates to:
  /// **'All'**
  String get stoneFilterAll;

  /// Localized app string for stoneFilterMaterial.
  ///
  /// In en, this message translates to:
  /// **'Material'**
  String get stoneFilterMaterial;

  /// Localized app string for stoneFilterColor.
  ///
  /// In en, this message translates to:
  /// **'Color'**
  String get stoneFilterColor;

  /// Localized app string for stoneFilterPattern.
  ///
  /// In en, this message translates to:
  /// **'Pattern'**
  String get stoneFilterPattern;

  /// Localized app string for stoneFilterAvailability.
  ///
  /// In en, this message translates to:
  /// **'Availability'**
  String get stoneFilterAvailability;

  /// Localized app string for stoneFilterReset.
  ///
  /// In en, this message translates to:
  /// **'Reset'**
  String get stoneFilterReset;

  /// Localized app string for noStonesMatchFilters.
  ///
  /// In en, this message translates to:
  /// **'No stones match filters'**
  String get noStonesMatchFilters;

  /// Localized app string for noStonesMatchFiltersMessage.
  ///
  /// In en, this message translates to:
  /// **'Clear or change filters to browse other stones.'**
  String get noStonesMatchFiltersMessage;

  /// Localized app string for stoneSortTitle.
  ///
  /// In en, this message translates to:
  /// **'Sort'**
  String get stoneSortTitle;

  /// Localized app string for stoneSortAction.
  ///
  /// In en, this message translates to:
  /// **'Sort'**
  String get stoneSortAction;

  /// Localized app string for stoneSortRecommended.
  ///
  /// In en, this message translates to:
  /// **'Recommended'**
  String get stoneSortRecommended;

  /// Localized app string for stoneSortNewest.
  ///
  /// In en, this message translates to:
  /// **'Newest'**
  String get stoneSortNewest;

  /// Localized app string for stoneSortPriceLowToHigh.
  ///
  /// In en, this message translates to:
  /// **'Price: Low to High'**
  String get stoneSortPriceLowToHigh;

  /// Localized app string for stoneSortPriceHighToLow.
  ///
  /// In en, this message translates to:
  /// **'Price: High to Low'**
  String get stoneSortPriceHighToLow;

  /// Localized app string for viewStoneDetails.
  ///
  /// In en, this message translates to:
  /// **'View Details'**
  String get viewStoneDetails;

  /// Localized app string for stoneDetailTitle.
  ///
  /// In en, this message translates to:
  /// **'Stone Detail'**
  String get stoneDetailTitle;

  /// Localized app string for stoneDetailDescriptionTitle.
  ///
  /// In en, this message translates to:
  /// **'Description'**
  String get stoneDetailDescriptionTitle;

  /// Localized app string for stoneDetailStoryTitle.
  ///
  /// In en, this message translates to:
  /// **'Story'**
  String get stoneDetailStoryTitle;

  /// Localized app string for stoneDetailSpecsTitle.
  ///
  /// In en, this message translates to:
  /// **'Details'**
  String get stoneDetailSpecsTitle;

  /// Localized app string for stoneDetailNotesTitle.
  ///
  /// In en, this message translates to:
  /// **'Notes'**
  String get stoneDetailNotesTitle;

  /// Localized app string for stoneDetailNotesMessage.
  ///
  /// In en, this message translates to:
  /// **'Natural stone color, pattern, and translucency vary by piece. Review the photos and details before ordering.'**
  String get stoneDetailNotesMessage;

  /// Localized app string for stoneDetailMaterialLabel.
  ///
  /// In en, this message translates to:
  /// **'Material'**
  String get stoneDetailMaterialLabel;

  /// Localized app string for stoneDetailSizeLabel.
  ///
  /// In en, this message translates to:
  /// **'Size'**
  String get stoneDetailSizeLabel;

  /// Localized app string for stoneDetailColorLabel.
  ///
  /// In en, this message translates to:
  /// **'Color'**
  String get stoneDetailColorLabel;

  /// Localized app string for stoneDetailPatternLabel.
  ///
  /// In en, this message translates to:
  /// **'Pattern'**
  String get stoneDetailPatternLabel;

  /// Localized app string for stoneDetailShapeLabel.
  ///
  /// In en, this message translates to:
  /// **'Shape'**
  String get stoneDetailShapeLabel;

  /// Localized app string for stoneDetailTextureLabel.
  ///
  /// In en, this message translates to:
  /// **'Texture'**
  String get stoneDetailTextureLabel;

  /// Localized app string for stoneDetailStatusLabel.
  ///
  /// In en, this message translates to:
  /// **'Status'**
  String get stoneDetailStatusLabel;

  /// Localized app string for selectStone.
  ///
  /// In en, this message translates to:
  /// **'Select Stone'**
  String get selectStone;

  /// Localized app string for selectStoneConfirmationTitle.
  ///
  /// In en, this message translates to:
  /// **'Select this stone?'**
  String get selectStoneConfirmationTitle;

  /// Localized app string for selectStoneConfirmationMessage.
  ///
  /// In en, this message translates to:
  /// **'Confirm this one-of-a-kind stone for your order draft. Availability will be checked again before checkout.'**
  String get selectStoneConfirmationMessage;

  /// Localized app string for selectStoneConfirm.
  ///
  /// In en, this message translates to:
  /// **'Confirm Selection'**
  String get selectStoneConfirm;

  /// Localized app string for stoneSelectedForOrderTitle.
  ///
  /// In en, this message translates to:
  /// **'Selected for order'**
  String get stoneSelectedForOrderTitle;

  /// Localized app string for stoneSelectedForOrderMessage.
  ///
  /// In en, this message translates to:
  /// **'This stone is now saved in the order draft.'**
  String get stoneSelectedForOrderMessage;

  /// Localized app string for stoneSelectedForOrderAction.
  ///
  /// In en, this message translates to:
  /// **'Selected for Order'**
  String get stoneSelectedForOrderAction;

  /// Localized app string for soldOutStoneTitle.
  ///
  /// In en, this message translates to:
  /// **'Stone unavailable'**
  String get soldOutStoneTitle;

  /// Localized app string for soldOutStoneMessage.
  ///
  /// In en, this message translates to:
  /// **'This stone is no longer available. Choose another stone before ordering.'**
  String get soldOutStoneMessage;

  /// Localized app string for stoneAvailable.
  ///
  /// In en, this message translates to:
  /// **'Available'**
  String get stoneAvailable;

  /// Localized app string for stoneUnavailable.
  ///
  /// In en, this message translates to:
  /// **'Unavailable'**
  String get stoneUnavailable;

  /// Localized app string for order.
  ///
  /// In en, this message translates to:
  /// **'Order'**
  String get order;

  /// Localized app string for noActiveDraft.
  ///
  /// In en, this message translates to:
  /// **'No active draft'**
  String get noActiveDraft;

  /// Localized app string for noActiveDraftMessage.
  ///
  /// In en, this message translates to:
  /// **'Choose a saved seal and a stone before checkout.'**
  String get noActiveDraftMessage;

  /// Localized app string for reviewSelection.
  ///
  /// In en, this message translates to:
  /// **'Review Selection'**
  String get reviewSelection;

  /// Localized app string for orderMissingSealTitle.
  ///
  /// In en, this message translates to:
  /// **'Seal design missing'**
  String get orderMissingSealTitle;

  /// Localized app string for orderMissingSealMessage.
  ///
  /// In en, this message translates to:
  /// **'Choose a saved seal design before continuing to checkout.'**
  String get orderMissingSealMessage;

  /// Localized app string for orderMissingSealNotice.
  ///
  /// In en, this message translates to:
  /// **'A seal design is required to complete this custom order.'**
  String get orderMissingSealNotice;

  /// Localized app string for orderChooseSealAction.
  ///
  /// In en, this message translates to:
  /// **'Choose a Seal'**
  String get orderChooseSealAction;

  /// Localized app string for orderChangeSealAction.
  ///
  /// In en, this message translates to:
  /// **'Change Seal'**
  String get orderChangeSealAction;

  /// Localized app string for orderMissingStoneTitle.
  ///
  /// In en, this message translates to:
  /// **'Stone missing'**
  String get orderMissingStoneTitle;

  /// Localized app string for orderMissingStoneMessage.
  ///
  /// In en, this message translates to:
  /// **'Choose a gemstone seal stone before continuing to checkout.'**
  String get orderMissingStoneMessage;

  /// Localized app string for orderChooseStoneAction.
  ///
  /// In en, this message translates to:
  /// **'Choose a Stone'**
  String get orderChooseStoneAction;

  /// Localized app string for orderChangeStoneAction.
  ///
  /// In en, this message translates to:
  /// **'Change Stone'**
  String get orderChangeStoneAction;

  /// Localized app string for orderReviewTitle.
  ///
  /// In en, this message translates to:
  /// **'Order Review'**
  String get orderReviewTitle;

  /// Localized app string for orderReviewMessage.
  ///
  /// In en, this message translates to:
  /// **'Review the selected seal design and one-of-a-kind stone before entering shipping details.'**
  String get orderReviewMessage;

  /// Localized app string for orderItemPriceLabel.
  ///
  /// In en, this message translates to:
  /// **'Item price'**
  String get orderItemPriceLabel;

  /// Localized app string for orderShippingFeeLabel.
  ///
  /// In en, this message translates to:
  /// **'Shipping'**
  String get orderShippingFeeLabel;

  /// Localized app string for orderShippingEstimateNote.
  ///
  /// In en, this message translates to:
  /// **'Shipping is an estimate and will be recalculated before payment.'**
  String get orderShippingEstimateNote;

  /// Localized app string for orderTotalLabel.
  ///
  /// In en, this message translates to:
  /// **'Total'**
  String get orderTotalLabel;

  /// Localized app string for orderCustomMadeNotice.
  ///
  /// In en, this message translates to:
  /// **'This product is made by combining your seal design with the selected stone as a custom one-of-a-kind item.'**
  String get orderCustomMadeNotice;

  /// Localized app string for continueToShipping.
  ///
  /// In en, this message translates to:
  /// **'Continue to Shipping'**
  String get continueToShipping;

  /// Localized app string for checkoutInputTitle.
  ///
  /// In en, this message translates to:
  /// **'Checkout Information'**
  String get checkoutInputTitle;

  /// Localized app string for checkoutInputMessage.
  ///
  /// In en, this message translates to:
  /// **'Enter the contact, shipping, and optional note details for this order draft.'**
  String get checkoutInputMessage;

  /// Localized app string for checkoutContactTitle.
  ///
  /// In en, this message translates to:
  /// **'Contact'**
  String get checkoutContactTitle;

  /// Localized app string for checkoutShippingTitle.
  ///
  /// In en, this message translates to:
  /// **'Shipping address'**
  String get checkoutShippingTitle;

  /// Localized app string for checkoutOrderNoteTitle.
  ///
  /// In en, this message translates to:
  /// **'Order note'**
  String get checkoutOrderNoteTitle;

  /// Localized app string for checkoutFullNameLabel.
  ///
  /// In en, this message translates to:
  /// **'Full name'**
  String get checkoutFullNameLabel;

  /// Localized app string for checkoutFullNameHint.
  ///
  /// In en, this message translates to:
  /// **'Michael Smith'**
  String get checkoutFullNameHint;

  /// Localized app string for checkoutPhoneLabel.
  ///
  /// In en, this message translates to:
  /// **'Phone number'**
  String get checkoutPhoneLabel;

  /// Localized app string for checkoutPhoneHint.
  ///
  /// In en, this message translates to:
  /// **'+1 000 000 0000'**
  String get checkoutPhoneHint;

  /// Localized app string for checkoutCountryLabel.
  ///
  /// In en, this message translates to:
  /// **'Country / Region'**
  String get checkoutCountryLabel;

  /// Localized app string for checkoutPostalCodeLabel.
  ///
  /// In en, this message translates to:
  /// **'Postal code'**
  String get checkoutPostalCodeLabel;

  /// Localized app string for checkoutPostalCodeHint.
  ///
  /// In en, this message translates to:
  /// **'10001'**
  String get checkoutPostalCodeHint;

  /// Localized app string for checkoutAddressLine1Label.
  ///
  /// In en, this message translates to:
  /// **'Address line 1'**
  String get checkoutAddressLine1Label;

  /// Localized app string for checkoutAddressLine1Hint.
  ///
  /// In en, this message translates to:
  /// **'123 Example Street'**
  String get checkoutAddressLine1Hint;

  /// Localized app string for checkoutAddressLine2Label.
  ///
  /// In en, this message translates to:
  /// **'Address line 2'**
  String get checkoutAddressLine2Label;

  /// Localized app string for checkoutAddressLine2Hint.
  ///
  /// In en, this message translates to:
  /// **'Apt 1'**
  String get checkoutAddressLine2Hint;

  /// Localized app string for checkoutCityLabel.
  ///
  /// In en, this message translates to:
  /// **'City'**
  String get checkoutCityLabel;

  /// Localized app string for checkoutCityHint.
  ///
  /// In en, this message translates to:
  /// **'New York'**
  String get checkoutCityHint;

  /// Localized app string for checkoutStateLabel.
  ///
  /// In en, this message translates to:
  /// **'State / Province'**
  String get checkoutStateLabel;

  /// Localized app string for checkoutStateHint.
  ///
  /// In en, this message translates to:
  /// **'NY'**
  String get checkoutStateHint;

  /// Localized app string for checkoutOrderNoteLabel.
  ///
  /// In en, this message translates to:
  /// **'Order note'**
  String get checkoutOrderNoteLabel;

  /// Localized app string for checkoutOrderNoteHint.
  ///
  /// In en, this message translates to:
  /// **'Optional production or delivery note'**
  String get checkoutOrderNoteHint;

  /// Localized app string for checkoutInputSaveAction.
  ///
  /// In en, this message translates to:
  /// **'Save Checkout Information'**
  String get checkoutInputSaveAction;

  /// Localized app string for checkoutInputSavingAction.
  ///
  /// In en, this message translates to:
  /// **'Saving...'**
  String get checkoutInputSavingAction;

  /// Localized app string for checkoutInputSavedMessage.
  ///
  /// In en, this message translates to:
  /// **'Checkout information was saved to this order draft.'**
  String get checkoutInputSavedMessage;

  /// Localized app string for checkoutValidationTitle.
  ///
  /// In en, this message translates to:
  /// **'Please review the highlighted fields.'**
  String get checkoutValidationTitle;

  /// Localized app string for checkoutValidationMessage.
  ///
  /// In en, this message translates to:
  /// **'Some information is missing or invalid.'**
  String get checkoutValidationMessage;

  /// Localized app string for checkoutEmailInvalidMessage.
  ///
  /// In en, this message translates to:
  /// **'Please enter a valid email address.'**
  String get checkoutEmailInvalidMessage;

  /// Localized app string for checkoutFullNameRequiredMessage.
  ///
  /// In en, this message translates to:
  /// **'Full name is required.'**
  String get checkoutFullNameRequiredMessage;

  /// Localized app string for checkoutPhoneInvalidMessage.
  ///
  /// In en, this message translates to:
  /// **'Please enter a valid phone number.'**
  String get checkoutPhoneInvalidMessage;

  /// Localized app string for checkoutCountryRequiredMessage.
  ///
  /// In en, this message translates to:
  /// **'Country / Region is required.'**
  String get checkoutCountryRequiredMessage;

  /// Localized app string for checkoutPostalCodeRequiredMessage.
  ///
  /// In en, this message translates to:
  /// **'Postal code is required.'**
  String get checkoutPostalCodeRequiredMessage;

  /// Localized app string for checkoutAddressLine1RequiredMessage.
  ///
  /// In en, this message translates to:
  /// **'Address line 1 is required.'**
  String get checkoutAddressLine1RequiredMessage;

  /// Localized app string for checkoutCityRequiredMessage.
  ///
  /// In en, this message translates to:
  /// **'City is required.'**
  String get checkoutCityRequiredMessage;

  /// Localized app string for checkoutStateRequiredMessage.
  ///
  /// In en, this message translates to:
  /// **'State / Province is required.'**
  String get checkoutStateRequiredMessage;

  /// Localized app string for orderConfirmationTitle.
  ///
  /// In en, this message translates to:
  /// **'Order Confirmation'**
  String get orderConfirmationTitle;

  /// Localized app string for orderConfirmationMessage.
  ///
  /// In en, this message translates to:
  /// **'Review the seal, gemstone, shipping details, and total before proceeding.'**
  String get orderConfirmationMessage;

  /// Localized app string for orderConfirmationMissingInputMessage.
  ///
  /// In en, this message translates to:
  /// **'Checkout information is incomplete. Return to the checkout form before confirming.'**
  String get orderConfirmationMissingInputMessage;

  /// Localized app string for orderConfirmationCheckoutTitle.
  ///
  /// In en, this message translates to:
  /// **'Checkout details'**
  String get orderConfirmationCheckoutTitle;

  /// Localized app string for orderConfirmationNoOrderNote.
  ///
  /// In en, this message translates to:
  /// **'No order note'**
  String get orderConfirmationNoOrderNote;

  /// Localized app string for editCheckoutInformation.
  ///
  /// In en, this message translates to:
  /// **'Edit Checkout Information'**
  String get editCheckoutInformation;

  /// Localized app string for customMadeAgreementTitle.
  ///
  /// In en, this message translates to:
  /// **'Custom-made Agreement'**
  String get customMadeAgreementTitle;

  /// Localized app string for customMadeAgreementMessage.
  ///
  /// In en, this message translates to:
  /// **'Each seal is handcrafted to order using natural gemstones. Please confirm the details before payment.'**
  String get customMadeAgreementMessage;

  /// Localized app string for confirmKanjiAndDesignLabel.
  ///
  /// In en, this message translates to:
  /// **'I confirm that the selected kanji and seal design are correct.'**
  String get confirmKanjiAndDesignLabel;

  /// Localized app string for confirmCustomMadePolicyLabel.
  ///
  /// In en, this message translates to:
  /// **'I understand that this is a custom-made item and cannot be changed after production begins.'**
  String get confirmCustomMadePolicyLabel;

  /// Localized app string for orderConfirmationSecurePaymentNote.
  ///
  /// In en, this message translates to:
  /// **'You will be redirected to Stripe Checkout for secure payment.'**
  String get orderConfirmationSecurePaymentNote;

  /// Localized app string for orderConfirmationAgreementRequiredMessage.
  ///
  /// In en, this message translates to:
  /// **'Both confirmation checks are required before payment.'**
  String get orderConfirmationAgreementRequiredMessage;

  /// Localized app string for orderConfirmationSavedMessage.
  ///
  /// In en, this message translates to:
  /// **'Order confirmation was saved to this order draft.'**
  String get orderConfirmationSavedMessage;

  /// Localized app string for proceedToSecurePayment.
  ///
  /// In en, this message translates to:
  /// **'Proceed to Secure Payment'**
  String get proceedToSecurePayment;

  /// Localized app string for checkoutProcessingTitle.
  ///
  /// In en, this message translates to:
  /// **'Preparing Checkout'**
  String get checkoutProcessingTitle;

  /// Localized app string for checkoutProcessingMessage.
  ///
  /// In en, this message translates to:
  /// **'Creating your order and secure Stripe Checkout session.'**
  String get checkoutProcessingMessage;

  /// Localized app string for checkoutProcessingOrderStep.
  ///
  /// In en, this message translates to:
  /// **'Creating order'**
  String get checkoutProcessingOrderStep;

  /// Localized app string for checkoutProcessingSessionStep.
  ///
  /// In en, this message translates to:
  /// **'Creating secure payment session'**
  String get checkoutProcessingSessionStep;

  /// Localized app string for checkoutProcessingReadyTitle.
  ///
  /// In en, this message translates to:
  /// **'Checkout session ready'**
  String get checkoutProcessingReadyTitle;

  /// Localized app string for checkoutProcessingReadyMessage.
  ///
  /// In en, this message translates to:
  /// **'Stripe Checkout is ready to open.'**
  String get checkoutProcessingReadyMessage;

  /// Localized app string for checkoutProcessingErrorTitle.
  ///
  /// In en, this message translates to:
  /// **'Couldn\'t prepare Checkout'**
  String get checkoutProcessingErrorTitle;

  /// Localized app string for checkoutProcessingErrorMessage.
  ///
  /// In en, this message translates to:
  /// **'Please go back and try again.'**
  String get checkoutProcessingErrorMessage;

  /// Localized app string for stripeCheckoutTitle.
  ///
  /// In en, this message translates to:
  /// **'Secure Payment'**
  String get stripeCheckoutTitle;

  /// Localized app string for stripeCheckoutOpeningTitle.
  ///
  /// In en, this message translates to:
  /// **'Opening Stripe Checkout'**
  String get stripeCheckoutOpeningTitle;

  /// Localized app string for stripeCheckoutOpeningMessage.
  ///
  /// In en, this message translates to:
  /// **'You will be redirected to Stripe Checkout.'**
  String get stripeCheckoutOpeningMessage;

  /// Localized app string for stripeCheckoutWaitingTitle.
  ///
  /// In en, this message translates to:
  /// **'Complete payment in Stripe Checkout'**
  String get stripeCheckoutWaitingTitle;

  /// Localized app string for stripeCheckoutWaitingMessage.
  ///
  /// In en, this message translates to:
  /// **'Return to this app after payment to continue.'**
  String get stripeCheckoutWaitingMessage;

  /// Localized app string for stripeCheckoutReturnedTitle.
  ///
  /// In en, this message translates to:
  /// **'Returned from Stripe Checkout'**
  String get stripeCheckoutReturnedTitle;

  /// Localized app string for stripeCheckoutReturnedMessage.
  ///
  /// In en, this message translates to:
  /// **'The return URL was received. Payment status will be verified next.'**
  String get stripeCheckoutReturnedMessage;

  /// Localized app string for stripeCheckoutCanceledTitle.
  ///
  /// In en, this message translates to:
  /// **'Checkout was canceled'**
  String get stripeCheckoutCanceledTitle;

  /// Localized app string for stripeCheckoutCanceledMessage.
  ///
  /// In en, this message translates to:
  /// **'Stripe returned without completing payment.'**
  String get stripeCheckoutCanceledMessage;

  /// Localized app string for stripeCheckoutReturnFailedTitle.
  ///
  /// In en, this message translates to:
  /// **'Payment failed'**
  String get stripeCheckoutReturnFailedTitle;

  /// Localized app string for stripeCheckoutReturnFailedMessage.
  ///
  /// In en, this message translates to:
  /// **'Stripe Checkout could not be completed. You can try Checkout again.'**
  String get stripeCheckoutReturnFailedMessage;

  /// Localized app string for stripeCheckoutLaunchFailedTitle.
  ///
  /// In en, this message translates to:
  /// **'Couldn\'t open Stripe Checkout'**
  String get stripeCheckoutLaunchFailedTitle;

  /// Localized app string for stripeCheckoutLaunchFailedMessage.
  ///
  /// In en, this message translates to:
  /// **'Check your browser settings and try again.'**
  String get stripeCheckoutLaunchFailedMessage;

  /// Localized app string for stripeCheckoutOpenAction.
  ///
  /// In en, this message translates to:
  /// **'Open Stripe Checkout'**
  String get stripeCheckoutOpenAction;

  /// Localized app string for stripeCheckoutRetryAction.
  ///
  /// In en, this message translates to:
  /// **'Try Again'**
  String get stripeCheckoutRetryAction;

  /// Localized app string for stripeCheckoutSecureNote.
  ///
  /// In en, this message translates to:
  /// **'Your payment information is secure and encrypted. Powered by Stripe.'**
  String get stripeCheckoutSecureNote;

  /// Localized app string for stripeCheckoutReturnOrderIdLabel.
  ///
  /// In en, this message translates to:
  /// **'Return order ID'**
  String get stripeCheckoutReturnOrderIdLabel;

  /// Localized app string for paymentStatusTitle.
  ///
  /// In en, this message translates to:
  /// **'Payment Status'**
  String get paymentStatusTitle;

  /// Localized app string for paymentStatusCheckingTitle.
  ///
  /// In en, this message translates to:
  /// **'Checking payment status'**
  String get paymentStatusCheckingTitle;

  /// Localized app string for paymentStatusCheckingMessage.
  ///
  /// In en, this message translates to:
  /// **'Confirming the latest order status after Stripe Checkout.'**
  String get paymentStatusCheckingMessage;

  /// Localized app string for paymentStatusPaidTitle.
  ///
  /// In en, this message translates to:
  /// **'Payment confirmed'**
  String get paymentStatusPaidTitle;

  /// Localized app string for paymentStatusPaidMessage.
  ///
  /// In en, this message translates to:
  /// **'Your payment is confirmed. We are preparing the order summary.'**
  String get paymentStatusPaidMessage;

  /// Localized app string for paymentStatusPendingTitle.
  ///
  /// In en, this message translates to:
  /// **'Payment pending'**
  String get paymentStatusPendingTitle;

  /// Localized app string for paymentStatusPendingMessage.
  ///
  /// In en, this message translates to:
  /// **'Stripe returned successfully, but payment confirmation is still pending.'**
  String get paymentStatusPendingMessage;

  /// Localized app string for paymentStatusPendingNotice.
  ///
  /// In en, this message translates to:
  /// **''**
  String get paymentStatusPendingNotice;

  /// Localized app string for paymentStatusFailedTitle.
  ///
  /// In en, this message translates to:
  /// **'Couldn\'t verify payment'**
  String get paymentStatusFailedTitle;

  /// Localized app string for paymentStatusFailedMessage.
  ///
  /// In en, this message translates to:
  /// **'Payment status could not be confirmed. Please check Order Lookup later.'**
  String get paymentStatusFailedMessage;

  /// Localized app string for orderCompleteTitle.
  ///
  /// In en, this message translates to:
  /// **'Order Complete'**
  String get orderCompleteTitle;

  /// Localized app string for orderCompleteStatusTitle.
  ///
  /// In en, this message translates to:
  /// **'Thank you for your order'**
  String get orderCompleteStatusTitle;

  /// Localized app string for orderCompleteMessage.
  ///
  /// In en, this message translates to:
  /// **'Payment is confirmed. Stripe will send the payment receipt to the email address used for checkout.'**
  String get orderCompleteMessage;

  /// Localized app string for orderCompleteStatusLabel.
  ///
  /// In en, this message translates to:
  /// **'Status'**
  String get orderCompleteStatusLabel;

  /// Localized app string for orderCompleteStatusValue.
  ///
  /// In en, this message translates to:
  /// **'Payment received'**
  String get orderCompleteStatusValue;

  /// Localized app string for orderCompleteEmailMessage.
  ///
  /// In en, this message translates to:
  /// **'Please check the Stripe payment receipt. Keep the order number for support and order lookup.'**
  String get orderCompleteEmailMessage;

  /// Localized app string for orderCompleteSummaryTitle.
  ///
  /// In en, this message translates to:
  /// **'Order summary'**
  String get orderCompleteSummaryTitle;

  /// Localized app string for orderCompleteLookupAction.
  ///
  /// In en, this message translates to:
  /// **'Open Order Lookup'**
  String get orderCompleteLookupAction;

  /// Localized app string for orderCompleteBackToDesignAction.
  ///
  /// In en, this message translates to:
  /// **'Back to Design'**
  String get orderCompleteBackToDesignAction;

  /// Localized app string for emailSentNoticeTitle.
  ///
  /// In en, this message translates to:
  /// **'Stripe payment email'**
  String get emailSentNoticeTitle;

  /// Localized app string for emailSentNoticeMessage.
  ///
  /// In en, this message translates to:
  /// **'Stripe sends the payment receipt to the email address on the order. Please check your inbox and spam folder.'**
  String get emailSentNoticeMessage;

  /// Localized app string for orderEmailMissingGuideTitle.
  ///
  /// In en, this message translates to:
  /// **'Can\'t find the Stripe email?'**
  String get orderEmailMissingGuideTitle;

  /// Localized app string for orderEmailMissingGuideMessage.
  ///
  /// In en, this message translates to:
  /// **'Here are a few quick things to check.'**
  String get orderEmailMissingGuideMessage;

  /// Localized app string for orderEmailMissingSpamCheck.
  ///
  /// In en, this message translates to:
  /// **'Check your spam or junk folder.'**
  String get orderEmailMissingSpamCheck;

  /// Localized app string for orderEmailMissingAddressCheck.
  ///
  /// In en, this message translates to:
  /// **'Make sure the email address on the order is correct.'**
  String get orderEmailMissingAddressCheck;

  /// Localized app string for orderEmailMissingDeliveryWait.
  ///
  /// In en, this message translates to:
  /// **'Please allow a few minutes for delivery.'**
  String get orderEmailMissingDeliveryWait;

  /// Localized app string for orderEmailMissingContactSupport.
  ///
  /// In en, this message translates to:
  /// **'If you still can\'t find it, contact support with your order number.'**
  String get orderEmailMissingContactSupport;

  /// Localized app string for contactSupportPromptTitle.
  ///
  /// In en, this message translates to:
  /// **'Need help?'**
  String get contactSupportPromptTitle;

  /// Localized app string for contactSupportPromptMessage.
  ///
  /// In en, this message translates to:
  /// **'Our support team can help with order, shipping, payment, and email questions. Include your order number for faster support.'**
  String get contactSupportPromptMessage;

  /// Localized app string for contactSupportPromptAction.
  ///
  /// In en, this message translates to:
  /// **'Contact Support'**
  String get contactSupportPromptAction;

  /// Localized app string for orderLookup.
  ///
  /// In en, this message translates to:
  /// **'Order Lookup'**
  String get orderLookup;

  /// Localized app string for orderNo.
  ///
  /// In en, this message translates to:
  /// **'Order No'**
  String get orderNo;

  /// Localized app string for orderNoHint.
  ///
  /// In en, this message translates to:
  /// **'HF-0001'**
  String get orderNoHint;

  /// Localized app string for email.
  ///
  /// In en, this message translates to:
  /// **'Email'**
  String get email;

  /// Localized app string for emailHint.
  ///
  /// In en, this message translates to:
  /// **'name@example.com'**
  String get emailHint;

  /// Localized app string for lookupOrder.
  ///
  /// In en, this message translates to:
  /// **'Lookup Order'**
  String get lookupOrder;

  /// Localized app string for orderLookupLoadingTitle.
  ///
  /// In en, this message translates to:
  /// **'Looking up your order'**
  String get orderLookupLoadingTitle;

  /// Localized app string for orderLookupLoadingMessage.
  ///
  /// In en, this message translates to:
  /// **'Checking the order number and email address.'**
  String get orderLookupLoadingMessage;

  /// Localized app string for orderLookupNotFoundTitle.
  ///
  /// In en, this message translates to:
  /// **'Order not found'**
  String get orderLookupNotFoundTitle;

  /// Localized app string for orderLookupNotFoundMessage.
  ///
  /// In en, this message translates to:
  /// **'We couldn\'t find an order matching that order number and email address.'**
  String get orderLookupNotFoundMessage;

  /// Localized app string for orderLookupErrorTitle.
  ///
  /// In en, this message translates to:
  /// **'Couldn\'t load order'**
  String get orderLookupErrorTitle;

  /// Localized app string for orderLookupErrorMessage.
  ///
  /// In en, this message translates to:
  /// **'Order Lookup could not be completed. Please try again.'**
  String get orderLookupErrorMessage;

  /// Localized app string for orderLookupResultTitle.
  ///
  /// In en, this message translates to:
  /// **'Order Status'**
  String get orderLookupResultTitle;

  /// Localized app string for orderLookupResultMessage.
  ///
  /// In en, this message translates to:
  /// **'Here\'s the latest update on your order.'**
  String get orderLookupResultMessage;

  /// Localized app string for orderLookupOrderStatusLabel.
  ///
  /// In en, this message translates to:
  /// **'Order status'**
  String get orderLookupOrderStatusLabel;

  /// Localized app string for orderLookupProgressTitle.
  ///
  /// In en, this message translates to:
  /// **'Order progress'**
  String get orderLookupProgressTitle;

  /// Localized app string for orderLookupProductionStatusLabel.
  ///
  /// In en, this message translates to:
  /// **'Production status'**
  String get orderLookupProductionStatusLabel;

  /// Localized app string for orderLookupShippingStatusLabel.
  ///
  /// In en, this message translates to:
  /// **'Shipping status'**
  String get orderLookupShippingStatusLabel;

  /// Localized app string for orderLookupFulfillmentStatusLabel.
  ///
  /// In en, this message translates to:
  /// **'Fulfillment status'**
  String get orderLookupFulfillmentStatusLabel;

  /// Localized app string for orderLookupTrackingNumberLabel.
  ///
  /// In en, this message translates to:
  /// **'Tracking number'**
  String get orderLookupTrackingNumberLabel;

  /// Localized app string for orderLookupTrackingDetailsTitle.
  ///
  /// In en, this message translates to:
  /// **'Tracking details'**
  String get orderLookupTrackingDetailsTitle;

  /// Localized app string for orderLookupCarrierLabel.
  ///
  /// In en, this message translates to:
  /// **'Carrier'**
  String get orderLookupCarrierLabel;

  /// Localized app string for orderLookupShippedAtLabel.
  ///
  /// In en, this message translates to:
  /// **'Shipped at'**
  String get orderLookupShippedAtLabel;

  /// Localized app string for orderLookupUpdatedAtLabel.
  ///
  /// In en, this message translates to:
  /// **'Last updated'**
  String get orderLookupUpdatedAtLabel;

  /// Localized app string for orderLookupNoTrackingValue.
  ///
  /// In en, this message translates to:
  /// **'Not available yet'**
  String get orderLookupNoTrackingValue;

  /// Localized app string for orderLookupOrderDateLabel.
  ///
  /// In en, this message translates to:
  /// **'Order date'**
  String get orderLookupOrderDateLabel;

  /// Localized app string for orderLookupContentTitle.
  ///
  /// In en, this message translates to:
  /// **'Order content'**
  String get orderLookupContentTitle;

  /// Localized app string for orderLookupSelectedSealLabel.
  ///
  /// In en, this message translates to:
  /// **'Selected seal'**
  String get orderLookupSelectedSealLabel;

  /// Localized app string for orderLookupGemstoneLabel.
  ///
  /// In en, this message translates to:
  /// **'Gemstone'**
  String get orderLookupGemstoneLabel;

  /// Localized app string for orderLookupLookupAnotherAction.
  ///
  /// In en, this message translates to:
  /// **'Lookup another order'**
  String get orderLookupLookupAnotherAction;

  /// Localized app string for language.
  ///
  /// In en, this message translates to:
  /// **'Language'**
  String get language;

  /// Localized app string for about.
  ///
  /// In en, this message translates to:
  /// **'About'**
  String get about;

  /// Localized app string for howItWorks.
  ///
  /// In en, this message translates to:
  /// **'How It Works'**
  String get howItWorks;

  /// Localized app string for faq.
  ///
  /// In en, this message translates to:
  /// **'FAQ'**
  String get faq;

  /// Localized app string for privacy.
  ///
  /// In en, this message translates to:
  /// **'Privacy'**
  String get privacy;

  /// Localized app string for terms.
  ///
  /// In en, this message translates to:
  /// **'Terms'**
  String get terms;

  /// Localized app string for contact.
  ///
  /// In en, this message translates to:
  /// **'Contact'**
  String get contact;

  /// Localized app string for version.
  ///
  /// In en, this message translates to:
  /// **'Version'**
  String get version;

  /// Localized app string for settingsLanguageTitle.
  ///
  /// In en, this message translates to:
  /// **'App language'**
  String get settingsLanguageTitle;

  /// Localized app string for settingsLanguageMessage.
  ///
  /// In en, this message translates to:
  /// **'Choose the language used in the app. Before you choose one, the app follows your smartphone language.'**
  String get settingsLanguageMessage;

  /// Localized app string for settingsLanguageEnglish.
  ///
  /// In en, this message translates to:
  /// **'English'**
  String get settingsLanguageEnglish;

  /// Localized app string for settingsLanguageJapanese.
  ///
  /// In en, this message translates to:
  /// **'Japanese'**
  String get settingsLanguageJapanese;

  /// Localized app string for settingsFaqIntro.
  ///
  /// In en, this message translates to:
  /// **'Find answers to common questions about kanji selection, production, delivery, and order lookup.'**
  String get settingsFaqIntro;

  /// Localized app string for settingsVersionTitle.
  ///
  /// In en, this message translates to:
  /// **'Installed app version'**
  String get settingsVersionTitle;

  /// Installed app version label. The version placeholder is supplied by the app package metadata.
  ///
  /// In en, this message translates to:
  /// **'Version {version}'**
  String settingsVersionMessage(String version);

  /// Prefix shown before short design-state guidance messages.
  ///
  /// In en, this message translates to:
  /// **'Tip: '**
  String get designTipPrefix;

  /// Order status label for paid orders.
  ///
  /// In en, this message translates to:
  /// **'Paid'**
  String get orderStatusPaid;

  /// Order status label for unpaid orders.
  ///
  /// In en, this message translates to:
  /// **'Unpaid'**
  String get orderStatusUnpaid;

  /// Order status label for pending orders.
  ///
  /// In en, this message translates to:
  /// **'Pending'**
  String get orderStatusPending;

  /// Order status label for orders waiting for payment.
  ///
  /// In en, this message translates to:
  /// **'Pending payment'**
  String get orderStatusPendingPayment;

  /// Order status label for failed orders.
  ///
  /// In en, this message translates to:
  /// **'Failed'**
  String get orderStatusFailed;

  /// Order status label for canceled orders.
  ///
  /// In en, this message translates to:
  /// **'Canceled'**
  String get orderStatusCanceled;

  /// Order status label for work that has not started.
  ///
  /// In en, this message translates to:
  /// **'Not started'**
  String get orderStatusNotStarted;

  /// Order status label for orders in production.
  ///
  /// In en, this message translates to:
  /// **'In production'**
  String get orderStatusInProduction;

  /// Order status label for completed production.
  ///
  /// In en, this message translates to:
  /// **'Completed'**
  String get orderStatusCompleted;

  /// Order status label for orders preparing shipment.
  ///
  /// In en, this message translates to:
  /// **'Preparing shipment'**
  String get orderStatusPreparingShipment;

  /// Order status label for orders not yet shipped.
  ///
  /// In en, this message translates to:
  /// **'Not shipped'**
  String get orderStatusNotShipped;

  /// Order status label for shipped orders.
  ///
  /// In en, this message translates to:
  /// **'Shipped'**
  String get orderStatusShipped;

  /// Order status label for fulfilled orders.
  ///
  /// In en, this message translates to:
  /// **'Fulfilled'**
  String get orderStatusFulfilled;

  /// Fallback order status label for an empty status value.
  ///
  /// In en, this message translates to:
  /// **'-'**
  String get orderStatusEmpty;
}

class _GeneratedHankoLocalizationsDelegate
    extends LocalizationsDelegate<GeneratedHankoLocalizations> {
  const _GeneratedHankoLocalizationsDelegate();

  @override
  Future<GeneratedHankoLocalizations> load(Locale locale) {
    return SynchronousFuture<GeneratedHankoLocalizations>(
      lookupGeneratedHankoLocalizations(locale),
    );
  }

  @override
  bool isSupported(Locale locale) =>
      <String>['en', 'ja', 'zh'].contains(locale.languageCode);

  @override
  bool shouldReload(_GeneratedHankoLocalizationsDelegate old) => false;
}

GeneratedHankoLocalizations lookupGeneratedHankoLocalizations(Locale locale) {
  // Lookup logic when language+script codes are specified.
  switch (locale.languageCode) {
    case 'zh':
      {
        switch (locale.scriptCode) {
          case 'Hant':
            return GeneratedHankoLocalizationsZhHant();
        }
        break;
      }
  }

  // Lookup logic when only language code is specified.
  switch (locale.languageCode) {
    case 'en':
      return GeneratedHankoLocalizationsEn();
    case 'ja':
      return GeneratedHankoLocalizationsJa();
    case 'zh':
      return GeneratedHankoLocalizationsZh();
  }

  throw FlutterError(
    'GeneratedHankoLocalizations.delegate failed to load unsupported locale "$locale". This is likely '
    'an issue with the localizations generation tool. Please file an issue '
    'on GitHub with a reproducible sample app and the gen-l10n configuration '
    'that was used.',
  );
}
