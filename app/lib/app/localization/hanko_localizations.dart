import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';

import '../../l10n/generated/generated_hanko_localizations.dart';

class HankoLocalizations {
  const HankoLocalizations._(this.locale, this._strings);

  final Locale locale;
  final _HankoStrings _strings;

  static const supportedLocales = [Locale('en'), Locale('ja')];

  static const delegate = _HankoLocalizationsDelegate();

  static const List<LocalizationsDelegate<dynamic>> localizationsDelegates = [
    GeneratedHankoLocalizations.delegate,
    delegate,
    GlobalMaterialLocalizations.delegate,
    GlobalCupertinoLocalizations.delegate,
    GlobalWidgetsLocalizations.delegate,
  ];

  static HankoLocalizations of(BuildContext context) {
    return Localizations.of<HankoLocalizations>(context, HankoLocalizations) ??
        forLocale(const Locale('en'));
  }

  static HankoLocalizations forLocale(Locale locale) {
    return _localizedValues[locale.languageCode] ?? _localizedValues['en']!;
  }

  String get appTitle => _strings.appTitle;
  String get splashPreparing => _strings.splashPreparing;
  String get onboardingTitle => _strings.onboardingTitle;
  String get onboardingHeroTitle => _strings.onboardingHeroTitle;
  String get onboardingTagline => _strings.onboardingTagline;
  String get onboardingKanjiTitle => _strings.onboardingKanjiTitle;
  String get onboardingKanjiMessage => _strings.onboardingKanjiMessage;
  String get onboardingAiTitle => _strings.onboardingAiTitle;
  String get onboardingAiMessage => _strings.onboardingAiMessage;
  String get onboardingStoneTitle => _strings.onboardingStoneTitle;
  String get onboardingStoneMessage => _strings.onboardingStoneMessage;
  String get onboardingStorageTitle => _strings.onboardingStorageTitle;
  String get onboardingStorageMessage => _strings.onboardingStorageMessage;
  String get onboardingGetStarted => _strings.onboardingGetStarted;
  String get onboardingSaving => _strings.onboardingSaving;
  String get onboardingSkip => _strings.onboardingSkip;
  String get onboardingSaveError => _strings.onboardingSaveError;
  String get design => _strings.design;
  String get mySeals => _strings.mySeals;
  String get stones => _strings.stones;
  String get settings => _strings.settings;
  String get createCustomSeal => _strings.createCustomSeal;
  String get customSealDescription => _strings.customSealDescription;
  String get startDesigning => _strings.startDesigning;
  String get designNameTitle => _strings.designNameTitle;
  String get designNameIntro => _strings.designNameIntro;
  String get designNameLabel => _strings.designNameLabel;
  String get designNameHint => _strings.designNameHint;
  String get designNameHelp => _strings.designNameHelp;
  String get designGenderLabel => _strings.designGenderLabel;
  String get designGenderUnspecified => _strings.designGenderUnspecified;
  String get designGenderMale => _strings.designGenderMale;
  String get designGenderFemale => _strings.designGenderFemale;
  String get designKanjiStyleLabel => _strings.designKanjiStyleLabel;
  String get designKanjiStyleJapanese => _strings.designKanjiStyleJapanese;
  String get designKanjiStyleChinese => _strings.designKanjiStyleChinese;
  String get designKanjiStyleTaiwanese => _strings.designKanjiStyleTaiwanese;
  String get suggestKanji => _strings.suggestKanji;
  String get designKanjiTipTitle => _strings.designKanjiTipTitle;
  String get designKanjiTipMessage => _strings.designKanjiTipMessage;
  String get designCandidateReadyTitle => _strings.designCandidateReadyTitle;
  String get designCandidateReadyMessage =>
      _strings.designCandidateReadyMessage;
  String get designRequestDetails => _strings.designRequestDetails;
  String get editName => _strings.editName;
  String get designLoadingTitle => _strings.designLoadingTitle;
  String get designLoadingMessage => _strings.designLoadingMessage;
  String get designInvalidNameSummary => _strings.designInvalidNameSummary;
  String get designInvalidNameMessage => _strings.designInvalidNameMessage;
  String get designSuggestionErrorTitle => _strings.designSuggestionErrorTitle;
  String get designSuggestionErrorMessage =>
      _strings.designSuggestionErrorMessage;
  String get designNoKanjiTitle => _strings.designNoKanjiTitle;
  String get designNoKanjiMessage => _strings.designNoKanjiMessage;
  String get designNoKanjiRuleCharacters =>
      _strings.designNoKanjiRuleCharacters;
  String get designNoKanjiRuleCommon => _strings.designNoKanjiRuleCommon;
  String get designNoKanjiRuleEngraving => _strings.designNoKanjiRuleEngraving;
  String get designErrorTip => _strings.designErrorTip;
  String get designNoKanjiTip => _strings.designNoKanjiTip;
  String get tryAgain => _strings.tryAgain;
  String get back => _strings.back;
  String get commonNetworkErrorTitle => _strings.commonNetworkErrorTitle;
  String get commonNetworkErrorMessage => _strings.commonNetworkErrorMessage;
  String get commonServerErrorTitle => _strings.commonServerErrorTitle;
  String get commonServerErrorMessage => _strings.commonServerErrorMessage;
  String get storageErrorTitle => _strings.storageErrorTitle;
  String get storageErrorMessage => _strings.storageErrorMessage;
  String get deepLinkErrorTitle => _strings.deepLinkErrorTitle;
  String get deepLinkErrorMessage => _strings.deepLinkErrorMessage;
  String get maintenanceTitle => _strings.maintenanceTitle;
  String get maintenanceMessage => _strings.maintenanceMessage;
  String get appUpdateRequiredTitle => _strings.appUpdateRequiredTitle;
  String get appUpdateRequiredMessage => _strings.appUpdateRequiredMessage;
  String get appUpdateRequiredAction => _strings.appUpdateRequiredAction;
  String get commonGenericErrorTitle => _strings.commonGenericErrorTitle;
  String get commonGenericErrorMessage => _strings.commonGenericErrorMessage;
  String get kanjiSuggestionsTitle => _strings.kanjiSuggestionsTitle;
  String get kanjiSuggestionsMessage => _strings.kanjiSuggestionsMessage;
  String get kanjiCandidateDetailTitle => _strings.kanjiCandidateDetailTitle;
  String get kanjiReadingLabel => _strings.kanjiReadingLabel;
  String get kanjiMeaningLabel => _strings.kanjiMeaningLabel;
  String get kanjiImpressionLabel => _strings.kanjiImpressionLabel;
  String get kanjiReasonLabel => _strings.kanjiReasonLabel;
  String get kanjiCharacterCountLabel => _strings.kanjiCharacterCountLabel;
  String get kanjiStrokeComplexityLabel => _strings.kanjiStrokeComplexityLabel;
  String get kanjiEngravingSuitabilityLabel =>
      _strings.kanjiEngravingSuitabilityLabel;
  String get selectKanji => _strings.selectKanji;
  String get kanjiSelectedTitle => _strings.kanjiSelectedTitle;
  String get kanjiSelectedMessage => _strings.kanjiSelectedMessage;
  String get sealStyleTitle => _strings.sealStyleTitle;
  String get sealStyleMessage => _strings.sealStyleMessage;
  String get sealStyleSelectedKanjiLabel =>
      _strings.sealStyleSelectedKanjiLabel;
  String get sealShapeLabel => _strings.sealShapeLabel;
  String get sealShapeSquare => _strings.sealShapeSquare;
  String get sealShapeRound => _strings.sealShapeRound;
  String get sealStyleNameLabel => _strings.sealStyleNameLabel;
  String get sealStyleTraditional => _strings.sealStyleTraditional;
  String get sealStyleElegant => _strings.sealStyleElegant;
  String get sealStyleSoft => _strings.sealStyleSoft;
  String get sealStyleBold => _strings.sealStyleBold;
  String get sealStrokeWeightLabel => _strings.sealStrokeWeightLabel;
  String get sealStrokeStandard => _strings.sealStrokeStandard;
  String get sealStrokeBold => _strings.sealStrokeBold;
  String get sealBalanceLabel => _strings.sealBalanceLabel;
  String get sealBalanceAiry => _strings.sealBalanceAiry;
  String get sealBalanceBalanced => _strings.sealBalanceBalanced;
  String get sealBalanceDense => _strings.sealBalanceDense;
  String get sealStyleSummaryTitle => _strings.sealStyleSummaryTitle;
  String get confirmStyle => _strings.confirmStyle;
  String get sealStyleConfirmedTitle => _strings.sealStyleConfirmedTitle;
  String get sealStyleConfirmedMessage => _strings.sealStyleConfirmedMessage;
  String get generateSeal => _strings.generateSeal;
  String get sealGenerationLoadingTitle => _strings.sealGenerationLoadingTitle;
  String get sealGenerationLoadingMessage =>
      _strings.sealGenerationLoadingMessage;
  String get sealGenerationLoadingDetail =>
      _strings.sealGenerationLoadingDetail;
  String get sealGenerationErrorTitle => _strings.sealGenerationErrorTitle;
  String get sealGenerationErrorMessage => _strings.sealGenerationErrorMessage;
  String get sealGenerationLimitTitle => _strings.sealGenerationLimitTitle;
  String get sealGenerationLimitMessage => _strings.sealGenerationLimitMessage;
  String get sealGenerationAttemptLabel => _strings.sealGenerationAttemptLabel;
  String get sealGenerationStyleDetails => _strings.sealGenerationStyleDetails;
  String get sealGenerationErrorTip => _strings.sealGenerationErrorTip;
  String get sealGenerationLimitTip => _strings.sealGenerationLimitTip;
  String get adjustStyle => _strings.adjustStyle;
  String get sealVariantSelectionTitle => _strings.sealVariantSelectionTitle;
  String get sealVariantSelectionMessage =>
      _strings.sealVariantSelectionMessage;
  String get sealVariantSelectedBadge => _strings.sealVariantSelectedBadge;
  String get sealVariantSelectedTitle => _strings.sealVariantSelectedTitle;
  String get sealVariantSelectedMessage => _strings.sealVariantSelectedMessage;
  String get regenerateSeal => _strings.regenerateSeal;
  String get sealPreviewTitle => _strings.sealPreviewTitle;
  String get sealPreviewMessage => _strings.sealPreviewMessage;
  String get sealPreviewVariantLabel => _strings.sealPreviewVariantLabel;
  String get saveSeal => _strings.saveSeal;
  String get chooseStone => _strings.chooseStone;
  String get sealSavedTitle => _strings.sealSavedTitle;
  String get sealSavedHeading => _strings.sealSavedHeading;
  String get sealSavedMessage => _strings.sealSavedMessage;
  String get goToMySeals => _strings.goToMySeals;
  String get createAnotherSeal => _strings.createAnotherSeal;
  String get savedSeals => _strings.savedSeals;
  String get savedSealsDescription => _strings.savedSealsDescription;
  String get savedOnThisDevice => _strings.savedOnThisDevice;
  String get savedSealsLoadingTitle => _strings.savedSealsLoadingTitle;
  String get savedSealsLoadingMessage => _strings.savedSealsLoadingMessage;
  String get savedSealsLoadErrorTitle => _strings.savedSealsLoadErrorTitle;
  String get savedSealsLoadErrorMessage => _strings.savedSealsLoadErrorMessage;
  String get chooseSavedSeal => _strings.chooseSavedSeal;
  String get viewSealDetails => _strings.viewSealDetails;
  String get favoriteSeal => _strings.favoriteSeal;
  String get removeFavoriteSeal => _strings.removeFavoriteSeal;
  String get sealDetailTitle => _strings.sealDetailTitle;
  String get kanjiLabel => _strings.kanjiLabel;
  String get createdAtLabel => _strings.createdAtLabel;
  String get compareSavedSeals => _strings.compareSavedSeals;
  String get compareSavedSealsTitle => _strings.compareSavedSealsTitle;
  String get compareSavedSealsMessage => _strings.compareSavedSealsMessage;
  String get editSavedSeal => _strings.editSavedSeal;
  String get editSavedSealTitle => _strings.editSavedSealTitle;
  String get editSavedSealMessage => _strings.editSavedSealMessage;
  String get chooseSealForOrder => _strings.chooseSealForOrder;
  String get sealSelectedForOrderTitle => _strings.sealSelectedForOrderTitle;
  String get sealSelectedForOrderMessage =>
      _strings.sealSelectedForOrderMessage;
  String get sealSelectedForOrderAction => _strings.sealSelectedForOrderAction;
  String get deleteSavedSeal => _strings.deleteSavedSeal;
  String get deleteSealTitle => _strings.deleteSealTitle;
  String get deleteSealMessage => _strings.deleteSealMessage;
  String get deleteSealConfirm => _strings.deleteSealConfirm;
  String get cancel => _strings.cancel;
  String get close => _strings.close;
  String get browseStones => _strings.browseStones;
  String get browseStonesDescription => _strings.browseStonesDescription;
  String get noSavedSeals => _strings.noSavedSeals;
  String get noSavedSealsMessage => _strings.noSavedSealsMessage;
  String get stonesLoadingTitle => _strings.stonesLoadingTitle;
  String get stonesLoadingMessage => _strings.stonesLoadingMessage;
  String get stonesLoadErrorTitle => _strings.stonesLoadErrorTitle;
  String get stonesLoadErrorMessage => _strings.stonesLoadErrorMessage;
  String get noStonesLoaded => _strings.noStonesLoaded;
  String get noStonesLoadedMessage => _strings.noStonesLoadedMessage;
  String get stoneFiltersTitle => _strings.stoneFiltersTitle;
  String get stoneFilterAll => _strings.stoneFilterAll;
  String get stoneFilterMaterial => _strings.stoneFilterMaterial;
  String get stoneFilterColor => _strings.stoneFilterColor;
  String get stoneFilterPattern => _strings.stoneFilterPattern;
  String get stoneFilterAvailability => _strings.stoneFilterAvailability;
  String get stoneFilterReset => _strings.stoneFilterReset;
  String get noStonesMatchFilters => _strings.noStonesMatchFilters;
  String get noStonesMatchFiltersMessage =>
      _strings.noStonesMatchFiltersMessage;
  String get stoneSortTitle => _strings.stoneSortTitle;
  String get stoneSortAction => _strings.stoneSortAction;
  String get stoneSortRecommended => _strings.stoneSortRecommended;
  String get stoneSortNewest => _strings.stoneSortNewest;
  String get stoneSortPriceLowToHigh => _strings.stoneSortPriceLowToHigh;
  String get stoneSortPriceHighToLow => _strings.stoneSortPriceHighToLow;
  String get viewStoneDetails => _strings.viewStoneDetails;
  String get stoneDetailTitle => _strings.stoneDetailTitle;
  String get stoneDetailDescriptionTitle =>
      _strings.stoneDetailDescriptionTitle;
  String get stoneDetailStoryTitle => _strings.stoneDetailStoryTitle;
  String get stoneDetailSpecsTitle => _strings.stoneDetailSpecsTitle;
  String get stoneDetailNotesTitle => _strings.stoneDetailNotesTitle;
  String get stoneDetailNotesMessage => _strings.stoneDetailNotesMessage;
  String get stoneDetailMaterialLabel => _strings.stoneDetailMaterialLabel;
  String get stoneDetailSizeLabel => _strings.stoneDetailSizeLabel;
  String get stoneDetailColorLabel => _strings.stoneDetailColorLabel;
  String get stoneDetailPatternLabel => _strings.stoneDetailPatternLabel;
  String get stoneDetailShapeLabel => _strings.stoneDetailShapeLabel;
  String get stoneDetailTextureLabel => _strings.stoneDetailTextureLabel;
  String get stoneDetailStatusLabel => _strings.stoneDetailStatusLabel;
  String get selectStone => _strings.selectStone;
  String get selectStoneConfirmationTitle =>
      _strings.selectStoneConfirmationTitle;
  String get selectStoneConfirmationMessage =>
      _strings.selectStoneConfirmationMessage;
  String get selectStoneConfirm => _strings.selectStoneConfirm;
  String get stoneSelectedForOrderTitle => _strings.stoneSelectedForOrderTitle;
  String get stoneSelectedForOrderMessage =>
      _strings.stoneSelectedForOrderMessage;
  String get stoneSelectedForOrderAction =>
      _strings.stoneSelectedForOrderAction;
  String get soldOutStoneTitle => _strings.soldOutStoneTitle;
  String get soldOutStoneMessage => _strings.soldOutStoneMessage;
  String get stoneAvailable => _strings.stoneAvailable;
  String get stoneUnavailable => _strings.stoneUnavailable;
  String get order => _strings.order;
  String get noActiveDraft => _strings.noActiveDraft;
  String get noActiveDraftMessage => _strings.noActiveDraftMessage;
  String get reviewSelection => _strings.reviewSelection;
  String get orderMissingSealTitle => _strings.orderMissingSealTitle;
  String get orderMissingSealMessage => _strings.orderMissingSealMessage;
  String get orderMissingSealNotice => _strings.orderMissingSealNotice;
  String get orderChooseSealAction => _strings.orderChooseSealAction;
  String get orderChangeSealAction => _strings.orderChangeSealAction;
  String get orderMissingStoneTitle => _strings.orderMissingStoneTitle;
  String get orderMissingStoneMessage => _strings.orderMissingStoneMessage;
  String get orderChooseStoneAction => _strings.orderChooseStoneAction;
  String get orderChangeStoneAction => _strings.orderChangeStoneAction;
  String get orderReviewTitle => _strings.orderReviewTitle;
  String get orderReviewMessage => _strings.orderReviewMessage;
  String get orderItemPriceLabel => _strings.orderItemPriceLabel;
  String get orderShippingFeeLabel => _strings.orderShippingFeeLabel;
  String get orderShippingEstimateNote => _strings.orderShippingEstimateNote;
  String get orderTotalLabel => _strings.orderTotalLabel;
  String get orderCustomMadeNotice => _strings.orderCustomMadeNotice;
  String get continueToShipping => _strings.continueToShipping;
  String get checkoutInputTitle => _strings.checkoutInputTitle;
  String get checkoutInputMessage => _strings.checkoutInputMessage;
  String get checkoutContactTitle => _strings.checkoutContactTitle;
  String get checkoutShippingTitle => _strings.checkoutShippingTitle;
  String get checkoutOrderNoteTitle => _strings.checkoutOrderNoteTitle;
  String get checkoutFullNameLabel => _strings.checkoutFullNameLabel;
  String get checkoutFullNameHint => _strings.checkoutFullNameHint;
  String get checkoutPhoneLabel => _strings.checkoutPhoneLabel;
  String get checkoutPhoneHint => _strings.checkoutPhoneHint;
  String get checkoutCountryLabel => _strings.checkoutCountryLabel;
  String get checkoutPostalCodeLabel => _strings.checkoutPostalCodeLabel;
  String get checkoutPostalCodeHint => _strings.checkoutPostalCodeHint;
  String get checkoutAddressLine1Label => _strings.checkoutAddressLine1Label;
  String get checkoutAddressLine1Hint => _strings.checkoutAddressLine1Hint;
  String get checkoutAddressLine2Label => _strings.checkoutAddressLine2Label;
  String get checkoutAddressLine2Hint => _strings.checkoutAddressLine2Hint;
  String get checkoutCityLabel => _strings.checkoutCityLabel;
  String get checkoutCityHint => _strings.checkoutCityHint;
  String get checkoutStateLabel => _strings.checkoutStateLabel;
  String get checkoutStateHint => _strings.checkoutStateHint;
  String get checkoutOrderNoteLabel => _strings.checkoutOrderNoteLabel;
  String get checkoutOrderNoteHint => _strings.checkoutOrderNoteHint;
  String get checkoutInputSaveAction => _strings.checkoutInputSaveAction;
  String get checkoutInputSavingAction => _strings.checkoutInputSavingAction;
  String get checkoutInputSavedMessage => _strings.checkoutInputSavedMessage;
  String get checkoutValidationTitle => _strings.checkoutValidationTitle;
  String get checkoutValidationMessage => _strings.checkoutValidationMessage;
  String get checkoutEmailInvalidMessage =>
      _strings.checkoutEmailInvalidMessage;
  String get checkoutFullNameRequiredMessage =>
      _strings.checkoutFullNameRequiredMessage;
  String get checkoutPhoneInvalidMessage =>
      _strings.checkoutPhoneInvalidMessage;
  String get checkoutCountryRequiredMessage =>
      _strings.checkoutCountryRequiredMessage;
  String get checkoutPostalCodeRequiredMessage =>
      _strings.checkoutPostalCodeRequiredMessage;
  String get checkoutAddressLine1RequiredMessage =>
      _strings.checkoutAddressLine1RequiredMessage;
  String get checkoutCityRequiredMessage =>
      _strings.checkoutCityRequiredMessage;
  String get checkoutStateRequiredMessage =>
      _strings.checkoutStateRequiredMessage;
  String get orderConfirmationTitle => _strings.orderConfirmationTitle;
  String get orderConfirmationMessage => _strings.orderConfirmationMessage;
  String get orderConfirmationMissingInputMessage =>
      _strings.orderConfirmationMissingInputMessage;
  String get orderConfirmationCheckoutTitle =>
      _strings.orderConfirmationCheckoutTitle;
  String get orderConfirmationNoOrderNote =>
      _strings.orderConfirmationNoOrderNote;
  String get editCheckoutInformation => _strings.editCheckoutInformation;
  String get customMadeAgreementTitle => _strings.customMadeAgreementTitle;
  String get customMadeAgreementMessage => _strings.customMadeAgreementMessage;
  String get confirmKanjiAndDesignLabel => _strings.confirmKanjiAndDesignLabel;
  String get confirmCustomMadePolicyLabel =>
      _strings.confirmCustomMadePolicyLabel;
  String get orderConfirmationSecurePaymentNote =>
      _strings.orderConfirmationSecurePaymentNote;
  String get orderConfirmationAgreementRequiredMessage =>
      _strings.orderConfirmationAgreementRequiredMessage;
  String get orderConfirmationSavedMessage =>
      _strings.orderConfirmationSavedMessage;
  String get proceedToSecurePayment => _strings.proceedToSecurePayment;
  String get checkoutProcessingTitle => _strings.checkoutProcessingTitle;
  String get checkoutProcessingMessage => _strings.checkoutProcessingMessage;
  String get checkoutProcessingOrderStep =>
      _strings.checkoutProcessingOrderStep;
  String get checkoutProcessingSessionStep =>
      _strings.checkoutProcessingSessionStep;
  String get checkoutProcessingReadyTitle =>
      _strings.checkoutProcessingReadyTitle;
  String get checkoutProcessingReadyMessage =>
      _strings.checkoutProcessingReadyMessage;
  String get checkoutProcessingErrorTitle =>
      _strings.checkoutProcessingErrorTitle;
  String get checkoutProcessingErrorMessage =>
      _strings.checkoutProcessingErrorMessage;
  String get stripeCheckoutTitle => _strings.stripeCheckoutTitle;
  String get stripeCheckoutOpeningTitle => _strings.stripeCheckoutOpeningTitle;
  String get stripeCheckoutOpeningMessage =>
      _strings.stripeCheckoutOpeningMessage;
  String get stripeCheckoutWaitingTitle => _strings.stripeCheckoutWaitingTitle;
  String get stripeCheckoutWaitingMessage =>
      _strings.stripeCheckoutWaitingMessage;
  String get stripeCheckoutReturnedTitle =>
      _strings.stripeCheckoutReturnedTitle;
  String get stripeCheckoutReturnedMessage =>
      _strings.stripeCheckoutReturnedMessage;
  String get stripeCheckoutCanceledTitle =>
      _strings.stripeCheckoutCanceledTitle;
  String get stripeCheckoutCanceledMessage =>
      _strings.stripeCheckoutCanceledMessage;
  String get stripeCheckoutReturnFailedTitle =>
      _strings.stripeCheckoutReturnFailedTitle;
  String get stripeCheckoutReturnFailedMessage =>
      _strings.stripeCheckoutReturnFailedMessage;
  String get stripeCheckoutLaunchFailedTitle =>
      _strings.stripeCheckoutLaunchFailedTitle;
  String get stripeCheckoutLaunchFailedMessage =>
      _strings.stripeCheckoutLaunchFailedMessage;
  String get stripeCheckoutOpenAction => _strings.stripeCheckoutOpenAction;
  String get stripeCheckoutRetryAction => _strings.stripeCheckoutRetryAction;
  String get stripeCheckoutSecureNote => _strings.stripeCheckoutSecureNote;
  String get stripeCheckoutReturnOrderIdLabel =>
      _strings.stripeCheckoutReturnOrderIdLabel;
  String get paymentStatusTitle => _strings.paymentStatusTitle;
  String get paymentStatusCheckingTitle => _strings.paymentStatusCheckingTitle;
  String get paymentStatusCheckingMessage =>
      _strings.paymentStatusCheckingMessage;
  String get paymentStatusPaidTitle => _strings.paymentStatusPaidTitle;
  String get paymentStatusPaidMessage => _strings.paymentStatusPaidMessage;
  String get paymentStatusPendingTitle => _strings.paymentStatusPendingTitle;
  String get paymentStatusPendingMessage =>
      _strings.paymentStatusPendingMessage;
  String get paymentStatusPendingNotice => _strings.paymentStatusPendingNotice;
  String get paymentStatusFailedTitle => _strings.paymentStatusFailedTitle;
  String get paymentStatusFailedMessage => _strings.paymentStatusFailedMessage;
  String get orderCompleteTitle => _strings.orderCompleteTitle;
  String get orderCompleteStatusTitle => _strings.orderCompleteStatusTitle;
  String get orderCompleteMessage => _strings.orderCompleteMessage;
  String get orderCompleteStatusLabel => _strings.orderCompleteStatusLabel;
  String get orderCompleteStatusValue => _strings.orderCompleteStatusValue;
  String get orderCompleteEmailMessage => _strings.orderCompleteEmailMessage;
  String get orderCompleteSummaryTitle => _strings.orderCompleteSummaryTitle;
  String get orderCompleteLookupAction => _strings.orderCompleteLookupAction;
  String get orderCompleteBackToDesignAction =>
      _strings.orderCompleteBackToDesignAction;
  String get emailSentNoticeTitle => _strings.emailSentNoticeTitle;
  String get emailSentNoticeMessage => _strings.emailSentNoticeMessage;
  String get orderEmailMissingGuideTitle =>
      _strings.orderEmailMissingGuideTitle;
  String get orderEmailMissingGuideMessage =>
      _strings.orderEmailMissingGuideMessage;
  String get orderEmailMissingSpamCheck => _strings.orderEmailMissingSpamCheck;
  String get orderEmailMissingAddressCheck =>
      _strings.orderEmailMissingAddressCheck;
  String get orderEmailMissingDeliveryWait =>
      _strings.orderEmailMissingDeliveryWait;
  String get orderEmailMissingContactSupport =>
      _strings.orderEmailMissingContactSupport;
  String get contactSupportPromptTitle => _strings.contactSupportPromptTitle;
  String get contactSupportPromptMessage =>
      _strings.contactSupportPromptMessage;
  String get contactSupportPromptAction => _strings.contactSupportPromptAction;
  String get orderLookup => _strings.orderLookup;
  String get orderNo => _strings.orderNo;
  String get orderNoHint => _strings.orderNoHint;
  String get email => _strings.email;
  String get emailHint => _strings.emailHint;
  String get lookupOrder => _strings.lookupOrder;
  String get orderLookupLoadingTitle => _strings.orderLookupLoadingTitle;
  String get orderLookupLoadingMessage => _strings.orderLookupLoadingMessage;
  String get orderLookupNotFoundTitle => _strings.orderLookupNotFoundTitle;
  String get orderLookupNotFoundMessage => _strings.orderLookupNotFoundMessage;
  String get orderLookupErrorTitle => _strings.orderLookupErrorTitle;
  String get orderLookupErrorMessage => _strings.orderLookupErrorMessage;
  String get orderLookupResultTitle => _strings.orderLookupResultTitle;
  String get orderLookupResultMessage => _strings.orderLookupResultMessage;
  String get orderLookupOrderStatusLabel =>
      _strings.orderLookupOrderStatusLabel;
  String get orderLookupProgressTitle => _strings.orderLookupProgressTitle;
  String get orderLookupProductionStatusLabel =>
      _strings.orderLookupProductionStatusLabel;
  String get orderLookupShippingStatusLabel =>
      _strings.orderLookupShippingStatusLabel;
  String get orderLookupFulfillmentStatusLabel =>
      _strings.orderLookupFulfillmentStatusLabel;
  String get orderLookupTrackingNumberLabel =>
      _strings.orderLookupTrackingNumberLabel;
  String get orderLookupTrackingDetailsTitle =>
      _strings.orderLookupTrackingDetailsTitle;
  String get orderLookupCarrierLabel => _strings.orderLookupCarrierLabel;
  String get orderLookupShippedAtLabel => _strings.orderLookupShippedAtLabel;
  String get orderLookupUpdatedAtLabel => _strings.orderLookupUpdatedAtLabel;
  String get orderLookupNoTrackingValue => _strings.orderLookupNoTrackingValue;
  String get orderLookupOrderDateLabel => _strings.orderLookupOrderDateLabel;
  String get orderLookupContentTitle => _strings.orderLookupContentTitle;
  String get orderLookupSelectedSealLabel =>
      _strings.orderLookupSelectedSealLabel;
  String get orderLookupGemstoneLabel => _strings.orderLookupGemstoneLabel;
  String get orderLookupLookupAnotherAction =>
      _strings.orderLookupLookupAnotherAction;
  String get language => _strings.language;
  String get about => _strings.about;
  String get howItWorks => _strings.howItWorks;
  String get faq => _strings.faq;
  String get privacy => _strings.privacy;
  String get terms => _strings.terms;
  String get contact => _strings.contact;
  String get version => _strings.version;
  String get settingsLanguageTitle => _strings.settingsLanguageTitle;
  String get settingsLanguageMessage => _strings.settingsLanguageMessage;
  String get settingsLanguageEnglish => _strings.settingsLanguageEnglish;
  String get settingsLanguageJapanese => _strings.settingsLanguageJapanese;
  String get settingsFaqIntro => _strings.settingsFaqIntro;
  String get settingsVersionTitle => _strings.settingsVersionTitle;
  String settingsVersionMessage(String version) {
    return _strings.settingsVersionMessageTemplate.replaceAll(
      '{version}',
      version,
    );
  }
}

extension HankoLocalizationsBuildContext on BuildContext {
  HankoLocalizations get l10n => HankoLocalizations.of(this);
}

class _HankoLocalizationsDelegate
    extends LocalizationsDelegate<HankoLocalizations> {
  const _HankoLocalizationsDelegate();

  @override
  bool isSupported(Locale locale) {
    return _localizedValues.containsKey(locale.languageCode);
  }

  @override
  Future<HankoLocalizations> load(Locale locale) {
    return SynchronousFuture(HankoLocalizations.forLocale(locale));
  }

  @override
  bool shouldReload(covariant LocalizationsDelegate<HankoLocalizations> old) {
    return false;
  }
}

class _HankoStrings {
  const _HankoStrings({
    required this.appTitle,
    required this.splashPreparing,
    required this.onboardingTitle,
    required this.onboardingHeroTitle,
    required this.onboardingTagline,
    required this.onboardingKanjiTitle,
    required this.onboardingKanjiMessage,
    required this.onboardingAiTitle,
    required this.onboardingAiMessage,
    required this.onboardingStoneTitle,
    required this.onboardingStoneMessage,
    required this.onboardingStorageTitle,
    required this.onboardingStorageMessage,
    required this.onboardingGetStarted,
    required this.onboardingSaving,
    required this.onboardingSkip,
    required this.onboardingSaveError,
    required this.design,
    required this.mySeals,
    required this.stones,
    required this.settings,
    required this.createCustomSeal,
    required this.customSealDescription,
    required this.startDesigning,
    required this.designNameTitle,
    required this.designNameIntro,
    required this.designNameLabel,
    required this.designNameHint,
    required this.designNameHelp,
    required this.designGenderLabel,
    required this.designGenderUnspecified,
    required this.designGenderMale,
    required this.designGenderFemale,
    required this.designKanjiStyleLabel,
    required this.designKanjiStyleJapanese,
    required this.designKanjiStyleChinese,
    required this.designKanjiStyleTaiwanese,
    required this.suggestKanji,
    required this.designKanjiTipTitle,
    required this.designKanjiTipMessage,
    required this.designCandidateReadyTitle,
    required this.designCandidateReadyMessage,
    required this.designRequestDetails,
    required this.editName,
    required this.designLoadingTitle,
    required this.designLoadingMessage,
    required this.designInvalidNameSummary,
    required this.designInvalidNameMessage,
    required this.designSuggestionErrorTitle,
    required this.designSuggestionErrorMessage,
    required this.designNoKanjiTitle,
    required this.designNoKanjiMessage,
    required this.designNoKanjiRuleCharacters,
    required this.designNoKanjiRuleCommon,
    required this.designNoKanjiRuleEngraving,
    required this.designErrorTip,
    required this.designNoKanjiTip,
    required this.tryAgain,
    required this.back,
    required this.commonNetworkErrorTitle,
    required this.commonNetworkErrorMessage,
    required this.commonServerErrorTitle,
    required this.commonServerErrorMessage,
    required this.storageErrorTitle,
    required this.storageErrorMessage,
    required this.deepLinkErrorTitle,
    required this.deepLinkErrorMessage,
    required this.maintenanceTitle,
    required this.maintenanceMessage,
    required this.appUpdateRequiredTitle,
    required this.appUpdateRequiredMessage,
    required this.appUpdateRequiredAction,
    required this.commonGenericErrorTitle,
    required this.commonGenericErrorMessage,
    required this.kanjiSuggestionsTitle,
    required this.kanjiSuggestionsMessage,
    required this.kanjiCandidateDetailTitle,
    required this.kanjiReadingLabel,
    required this.kanjiMeaningLabel,
    required this.kanjiImpressionLabel,
    required this.kanjiReasonLabel,
    required this.kanjiCharacterCountLabel,
    required this.kanjiStrokeComplexityLabel,
    required this.kanjiEngravingSuitabilityLabel,
    required this.selectKanji,
    required this.kanjiSelectedTitle,
    required this.kanjiSelectedMessage,
    required this.sealStyleTitle,
    required this.sealStyleMessage,
    required this.sealStyleSelectedKanjiLabel,
    required this.sealShapeLabel,
    required this.sealShapeSquare,
    required this.sealShapeRound,
    required this.sealStyleNameLabel,
    required this.sealStyleTraditional,
    required this.sealStyleElegant,
    required this.sealStyleSoft,
    required this.sealStyleBold,
    required this.sealStrokeWeightLabel,
    required this.sealStrokeStandard,
    required this.sealStrokeBold,
    required this.sealBalanceLabel,
    required this.sealBalanceAiry,
    required this.sealBalanceBalanced,
    required this.sealBalanceDense,
    required this.sealStyleSummaryTitle,
    required this.confirmStyle,
    required this.sealStyleConfirmedTitle,
    required this.sealStyleConfirmedMessage,
    required this.generateSeal,
    required this.sealGenerationLoadingTitle,
    required this.sealGenerationLoadingMessage,
    required this.sealGenerationLoadingDetail,
    required this.sealGenerationErrorTitle,
    required this.sealGenerationErrorMessage,
    required this.sealGenerationLimitTitle,
    required this.sealGenerationLimitMessage,
    required this.sealGenerationAttemptLabel,
    required this.sealGenerationStyleDetails,
    required this.sealGenerationErrorTip,
    required this.sealGenerationLimitTip,
    required this.adjustStyle,
    required this.sealVariantSelectionTitle,
    required this.sealVariantSelectionMessage,
    required this.sealVariantSelectedBadge,
    required this.sealVariantSelectedTitle,
    required this.sealVariantSelectedMessage,
    required this.regenerateSeal,
    required this.sealPreviewTitle,
    required this.sealPreviewMessage,
    required this.sealPreviewVariantLabel,
    required this.saveSeal,
    required this.chooseStone,
    required this.sealSavedTitle,
    required this.sealSavedHeading,
    required this.sealSavedMessage,
    required this.goToMySeals,
    required this.createAnotherSeal,
    required this.savedSeals,
    required this.savedSealsDescription,
    required this.savedOnThisDevice,
    required this.savedSealsLoadingTitle,
    required this.savedSealsLoadingMessage,
    required this.savedSealsLoadErrorTitle,
    required this.savedSealsLoadErrorMessage,
    required this.chooseSavedSeal,
    required this.viewSealDetails,
    required this.favoriteSeal,
    required this.removeFavoriteSeal,
    required this.sealDetailTitle,
    required this.kanjiLabel,
    required this.createdAtLabel,
    required this.compareSavedSeals,
    required this.compareSavedSealsTitle,
    required this.compareSavedSealsMessage,
    required this.editSavedSeal,
    required this.editSavedSealTitle,
    required this.editSavedSealMessage,
    required this.chooseSealForOrder,
    required this.sealSelectedForOrderTitle,
    required this.sealSelectedForOrderMessage,
    required this.sealSelectedForOrderAction,
    required this.deleteSavedSeal,
    required this.deleteSealTitle,
    required this.deleteSealMessage,
    required this.deleteSealConfirm,
    required this.cancel,
    required this.close,
    required this.browseStones,
    required this.browseStonesDescription,
    required this.noSavedSeals,
    required this.noSavedSealsMessage,
    required this.stonesLoadingTitle,
    required this.stonesLoadingMessage,
    required this.stonesLoadErrorTitle,
    required this.stonesLoadErrorMessage,
    required this.noStonesLoaded,
    required this.noStonesLoadedMessage,
    required this.stoneFiltersTitle,
    required this.stoneFilterAll,
    required this.stoneFilterMaterial,
    required this.stoneFilterColor,
    required this.stoneFilterPattern,
    required this.stoneFilterAvailability,
    required this.stoneFilterReset,
    required this.noStonesMatchFilters,
    required this.noStonesMatchFiltersMessage,
    required this.stoneSortTitle,
    required this.stoneSortAction,
    required this.stoneSortRecommended,
    required this.stoneSortNewest,
    required this.stoneSortPriceLowToHigh,
    required this.stoneSortPriceHighToLow,
    required this.viewStoneDetails,
    required this.stoneDetailTitle,
    required this.stoneDetailDescriptionTitle,
    required this.stoneDetailStoryTitle,
    required this.stoneDetailSpecsTitle,
    required this.stoneDetailNotesTitle,
    required this.stoneDetailNotesMessage,
    required this.stoneDetailMaterialLabel,
    required this.stoneDetailSizeLabel,
    required this.stoneDetailColorLabel,
    required this.stoneDetailPatternLabel,
    required this.stoneDetailShapeLabel,
    required this.stoneDetailTextureLabel,
    required this.stoneDetailStatusLabel,
    required this.selectStone,
    required this.selectStoneConfirmationTitle,
    required this.selectStoneConfirmationMessage,
    required this.selectStoneConfirm,
    required this.stoneSelectedForOrderTitle,
    required this.stoneSelectedForOrderMessage,
    required this.stoneSelectedForOrderAction,
    required this.soldOutStoneTitle,
    required this.soldOutStoneMessage,
    required this.stoneAvailable,
    required this.stoneUnavailable,
    required this.order,
    required this.noActiveDraft,
    required this.noActiveDraftMessage,
    required this.reviewSelection,
    required this.orderMissingSealTitle,
    required this.orderMissingSealMessage,
    required this.orderMissingSealNotice,
    required this.orderChooseSealAction,
    required this.orderChangeSealAction,
    required this.orderMissingStoneTitle,
    required this.orderMissingStoneMessage,
    required this.orderChooseStoneAction,
    required this.orderChangeStoneAction,
    required this.orderReviewTitle,
    required this.orderReviewMessage,
    required this.orderItemPriceLabel,
    required this.orderShippingFeeLabel,
    required this.orderShippingEstimateNote,
    required this.orderTotalLabel,
    required this.orderCustomMadeNotice,
    required this.continueToShipping,
    required this.checkoutInputTitle,
    required this.checkoutInputMessage,
    required this.checkoutContactTitle,
    required this.checkoutShippingTitle,
    required this.checkoutOrderNoteTitle,
    required this.checkoutFullNameLabel,
    required this.checkoutFullNameHint,
    required this.checkoutPhoneLabel,
    required this.checkoutPhoneHint,
    required this.checkoutCountryLabel,
    required this.checkoutPostalCodeLabel,
    required this.checkoutPostalCodeHint,
    required this.checkoutAddressLine1Label,
    required this.checkoutAddressLine1Hint,
    required this.checkoutAddressLine2Label,
    required this.checkoutAddressLine2Hint,
    required this.checkoutCityLabel,
    required this.checkoutCityHint,
    required this.checkoutStateLabel,
    required this.checkoutStateHint,
    required this.checkoutOrderNoteLabel,
    required this.checkoutOrderNoteHint,
    required this.checkoutInputSaveAction,
    required this.checkoutInputSavingAction,
    required this.checkoutInputSavedMessage,
    required this.checkoutValidationTitle,
    required this.checkoutValidationMessage,
    required this.checkoutEmailInvalidMessage,
    required this.checkoutFullNameRequiredMessage,
    required this.checkoutPhoneInvalidMessage,
    required this.checkoutCountryRequiredMessage,
    required this.checkoutPostalCodeRequiredMessage,
    required this.checkoutAddressLine1RequiredMessage,
    required this.checkoutCityRequiredMessage,
    required this.checkoutStateRequiredMessage,
    required this.orderConfirmationTitle,
    required this.orderConfirmationMessage,
    required this.orderConfirmationMissingInputMessage,
    required this.orderConfirmationCheckoutTitle,
    required this.orderConfirmationNoOrderNote,
    required this.editCheckoutInformation,
    required this.customMadeAgreementTitle,
    required this.customMadeAgreementMessage,
    required this.confirmKanjiAndDesignLabel,
    required this.confirmCustomMadePolicyLabel,
    required this.orderConfirmationSecurePaymentNote,
    required this.orderConfirmationAgreementRequiredMessage,
    required this.orderConfirmationSavedMessage,
    required this.proceedToSecurePayment,
    required this.checkoutProcessingTitle,
    required this.checkoutProcessingMessage,
    required this.checkoutProcessingOrderStep,
    required this.checkoutProcessingSessionStep,
    required this.checkoutProcessingReadyTitle,
    required this.checkoutProcessingReadyMessage,
    required this.checkoutProcessingErrorTitle,
    required this.checkoutProcessingErrorMessage,
    required this.stripeCheckoutTitle,
    required this.stripeCheckoutOpeningTitle,
    required this.stripeCheckoutOpeningMessage,
    required this.stripeCheckoutWaitingTitle,
    required this.stripeCheckoutWaitingMessage,
    required this.stripeCheckoutReturnedTitle,
    required this.stripeCheckoutReturnedMessage,
    required this.stripeCheckoutCanceledTitle,
    required this.stripeCheckoutCanceledMessage,
    required this.stripeCheckoutReturnFailedTitle,
    required this.stripeCheckoutReturnFailedMessage,
    required this.stripeCheckoutLaunchFailedTitle,
    required this.stripeCheckoutLaunchFailedMessage,
    required this.stripeCheckoutOpenAction,
    required this.stripeCheckoutRetryAction,
    required this.stripeCheckoutSecureNote,
    required this.stripeCheckoutReturnOrderIdLabel,
    required this.paymentStatusTitle,
    required this.paymentStatusCheckingTitle,
    required this.paymentStatusCheckingMessage,
    required this.paymentStatusPaidTitle,
    required this.paymentStatusPaidMessage,
    required this.paymentStatusPendingTitle,
    required this.paymentStatusPendingMessage,
    required this.paymentStatusPendingNotice,
    required this.paymentStatusFailedTitle,
    required this.paymentStatusFailedMessage,
    required this.orderCompleteTitle,
    required this.orderCompleteStatusTitle,
    required this.orderCompleteMessage,
    required this.orderCompleteStatusLabel,
    required this.orderCompleteStatusValue,
    required this.orderCompleteEmailMessage,
    required this.orderCompleteSummaryTitle,
    required this.orderCompleteLookupAction,
    required this.orderCompleteBackToDesignAction,
    required this.emailSentNoticeTitle,
    required this.emailSentNoticeMessage,
    required this.orderEmailMissingGuideTitle,
    required this.orderEmailMissingGuideMessage,
    required this.orderEmailMissingSpamCheck,
    required this.orderEmailMissingAddressCheck,
    required this.orderEmailMissingDeliveryWait,
    required this.orderEmailMissingContactSupport,
    required this.contactSupportPromptTitle,
    required this.contactSupportPromptMessage,
    required this.contactSupportPromptAction,
    required this.orderLookup,
    required this.orderNo,
    required this.orderNoHint,
    required this.email,
    required this.emailHint,
    required this.lookupOrder,
    required this.orderLookupLoadingTitle,
    required this.orderLookupLoadingMessage,
    required this.orderLookupNotFoundTitle,
    required this.orderLookupNotFoundMessage,
    required this.orderLookupErrorTitle,
    required this.orderLookupErrorMessage,
    required this.orderLookupResultTitle,
    required this.orderLookupResultMessage,
    required this.orderLookupOrderStatusLabel,
    required this.orderLookupProgressTitle,
    required this.orderLookupProductionStatusLabel,
    required this.orderLookupShippingStatusLabel,
    required this.orderLookupFulfillmentStatusLabel,
    required this.orderLookupTrackingNumberLabel,
    required this.orderLookupTrackingDetailsTitle,
    required this.orderLookupCarrierLabel,
    required this.orderLookupShippedAtLabel,
    required this.orderLookupUpdatedAtLabel,
    required this.orderLookupNoTrackingValue,
    required this.orderLookupOrderDateLabel,
    required this.orderLookupContentTitle,
    required this.orderLookupSelectedSealLabel,
    required this.orderLookupGemstoneLabel,
    required this.orderLookupLookupAnotherAction,
    required this.language,
    required this.about,
    required this.howItWorks,
    required this.faq,
    required this.privacy,
    required this.terms,
    required this.contact,
    required this.version,
    required this.settingsLanguageTitle,
    required this.settingsLanguageMessage,
    required this.settingsLanguageEnglish,
    required this.settingsLanguageJapanese,
    required this.settingsFaqIntro,
    required this.settingsVersionTitle,
    required this.settingsVersionMessageTemplate,
  });

  final String appTitle;
  final String splashPreparing;
  final String onboardingTitle;
  final String onboardingHeroTitle;
  final String onboardingTagline;
  final String onboardingKanjiTitle;
  final String onboardingKanjiMessage;
  final String onboardingAiTitle;
  final String onboardingAiMessage;
  final String onboardingStoneTitle;
  final String onboardingStoneMessage;
  final String onboardingStorageTitle;
  final String onboardingStorageMessage;
  final String onboardingGetStarted;
  final String onboardingSaving;
  final String onboardingSkip;
  final String onboardingSaveError;
  final String design;
  final String mySeals;
  final String stones;
  final String settings;
  final String createCustomSeal;
  final String customSealDescription;
  final String startDesigning;
  final String designNameTitle;
  final String designNameIntro;
  final String designNameLabel;
  final String designNameHint;
  final String designNameHelp;
  final String designGenderLabel;
  final String designGenderUnspecified;
  final String designGenderMale;
  final String designGenderFemale;
  final String designKanjiStyleLabel;
  final String designKanjiStyleJapanese;
  final String designKanjiStyleChinese;
  final String designKanjiStyleTaiwanese;
  final String suggestKanji;
  final String designKanjiTipTitle;
  final String designKanjiTipMessage;
  final String designCandidateReadyTitle;
  final String designCandidateReadyMessage;
  final String designRequestDetails;
  final String editName;
  final String designLoadingTitle;
  final String designLoadingMessage;
  final String designInvalidNameSummary;
  final String designInvalidNameMessage;
  final String designSuggestionErrorTitle;
  final String designSuggestionErrorMessage;
  final String designNoKanjiTitle;
  final String designNoKanjiMessage;
  final String designNoKanjiRuleCharacters;
  final String designNoKanjiRuleCommon;
  final String designNoKanjiRuleEngraving;
  final String designErrorTip;
  final String designNoKanjiTip;
  final String tryAgain;
  final String back;
  final String commonNetworkErrorTitle;
  final String commonNetworkErrorMessage;
  final String commonServerErrorTitle;
  final String commonServerErrorMessage;
  final String storageErrorTitle;
  final String storageErrorMessage;
  final String deepLinkErrorTitle;
  final String deepLinkErrorMessage;
  final String maintenanceTitle;
  final String maintenanceMessage;
  final String appUpdateRequiredTitle;
  final String appUpdateRequiredMessage;
  final String appUpdateRequiredAction;
  final String commonGenericErrorTitle;
  final String commonGenericErrorMessage;
  final String kanjiSuggestionsTitle;
  final String kanjiSuggestionsMessage;
  final String kanjiCandidateDetailTitle;
  final String kanjiReadingLabel;
  final String kanjiMeaningLabel;
  final String kanjiImpressionLabel;
  final String kanjiReasonLabel;
  final String kanjiCharacterCountLabel;
  final String kanjiStrokeComplexityLabel;
  final String kanjiEngravingSuitabilityLabel;
  final String selectKanji;
  final String kanjiSelectedTitle;
  final String kanjiSelectedMessage;
  final String sealStyleTitle;
  final String sealStyleMessage;
  final String sealStyleSelectedKanjiLabel;
  final String sealShapeLabel;
  final String sealShapeSquare;
  final String sealShapeRound;
  final String sealStyleNameLabel;
  final String sealStyleTraditional;
  final String sealStyleElegant;
  final String sealStyleSoft;
  final String sealStyleBold;
  final String sealStrokeWeightLabel;
  final String sealStrokeStandard;
  final String sealStrokeBold;
  final String sealBalanceLabel;
  final String sealBalanceAiry;
  final String sealBalanceBalanced;
  final String sealBalanceDense;
  final String sealStyleSummaryTitle;
  final String confirmStyle;
  final String sealStyleConfirmedTitle;
  final String sealStyleConfirmedMessage;
  final String generateSeal;
  final String sealGenerationLoadingTitle;
  final String sealGenerationLoadingMessage;
  final String sealGenerationLoadingDetail;
  final String sealGenerationErrorTitle;
  final String sealGenerationErrorMessage;
  final String sealGenerationLimitTitle;
  final String sealGenerationLimitMessage;
  final String sealGenerationAttemptLabel;
  final String sealGenerationStyleDetails;
  final String sealGenerationErrorTip;
  final String sealGenerationLimitTip;
  final String adjustStyle;
  final String sealVariantSelectionTitle;
  final String sealVariantSelectionMessage;
  final String sealVariantSelectedBadge;
  final String sealVariantSelectedTitle;
  final String sealVariantSelectedMessage;
  final String regenerateSeal;
  final String sealPreviewTitle;
  final String sealPreviewMessage;
  final String sealPreviewVariantLabel;
  final String saveSeal;
  final String chooseStone;
  final String sealSavedTitle;
  final String sealSavedHeading;
  final String sealSavedMessage;
  final String goToMySeals;
  final String createAnotherSeal;
  final String savedSeals;
  final String savedSealsDescription;
  final String savedOnThisDevice;
  final String savedSealsLoadingTitle;
  final String savedSealsLoadingMessage;
  final String savedSealsLoadErrorTitle;
  final String savedSealsLoadErrorMessage;
  final String chooseSavedSeal;
  final String viewSealDetails;
  final String favoriteSeal;
  final String removeFavoriteSeal;
  final String sealDetailTitle;
  final String kanjiLabel;
  final String createdAtLabel;
  final String compareSavedSeals;
  final String compareSavedSealsTitle;
  final String compareSavedSealsMessage;
  final String editSavedSeal;
  final String editSavedSealTitle;
  final String editSavedSealMessage;
  final String chooseSealForOrder;
  final String sealSelectedForOrderTitle;
  final String sealSelectedForOrderMessage;
  final String sealSelectedForOrderAction;
  final String deleteSavedSeal;
  final String deleteSealTitle;
  final String deleteSealMessage;
  final String deleteSealConfirm;
  final String cancel;
  final String close;
  final String browseStones;
  final String browseStonesDescription;
  final String noSavedSeals;
  final String noSavedSealsMessage;
  final String stonesLoadingTitle;
  final String stonesLoadingMessage;
  final String stonesLoadErrorTitle;
  final String stonesLoadErrorMessage;
  final String noStonesLoaded;
  final String noStonesLoadedMessage;
  final String stoneFiltersTitle;
  final String stoneFilterAll;
  final String stoneFilterMaterial;
  final String stoneFilterColor;
  final String stoneFilterPattern;
  final String stoneFilterAvailability;
  final String stoneFilterReset;
  final String noStonesMatchFilters;
  final String noStonesMatchFiltersMessage;
  final String stoneSortTitle;
  final String stoneSortAction;
  final String stoneSortRecommended;
  final String stoneSortNewest;
  final String stoneSortPriceLowToHigh;
  final String stoneSortPriceHighToLow;
  final String viewStoneDetails;
  final String stoneDetailTitle;
  final String stoneDetailDescriptionTitle;
  final String stoneDetailStoryTitle;
  final String stoneDetailSpecsTitle;
  final String stoneDetailNotesTitle;
  final String stoneDetailNotesMessage;
  final String stoneDetailMaterialLabel;
  final String stoneDetailSizeLabel;
  final String stoneDetailColorLabel;
  final String stoneDetailPatternLabel;
  final String stoneDetailShapeLabel;
  final String stoneDetailTextureLabel;
  final String stoneDetailStatusLabel;
  final String selectStone;
  final String selectStoneConfirmationTitle;
  final String selectStoneConfirmationMessage;
  final String selectStoneConfirm;
  final String stoneSelectedForOrderTitle;
  final String stoneSelectedForOrderMessage;
  final String stoneSelectedForOrderAction;
  final String soldOutStoneTitle;
  final String soldOutStoneMessage;
  final String stoneAvailable;
  final String stoneUnavailable;
  final String order;
  final String noActiveDraft;
  final String noActiveDraftMessage;
  final String reviewSelection;
  final String orderMissingSealTitle;
  final String orderMissingSealMessage;
  final String orderMissingSealNotice;
  final String orderChooseSealAction;
  final String orderChangeSealAction;
  final String orderMissingStoneTitle;
  final String orderMissingStoneMessage;
  final String orderChooseStoneAction;
  final String orderChangeStoneAction;
  final String orderReviewTitle;
  final String orderReviewMessage;
  final String orderItemPriceLabel;
  final String orderShippingFeeLabel;
  final String orderShippingEstimateNote;
  final String orderTotalLabel;
  final String orderCustomMadeNotice;
  final String continueToShipping;
  final String checkoutInputTitle;
  final String checkoutInputMessage;
  final String checkoutContactTitle;
  final String checkoutShippingTitle;
  final String checkoutOrderNoteTitle;
  final String checkoutFullNameLabel;
  final String checkoutFullNameHint;
  final String checkoutPhoneLabel;
  final String checkoutPhoneHint;
  final String checkoutCountryLabel;
  final String checkoutPostalCodeLabel;
  final String checkoutPostalCodeHint;
  final String checkoutAddressLine1Label;
  final String checkoutAddressLine1Hint;
  final String checkoutAddressLine2Label;
  final String checkoutAddressLine2Hint;
  final String checkoutCityLabel;
  final String checkoutCityHint;
  final String checkoutStateLabel;
  final String checkoutStateHint;
  final String checkoutOrderNoteLabel;
  final String checkoutOrderNoteHint;
  final String checkoutInputSaveAction;
  final String checkoutInputSavingAction;
  final String checkoutInputSavedMessage;
  final String checkoutValidationTitle;
  final String checkoutValidationMessage;
  final String checkoutEmailInvalidMessage;
  final String checkoutFullNameRequiredMessage;
  final String checkoutPhoneInvalidMessage;
  final String checkoutCountryRequiredMessage;
  final String checkoutPostalCodeRequiredMessage;
  final String checkoutAddressLine1RequiredMessage;
  final String checkoutCityRequiredMessage;
  final String checkoutStateRequiredMessage;
  final String orderConfirmationTitle;
  final String orderConfirmationMessage;
  final String orderConfirmationMissingInputMessage;
  final String orderConfirmationCheckoutTitle;
  final String orderConfirmationNoOrderNote;
  final String editCheckoutInformation;
  final String customMadeAgreementTitle;
  final String customMadeAgreementMessage;
  final String confirmKanjiAndDesignLabel;
  final String confirmCustomMadePolicyLabel;
  final String orderConfirmationSecurePaymentNote;
  final String orderConfirmationAgreementRequiredMessage;
  final String orderConfirmationSavedMessage;
  final String proceedToSecurePayment;
  final String checkoutProcessingTitle;
  final String checkoutProcessingMessage;
  final String checkoutProcessingOrderStep;
  final String checkoutProcessingSessionStep;
  final String checkoutProcessingReadyTitle;
  final String checkoutProcessingReadyMessage;
  final String checkoutProcessingErrorTitle;
  final String checkoutProcessingErrorMessage;
  final String stripeCheckoutTitle;
  final String stripeCheckoutOpeningTitle;
  final String stripeCheckoutOpeningMessage;
  final String stripeCheckoutWaitingTitle;
  final String stripeCheckoutWaitingMessage;
  final String stripeCheckoutReturnedTitle;
  final String stripeCheckoutReturnedMessage;
  final String stripeCheckoutCanceledTitle;
  final String stripeCheckoutCanceledMessage;
  final String stripeCheckoutReturnFailedTitle;
  final String stripeCheckoutReturnFailedMessage;
  final String stripeCheckoutLaunchFailedTitle;
  final String stripeCheckoutLaunchFailedMessage;
  final String stripeCheckoutOpenAction;
  final String stripeCheckoutRetryAction;
  final String stripeCheckoutSecureNote;
  final String stripeCheckoutReturnOrderIdLabel;
  final String paymentStatusTitle;
  final String paymentStatusCheckingTitle;
  final String paymentStatusCheckingMessage;
  final String paymentStatusPaidTitle;
  final String paymentStatusPaidMessage;
  final String paymentStatusPendingTitle;
  final String paymentStatusPendingMessage;
  final String paymentStatusPendingNotice;
  final String paymentStatusFailedTitle;
  final String paymentStatusFailedMessage;
  final String orderCompleteTitle;
  final String orderCompleteStatusTitle;
  final String orderCompleteMessage;
  final String orderCompleteStatusLabel;
  final String orderCompleteStatusValue;
  final String orderCompleteEmailMessage;
  final String orderCompleteSummaryTitle;
  final String orderCompleteLookupAction;
  final String orderCompleteBackToDesignAction;
  final String emailSentNoticeTitle;
  final String emailSentNoticeMessage;
  final String orderEmailMissingGuideTitle;
  final String orderEmailMissingGuideMessage;
  final String orderEmailMissingSpamCheck;
  final String orderEmailMissingAddressCheck;
  final String orderEmailMissingDeliveryWait;
  final String orderEmailMissingContactSupport;
  final String contactSupportPromptTitle;
  final String contactSupportPromptMessage;
  final String contactSupportPromptAction;
  final String orderLookup;
  final String orderNo;
  final String orderNoHint;
  final String email;
  final String emailHint;
  final String lookupOrder;
  final String orderLookupLoadingTitle;
  final String orderLookupLoadingMessage;
  final String orderLookupNotFoundTitle;
  final String orderLookupNotFoundMessage;
  final String orderLookupErrorTitle;
  final String orderLookupErrorMessage;
  final String orderLookupResultTitle;
  final String orderLookupResultMessage;
  final String orderLookupOrderStatusLabel;
  final String orderLookupProgressTitle;
  final String orderLookupProductionStatusLabel;
  final String orderLookupShippingStatusLabel;
  final String orderLookupFulfillmentStatusLabel;
  final String orderLookupTrackingNumberLabel;
  final String orderLookupTrackingDetailsTitle;
  final String orderLookupCarrierLabel;
  final String orderLookupShippedAtLabel;
  final String orderLookupUpdatedAtLabel;
  final String orderLookupNoTrackingValue;
  final String orderLookupOrderDateLabel;
  final String orderLookupContentTitle;
  final String orderLookupSelectedSealLabel;
  final String orderLookupGemstoneLabel;
  final String orderLookupLookupAnotherAction;
  final String language;
  final String about;
  final String howItWorks;
  final String faq;
  final String privacy;
  final String terms;
  final String contact;
  final String version;
  final String settingsLanguageTitle;
  final String settingsLanguageMessage;
  final String settingsLanguageEnglish;
  final String settingsLanguageJapanese;
  final String settingsFaqIntro;
  final String settingsVersionTitle;
  final String settingsVersionMessageTemplate;
}

const _localizedValues = {
  'en': HankoLocalizations._(
    Locale('en'),
    _HankoStrings(
      appTitle: 'STONE SIGNATURE',
      splashPreparing: 'Preparing your design experience.',
      onboardingTitle: 'Welcome',
      onboardingHeroTitle: 'Create your\nseal in minutes',
      onboardingTagline: 'Personalized. Timeless.\nUniquely yours.',
      onboardingKanjiTitle: 'Choose kanji from your name',
      onboardingKanjiMessage: 'We suggest meaningful kanji based on your name.',
      onboardingAiTitle: 'Generate a seal design with AI',
      onboardingAiMessage:
          'Our AI creates beautiful, balanced seal designs just for you.',
      onboardingStoneTitle: 'Select a gemstone and order',
      onboardingStoneMessage:
          "Pick your favorite gemstone and we'll craft your seal with care.",
      onboardingStorageTitle: 'Saved on this device',
      onboardingStorageMessage:
          'Saved seal designs and preview images stay on this device. Payment details and checkout secrets are never saved locally.',
      onboardingGetStarted: 'Get Started',
      onboardingSaving: 'Saving...',
      onboardingSkip: 'Skip',
      onboardingSaveError:
          'Could not save onboarding status. Please try again.',
      design: 'Design',
      mySeals: 'My Seals',
      stones: 'Stones',
      settings: 'Settings',
      createCustomSeal: 'Create your\ncustom seal',
      customSealDescription:
          'Turn your name into a\npersonalized gemstone seal.',
      startDesigning: 'Start Designing',
      designNameTitle: 'Enter Your Name',
      designNameIntro: "We'll suggest kanji based on your preferences.",
      designNameLabel: 'Your name',
      designNameHint: 'Michael Smith',
      designNameHelp: '1-2 kanji will be suggested for a small personal seal.',
      designGenderLabel: 'Gender preference',
      designGenderUnspecified: 'No preference',
      designGenderMale: 'Masculine',
      designGenderFemale: 'Feminine',
      designKanjiStyleLabel: 'Kanji style',
      designKanjiStyleJapanese: 'Japanese style',
      designKanjiStyleChinese: 'Chinese style',
      designKanjiStyleTaiwanese: 'Taiwanese style',
      suggestKanji: 'Suggest Kanji',
      designKanjiTipTitle: 'Simple kanji work best for gemstone engraving.',
      designKanjiTipMessage:
          'Clear, balanced characters create the most beautiful and timeless results.',
      designCandidateReadyTitle: 'Ready to Suggest Kanji',
      designCandidateReadyMessage:
          'Candidate generation can now use this name and preference set.',
      designRequestDetails: 'Request details',
      editName: 'Edit Name',
      designLoadingTitle: 'Finding Kanji',
      designLoadingMessage: 'Creating kanji suggestions...',
      designInvalidNameSummary: 'Enter your name to continue.',
      designInvalidNameMessage:
          'Please enter a valid first name or short name.',
      designSuggestionErrorTitle: "We couldn't suggest kanji",
      designSuggestionErrorMessage:
          'Something went wrong while generating kanji suggestions for your name. Please try again.',
      designNoKanjiTitle: "We couldn't find a suitable kanji",
      designNoKanjiMessage:
          'Your name did not return any kanji suggestions that fit our engraving rules.',
      designNoKanjiRuleCharacters: '1-2 characters only',
      designNoKanjiRuleCommon: 'Simple, common kanji',
      designNoKanjiRuleEngraving: 'Suitable for seal engraving',
      designErrorTip: 'Use a simple first name or short name.',
      designNoKanjiTip: 'Try a shorter name or nickname.',
      tryAgain: 'Try Again',
      back: 'Back',
      commonNetworkErrorTitle: 'Network Error',
      commonNetworkErrorMessage:
          "We're unable to connect to the server. Please check your internet connection and try again.",
      commonServerErrorTitle: 'Server Error',
      commonServerErrorMessage:
          "We're experiencing a temporary issue on our end. Please wait a moment and try again.",
      storageErrorTitle: "Couldn't Save Seal",
      storageErrorMessage:
          "The seal image couldn't be saved on this device. Check storage permissions and available space, then try again.",
      deepLinkErrorTitle: 'Checkout Return Link Error',
      deepLinkErrorMessage:
          "The Stripe Checkout return link couldn't be processed. Please open Checkout again or contact support if payment may have completed.",
      maintenanceTitle: 'Temporarily Unavailable',
      maintenanceMessage:
          'Stone Signature is currently undergoing maintenance. Please check back in a little while.',
      appUpdateRequiredTitle: 'Update Required',
      appUpdateRequiredMessage:
          'A newer app version is required to continue. Please update the app, then open Stone Signature again.',
      appUpdateRequiredAction: 'Update App',
      commonGenericErrorTitle: 'Something Went Wrong',
      commonGenericErrorMessage:
          'An unexpected error occurred. Please try again in a few moments.',
      kanjiSuggestionsTitle: 'Kanji Suggestions',
      kanjiSuggestionsMessage: 'Choose the kanji that best fits your seal.',
      kanjiCandidateDetailTitle: 'Kanji Detail',
      kanjiReadingLabel: 'Reading',
      kanjiMeaningLabel: 'Meaning',
      kanjiImpressionLabel: 'Impression',
      kanjiReasonLabel: 'Reason',
      kanjiCharacterCountLabel: 'Characters',
      kanjiStrokeComplexityLabel: 'Stroke complexity',
      kanjiEngravingSuitabilityLabel: 'Engraving suitability',
      selectKanji: 'Select Kanji',
      kanjiSelectedTitle: 'Kanji selected',
      kanjiSelectedMessage: 'This kanji is ready for seal style selection.',
      sealStyleTitle: 'Seal Style',
      sealStyleMessage: 'Customize your seal style.',
      sealStyleSelectedKanjiLabel: 'Selected kanji',
      sealShapeLabel: 'Shape',
      sealShapeSquare: 'Square',
      sealShapeRound: 'Round',
      sealStyleNameLabel: 'Style',
      sealStyleTraditional: 'Traditional',
      sealStyleElegant: 'Elegant',
      sealStyleSoft: 'Soft',
      sealStyleBold: 'Bold',
      sealStrokeWeightLabel: 'Stroke Weight',
      sealStrokeStandard: 'Standard',
      sealStrokeBold: 'Bold',
      sealBalanceLabel: 'Balance',
      sealBalanceAiry: 'Airy',
      sealBalanceBalanced: 'Balanced',
      sealBalanceDense: 'Dense',
      sealStyleSummaryTitle: 'Current style',
      confirmStyle: 'Confirm Style',
      sealStyleConfirmedTitle: 'Style selected',
      sealStyleConfirmedMessage:
          'These style choices are ready for AI seal generation.',
      generateSeal: 'Generate Seal',
      sealGenerationLoadingTitle: 'Generating Seal',
      sealGenerationLoadingMessage:
          'Creating three AI seal design directions...',
      sealGenerationLoadingDetail:
          'We are checking the kanji and style before saving previews.',
      sealGenerationErrorTitle: "We couldn't generate seal designs",
      sealGenerationErrorMessage:
          'Something went wrong while creating AI seal previews. Please try again.',
      sealGenerationLimitTitle: 'Generation limit reached',
      sealGenerationLimitMessage:
          'You have used all generation attempts for this style set. Adjust the style before trying again.',
      sealGenerationAttemptLabel: 'Attempts',
      sealGenerationStyleDetails: 'Generation details',
      sealGenerationErrorTip:
          'Try again once. If it still fails, adjust the style or choose a simpler kanji.',
      sealGenerationLimitTip:
          'Choose a different balance, stroke weight, or kanji to start a fresh generation.',
      adjustStyle: 'Adjust Style',
      sealVariantSelectionTitle: 'Seal Options',
      sealVariantSelectionMessage: 'Choose one AI seal design.',
      sealVariantSelectedBadge: 'Selected',
      sealVariantSelectedTitle: 'Seal design selected',
      sealVariantSelectedMessage:
          'This AI seal design is ready for preview and saving.',
      regenerateSeal: 'Regenerate Seal',
      sealPreviewTitle: 'Seal Preview',
      sealPreviewMessage: 'Review your selected seal design before saving.',
      sealPreviewVariantLabel: 'AI Variant',
      saveSeal: 'Save Seal',
      chooseStone: 'Choose a Stone',
      sealSavedTitle: 'Seal Saved',
      sealSavedHeading: 'Seal saved to My Seals',
      sealSavedMessage:
          'Your custom seal design is ready for comparison and ordering.',
      goToMySeals: 'Go to My Seals',
      createAnotherSeal: 'Create Another Seal',
      savedSeals: 'Saved Seals',
      savedSealsDescription: 'View and manage your\nsaved seal designs.',
      savedOnThisDevice: 'Saved on this device',
      savedSealsLoadingTitle: 'Loading saved seals',
      savedSealsLoadingMessage: 'Checking seal designs saved on this device.',
      savedSealsLoadErrorTitle: "Couldn't load saved seals",
      savedSealsLoadErrorMessage:
          'Open My Seals again, or create a new seal design.',
      chooseSavedSeal: 'Choose',
      viewSealDetails: 'View Details',
      favoriteSeal: 'Favorite seal',
      removeFavoriteSeal: 'Remove favorite',
      sealDetailTitle: 'Seal Detail',
      kanjiLabel: 'Kanji',
      createdAtLabel: 'Created',
      compareSavedSeals: 'Compare Seals',
      compareSavedSealsTitle: 'Compare saved seals',
      compareSavedSealsMessage:
          'Review saved seal previews, kanji meanings, and style choices side by side.',
      editSavedSeal: 'Edit / Regenerate',
      editSavedSealTitle: 'Create a new version from Design',
      editSavedSealMessage:
          'Saved seals stay unchanged. To try different kanji or style choices, start a new design and save it.',
      chooseSealForOrder: 'Choose for Order',
      sealSelectedForOrderTitle: 'Selected for order',
      sealSelectedForOrderMessage: 'This seal is now saved in the order draft.',
      sealSelectedForOrderAction: 'Selected for Order',
      deleteSavedSeal: 'Delete Seal',
      deleteSealTitle: 'Delete saved seal?',
      deleteSealMessage:
          'This removes the seal design from this device. This action cannot be undone.',
      deleteSealConfirm: 'Delete',
      cancel: 'Cancel',
      close: 'Close',
      browseStones: 'Browse Stones',
      browseStonesDescription: 'Explore our collection of\nnatural gemstones.',
      noSavedSeals: 'No saved seals',
      noSavedSealsMessage:
          'Saved seal designs will appear here after you create one.',
      stonesLoadingTitle: 'Loading stones',
      stonesLoadingMessage: 'Checking available one-of-a-kind seal stones.',
      stonesLoadErrorTitle: "Couldn't load stones",
      stonesLoadErrorMessage:
          'Try again to refresh the available stone listings.',
      noStonesLoaded: 'No stones loaded',
      noStonesLoadedMessage:
          'Available one-of-a-kind stones will be shown here.',
      stoneFiltersTitle: 'Filters',
      stoneFilterAll: 'All',
      stoneFilterMaterial: 'Material',
      stoneFilterColor: 'Color',
      stoneFilterPattern: 'Pattern',
      stoneFilterAvailability: 'Availability',
      stoneFilterReset: 'Reset',
      noStonesMatchFilters: 'No stones match filters',
      noStonesMatchFiltersMessage:
          'Clear or change filters to browse other stones.',
      stoneSortTitle: 'Sort',
      stoneSortAction: 'Sort',
      stoneSortRecommended: 'Recommended',
      stoneSortNewest: 'Newest',
      stoneSortPriceLowToHigh: 'Price: Low to High',
      stoneSortPriceHighToLow: 'Price: High to Low',
      viewStoneDetails: 'View Details',
      stoneDetailTitle: 'Stone Detail',
      stoneDetailDescriptionTitle: 'Description',
      stoneDetailStoryTitle: 'Story',
      stoneDetailSpecsTitle: 'Details',
      stoneDetailNotesTitle: 'Notes',
      stoneDetailNotesMessage:
          'Natural stone color, pattern, and translucency vary by piece. Review the photos and details before ordering.',
      stoneDetailMaterialLabel: 'Material',
      stoneDetailSizeLabel: 'Size',
      stoneDetailColorLabel: 'Color',
      stoneDetailPatternLabel: 'Pattern',
      stoneDetailShapeLabel: 'Shape',
      stoneDetailTextureLabel: 'Texture',
      stoneDetailStatusLabel: 'Status',
      selectStone: 'Select Stone',
      selectStoneConfirmationTitle: 'Select this stone?',
      selectStoneConfirmationMessage:
          'Confirm this one-of-a-kind stone for your order draft. Availability will be checked again before checkout.',
      selectStoneConfirm: 'Confirm Selection',
      stoneSelectedForOrderTitle: 'Selected for order',
      stoneSelectedForOrderMessage:
          'This stone is now saved in the order draft.',
      stoneSelectedForOrderAction: 'Selected for Order',
      soldOutStoneTitle: 'Stone unavailable',
      soldOutStoneMessage:
          'This stone is no longer available. Choose another stone before ordering.',
      stoneAvailable: 'Available',
      stoneUnavailable: 'Unavailable',
      order: 'Order',
      noActiveDraft: 'No active draft',
      noActiveDraftMessage: 'Choose a saved seal and a stone before checkout.',
      reviewSelection: 'Review Selection',
      orderMissingSealTitle: 'Seal design missing',
      orderMissingSealMessage:
          'Choose a saved seal design before continuing to checkout.',
      orderMissingSealNotice:
          'A seal design is required to complete this custom order.',
      orderChooseSealAction: 'Choose a Seal',
      orderChangeSealAction: 'Change Seal',
      orderMissingStoneTitle: 'Stone missing',
      orderMissingStoneMessage:
          'Choose a gemstone seal stone before continuing to checkout.',
      orderChooseStoneAction: 'Choose a Stone',
      orderChangeStoneAction: 'Change Stone',
      orderReviewTitle: 'Order Review',
      orderReviewMessage:
          'Review the selected seal design and one-of-a-kind stone before entering shipping details.',
      orderItemPriceLabel: 'Item price',
      orderShippingFeeLabel: 'Shipping',
      orderShippingEstimateNote:
          'Shipping is an estimate and will be recalculated before payment.',
      orderTotalLabel: 'Total',
      orderCustomMadeNotice:
          'This product is made by combining your seal design with the selected stone as a custom one-of-a-kind item.',
      continueToShipping: 'Continue to Shipping',
      checkoutInputTitle: 'Checkout Information',
      checkoutInputMessage:
          'Enter the contact, shipping, and optional note details for this order draft.',
      checkoutContactTitle: 'Contact',
      checkoutShippingTitle: 'Shipping address',
      checkoutOrderNoteTitle: 'Order note',
      checkoutFullNameLabel: 'Full name',
      checkoutFullNameHint: 'Michael Smith',
      checkoutPhoneLabel: 'Phone number',
      checkoutPhoneHint: '+1 000 000 0000',
      checkoutCountryLabel: 'Country / Region',
      checkoutPostalCodeLabel: 'Postal code',
      checkoutPostalCodeHint: '10001',
      checkoutAddressLine1Label: 'Address line 1',
      checkoutAddressLine1Hint: '123 Example Street',
      checkoutAddressLine2Label: 'Address line 2',
      checkoutAddressLine2Hint: 'Apt 1',
      checkoutCityLabel: 'City',
      checkoutCityHint: 'New York',
      checkoutStateLabel: 'State / Province',
      checkoutStateHint: 'NY',
      checkoutOrderNoteLabel: 'Order note',
      checkoutOrderNoteHint: 'Optional production or delivery note',
      checkoutInputSaveAction: 'Save Checkout Information',
      checkoutInputSavingAction: 'Saving...',
      checkoutInputSavedMessage:
          'Checkout information was saved to this order draft.',
      checkoutValidationTitle: 'Please review the highlighted fields.',
      checkoutValidationMessage: 'Some information is missing or invalid.',
      checkoutEmailInvalidMessage: 'Please enter a valid email address.',
      checkoutFullNameRequiredMessage: 'Full name is required.',
      checkoutPhoneInvalidMessage: 'Please enter a valid phone number.',
      checkoutCountryRequiredMessage: 'Country / Region is required.',
      checkoutPostalCodeRequiredMessage: 'Postal code is required.',
      checkoutAddressLine1RequiredMessage: 'Address line 1 is required.',
      checkoutCityRequiredMessage: 'City is required.',
      checkoutStateRequiredMessage: 'State / Province is required.',
      orderConfirmationTitle: 'Order Confirmation',
      orderConfirmationMessage:
          'Review the seal, gemstone, shipping details, and total before proceeding.',
      orderConfirmationMissingInputMessage:
          'Checkout information is incomplete. Return to the checkout form before confirming.',
      orderConfirmationCheckoutTitle: 'Checkout details',
      orderConfirmationNoOrderNote: 'No order note',
      editCheckoutInformation: 'Edit Checkout Information',
      customMadeAgreementTitle: 'Custom-made Agreement',
      customMadeAgreementMessage:
          'Each seal is handcrafted to order using natural gemstones. Please confirm the details before payment.',
      confirmKanjiAndDesignLabel:
          'I confirm that the selected kanji and seal design are correct.',
      confirmCustomMadePolicyLabel:
          'I understand that this is a custom-made item and cannot be changed after production begins.',
      orderConfirmationSecurePaymentNote:
          'You will be redirected to Stripe Checkout for secure payment.',
      orderConfirmationAgreementRequiredMessage:
          'Both confirmation checks are required before payment.',
      orderConfirmationSavedMessage:
          'Order confirmation was saved to this order draft.',
      proceedToSecurePayment: 'Proceed to Secure Payment',
      checkoutProcessingTitle: 'Preparing Checkout',
      checkoutProcessingMessage:
          'Creating your order and secure Stripe Checkout session.',
      checkoutProcessingOrderStep: 'Creating order',
      checkoutProcessingSessionStep: 'Creating secure payment session',
      checkoutProcessingReadyTitle: 'Checkout session ready',
      checkoutProcessingReadyMessage: 'Stripe Checkout is ready to open.',
      checkoutProcessingErrorTitle: "Couldn't prepare Checkout",
      checkoutProcessingErrorMessage: 'Please go back and try again.',
      stripeCheckoutTitle: 'Secure Payment',
      stripeCheckoutOpeningTitle: 'Opening Stripe Checkout',
      stripeCheckoutOpeningMessage:
          'You will be redirected to Stripe Checkout.',
      stripeCheckoutWaitingTitle: 'Complete payment in Stripe Checkout',
      stripeCheckoutWaitingMessage:
          'Return to this app after payment to continue.',
      stripeCheckoutReturnedTitle: 'Returned from Stripe Checkout',
      stripeCheckoutReturnedMessage:
          'The return URL was received. Payment status will be verified next.',
      stripeCheckoutCanceledTitle: 'Checkout was canceled',
      stripeCheckoutCanceledMessage:
          'Stripe returned without completing payment.',
      stripeCheckoutReturnFailedTitle: 'Payment failed',
      stripeCheckoutReturnFailedMessage:
          'Stripe Checkout could not be completed. You can try Checkout again.',
      stripeCheckoutLaunchFailedTitle: "Couldn't open Stripe Checkout",
      stripeCheckoutLaunchFailedMessage:
          'Check your browser settings and try again.',
      stripeCheckoutOpenAction: 'Open Stripe Checkout',
      stripeCheckoutRetryAction: 'Try Again',
      stripeCheckoutSecureNote:
          'Your payment information is secure and encrypted. Powered by Stripe.',
      stripeCheckoutReturnOrderIdLabel: 'Return order ID',
      paymentStatusTitle: 'Payment Status',
      paymentStatusCheckingTitle: 'Checking payment status',
      paymentStatusCheckingMessage:
          'Confirming the latest order status after Stripe Checkout.',
      paymentStatusPaidTitle: 'Payment confirmed',
      paymentStatusPaidMessage:
          'Your payment is confirmed. We are preparing the order summary.',
      paymentStatusPendingTitle: 'Payment pending',
      paymentStatusPendingMessage:
          'Stripe returned successfully, but payment confirmation is still pending.',
      paymentStatusPendingNotice: '',
      paymentStatusFailedTitle: "Couldn't verify payment",
      paymentStatusFailedMessage:
          'Payment status could not be confirmed. Please check Order Lookup later.',
      orderCompleteTitle: 'Order Complete',
      orderCompleteStatusTitle: 'Thank you for your order',
      orderCompleteMessage:
          'Payment is confirmed. Stripe will send the payment receipt to the email address used for checkout.',
      orderCompleteStatusLabel: 'Status',
      orderCompleteStatusValue: 'Payment received',
      orderCompleteEmailMessage:
          'Please check the Stripe payment receipt. Keep the order number for support and order lookup.',
      orderCompleteSummaryTitle: 'Order summary',
      orderCompleteLookupAction: 'Open Order Lookup',
      orderCompleteBackToDesignAction: 'Back to Design',
      emailSentNoticeTitle: 'Stripe payment email',
      emailSentNoticeMessage:
          'Stripe sends the payment receipt to the email address on the order. Please check your inbox and spam folder.',
      orderEmailMissingGuideTitle: "Can't find the Stripe email?",
      orderEmailMissingGuideMessage: 'Here are a few quick things to check.',
      orderEmailMissingSpamCheck: 'Check your spam or junk folder.',
      orderEmailMissingAddressCheck:
          'Make sure the email address on the order is correct.',
      orderEmailMissingDeliveryWait: 'Please allow a few minutes for delivery.',
      orderEmailMissingContactSupport:
          "If you still can't find it, contact support with your order number.",
      contactSupportPromptTitle: 'Need help?',
      contactSupportPromptMessage:
          'Our support team can help with order, shipping, payment, and email questions. Include your order number for faster support.',
      contactSupportPromptAction: 'Contact Support',
      orderLookup: 'Order Lookup',
      orderNo: 'Order No',
      orderNoHint: 'HF-0001',
      email: 'Email',
      emailHint: 'name@example.com',
      lookupOrder: 'Lookup Order',
      orderLookupLoadingTitle: 'Looking up your order',
      orderLookupLoadingMessage: 'Checking the order number and email address.',
      orderLookupNotFoundTitle: 'Order not found',
      orderLookupNotFoundMessage:
          "We couldn't find an order matching that order number and email address.",
      orderLookupErrorTitle: "Couldn't load order",
      orderLookupErrorMessage:
          'Order Lookup could not be completed. Please try again.',
      orderLookupResultTitle: 'Order Status',
      orderLookupResultMessage: "Here's the latest update on your order.",
      orderLookupOrderStatusLabel: 'Order status',
      orderLookupProgressTitle: 'Order progress',
      orderLookupProductionStatusLabel: 'Production status',
      orderLookupShippingStatusLabel: 'Shipping status',
      orderLookupFulfillmentStatusLabel: 'Fulfillment status',
      orderLookupTrackingNumberLabel: 'Tracking number',
      orderLookupTrackingDetailsTitle: 'Tracking details',
      orderLookupCarrierLabel: 'Carrier',
      orderLookupShippedAtLabel: 'Shipped at',
      orderLookupUpdatedAtLabel: 'Last updated',
      orderLookupNoTrackingValue: 'Not available yet',
      orderLookupOrderDateLabel: 'Order date',
      orderLookupContentTitle: 'Order content',
      orderLookupSelectedSealLabel: 'Selected seal',
      orderLookupGemstoneLabel: 'Gemstone',
      orderLookupLookupAnotherAction: 'Lookup another order',
      language: 'Language',
      about: 'About',
      howItWorks: 'How It Works',
      faq: 'FAQ',
      privacy: 'Privacy',
      terms: 'Terms',
      contact: 'Contact',
      version: 'Version',
      settingsLanguageTitle: 'App language',
      settingsLanguageMessage:
          'Choose the language used in the app. Before you choose one, the app follows your smartphone language.',
      settingsLanguageEnglish: 'English',
      settingsLanguageJapanese: 'Japanese',
      settingsFaqIntro:
          'Find answers to common questions about kanji selection, production, delivery, and order lookup.',
      settingsVersionTitle: 'Installed app version',
      settingsVersionMessageTemplate: 'Version {version}',
    ),
  ),
  'ja': HankoLocalizations._(
    Locale('ja'),
    _HankoStrings(
      appTitle: 'STONE SIGNATURE',
      splashPreparing: 'デザイン体験を準備しています。',
      onboardingTitle: 'ようこそ',
      onboardingHeroTitle: '数分で\n印影を作成',
      onboardingTagline: 'あなたらしく。時を超えて。\n唯一無二の印鑑へ。',
      onboardingKanjiTitle: '名前から漢字を選ぶ',
      onboardingKanjiMessage: '名前に合わせて意味のある漢字候補を提案します。',
      onboardingAiTitle: 'AIで印影を生成',
      onboardingAiMessage: '美しく整った印影デザインをAIが作成します。',
      onboardingStoneTitle: '天然石を選んで注文',
      onboardingStoneMessage: 'お気に入りの天然石を選び、職人が丁寧に制作します。',
      onboardingStorageTitle: 'この端末に保存されます',
      onboardingStorageMessage:
          '保存済み印影とプレビュー画像はこの端末に保存されます。カード情報やCheckoutの秘密情報は端末に保存しません。',
      onboardingGetStarted: 'はじめる',
      onboardingSaving: '保存中...',
      onboardingSkip: 'スキップ',
      onboardingSaveError: '初回案内の完了状態を保存できませんでした。もう一度お試しください。',
      design: 'デザイン',
      mySeals: 'マイ印影',
      stones: '石',
      settings: '設定',
      createCustomSeal: 'あなた専用の\n印影を作成',
      customSealDescription: '名前から、あなただけの\n天然石印鑑を作ります。',
      startDesigning: '作成をはじめる',
      designNameTitle: '名前を入力',
      designNameIntro: 'あなたの希望に合わせた漢字を提案します。',
      designNameLabel: 'お名前',
      designNameHint: '山田 太郎',
      designNameHelp: '小さな印鑑に合う1-2文字の漢字を提案します。',
      designGenderLabel: '性別の希望',
      designGenderUnspecified: '指定なし',
      designGenderMale: '男性的',
      designGenderFemale: '女性的',
      designKanjiStyleLabel: '漢字スタイル',
      designKanjiStyleJapanese: '日本スタイル',
      designKanjiStyleChinese: '中国スタイル',
      designKanjiStyleTaiwanese: '台湾スタイル',
      suggestKanji: '漢字を提案',
      designKanjiTipTitle: '天然石の彫刻にはシンプルな漢字が適しています。',
      designKanjiTipMessage: '明快で整った文字が、美しく長く愛せる印影になります。',
      designCandidateReadyTitle: '漢字候補の準備ができました',
      designCandidateReadyMessage: 'この名前と希望条件で漢字候補生成へ進めます。',
      designRequestDetails: '入力内容',
      editName: '名前を修正',
      designLoadingTitle: '漢字を検索中',
      designLoadingMessage: '漢字候補を作成しています...',
      designInvalidNameSummary: '名前を入力してください。',
      designInvalidNameMessage: '短いお名前またはニックネームを入力してください。',
      designSuggestionErrorTitle: '漢字候補を提案できませんでした',
      designSuggestionErrorMessage: '漢字候補の生成中に問題が発生しました。もう一度お試しください。',
      designNoKanjiTitle: '条件に合う漢字が見つかりませんでした',
      designNoKanjiMessage: '入力された名前では、彫刻条件に合う漢字候補を出せませんでした。',
      designNoKanjiRuleCharacters: '1-2文字のみ',
      designNoKanjiRuleCommon: 'シンプルで一般的な漢字',
      designNoKanjiRuleEngraving: '印鑑の彫刻に適している',
      designErrorTip: '短い名またはニックネームをお試しください。',
      designNoKanjiTip: 'より短い名前やニックネームをお試しください。',
      tryAgain: 'もう一度試す',
      back: '戻る',
      commonNetworkErrorTitle: '通信エラー',
      commonNetworkErrorMessage: 'サーバーに接続できません。通信環境を確認して、もう一度お試しください。',
      commonServerErrorTitle: 'サーバーエラー',
      commonServerErrorMessage: '一時的な問題が発生しています。しばらく待ってからもう一度お試しください。',
      storageErrorTitle: '印影を保存できませんでした',
      storageErrorMessage:
          'この端末に印影画像を保存できませんでした。ストレージ権限と空き容量を確認して、もう一度お試しください。',
      deepLinkErrorTitle: 'Checkout戻りURLを処理できませんでした',
      deepLinkErrorMessage:
          'Stripe Checkoutからの戻りURLを処理できませんでした。もう一度Checkoutを開くか、決済済みの可能性がある場合はお問い合わせください。',
      maintenanceTitle: 'ただいまご利用いただけません',
      maintenanceMessage:
          'Stone Signatureは現在メンテナンス中です。しばらく時間をおいてからもう一度お試しください。',
      appUpdateRequiredTitle: 'アプリの更新が必要です',
      appUpdateRequiredMessage:
          '続けるには新しいバージョンのアプリが必要です。アプリを更新してから、もう一度Stone Signatureを開いてください。',
      appUpdateRequiredAction: 'アプリを更新',
      commonGenericErrorTitle: '問題が発生しました',
      commonGenericErrorMessage: '予期しないエラーが発生しました。時間をおいてもう一度お試しください。',
      kanjiSuggestionsTitle: '漢字候補',
      kanjiSuggestionsMessage: '印鑑に使う漢字を選んでください。',
      kanjiCandidateDetailTitle: '漢字の詳細',
      kanjiReadingLabel: '読み',
      kanjiMeaningLabel: '意味',
      kanjiImpressionLabel: '印象',
      kanjiReasonLabel: '理由',
      kanjiCharacterCountLabel: '文字数',
      kanjiStrokeComplexityLabel: '画数の複雑さ',
      kanjiEngravingSuitabilityLabel: '彫刻適性',
      selectKanji: '漢字を選択',
      kanjiSelectedTitle: '漢字を選択しました',
      kanjiSelectedMessage: 'この漢字で印影スタイル選択へ進めます。',
      sealStyleTitle: '印影スタイル',
      sealStyleMessage: '印影スタイルをカスタマイズしてください。',
      sealStyleSelectedKanjiLabel: '選択中の漢字',
      sealShapeLabel: '形',
      sealShapeSquare: '角印',
      sealShapeRound: '丸印',
      sealStyleNameLabel: '雰囲気',
      sealStyleTraditional: '伝統的',
      sealStyleElegant: '上品',
      sealStyleSoft: 'やわらかい',
      sealStyleBold: '力強い',
      sealStrokeWeightLabel: '線の太さ',
      sealStrokeStandard: '標準',
      sealStrokeBold: '太め',
      sealBalanceLabel: '余白感',
      sealBalanceAiry: '余白多め',
      sealBalanceBalanced: '標準',
      sealBalanceDense: '密度高め',
      sealStyleSummaryTitle: '現在のスタイル',
      confirmStyle: 'スタイルを確定',
      sealStyleConfirmedTitle: 'スタイルを選択しました',
      sealStyleConfirmedMessage: 'このスタイルでAI印影生成へ進めます。',
      generateSeal: '印影を生成',
      sealGenerationLoadingTitle: '印影を生成中',
      sealGenerationLoadingMessage: 'AI印影候補を3件作成しています...',
      sealGenerationLoadingDetail: '漢字、スタイルを確認しながらプレビューを保存しています。',
      sealGenerationErrorTitle: '印影を生成できませんでした',
      sealGenerationErrorMessage: 'AI印影候補の作成中に問題が発生しました。もう一度お試しください。',
      sealGenerationLimitTitle: '再生成の上限に達しました',
      sealGenerationLimitMessage:
          'このスタイルで利用できる生成回数を使い切りました。スタイルを調整してから再度お試しください。',
      sealGenerationAttemptLabel: '生成回数',
      sealGenerationStyleDetails: '生成内容',
      sealGenerationErrorTip: '一度再試行し、続けて失敗する場合はスタイルや漢字をよりシンプルにしてください。',
      sealGenerationLimitTip: '余白感、線の太さ、または漢字を変えると新しく生成できます。',
      adjustStyle: 'スタイルを調整',
      sealVariantSelectionTitle: '印影候補',
      sealVariantSelectionMessage: 'AI印影候補から1件を選択してください。',
      sealVariantSelectedBadge: '選択中',
      sealVariantSelectedTitle: '印影を選択しました',
      sealVariantSelectedMessage: 'このAI印影をプレビュー確認と保存に進められます。',
      regenerateSeal: '印影を再生成',
      sealPreviewTitle: '印影プレビュー',
      sealPreviewMessage: '保存前に選択した印影デザインを確認してください。',
      sealPreviewVariantLabel: 'AI候補',
      saveSeal: '印影を保存',
      chooseStone: '石を選ぶ',
      sealSavedTitle: '保存完了',
      sealSavedHeading: '印影を保存しました',
      sealSavedMessage: '保存した印影は比較や注文に使えます。',
      goToMySeals: 'マイ印影へ',
      createAnotherSeal: '別の印影を作成',
      savedSeals: '保存済み印影',
      savedSealsDescription: '保存した印影デザインを\n確認・管理できます。',
      savedOnThisDevice: 'この端末に保存',
      savedSealsLoadingTitle: '保存済み印影を読み込み中',
      savedSealsLoadingMessage: 'この端末に保存された印影を確認しています。',
      savedSealsLoadErrorTitle: '保存済み印影を読み込めません',
      savedSealsLoadErrorMessage: 'マイ印影を開き直すか、新しい印影を作成してください。',
      chooseSavedSeal: '選択',
      viewSealDetails: '詳細を見る',
      favoriteSeal: 'お気に入りに追加',
      removeFavoriteSeal: 'お気に入りを解除',
      sealDetailTitle: '印影詳細',
      kanjiLabel: '漢字',
      createdAtLabel: '作成日',
      compareSavedSeals: '印影を比較',
      compareSavedSealsTitle: '保存済み印影の比較',
      compareSavedSealsMessage: '保存済み印影のプレビュー、漢字の意味、スタイルを横並びで比較できます。',
      editSavedSeal: '編集 / 再生成',
      editSavedSealTitle: 'Designから新しい案を作成',
      editSavedSealMessage:
          '保存済み印影はそのまま残ります。漢字やスタイルを変える場合は、Designから新しい印影を作成して保存してください。',
      chooseSealForOrder: '注文に使う',
      sealSelectedForOrderTitle: '注文用に選択済み',
      sealSelectedForOrderMessage: 'この印影を注文下書きに反映しました。',
      sealSelectedForOrderAction: '選択済み',
      deleteSavedSeal: '印影を削除',
      deleteSealTitle: '保存済み印影を削除しますか？',
      deleteSealMessage: 'この端末から印影デザインを削除します。この操作は元に戻せません。',
      deleteSealConfirm: '削除',
      cancel: 'キャンセル',
      close: '閉じる',
      browseStones: '石を探す',
      browseStonesDescription: '天然石コレクションを\nご覧ください。',
      noSavedSeals: '保存済み印影はありません',
      noSavedSealsMessage: '作成した印影デザインがここに表示されます。',
      stonesLoadingTitle: '石を読み込み中',
      stonesLoadingMessage: '販売中の一点物の石を確認しています。',
      stonesLoadErrorTitle: '石を読み込めません',
      stonesLoadErrorMessage: '販売中の石一覧を再読み込みしてください。',
      noStonesLoaded: '石を読み込んでいません',
      noStonesLoadedMessage: '販売中の一点物の石がここに表示されます。',
      stoneFiltersTitle: '絞り込み',
      stoneFilterAll: 'すべて',
      stoneFilterMaterial: '素材',
      stoneFilterColor: '色',
      stoneFilterPattern: '模様',
      stoneFilterAvailability: '在庫',
      stoneFilterReset: 'リセット',
      noStonesMatchFilters: '条件に合う石がありません',
      noStonesMatchFiltersMessage: '絞り込み条件を変更するか解除してください。',
      stoneSortTitle: '並び替え',
      stoneSortAction: '並び替え',
      stoneSortRecommended: 'おすすめ順',
      stoneSortNewest: '新着順',
      stoneSortPriceLowToHigh: '価格が安い順',
      stoneSortPriceHighToLow: '価格が高い順',
      viewStoneDetails: '詳細を見る',
      stoneDetailTitle: '石の詳細',
      stoneDetailDescriptionTitle: '説明',
      stoneDetailStoryTitle: 'ストーリー',
      stoneDetailSpecsTitle: '詳細',
      stoneDetailNotesTitle: '注意事項',
      stoneDetailNotesMessage: '天然石は一点ごとに色、模様、透け感が異なります。注文前に写真と詳細を確認してください。',
      stoneDetailMaterialLabel: '素材',
      stoneDetailSizeLabel: 'サイズ',
      stoneDetailColorLabel: '色',
      stoneDetailPatternLabel: '模様',
      stoneDetailShapeLabel: '形状',
      stoneDetailTextureLabel: '質感',
      stoneDetailStatusLabel: '在庫',
      selectStone: '石を選択',
      selectStoneConfirmationTitle: 'この石を選択しますか？',
      selectStoneConfirmationMessage: 'この一点物の石を注文下書きに反映します。注文前に在庫をもう一度確認します。',
      selectStoneConfirm: '選択を確定',
      stoneSelectedForOrderTitle: '注文用に選択済み',
      stoneSelectedForOrderMessage: 'この石を注文下書きに反映しました。',
      stoneSelectedForOrderAction: '選択済み',
      soldOutStoneTitle: 'この石は選択できません',
      soldOutStoneMessage: 'この石は現在販売できません。注文前に別の石を選択してください。',
      stoneAvailable: '販売中',
      stoneUnavailable: '選択不可',
      order: '注文',
      noActiveDraft: '進行中の下書きはありません',
      noActiveDraftMessage: '注文前に保存済み印影と石を選択してください。',
      reviewSelection: '選択内容を確認',
      orderMissingSealTitle: '印影が未選択です',
      orderMissingSealMessage: 'ご注文に進む前に、印影デザインを選択してください。',
      orderMissingSealNotice: '印影デザインは、ご注文を完了するために必要な項目です。必ずご選択ください。',
      orderChooseSealAction: '印影を選ぶ',
      orderChangeSealAction: '印影を変更',
      orderMissingStoneTitle: '宝石材が未選択です',
      orderMissingStoneMessage: 'ご注文に進む前に、宝石材を選択してください。',
      orderChooseStoneAction: '宝石材を選ぶ',
      orderChangeStoneAction: '宝石材を変更',
      orderReviewTitle: 'ご注文内容確認',
      orderReviewMessage: '配送先入力へ進む前に、選択した印影と一点物の石を確認してください。',
      orderItemPriceLabel: '商品価格',
      orderShippingFeeLabel: '送料',
      orderShippingEstimateNote: '送料は概算です。決済前に配送先にもとづいて再計算されます。',
      orderTotalLabel: '合計',
      orderCustomMadeNotice: '本商品は、印影と石を組み合わせてお作りする、お客様だけの一点物の特注品です。',
      continueToShipping: '配送先入力へ進む',
      checkoutInputTitle: 'Checkout情報',
      checkoutInputMessage: 'この注文下書きに使う連絡先、配送先、任意の注文メモを入力してください。',
      checkoutContactTitle: '連絡先',
      checkoutShippingTitle: '配送先住所',
      checkoutOrderNoteTitle: '注文メモ',
      checkoutFullNameLabel: '氏名',
      checkoutFullNameHint: '山田 太郎',
      checkoutPhoneLabel: '電話番号',
      checkoutPhoneHint: '+81 90 0000 0000',
      checkoutCountryLabel: '国 / 地域',
      checkoutPostalCodeLabel: '郵便番号',
      checkoutPostalCodeHint: '100-0001',
      checkoutAddressLine1Label: '住所1',
      checkoutAddressLine1Hint: '千代田1-1',
      checkoutAddressLine2Label: '住所2',
      checkoutAddressLine2Hint: 'マンション名・部屋番号',
      checkoutCityLabel: '市区町村',
      checkoutCityHint: '千代田区',
      checkoutStateLabel: '都道府県 / 州',
      checkoutStateHint: '東京都',
      checkoutOrderNoteLabel: '注文メモ',
      checkoutOrderNoteHint: '製作や配送についての任意メモ',
      checkoutInputSaveAction: 'Checkout情報を保存',
      checkoutInputSavingAction: '保存中...',
      checkoutInputSavedMessage: 'Checkout情報を注文下書きに保存しました。',
      checkoutValidationTitle: '入力内容を確認してください。',
      checkoutValidationMessage: '未入力または形式が正しくない項目があります。',
      checkoutEmailInvalidMessage: '有効なメールアドレスを入力してください。',
      checkoutFullNameRequiredMessage: '氏名を入力してください。',
      checkoutPhoneInvalidMessage: '有効な電話番号を入力してください。',
      checkoutCountryRequiredMessage: '国 / 地域を選択してください。',
      checkoutPostalCodeRequiredMessage: '郵便番号を入力してください。',
      checkoutAddressLine1RequiredMessage: '住所1を入力してください。',
      checkoutCityRequiredMessage: '市区町村を入力してください。',
      checkoutStateRequiredMessage: '都道府県 / 州を入力してください。',
      orderConfirmationTitle: 'ご注文内容確認',
      orderConfirmationMessage: '決済へ進む前に、印影、宝石材、配送先、合計金額を確認してください。',
      orderConfirmationMissingInputMessage:
          'Checkout情報に未入力があります。確認前に入力画面へ戻ってください。',
      orderConfirmationCheckoutTitle: 'Checkout詳細',
      orderConfirmationNoOrderNote: '注文メモなし',
      editCheckoutInformation: 'Checkout情報を編集',
      customMadeAgreementTitle: 'オーダーメイド確認',
      customMadeAgreementMessage: '天然石を使い、お客様の印影に合わせて一点ずつ製作します。決済前に内容をご確認ください。',
      confirmKanjiAndDesignLabel: '選択した漢字と印影デザインに間違いがないことを確認しました。',
      confirmCustomMadePolicyLabel: '本商品がオーダーメイド品であり、製作開始後は変更できないことを理解しました。',
      orderConfirmationSecurePaymentNote: '安全な決済のため、Stripe Checkoutへ移動します。',
      orderConfirmationAgreementRequiredMessage: '決済へ進むには2つの確認チェックが必要です。',
      orderConfirmationSavedMessage: '注文前確認を注文下書きに保存しました。',
      proceedToSecurePayment: '安全な決済へ進む',
      checkoutProcessingTitle: 'Checkout準備中',
      checkoutProcessingMessage: '注文を作成し、安全なStripe Checkout Sessionを準備しています。',
      checkoutProcessingOrderStep: '注文を作成中',
      checkoutProcessingSessionStep: '決済Sessionを作成中',
      checkoutProcessingReadyTitle: 'Checkout Session準備完了',
      checkoutProcessingReadyMessage: 'Stripe Checkoutを開く準備ができました。',
      checkoutProcessingErrorTitle: 'Checkoutを準備できませんでした',
      checkoutProcessingErrorMessage: '戻ってからもう一度お試しください。',
      stripeCheckoutTitle: '安全な決済',
      stripeCheckoutOpeningTitle: 'Stripe Checkoutを開いています',
      stripeCheckoutOpeningMessage: 'Stripe Checkoutへ移動します。',
      stripeCheckoutWaitingTitle: 'Stripe Checkoutで決済を完了してください',
      stripeCheckoutWaitingMessage: '決済後、このアプリに戻ると続きへ進みます。',
      stripeCheckoutReturnedTitle: 'Stripe Checkoutから戻りました',
      stripeCheckoutReturnedMessage: '戻りURLを受け取りました。次に注文状態を確認します。',
      stripeCheckoutCanceledTitle: 'Checkoutがキャンセルされました',
      stripeCheckoutCanceledMessage: 'Stripeから未決済の状態で戻りました。',
      stripeCheckoutReturnFailedTitle: '決済に失敗しました',
      stripeCheckoutReturnFailedMessage:
          'Stripe Checkoutを完了できませんでした。もう一度Checkoutをお試しください。',
      stripeCheckoutLaunchFailedTitle: 'Stripe Checkoutを開けませんでした',
      stripeCheckoutLaunchFailedMessage: 'ブラウザ設定を確認して、もう一度お試しください。',
      stripeCheckoutOpenAction: 'Stripe Checkoutを開く',
      stripeCheckoutRetryAction: 'もう一度試す',
      stripeCheckoutSecureNote: '決済情報は暗号化され、安全に処理されます。Powered by Stripe.',
      stripeCheckoutReturnOrderIdLabel: '戻り注文ID',
      paymentStatusTitle: '決済状態',
      paymentStatusCheckingTitle: '決済状態を確認しています',
      paymentStatusCheckingMessage: 'Stripe Checkoutから戻った後の最新注文状態を確認しています。',
      paymentStatusPaidTitle: '決済を確認しました',
      paymentStatusPaidMessage: '決済が確認できました。注文完了画面を準備しています。',
      paymentStatusPendingTitle: '決済確認待ち',
      paymentStatusPendingMessage: 'Stripeから正常に戻りましたが、決済確定の反映がまだ完了していません。',
      paymentStatusPendingNotice: '',
      paymentStatusFailedTitle: '決済状態を確認できませんでした',
      paymentStatusFailedMessage: '決済状態を確認できませんでした。後ほど注文照会をご確認ください。',
      orderCompleteTitle: '注文完了',
      orderCompleteStatusTitle: 'ご注文ありがとうございます',
      orderCompleteMessage: '決済が確認できました。決済完了メールはStripeから注文時のメールアドレスへ送信されます。',
      orderCompleteStatusLabel: '状態',
      orderCompleteStatusValue: '決済を受け付けました',
      orderCompleteEmailMessage:
          'Stripeの決済完了メールをご確認ください。注文番号はお問い合わせや注文照会で使用します。',
      orderCompleteSummaryTitle: '注文概要',
      orderCompleteLookupAction: '注文照会を開く',
      orderCompleteBackToDesignAction: 'Designへ戻る',
      emailSentNoticeTitle: 'Stripeの決済完了メール',
      emailSentNoticeMessage:
          '決済完了メールはStripeから注文時のメールアドレスへ送信されます。受信箱と迷惑メールフォルダをご確認ください。',
      orderEmailMissingGuideTitle: 'Stripeのメールが見つからない場合',
      orderEmailMissingGuideMessage: 'まず次の項目をご確認ください。',
      orderEmailMissingSpamCheck: '迷惑メールフォルダやプロモーションフォルダを確認してください。',
      orderEmailMissingAddressCheck: '注文時のメールアドレスに誤りがないか確認してください。',
      orderEmailMissingDeliveryWait: '配信まで数分かかる場合があります。',
      orderEmailMissingContactSupport: 'それでも見つからない場合は、注文番号を添えてお問い合わせください。',
      contactSupportPromptTitle: 'お困りですか？',
      contactSupportPromptMessage:
          '注文、配送、決済、メールに関する質問はサポートへお問い合わせください。注文番号を添えると確認がスムーズです。',
      contactSupportPromptAction: 'お問い合わせ',
      orderLookup: '注文照会',
      orderNo: '注文番号',
      orderNoHint: 'HF-0001',
      email: 'メールアドレス',
      emailHint: 'name@example.com',
      lookupOrder: '注文を照会',
      orderLookupLoadingTitle: '注文を照会中',
      orderLookupLoadingMessage: '注文番号とメールアドレスを確認しています。',
      orderLookupNotFoundTitle: '注文が見つかりません',
      orderLookupNotFoundMessage: '入力された注文番号とメールアドレスに一致する注文が見つかりませんでした。',
      orderLookupErrorTitle: '注文照会に失敗しました',
      orderLookupErrorMessage: '注文照会を完了できませんでした。もう一度お試しください。',
      orderLookupResultTitle: '注文状況',
      orderLookupResultMessage: 'ご注文の最新状況です。',
      orderLookupOrderStatusLabel: '注文ステータス',
      orderLookupProgressTitle: '進行状況',
      orderLookupProductionStatusLabel: '制作ステータス',
      orderLookupShippingStatusLabel: '発送ステータス',
      orderLookupFulfillmentStatusLabel: '配送手配ステータス',
      orderLookupTrackingNumberLabel: '追跡番号',
      orderLookupTrackingDetailsTitle: '追跡詳細',
      orderLookupCarrierLabel: '配送会社',
      orderLookupShippedAtLabel: '発送日時',
      orderLookupUpdatedAtLabel: '最終更新',
      orderLookupNoTrackingValue: '未登録',
      orderLookupOrderDateLabel: '注文日',
      orderLookupContentTitle: '注文内容',
      orderLookupSelectedSealLabel: '印影',
      orderLookupGemstoneLabel: '宝石材',
      orderLookupLookupAnotherAction: '別の注文を照会',
      language: '言語',
      about: 'このアプリについて',
      howItWorks: '使い方',
      faq: 'FAQ',
      privacy: 'プライバシー',
      terms: '利用規約',
      contact: 'お問い合わせ',
      version: 'バージョン',
      settingsLanguageTitle: 'アプリの言語',
      settingsLanguageMessage: 'アプリ内で使う言語を選択できます。未選択の場合はスマートフォンの言語に合わせます。',
      settingsLanguageEnglish: '英語',
      settingsLanguageJapanese: '日本語',
      settingsFaqIntro: '漢字選択、製作、配送、注文照会についてのよくある質問です。',
      settingsVersionTitle: 'インストール済みアプリのバージョン',
      settingsVersionMessageTemplate: 'バージョン {version}',
    ),
  ),
};
