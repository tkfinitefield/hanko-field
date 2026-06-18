import 'package:flutter/material.dart';

import '../../../app/localization/app_localization.dart';
import '../../../app/localization/language_registry.dart';
import '../../../app/theme/app_theme.dart';
import '../../../core/errors/core_errors.dart';
import '../../../core/widgets/core_widgets.dart';
import '../data/kanji_candidates_repository.dart';
import '../domain/kanji_candidate.dart';
import '../domain/seal_generation.dart';
import '../domain/seal_style_selection.dart';

class DesignHomeScreen extends StatelessWidget {
  const DesignHomeScreen({
    super.key,
    this.onOpenSettings,
    this.onStartDesign,
    this.onOpenMySeals,
    this.onOpenStones,
  });

  final VoidCallback? onOpenSettings;
  final VoidCallback? onStartDesign;
  final VoidCallback? onOpenMySeals;
  final VoidCallback? onOpenStones;

  @override
  Widget build(BuildContext context) {
    return DesignStartScreen(
      onOpenSettings: onOpenSettings,
      onStartDesign: onStartDesign,
      onOpenMySeals: onOpenMySeals,
      onOpenStones: onOpenStones,
    );
  }
}

class DesignStartScreen extends StatelessWidget {
  const DesignStartScreen({
    super.key,
    this.onOpenSettings,
    this.onStartDesign,
    this.onOpenMySeals,
    this.onOpenStones,
  });

  final VoidCallback? onOpenSettings;
  final VoidCallback? onStartDesign;
  final VoidCallback? onOpenMySeals;
  final VoidCallback? onOpenStones;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;

    return LayoutBuilder(
      builder: (context, constraints) {
        final width = constraints.maxWidth;
        final horizontalPadding = width < 410 ? 14.0 : 16.0;
        final cardGap = width < 410 ? 12.0 : 14.0;
        final heroHeight = width < 410 ? 372.0 : 386.0;
        final featureCardHeight = width < 410 ? 252.0 : 260.0;

        return Semantics(
          scopesRoute: true,
          namesRoute: true,
          label: l10n.design,
          explicitChildNodes: true,
          child: SingleChildScrollView(
            physics: const BouncingScrollPhysics(),
            child: ConstrainedBox(
              constraints: BoxConstraints(minHeight: constraints.maxHeight),
              child: Padding(
                padding: EdgeInsets.fromLTRB(
                  horizontalPadding,
                  48,
                  horizontalPadding,
                  31,
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    _DesignHeader(onOpenSettings: onOpenSettings),
                    const SizedBox(height: 17),
                    _HeroDesignCard(
                      height: heroHeight,
                      onStartDesign: onStartDesign,
                    ),
                    const SizedBox(height: 23),
                    Row(
                      children: [
                        Expanded(
                          child: _FeatureCard(
                            title: l10n.savedSeals,
                            body: l10n.savedSealsDescription,
                            assetPath: 'assets/design/com003_saved_seal.png',
                            icon: _FeatureIcon.saved,
                            imageTop: 31,
                            imageRight: -2,
                            imageWidth: 108,
                            height: featureCardHeight,
                            onTap: onOpenMySeals,
                          ),
                        ),
                        SizedBox(width: cardGap),
                        Expanded(
                          child: _FeatureCard(
                            title: l10n.browseStones,
                            body: l10n.browseStonesDescription,
                            assetPath: 'assets/design/com003_gemstones.png',
                            icon: _FeatureIcon.diamond,
                            imageTop: 26,
                            imageRight: 0,
                            imageWidth: 123,
                            height: featureCardHeight,
                            onTap: onOpenStones,
                          ),
                        ),
                      ],
                    ),
                  ],
                ),
              ),
            ),
          ),
        );
      },
    );
  }
}

class NameInputScreen extends StatefulWidget {
  const NameInputScreen({
    super.key,
    required this.onBack,
    required this.onSubmit,
  });

  final VoidCallback onBack;
  final ValueChanged<KanjiCandidatesRequest> onSubmit;

  @override
  State<NameInputScreen> createState() => _NameInputScreenState();
}

class _NameInputScreenState extends State<NameInputScreen> {
  final _nameController = TextEditingController();

  var _gender = KanjiCandidateGender.unspecified;
  var _kanjiStyle = KanjiNameStyle.japanese;
  var _showNameError = false;

  @override
  void initState() {
    super.initState();
    _nameController.addListener(_handleNameChanged);
  }

  @override
  void dispose() {
    _nameController
      ..removeListener(_handleNameChanged)
      ..dispose();
    super.dispose();
  }

  bool get _isNameValid {
    final realName = _nameController.text.trim();
    return realName.isNotEmpty && realName.length <= 80;
  }

  void _handleNameChanged() {
    setState(() {});
  }

  void _submit() {
    final realName = _nameController.text.trim();
    if (!_isNameValid) {
      setState(() => _showNameError = true);
      return;
    }

    widget.onSubmit(
      KanjiCandidatesRequest(
        realName: realName,
        reasonLanguage: _reasonLanguageFor(context),
        gender: _gender,
        kanjiStyle: _kanjiStyle,
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;

    return _DesignStepScaffold(
      title: l10n.designNameTitle,
      onBack: widget.onBack,
      children: [
        HankoSurfaceCard(
          padding: const EdgeInsets.fromLTRB(26, 30, 26, 28),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const Center(child: _SealMedallion()),
              const SizedBox(height: 22),
              const Center(child: _DividerMark()),
              const SizedBox(height: 26),
              Text(
                l10n.designNameIntro,
                textAlign: TextAlign.center,
                style: HankoTextStyles.sectionTitle,
              ),
              if (_showNameError && !_isNameValid) ...[
                const SizedBox(height: 18),
                _InlineAlert(message: l10n.designInvalidNameSummary),
              ],
              const SizedBox(height: 26),
              HankoTextField(
                label: l10n.designNameLabel,
                hintText: l10n.designNameHint,
                controller: _nameController,
                keyboardType: TextInputType.name,
                textInputAction: TextInputAction.done,
                onFieldSubmitted: (_) => _submit(),
                errorText: _showNameError && !_isNameValid
                    ? l10n.designInvalidNameMessage
                    : null,
              ),
              const SizedBox(height: 10),
              Text(l10n.designNameHelp, style: HankoTextStyles.compactBody),
              const SizedBox(height: 20),
              DropdownButtonFormField<KanjiCandidateGender>(
                initialValue: _gender,
                isExpanded: true,
                decoration: InputDecoration(labelText: l10n.designGenderLabel),
                items: [
                  for (final gender in KanjiCandidateGender.values)
                    DropdownMenuItem(
                      value: gender,
                      child: Text(_genderLabel(l10n, gender)),
                    ),
                ],
                onChanged: (gender) {
                  if (gender == null) {
                    return;
                  }
                  setState(() => _gender = gender);
                },
              ),
              const SizedBox(height: 14),
              DropdownButtonFormField<KanjiNameStyle>(
                initialValue: _kanjiStyle,
                isExpanded: true,
                decoration: InputDecoration(
                  labelText: l10n.designKanjiStyleLabel,
                ),
                items: [
                  for (final style in KanjiNameStyle.values)
                    DropdownMenuItem(
                      value: style,
                      child: Text(_kanjiStyleLabel(l10n, style)),
                    ),
                ],
                onChanged: (style) {
                  if (style == null) {
                    return;
                  }
                  setState(() => _kanjiStyle = style);
                },
              ),
              const SizedBox(height: 24),
              HankoPrimaryButton(label: l10n.suggestKanji, onPressed: _submit),
            ],
          ),
        ),
        const SizedBox(height: 22),
        HankoSurfaceCard(
          padding: const EdgeInsets.fromLTRB(24, 24, 24, 24),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.center,
            children: [
              const _TipBadge(),
              const SizedBox(width: 20),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      l10n.designKanjiTipTitle,
                      style: HankoTextStyles.cardTitle,
                    ),
                    const SizedBox(height: 10),
                    Text(
                      l10n.designKanjiTipMessage,
                      style: HankoTextStyles.body,
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ],
    );
  }
}

class KanjiSuggestionLoadingScreen extends StatefulWidget {
  const KanjiSuggestionLoadingScreen({
    super.key,
    required this.request,
    required this.generateCandidates,
    required this.onLoaded,
    required this.onError,
    required this.onBack,
  });

  final KanjiCandidatesRequest request;
  final KanjiCandidatesGenerator generateCandidates;
  final ValueChanged<KanjiCandidatesResult> onLoaded;
  final ValueChanged<Object> onError;
  final VoidCallback onBack;

  @override
  State<KanjiSuggestionLoadingScreen> createState() =>
      _KanjiSuggestionLoadingScreenState();
}

class _KanjiSuggestionLoadingScreenState
    extends State<KanjiSuggestionLoadingScreen> {
  @override
  void initState() {
    super.initState();
    _loadCandidates();
  }

  Future<void> _loadCandidates() async {
    try {
      final result = await widget.generateCandidates(widget.request);
      if (!mounted) {
        return;
      }
      widget.onLoaded(result);
    } catch (error) {
      if (!mounted) {
        return;
      }
      widget.onError(error);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;

    return _DesignStepScaffold(
      title: l10n.designLoadingTitle,
      onBack: widget.onBack,
      children: [
        HankoSurfaceCard(
          padding: const EdgeInsets.fromLTRB(26, 30, 26, 28),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const Center(child: _SealMedallion(icon: Icons.search)),
              const SizedBox(height: 24),
              const Center(child: _DividerMark()),
              const SizedBox(height: 26),
              Text(
                l10n.designLoadingMessage,
                textAlign: TextAlign.center,
                style: HankoTextStyles.sectionTitle,
              ),
              const SizedBox(height: 26),
              const Center(
                child: SizedBox.square(
                  dimension: 48,
                  child: CircularProgressIndicator(
                    strokeWidth: 3,
                    color: HankoColors.gold,
                  ),
                ),
              ),
              const SizedBox(height: 28),
              _RequestSummaryCard(request: widget.request),
            ],
          ),
        ),
      ],
    );
  }
}

class KanjiSuggestionsScreen extends StatelessWidget {
  const KanjiSuggestionsScreen({
    super.key,
    required this.result,
    required this.onOpenCandidate,
    required this.onBack,
  });

  final KanjiCandidatesResult result;
  final ValueChanged<KanjiCandidate> onOpenCandidate;
  final VoidCallback onBack;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;

    return _DesignStepScaffold(
      title: l10n.kanjiSuggestionsTitle,
      onBack: onBack,
      children: [
        HankoSurfaceCard(
          padding: const EdgeInsets.fromLTRB(26, 30, 26, 28),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const Center(child: _SealMedallion(icon: Icons.auto_awesome)),
              const SizedBox(height: 22),
              const Center(child: _DividerMark()),
              const SizedBox(height: 26),
              Text(
                l10n.kanjiSuggestionsMessage,
                textAlign: TextAlign.center,
                style: HankoTextStyles.sectionTitle,
              ),
              const SizedBox(height: 24),
              for (final candidate in result.candidates) ...[
                _KanjiCandidateCard(
                  candidate: candidate,
                  onTap: () => onOpenCandidate(candidate),
                ),
                const SizedBox(height: 14),
              ],
            ],
          ),
        ),
      ],
    );
  }
}

class KanjiCandidateDetailScreen extends StatefulWidget {
  const KanjiCandidateDetailScreen({
    super.key,
    required this.candidate,
    required this.onBack,
    this.onSelected,
  });

  final KanjiCandidate candidate;
  final VoidCallback onBack;
  final ValueChanged<KanjiCandidate>? onSelected;

  @override
  State<KanjiCandidateDetailScreen> createState() =>
      _KanjiCandidateDetailScreenState();
}

class _KanjiCandidateDetailScreenState
    extends State<KanjiCandidateDetailScreen> {
  var _isSelected = false;

  void _selectCandidate() {
    if (widget.onSelected != null) {
      widget.onSelected!(widget.candidate);
      return;
    }
    setState(() => _isSelected = true);
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final candidate = widget.candidate;
    final meaning = candidate.meaning?.trim();
    final reason = candidate.reason.trim();
    final impressions = _candidateImpressions(candidate);

    return _DesignStepScaffold(
      title: l10n.kanjiCandidateDetailTitle,
      onBack: widget.onBack,
      children: [
        HankoSurfaceCard(
          padding: const EdgeInsets.fromLTRB(26, 30, 26, 28),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Center(
                child: DecoratedBox(
                  decoration: BoxDecoration(
                    color: HankoColors.medallion,
                    shape: BoxShape.circle,
                    border: Border.all(color: HankoColors.gold, width: 1.2),
                  ),
                  child: SizedBox.square(
                    dimension: 118,
                    child: Center(
                      child: Text(
                        candidate.kanji,
                        textAlign: TextAlign.center,
                        style: HankoTextStyles.pageTitle.copyWith(
                          fontSize: candidate.kanji.length <= 1 ? 48 : 38,
                        ),
                      ),
                    ),
                  ),
                ),
              ),
              const SizedBox(height: 24),
              const Center(child: _DividerMark()),
              const SizedBox(height: 24),
              _CandidateDetailLine(
                label: l10n.kanjiReadingLabel,
                value: candidate.reading,
              ),
              if (meaning != null && meaning.isNotEmpty) ...[
                const SizedBox(height: 14),
                _CandidateDetailBlock(
                  label: l10n.kanjiMeaningLabel,
                  value: meaning,
                ),
              ],
              if (impressions.isNotEmpty) ...[
                const SizedBox(height: 18),
                Text(l10n.kanjiImpressionLabel, style: HankoTextStyles.label),
                const SizedBox(height: 10),
                Wrap(
                  spacing: 8,
                  runSpacing: 8,
                  children: [
                    for (final impression in impressions)
                      _CandidatePill(label: impression),
                  ],
                ),
              ],
              if (reason.isNotEmpty) ...[
                const SizedBox(height: 18),
                _CandidateDetailBlock(
                  label: l10n.kanjiReasonLabel,
                  value: reason,
                ),
              ],
              if (_isSelected) ...[
                const SizedBox(height: 22),
                _InlineConfirmation(
                  title: l10n.kanjiSelectedTitle,
                  message: l10n.kanjiSelectedMessage,
                ),
              ],
              const SizedBox(height: 24),
              HankoPrimaryButton(
                label: _isSelected ? l10n.kanjiSelectedTitle : l10n.selectKanji,
                icon: _isSelected ? Icons.check : Icons.arrow_forward,
                onPressed: _selectCandidate,
              ),
            ],
          ),
        ),
      ],
    );
  }
}

class SealStyleSelectionScreen extends StatefulWidget {
  const SealStyleSelectionScreen({
    super.key,
    required this.candidate,
    required this.onBack,
    this.initialSelection = const SealStyleSelection(),
    this.onConfirmed,
    this.onGenerate,
  });

  final KanjiCandidate candidate;
  final VoidCallback onBack;
  final SealStyleSelection initialSelection;
  final ValueChanged<SealStyleSelection>? onConfirmed;
  final ValueChanged<SealStyleSelection>? onGenerate;

  @override
  State<SealStyleSelectionScreen> createState() =>
      _SealStyleSelectionScreenState();
}

class _SealStyleSelectionScreenState extends State<SealStyleSelectionScreen> {
  late var _selection = widget.initialSelection;
  var _isConfirmed = false;

  void _updateSelection(SealStyleSelection selection) {
    setState(() {
      _selection = selection;
      _isConfirmed = false;
    });
  }

  void _confirmSelection() {
    setState(() => _isConfirmed = true);
    widget.onConfirmed?.call(_selection);
  }

  void _generateSeal() {
    widget.onGenerate?.call(_selection);
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;

    return _DesignStepScaffold(
      title: l10n.sealStyleTitle,
      onBack: widget.onBack,
      children: [
        HankoSurfaceCard(
          padding: const EdgeInsets.fromLTRB(26, 30, 26, 28),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const Center(child: _SealMedallion(icon: Icons.tune)),
              const SizedBox(height: 22),
              const Center(child: _DividerMark()),
              const SizedBox(height: 26),
              Text(
                l10n.sealStyleMessage,
                textAlign: TextAlign.center,
                style: HankoTextStyles.sectionTitle,
              ),
              const SizedBox(height: 24),
              _SelectedKanjiStyleCard(candidate: widget.candidate),
              const SizedBox(height: 24),
              _StyleOptionGroup<SealShape>(
                key: const Key('DES-006-seal-shape-options'),
                label: l10n.sealShapeLabel,
                selectedValue: _selection.shape,
                options: [
                  _StyleOption(
                    value: SealShape.square,
                    label: _sealShapeLabel(l10n, SealShape.square),
                    icon: Icons.crop_square,
                  ),
                  _StyleOption(
                    value: SealShape.round,
                    label: _sealShapeLabel(l10n, SealShape.round),
                    icon: Icons.circle_outlined,
                  ),
                ],
                onChanged: (shape) {
                  _updateSelection(_selection.copyWith(shape: shape));
                },
              ),
              const SizedBox(height: 20),
              _StyleOptionGroup<SealStyleName>(
                key: const Key('DES-006-seal-style-options'),
                label: l10n.sealStyleNameLabel,
                selectedValue: _selection.style,
                options: [
                  _StyleOption(
                    value: SealStyleName.traditional,
                    label: _sealStyleNameLabel(l10n, SealStyleName.traditional),
                    icon: Icons.temple_buddhist_outlined,
                  ),
                  _StyleOption(
                    value: SealStyleName.elegant,
                    label: _sealStyleNameLabel(l10n, SealStyleName.elegant),
                    icon: Icons.auto_awesome,
                  ),
                  _StyleOption(
                    value: SealStyleName.soft,
                    label: _sealStyleNameLabel(l10n, SealStyleName.soft),
                    icon: Icons.spa_outlined,
                  ),
                  _StyleOption(
                    value: SealStyleName.bold,
                    label: _sealStyleNameLabel(l10n, SealStyleName.bold),
                    icon: Icons.format_bold,
                  ),
                ],
                onChanged: (style) {
                  _updateSelection(_selection.copyWith(style: style));
                },
              ),
              const SizedBox(height: 20),
              _StyleOptionGroup<SealStrokeWeight>(
                key: const Key('DES-006-seal-stroke-options'),
                label: l10n.sealStrokeWeightLabel,
                selectedValue: _selection.strokeWeight,
                options: [
                  _StyleOption(
                    value: SealStrokeWeight.standard,
                    label: _sealStrokeWeightLabel(
                      l10n,
                      SealStrokeWeight.standard,
                    ),
                    icon: Icons.line_weight,
                  ),
                  _StyleOption(
                    value: SealStrokeWeight.bold,
                    label: _sealStrokeWeightLabel(l10n, SealStrokeWeight.bold),
                    icon: Icons.format_bold,
                  ),
                ],
                onChanged: (strokeWeight) {
                  _updateSelection(
                    _selection.copyWith(strokeWeight: strokeWeight),
                  );
                },
              ),
              const SizedBox(height: 20),
              _StyleOptionGroup<SealBalance>(
                key: const Key('DES-006-seal-balance-options'),
                label: l10n.sealBalanceLabel,
                selectedValue: _selection.balance,
                options: [
                  _StyleOption(
                    value: SealBalance.airy,
                    label: _sealBalanceLabel(l10n, SealBalance.airy),
                    icon: Icons.air,
                  ),
                  _StyleOption(
                    value: SealBalance.balanced,
                    label: _sealBalanceLabel(l10n, SealBalance.balanced),
                    icon: Icons.balance,
                  ),
                  _StyleOption(
                    value: SealBalance.dense,
                    label: _sealBalanceLabel(l10n, SealBalance.dense),
                    icon: Icons.density_medium,
                  ),
                ],
                onChanged: (balance) {
                  _updateSelection(_selection.copyWith(balance: balance));
                },
              ),
              const SizedBox(height: 24),
              _SealStyleSummaryCard(selection: _selection),
              if (_isConfirmed) ...[
                const SizedBox(height: 22),
                _InlineConfirmation(
                  title: l10n.sealStyleConfirmedTitle,
                  message: l10n.sealStyleConfirmedMessage,
                ),
              ],
              const SizedBox(height: 24),
              HankoPrimaryButton(
                label: _isConfirmed ? l10n.generateSeal : l10n.confirmStyle,
                icon: _isConfirmed ? Icons.auto_fix_high : Icons.check,
                onPressed: _isConfirmed ? _generateSeal : _confirmSelection,
              ),
            ],
          ),
        ),
      ],
    );
  }
}

class SealGenerationLoadingScreen extends StatefulWidget {
  const SealGenerationLoadingScreen({
    super.key,
    required this.request,
    required this.generateSealDesigns,
    required this.onGenerated,
    required this.onError,
    required this.onBack,
  });

  final SealGenerationRequest request;
  final SealDesignsGenerator generateSealDesigns;
  final ValueChanged<SealGenerationResult> onGenerated;
  final ValueChanged<Object> onError;
  final VoidCallback onBack;

  @override
  State<SealGenerationLoadingScreen> createState() =>
      _SealGenerationLoadingScreenState();
}

class _SealGenerationLoadingScreenState
    extends State<SealGenerationLoadingScreen> {
  @override
  void initState() {
    super.initState();
    _generateSealDesigns();
  }

  Future<void> _generateSealDesigns() async {
    try {
      final result = await widget.generateSealDesigns(widget.request);
      if (!mounted) {
        return;
      }
      widget.onGenerated(result);
    } catch (error) {
      if (!mounted) {
        return;
      }
      widget.onError(error);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;

    return _DesignStepScaffold(
      title: l10n.sealGenerationLoadingTitle,
      onBack: widget.onBack,
      children: [
        HankoSurfaceCard(
          padding: const EdgeInsets.fromLTRB(26, 30, 26, 28),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const Center(child: _SealMedallion(icon: Icons.auto_fix_high)),
              const SizedBox(height: 24),
              const Center(child: _DividerMark()),
              const SizedBox(height: 26),
              Text(
                l10n.sealGenerationLoadingMessage,
                textAlign: TextAlign.center,
                style: HankoTextStyles.sectionTitle,
              ),
              const SizedBox(height: 12),
              Text(
                l10n.sealGenerationLoadingDetail,
                textAlign: TextAlign.center,
                style: HankoTextStyles.body,
              ),
              const SizedBox(height: 26),
              const Center(
                child: SizedBox.square(
                  dimension: 48,
                  child: CircularProgressIndicator(
                    strokeWidth: 3,
                    color: HankoColors.gold,
                  ),
                ),
              ),
              const SizedBox(height: 28),
              _SealGenerationSummaryCard(request: widget.request),
            ],
          ),
        ),
      ],
    );
  }
}

class SealVariantSelectionScreen extends StatefulWidget {
  const SealVariantSelectionScreen({
    super.key,
    required this.result,
    required this.onSelected,
    required this.onBack,
    this.onRegenerate,
  });

  final SealGenerationResult result;
  final ValueChanged<SealDesignVariant> onSelected;
  final VoidCallback onBack;
  final VoidCallback? onRegenerate;

  @override
  State<SealVariantSelectionScreen> createState() =>
      _SealVariantSelectionScreenState();
}

class _SealVariantSelectionScreenState
    extends State<SealVariantSelectionScreen> {
  SealDesignVariant? _selectedVariant;

  void _selectVariant(SealDesignVariant variant) {
    setState(() => _selectedVariant = variant);
    widget.onSelected(variant);
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final variants = widget.result.variants;
    final selectedVariant = _selectedVariant;

    return _DesignStepScaffold(
      title: l10n.sealVariantSelectionTitle,
      onBack: widget.onBack,
      children: [
        HankoSurfaceCard(
          padding: const EdgeInsets.fromLTRB(26, 30, 26, 28),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const Center(child: _SealMedallion(icon: Icons.dashboard)),
              const SizedBox(height: 24),
              const Center(child: _DividerMark()),
              const SizedBox(height: 26),
              Text(
                l10n.sealVariantSelectionMessage,
                textAlign: TextAlign.center,
                style: HankoTextStyles.sectionTitle,
              ),
              const SizedBox(height: 24),
              _SealGenerationSummaryCard(request: widget.result.request),
            ],
          ),
        ),
        const SizedBox(height: 18),
        for (final variant in variants) ...[
          _SealVariantCard(
            variant: variant,
            selected: selectedVariant?.id == variant.id,
            onTap: () => _selectVariant(variant),
          ),
          const SizedBox(height: 14),
        ],
        if (selectedVariant != null) ...[
          _InlineConfirmation(
            title: l10n.sealVariantSelectedTitle,
            message: l10n.sealVariantSelectedMessage,
          ),
          const SizedBox(height: 14),
        ],
        if (widget.onRegenerate != null)
          _SecondaryActionButton(
            label: l10n.regenerateSeal,
            onPressed: widget.onRegenerate!,
          ),
      ],
    );
  }
}

class SealPreviewDetailScreen extends StatelessWidget {
  const SealPreviewDetailScreen({
    super.key,
    required this.result,
    required this.variant,
    required this.onSave,
    required this.onChooseStone,
    required this.onBack,
  });

  final SealGenerationResult result;
  final SealDesignVariant variant;
  final VoidCallback onSave;
  final VoidCallback onChooseStone;
  final VoidCallback onBack;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final request = result.request;
    final candidate = request.candidate;
    final style = request.style;

    return _DesignStepScaffold(
      title: l10n.sealPreviewTitle,
      onBack: onBack,
      children: [
        HankoSurfaceCard(
          padding: const EdgeInsets.fromLTRB(24, 28, 24, 26),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(
                l10n.sealPreviewMessage,
                textAlign: TextAlign.center,
                style: HankoTextStyles.body,
              ),
              const SizedBox(height: 22),
              _SealPreviewFrame(variant: variant, shape: style.shape),
            ],
          ),
        ),
        const SizedBox(height: 18),
        _SealPreviewDetailsCard(
          rows: [
            (
              icon: Icons.draw_outlined,
              label: l10n.kanjiSuggestionsTitle,
              value: candidate.kanji,
            ),
            (
              icon: Icons.translate,
              label: l10n.kanjiMeaningLabel,
              value: _previewDetailValue(candidate.meaning, candidate.reason),
            ),
            (
              icon: Icons.auto_awesome_outlined,
              label: l10n.sealPreviewVariantLabel,
              value: variant.label,
            ),
            (
              icon: Icons.category_outlined,
              label: l10n.sealShapeLabel,
              value: _sealShapeLabel(l10n, style.shape),
            ),
            (
              icon: Icons.brush_outlined,
              label: l10n.sealStyleNameLabel,
              value: _sealStyleNameLabel(l10n, style.style),
            ),
            (
              icon: Icons.line_weight,
              label: l10n.sealStrokeWeightLabel,
              value: _sealStrokeWeightLabel(l10n, style.strokeWeight),
            ),
            (
              icon: Icons.balance_outlined,
              label: l10n.sealBalanceLabel,
              value: _sealBalanceLabel(l10n, style.balance),
            ),
          ],
        ),
        const SizedBox(height: 20),
        HankoPrimaryButton(
          label: l10n.saveSeal,
          icon: Icons.bookmark_add_outlined,
          onPressed: onSave,
        ),
        const SizedBox(height: 12),
        _SecondaryActionButton(
          label: l10n.chooseStone,
          onPressed: onChooseStone,
        ),
      ],
    );
  }
}

class SealSaveConfirmationScreen extends StatelessWidget {
  const SealSaveConfirmationScreen({
    super.key,
    required this.result,
    required this.variant,
    required this.onOpenMySeals,
    required this.onChooseStone,
    required this.onCreateAnother,
    required this.onBack,
  });

  final SealGenerationResult result;
  final SealDesignVariant variant;
  final VoidCallback onOpenMySeals;
  final VoidCallback onChooseStone;
  final VoidCallback onCreateAnother;
  final VoidCallback onBack;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;

    return _DesignStepScaffold(
      title: l10n.sealSavedTitle,
      onBack: onBack,
      children: [
        HankoSurfaceCard(
          padding: const EdgeInsets.fromLTRB(26, 30, 26, 28),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const Center(child: _SealMedallion(icon: Icons.check)),
              const SizedBox(height: 24),
              const Center(child: _DividerMark()),
              const SizedBox(height: 26),
              Text(
                l10n.sealSavedHeading,
                textAlign: TextAlign.center,
                style: HankoTextStyles.sectionTitle,
              ),
              const SizedBox(height: 12),
              Text(
                l10n.sealSavedMessage,
                textAlign: TextAlign.center,
                style: HankoTextStyles.body,
              ),
            ],
          ),
        ),
        const SizedBox(height: 18),
        _SavedSealSummaryCard(result: result, variant: variant),
        const SizedBox(height: 20),
        HankoPrimaryButton(
          label: l10n.chooseStone,
          icon: Icons.diamond_outlined,
          onPressed: onChooseStone,
        ),
        const SizedBox(height: 12),
        _SecondaryActionButton(
          label: l10n.goToMySeals,
          onPressed: onOpenMySeals,
        ),
        const SizedBox(height: 18),
        Center(
          child: TextButton(
            onPressed: onCreateAnother,
            style: TextButton.styleFrom(
              foregroundColor: HankoColors.red,
              textStyle: HankoTextStyles.buttonLabel,
            ),
            child: Text(l10n.createAnotherSeal),
          ),
        ),
      ],
    );
  }
}

class SealSaveErrorScreen extends StatelessWidget {
  const SealSaveErrorScreen({
    super.key,
    required this.result,
    required this.variant,
    required this.error,
    required this.onRetry,
    required this.onBack,
  });

  final SealGenerationResult result;
  final SealDesignVariant variant;
  final Object error;
  final VoidCallback onRetry;
  final VoidCallback onBack;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;

    return _DesignStepScaffold(
      title: l10n.design,
      onBack: onBack,
      children: [
        HankoErrorStateView(
          appError: HankoAppError.storage(cause: error),
          actionLabel: l10n.tryAgain,
          onAction: onRetry,
        ),
        const SizedBox(height: 18),
        _SavedSealSummaryCard(result: result, variant: variant),
      ],
    );
  }
}

class SealGenerationErrorScreen extends StatelessWidget {
  const SealGenerationErrorScreen({
    super.key,
    required this.request,
    required this.onRetry,
    required this.onBack,
    this.error,
  });

  final SealGenerationRequest request;
  final Object? error;
  final VoidCallback onRetry;
  final VoidCallback onBack;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final appError = error == null ? null : HankoAppError.fromObject(error);

    return _DesignStepScaffold(
      title: l10n.design,
      onBack: onBack,
      children: [
        _DesignStateCard(
          icon: Icons.broken_image_outlined,
          title: appError?.title(l10n) ?? l10n.sealGenerationErrorTitle,
          message: appError?.message(l10n) ?? l10n.sealGenerationErrorMessage,
          primaryLabel: l10n.tryAgain,
          onPrimary: onRetry,
          secondaryLabel: l10n.back,
          onSecondary: onBack,
          child: _SealGenerationSummaryCard(request: request),
        ),
        const SizedBox(height: 22),
        _StateTipCard(message: l10n.sealGenerationErrorTip),
      ],
    );
  }
}

class SealGenerationLimitScreen extends StatelessWidget {
  const SealGenerationLimitScreen({
    super.key,
    required this.request,
    required this.onAdjustStyle,
    required this.onBack,
  });

  final SealGenerationRequest request;
  final VoidCallback onAdjustStyle;
  final VoidCallback onBack;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;

    return _DesignStepScaffold(
      title: l10n.design,
      onBack: onBack,
      children: [
        _DesignStateCard(
          icon: Icons.hourglass_disabled_outlined,
          title: l10n.sealGenerationLimitTitle,
          message: l10n.sealGenerationLimitMessage,
          primaryLabel: l10n.adjustStyle,
          onPrimary: onAdjustStyle,
          secondaryLabel: l10n.back,
          onSecondary: onBack,
          child: _SealGenerationSummaryCard(request: request),
        ),
        const SizedBox(height: 22),
        _StateTipCard(message: l10n.sealGenerationLimitTip),
      ],
    );
  }
}

class KanjiSuggestionErrorScreen extends StatelessWidget {
  const KanjiSuggestionErrorScreen({
    super.key,
    required this.onRetry,
    required this.onBack,
    this.request,
    this.error,
  });

  final KanjiCandidatesRequest? request;
  final Object? error;
  final VoidCallback onRetry;
  final VoidCallback onBack;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final appError = error == null ? null : HankoAppError.fromObject(error);

    return _DesignStepScaffold(
      title: l10n.design,
      onBack: onBack,
      children: [
        _DesignStateCard(
          icon: Icons.error_outline,
          title: appError?.title(l10n) ?? l10n.designSuggestionErrorTitle,
          message: appError?.message(l10n) ?? l10n.designSuggestionErrorMessage,
          primaryLabel: l10n.tryAgain,
          onPrimary: onRetry,
          secondaryLabel: l10n.back,
          onSecondary: onBack,
          child: request == null
              ? null
              : _RequestSummaryCard(request: request!),
        ),
        const SizedBox(height: 22),
        _StateTipCard(message: l10n.designErrorTip),
      ],
    );
  }
}

class _KanjiCandidateCard extends StatelessWidget {
  const _KanjiCandidateCard({required this.candidate, required this.onTap});

  final KanjiCandidate candidate;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final reason = candidate.reason.trim();
    final meaning = candidate.meaning?.trim();
    final impressions = _candidateImpressions(candidate);

    return Material(
      color: Colors.transparent,
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(HankoRadii.sm),
        child: DecoratedBox(
          decoration: BoxDecoration(
            color: HankoColors.background.withValues(alpha: 0.56),
            border: Border.all(color: HankoColors.surfaceBorder),
            borderRadius: BorderRadius.circular(HankoRadii.sm),
          ),
          child: Padding(
            padding: const EdgeInsets.fromLTRB(18, 18, 18, 18),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    DecoratedBox(
                      decoration: const BoxDecoration(
                        color: HankoColors.medallion,
                        shape: BoxShape.circle,
                      ),
                      child: SizedBox.square(
                        dimension: 74,
                        child: Center(
                          child: Text(
                            candidate.kanji,
                            textAlign: TextAlign.center,
                            style: HankoTextStyles.sectionTitle.copyWith(
                              color: HankoColors.red,
                              fontSize: candidate.kanji.length <= 1 ? 36 : 28,
                            ),
                          ),
                        ),
                      ),
                    ),
                    const SizedBox(width: 18),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            l10n.kanjiReadingLabel,
                            style: HankoTextStyles.compactBody,
                          ),
                          const SizedBox(height: 4),
                          Text(
                            candidate.reading,
                            style: HankoTextStyles.cardTitle,
                          ),
                          if (meaning != null && meaning.isNotEmpty) ...[
                            const SizedBox(height: 10),
                            _CandidateDetailLine(
                              label: l10n.kanjiMeaningLabel,
                              value: meaning,
                            ),
                          ],
                        ],
                      ),
                    ),
                    const SizedBox(width: 8),
                    const Icon(
                      Icons.chevron_right,
                      color: HankoColors.gold,
                      size: 26,
                    ),
                  ],
                ),
                if (impressions.isNotEmpty) ...[
                  const SizedBox(height: 16),
                  Text(l10n.kanjiImpressionLabel, style: HankoTextStyles.label),
                  const SizedBox(height: 10),
                  Wrap(
                    spacing: 8,
                    runSpacing: 8,
                    children: [
                      for (final impression in impressions)
                        _CandidatePill(label: impression),
                    ],
                  ),
                ],
                if (reason.isNotEmpty) ...[
                  const SizedBox(height: 16),
                  _CandidateDetailBlock(
                    label: l10n.kanjiReasonLabel,
                    value: reason,
                  ),
                ],
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _CandidateDetailLine extends StatelessWidget {
  const _CandidateDetailLine({required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    return Text.rich(
      TextSpan(
        children: [
          TextSpan(
            text: '$label: ',
            style: HankoTextStyles.compactBody.copyWith(
              color: HankoColors.gold,
              fontWeight: FontWeight.w700,
            ),
          ),
          TextSpan(text: value, style: HankoTextStyles.compactBody),
        ],
      ),
    );
  }
}

class _CandidateDetailBlock extends StatelessWidget {
  const _CandidateDetailBlock({required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(label, style: HankoTextStyles.label),
        const SizedBox(height: 8),
        Text(value, style: HankoTextStyles.compactBody),
      ],
    );
  }
}

class _CandidatePill extends StatelessWidget {
  const _CandidatePill({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    return DecoratedBox(
      decoration: BoxDecoration(
        color: HankoColors.medallion,
        borderRadius: BorderRadius.circular(HankoRadii.sm),
      ),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 7),
        child: Text(
          label,
          style: HankoTextStyles.compactBody.copyWith(color: HankoColors.ink),
        ),
      ),
    );
  }
}

class _SelectedKanjiStyleCard extends StatelessWidget {
  const _SelectedKanjiStyleCard({required this.candidate});

  final KanjiCandidate candidate;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final meaning = candidate.meaning?.trim();

    return DecoratedBox(
      decoration: BoxDecoration(
        color: HankoColors.background.withValues(alpha: 0.56),
        border: Border.all(color: HankoColors.surfaceBorder),
        borderRadius: BorderRadius.circular(HankoRadii.sm),
      ),
      child: Padding(
        padding: const EdgeInsets.fromLTRB(18, 16, 18, 16),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.center,
          children: [
            DecoratedBox(
              decoration: const BoxDecoration(
                color: HankoColors.medallion,
                shape: BoxShape.circle,
              ),
              child: SizedBox.square(
                dimension: 66,
                child: Center(
                  child: Text(
                    candidate.kanji,
                    textAlign: TextAlign.center,
                    style: HankoTextStyles.sectionTitle.copyWith(
                      color: HankoColors.red,
                      fontSize: candidate.kanji.length <= 1 ? 32 : 25,
                    ),
                  ),
                ),
              ),
            ),
            const SizedBox(width: 16),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    l10n.sealStyleSelectedKanjiLabel,
                    style: HankoTextStyles.compactBody,
                  ),
                  const SizedBox(height: 5),
                  Text(candidate.reading, style: HankoTextStyles.cardTitle),
                  if (meaning != null && meaning.isNotEmpty) ...[
                    const SizedBox(height: 8),
                    Text(meaning, style: HankoTextStyles.compactBody),
                  ],
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _StyleOption<T> {
  const _StyleOption({
    required this.value,
    required this.label,
    required this.icon,
  });

  final T value;
  final String label;
  final IconData icon;
}

class _StyleOptionGroup<T> extends StatelessWidget {
  const _StyleOptionGroup({
    super.key,
    required this.label,
    required this.options,
    required this.selectedValue,
    required this.onChanged,
  });

  final String label;
  final List<_StyleOption<T>> options;
  final T selectedValue;
  final ValueChanged<T> onChanged;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(label, style: HankoTextStyles.label),
        const SizedBox(height: 10),
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: [
            for (final option in options)
              _StyleChoiceChip(
                label: option.label,
                icon: option.icon,
                selected: option.value == selectedValue,
                onSelected: () => onChanged(option.value),
              ),
          ],
        ),
      ],
    );
  }
}

class _StyleChoiceChip extends StatelessWidget {
  const _StyleChoiceChip({
    required this.label,
    required this.icon,
    required this.selected,
    required this.onSelected,
  });

  final String label;
  final IconData icon;
  final bool selected;
  final VoidCallback onSelected;

  @override
  Widget build(BuildContext context) {
    final foreground = selected ? Colors.white : HankoColors.ink;

    return ChoiceChip(
      selected: selected,
      showCheckmark: false,
      selectedColor: HankoColors.red,
      backgroundColor: HankoColors.surface,
      side: BorderSide(
        color: selected ? HankoColors.red : HankoColors.surfaceBorder,
      ),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(HankoRadii.sm),
      ),
      labelPadding: EdgeInsets.zero,
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 9),
      label: SizedBox(
        width: 128,
        child: Row(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(icon, size: 18, color: foreground),
            const SizedBox(width: 7),
            Flexible(
              child: Text(
                label,
                overflow: TextOverflow.ellipsis,
                style: HankoTextStyles.label.copyWith(color: foreground),
              ),
            ),
          ],
        ),
      ),
      onSelected: (_) => onSelected(),
    );
  }
}

class _SealStyleSummaryCard extends StatelessWidget {
  const _SealStyleSummaryCard({required this.selection});

  final SealStyleSelection selection;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;

    return DecoratedBox(
      decoration: BoxDecoration(
        color: HankoColors.background.withValues(alpha: 0.56),
        border: Border.all(color: HankoColors.surfaceBorder),
        borderRadius: BorderRadius.circular(HankoRadii.sm),
      ),
      child: Padding(
        padding: const EdgeInsets.fromLTRB(18, 16, 18, 16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(l10n.sealStyleSummaryTitle, style: HankoTextStyles.label),
            const SizedBox(height: 12),
            _RequestSummaryRow(
              label: l10n.sealShapeLabel,
              value: _sealShapeLabel(l10n, selection.shape),
            ),
            _RequestSummaryRow(
              label: l10n.sealStyleNameLabel,
              value: _sealStyleNameLabel(l10n, selection.style),
            ),
            _RequestSummaryRow(
              label: l10n.sealStrokeWeightLabel,
              value: _sealStrokeWeightLabel(l10n, selection.strokeWeight),
            ),
            _RequestSummaryRow(
              label: l10n.sealBalanceLabel,
              value: _sealBalanceLabel(l10n, selection.balance),
            ),
          ],
        ),
      ),
    );
  }
}

class _SealGenerationSummaryCard extends StatelessWidget {
  const _SealGenerationSummaryCard({required this.request});

  final SealGenerationRequest request;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final candidate = request.candidate;
    final selection = request.style;

    return DecoratedBox(
      decoration: BoxDecoration(
        color: HankoColors.background.withValues(alpha: 0.56),
        border: Border.all(color: HankoColors.surfaceBorder),
        borderRadius: BorderRadius.circular(HankoRadii.sm),
      ),
      child: Padding(
        padding: const EdgeInsets.fromLTRB(18, 16, 18, 16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(l10n.sealGenerationStyleDetails, style: HankoTextStyles.label),
            const SizedBox(height: 12),
            _RequestSummaryRow(
              label: l10n.sealStyleSelectedKanjiLabel,
              value: candidate.kanji,
            ),
            _RequestSummaryRow(
              label: l10n.kanjiReadingLabel,
              value: candidate.reading,
            ),
            _RequestSummaryRow(
              label: l10n.sealShapeLabel,
              value: _sealShapeLabel(l10n, selection.shape),
            ),
            _RequestSummaryRow(
              label: l10n.sealStyleNameLabel,
              value: _sealStyleNameLabel(l10n, selection.style),
            ),
            _RequestSummaryRow(
              label: l10n.sealStrokeWeightLabel,
              value: _sealStrokeWeightLabel(l10n, selection.strokeWeight),
            ),
            _RequestSummaryRow(
              label: l10n.sealBalanceLabel,
              value: _sealBalanceLabel(l10n, selection.balance),
            ),
            _RequestSummaryRow(
              label: l10n.sealGenerationAttemptLabel,
              value: '${request.attemptNumber}/${request.maxAttempts}',
            ),
          ],
        ),
      ),
    );
  }
}

class _SealVariantCard extends StatelessWidget {
  const _SealVariantCard({
    required this.variant,
    required this.selected,
    required this.onTap,
  });

  final SealDesignVariant variant;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final borderColor = selected ? HankoColors.red : HankoColors.surfaceBorder;

    return Material(
      color: Colors.transparent,
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(HankoRadii.sm),
        child: DecoratedBox(
          decoration: BoxDecoration(
            color: HankoColors.surface,
            border: Border.all(color: borderColor, width: selected ? 1.4 : 0.7),
            borderRadius: BorderRadius.circular(HankoRadii.sm),
            boxShadow: const [
              BoxShadow(
                color: Color(0x10000000),
                blurRadius: 18,
                offset: Offset(0, 9),
              ),
            ],
          ),
          child: Padding(
            padding: const EdgeInsets.fromLTRB(16, 16, 16, 16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                AspectRatio(
                  aspectRatio: 1,
                  child: ClipRRect(
                    borderRadius: BorderRadius.circular(HankoRadii.sm),
                    child: _SealVariantImage(variant: variant),
                  ),
                ),
                const SizedBox(height: 14),
                Row(
                  crossAxisAlignment: CrossAxisAlignment.center,
                  children: [
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(variant.label, style: HankoTextStyles.cardTitle),
                          const SizedBox(height: 7),
                          Text(variant.id, style: HankoTextStyles.compactBody),
                        ],
                      ),
                    ),
                    const SizedBox(width: 12),
                    AnimatedSwitcher(
                      duration: const Duration(milliseconds: 160),
                      child: selected
                          ? _VariantSelectedBadge(
                              label: l10n.sealVariantSelectedBadge,
                            )
                          : _SmallBadge(icon: Icons.touch_app_outlined),
                    ),
                  ],
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _SealVariantImage extends StatelessWidget {
  const _SealVariantImage({required this.variant});

  final SealDesignVariant variant;

  @override
  Widget build(BuildContext context) {
    final url = variant.downloadUrl.trim();
    if (url.isEmpty) {
      return _SealVariantPlaceholder(variant: variant);
    }

    return Image.network(
      url,
      fit: BoxFit.cover,
      errorBuilder: (context, error, stackTrace) {
        return _SealVariantPlaceholder(variant: variant);
      },
      loadingBuilder: (context, child, loadingProgress) {
        if (loadingProgress == null) {
          return child;
        }
        return _SealVariantPlaceholder(variant: variant, loading: true);
      },
    );
  }
}

class _SealVariantPlaceholder extends StatelessWidget {
  const _SealVariantPlaceholder({required this.variant, this.loading = false});

  final SealDesignVariant variant;
  final bool loading;

  @override
  Widget build(BuildContext context) {
    return DecoratedBox(
      decoration: const BoxDecoration(color: HankoColors.medallion),
      child: Center(
        child: SizedBox.square(
          dimension: 124,
          child: DecoratedBox(
            decoration: BoxDecoration(
              color: HankoColors.surface,
              shape: BoxShape.circle,
              border: Border.all(color: HankoColors.gold, width: 1.2),
            ),
            child: Center(
              child: loading
                  ? const SizedBox.square(
                      dimension: 34,
                      child: CircularProgressIndicator(
                        strokeWidth: 2.4,
                        color: HankoColors.gold,
                      ),
                    )
                  : Text(
                      String.fromCharCodes(variant.label.runes.take(2)),
                      textAlign: TextAlign.center,
                      style: HankoTextStyles.sectionTitle.copyWith(
                        color: HankoColors.red,
                        fontSize: 34,
                      ),
                    ),
            ),
          ),
        ),
      ),
    );
  }
}

class _SealPreviewFrame extends StatelessWidget {
  const _SealPreviewFrame({required this.variant, required this.shape});

  final SealDesignVariant variant;
  final SealShape shape;

  @override
  Widget build(BuildContext context) {
    final previewImage = shape == SealShape.round
        ? ClipOval(child: _SealVariantImage(variant: variant))
        : ClipRRect(
            borderRadius: BorderRadius.circular(HankoRadii.sm),
            child: _SealVariantImage(variant: variant),
          );

    return DecoratedBox(
      decoration: BoxDecoration(
        color: HankoColors.medallion,
        border: Border.all(color: HankoColors.surfaceBorder),
        borderRadius: BorderRadius.circular(HankoRadii.sm),
      ),
      child: Padding(
        padding: const EdgeInsets.all(18),
        child: AspectRatio(
          aspectRatio: 1,
          child: DecoratedBox(
            decoration: BoxDecoration(
              color: HankoColors.surface,
              shape: shape == SealShape.round
                  ? BoxShape.circle
                  : BoxShape.rectangle,
              borderRadius: shape == SealShape.round
                  ? null
                  : BorderRadius.circular(HankoRadii.sm),
              border: Border.all(color: HankoColors.gold, width: 1.2),
            ),
            child: ClipRRect(
              borderRadius: shape == SealShape.round
                  ? BorderRadius.zero
                  : BorderRadius.circular(HankoRadii.sm),
              child: previewImage,
            ),
          ),
        ),
      ),
    );
  }
}

class _SealPreviewDetailsCard extends StatelessWidget {
  const _SealPreviewDetailsCard({required this.rows});

  final List<({IconData icon, String label, String value})> rows;

  @override
  Widget build(BuildContext context) {
    return HankoSurfaceCard(
      padding: const EdgeInsets.fromLTRB(18, 8, 18, 8),
      child: Column(
        children: [
          for (var index = 0; index < rows.length; index++) ...[
            _SealPreviewDetailRow(row: rows[index]),
            if (index < rows.length - 1)
              const Divider(color: HankoColors.surfaceBorder, height: 1),
          ],
        ],
      ),
    );
  }
}

class _SealPreviewDetailRow extends StatelessWidget {
  const _SealPreviewDetailRow({required this.row});

  final ({IconData icon, String label, String value}) row;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 12),
      child: Row(
        children: [
          _SmallBadge(icon: row.icon),
          const SizedBox(width: 14),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(row.label, style: HankoTextStyles.label),
                const SizedBox(height: 7),
                Text(
                  row.value,
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                  style: HankoTextStyles.compactBody,
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _SavedSealSummaryCard extends StatelessWidget {
  const _SavedSealSummaryCard({required this.result, required this.variant});

  final SealGenerationResult result;
  final SealDesignVariant variant;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final request = result.request;
    final style = request.style;

    return HankoSurfaceCard(
      padding: const EdgeInsets.fromLTRB(18, 18, 18, 18),
      child: Row(
        children: [
          SizedBox.square(
            dimension: 104,
            child: ClipRRect(
              borderRadius: BorderRadius.circular(HankoRadii.sm),
              child: _SealVariantImage(variant: variant),
            ),
          ),
          const SizedBox(width: 16),
          Container(width: 1, height: 92, color: HankoColors.gold),
          const SizedBox(width: 16),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  request.candidate.kanji,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: HankoTextStyles.sectionTitle.copyWith(
                    fontFamily: HankoFonts.serif,
                    color: HankoColors.ink,
                  ),
                ),
                const SizedBox(height: 12),
                Text(
                  [
                    _sealStyleNameLabel(l10n, style.style),
                    _sealStrokeWeightLabel(l10n, style.strokeWeight),
                    _sealBalanceLabel(l10n, style.balance),
                  ].join(' / '),
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                  style: HankoTextStyles.compactBody,
                ),
                const SizedBox(height: 12),
                Row(
                  children: [
                    const Icon(
                      Icons.auto_awesome,
                      color: HankoColors.gold,
                      size: 18,
                    ),
                    const SizedBox(width: 8),
                    Expanded(
                      child: Text(
                        variant.label,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: HankoTextStyles.label,
                      ),
                    ),
                  ],
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _VariantSelectedBadge extends StatelessWidget {
  const _VariantSelectedBadge({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    return DecoratedBox(
      decoration: BoxDecoration(
        color: HankoColors.red,
        borderRadius: BorderRadius.circular(HankoRadii.sm),
      ),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 9),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(Icons.check, color: Colors.white, size: 17),
            const SizedBox(width: 6),
            Text(
              label,
              style: HankoTextStyles.label.copyWith(color: Colors.white),
            ),
          ],
        ),
      ),
    );
  }
}

class UnsupportedKanjiResultScreen extends StatelessWidget {
  const UnsupportedKanjiResultScreen({
    super.key,
    required this.request,
    required this.onRetry,
    required this.onEditName,
    required this.onBack,
  });

  final KanjiCandidatesRequest request;
  final VoidCallback onRetry;
  final VoidCallback onEditName;
  final VoidCallback onBack;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;

    return _DesignStepScaffold(
      title: l10n.design,
      onBack: onBack,
      children: [
        _DesignStateCard(
          icon: Icons.search_off,
          title: l10n.designNoKanjiTitle,
          message: l10n.designNoKanjiMessage,
          primaryLabel: l10n.editName,
          onPrimary: onEditName,
          secondaryLabel: l10n.tryAgain,
          onSecondary: onRetry,
          child: _KanjiRulesList(),
        ),
        const SizedBox(height: 22),
        _StateTipCard(message: l10n.designNoKanjiTip),
      ],
    );
  }
}

class _DesignStateCard extends StatelessWidget {
  const _DesignStateCard({
    required this.icon,
    required this.title,
    required this.message,
    required this.primaryLabel,
    required this.onPrimary,
    required this.secondaryLabel,
    required this.onSecondary,
    this.child,
  });

  final IconData icon;
  final String title;
  final String message;
  final String primaryLabel;
  final VoidCallback onPrimary;
  final String secondaryLabel;
  final VoidCallback onSecondary;
  final Widget? child;

  @override
  Widget build(BuildContext context) {
    return HankoSurfaceCard(
      padding: const EdgeInsets.fromLTRB(26, 30, 26, 28),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Center(child: _SealMedallion(icon: icon)),
          const SizedBox(height: 24),
          const Center(child: _DividerMark()),
          const SizedBox(height: 26),
          Text(
            title,
            textAlign: TextAlign.center,
            style: HankoTextStyles.sectionTitle,
          ),
          const SizedBox(height: 12),
          Text(
            message,
            textAlign: TextAlign.center,
            style: HankoTextStyles.body,
          ),
          if (child != null) ...[const SizedBox(height: 24), child!],
          const SizedBox(height: 24),
          HankoPrimaryButton(label: primaryLabel, onPressed: onPrimary),
          const SizedBox(height: 12),
          _SecondaryActionButton(label: secondaryLabel, onPressed: onSecondary),
        ],
      ),
    );
  }
}

class _RequestSummaryCard extends StatelessWidget {
  const _RequestSummaryCard({required this.request});

  final KanjiCandidatesRequest request;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;

    return DecoratedBox(
      decoration: BoxDecoration(
        color: HankoColors.background.withValues(alpha: 0.56),
        border: Border.all(color: HankoColors.surfaceBorder),
        borderRadius: BorderRadius.circular(HankoRadii.sm),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(18, 16, 18, 16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(l10n.designRequestDetails, style: HankoTextStyles.label),
                const SizedBox(height: 12),
                _RequestSummaryRow(
                  label: l10n.designNameLabel,
                  value: request.realName,
                ),
                _RequestSummaryRow(
                  label: l10n.designGenderLabel,
                  value: _genderLabel(l10n, request.gender),
                ),
                _RequestSummaryRow(
                  label: l10n.designKanjiStyleLabel,
                  value: _kanjiStyleLabel(l10n, request.kanjiStyle),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _KanjiRulesList extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final rules = [
      (icon: Icons.looks_two_outlined, label: l10n.designNoKanjiRuleCharacters),
      (icon: Icons.auto_awesome, label: l10n.designNoKanjiRuleCommon),
      (
        icon: Icons.verified_user_outlined,
        label: l10n.designNoKanjiRuleEngraving,
      ),
    ];

    return DecoratedBox(
      decoration: BoxDecoration(
        color: HankoColors.background.withValues(alpha: 0.56),
        border: Border.all(color: HankoColors.surfaceBorder),
        borderRadius: BorderRadius.circular(HankoRadii.sm),
      ),
      child: Column(
        children: [
          for (final rule in rules)
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
              child: Row(
                children: [
                  _SmallBadge(icon: rule.icon),
                  const SizedBox(width: 14),
                  Expanded(
                    child: Text(rule.label, style: HankoTextStyles.body),
                  ),
                ],
              ),
            ),
        ],
      ),
    );
  }
}

class _InlineAlert extends StatelessWidget {
  const _InlineAlert({required this.message});

  final String message;

  @override
  Widget build(BuildContext context) {
    return DecoratedBox(
      decoration: BoxDecoration(
        color: const Color(0xFFFFF7F6),
        border: Border.all(color: HankoColors.error),
        borderRadius: BorderRadius.circular(HankoRadii.sm),
      ),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
        child: Row(
          children: [
            const Icon(Icons.error_outline, color: HankoColors.error, size: 20),
            const SizedBox(width: 10),
            Expanded(
              child: Text(
                message,
                style: HankoTextStyles.label.copyWith(color: HankoColors.error),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _InlineConfirmation extends StatelessWidget {
  const _InlineConfirmation({required this.title, required this.message});

  final String title;
  final String message;

  @override
  Widget build(BuildContext context) {
    return DecoratedBox(
      decoration: BoxDecoration(
        color: const Color(0xFFF7FBF5),
        border: Border.all(color: HankoColors.gold),
        borderRadius: BorderRadius.circular(HankoRadii.sm),
      ),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 13),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Icon(Icons.check_circle_outline, color: HankoColors.gold),
            const SizedBox(width: 10),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(title, style: HankoTextStyles.label),
                  const SizedBox(height: 6),
                  Text(message, style: HankoTextStyles.compactBody),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _StateTipCard extends StatelessWidget {
  const _StateTipCard({required this.message});

  final String message;

  @override
  Widget build(BuildContext context) {
    final tipPrefix = context.l10n.designTipPrefix;

    return HankoSurfaceCard(
      padding: const EdgeInsets.fromLTRB(20, 18, 20, 18),
      child: Row(
        children: [
          const _SmallBadge(icon: Icons.tips_and_updates_outlined),
          const SizedBox(width: 16),
          Expanded(
            child: Text.rich(
              TextSpan(
                children: [
                  TextSpan(
                    text: tipPrefix,
                    style: HankoTextStyles.label.copyWith(
                      color: HankoColors.gold,
                    ),
                  ),
                  TextSpan(text: message),
                ],
              ),
              style: HankoTextStyles.body,
            ),
          ),
        ],
      ),
    );
  }
}

class _SmallBadge extends StatelessWidget {
  const _SmallBadge({required this.icon});

  final IconData icon;

  @override
  Widget build(BuildContext context) {
    return DecoratedBox(
      decoration: const BoxDecoration(
        color: HankoColors.medallion,
        shape: BoxShape.circle,
      ),
      child: SizedBox.square(
        dimension: 48,
        child: Icon(icon, color: HankoColors.gold, size: 24),
      ),
    );
  }
}

class _SecondaryActionButton extends StatelessWidget {
  const _SecondaryActionButton({required this.label, required this.onPressed});

  final String label;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: 52,
      child: OutlinedButton(
        onPressed: onPressed,
        style: OutlinedButton.styleFrom(
          foregroundColor: HankoColors.gold,
          side: const BorderSide(color: HankoColors.gold),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(HankoRadii.sm),
          ),
        ),
        child: Text(label, style: HankoTextStyles.buttonLabel),
      ),
    );
  }
}

class _RequestSummaryRow extends StatelessWidget {
  const _RequestSummaryRow({required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 118,
            child: Text(label, style: HankoTextStyles.compactBody),
          ),
          Expanded(child: Text(value, style: HankoTextStyles.label)),
        ],
      ),
    );
  }
}

class _DesignStepScaffold extends StatelessWidget {
  const _DesignStepScaffold({
    required this.title,
    required this.onBack,
    required this.children,
  });

  final String title;
  final VoidCallback onBack;
  final List<Widget> children;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final width = constraints.maxWidth;
        final horizontalPadding = width < 410 ? 14.0 : 16.0;

        return SingleChildScrollView(
          physics: const BouncingScrollPhysics(),
          child: ConstrainedBox(
            constraints: BoxConstraints(minHeight: constraints.maxHeight),
            child: Padding(
              padding: EdgeInsets.fromLTRB(
                horizontalPadding,
                48,
                horizontalPadding,
                31,
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Row(
                    children: [
                      SizedBox(
                        width: 48,
                        height: 48,
                        child: IconButton(
                          onPressed: onBack,
                          tooltip: MaterialLocalizations.of(
                            context,
                          ).backButtonTooltip,
                          icon: const Icon(Icons.arrow_back_ios_new),
                          color: HankoColors.red,
                        ),
                      ),
                      const SizedBox(width: 8),
                      Expanded(
                        child: FittedBox(
                          fit: BoxFit.scaleDown,
                          child: Text(
                            title,
                            maxLines: 1,
                            style: HankoTextStyles.pageTitle.copyWith(
                              fontSize: 32,
                            ),
                          ),
                        ),
                      ),
                      const SizedBox(width: 56),
                    ],
                  ),
                  const SizedBox(height: 30),
                  ...children,
                ],
              ),
            ),
          ),
        );
      },
    );
  }
}

class _SealMedallion extends StatelessWidget {
  const _SealMedallion({this.icon = Icons.auto_awesome});

  final IconData icon;

  @override
  Widget build(BuildContext context) {
    return DecoratedBox(
      decoration: BoxDecoration(
        color: HankoColors.red,
        shape: BoxShape.circle,
        border: Border.all(color: HankoColors.gold, width: 2),
        boxShadow: const [
          BoxShadow(
            color: Color(0x26961B1D),
            blurRadius: 18,
            offset: Offset(0, 9),
          ),
        ],
      ),
      child: SizedBox.square(
        dimension: 92,
        child: Icon(icon, color: Colors.white, size: 42),
      ),
    );
  }
}

class _TipBadge extends StatelessWidget {
  const _TipBadge();

  @override
  Widget build(BuildContext context) {
    return const DecoratedBox(
      decoration: BoxDecoration(
        color: HankoColors.medallion,
        shape: BoxShape.circle,
      ),
      child: SizedBox.square(
        dimension: 76,
        child: Icon(Icons.auto_awesome, color: HankoColors.gold, size: 34),
      ),
    );
  }
}

String _reasonLanguageFor(BuildContext context) {
  return fallbackRouteCodeForLocale(context.l10n.locale);
}

String _genderLabel(HankoLocalizations l10n, KanjiCandidateGender gender) {
  return switch (gender) {
    KanjiCandidateGender.unspecified => l10n.designGenderUnspecified,
    KanjiCandidateGender.male => l10n.designGenderMale,
    KanjiCandidateGender.female => l10n.designGenderFemale,
  };
}

String _kanjiStyleLabel(HankoLocalizations l10n, KanjiNameStyle style) {
  return switch (style) {
    KanjiNameStyle.japanese => l10n.designKanjiStyleJapanese,
    KanjiNameStyle.chinese => l10n.designKanjiStyleChinese,
    KanjiNameStyle.taiwanese => l10n.designKanjiStyleTaiwanese,
  };
}

String _sealShapeLabel(HankoLocalizations l10n, SealShape shape) {
  return switch (shape) {
    SealShape.square => l10n.sealShapeSquare,
    SealShape.round => l10n.sealShapeRound,
  };
}

String _sealStyleNameLabel(HankoLocalizations l10n, SealStyleName style) {
  return switch (style) {
    SealStyleName.traditional => l10n.sealStyleTraditional,
    SealStyleName.elegant => l10n.sealStyleElegant,
    SealStyleName.soft => l10n.sealStyleSoft,
    SealStyleName.bold => l10n.sealStyleBold,
  };
}

String _sealStrokeWeightLabel(
  HankoLocalizations l10n,
  SealStrokeWeight strokeWeight,
) {
  return switch (strokeWeight) {
    SealStrokeWeight.standard => l10n.sealStrokeStandard,
    SealStrokeWeight.bold => l10n.sealStrokeBold,
  };
}

String _sealBalanceLabel(HankoLocalizations l10n, SealBalance balance) {
  return switch (balance) {
    SealBalance.airy => l10n.sealBalanceAiry,
    SealBalance.balanced => l10n.sealBalanceBalanced,
    SealBalance.dense => l10n.sealBalanceDense,
  };
}

List<String> _candidateImpressions(KanjiCandidate candidate) {
  return candidate.impression
      .map((impression) => impression.trim())
      .where((impression) => impression.isNotEmpty)
      .toList(growable: false);
}

String _previewDetailValue(String? value, String fallback) {
  final trimmed = value?.trim();
  if (trimmed == null || trimmed.isEmpty) {
    return fallback;
  }
  return trimmed;
}

class _DesignHeader extends StatelessWidget {
  const _DesignHeader({required this.onOpenSettings});

  final VoidCallback? onOpenSettings;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;

    return Row(
      crossAxisAlignment: CrossAxisAlignment.center,
      children: [
        Expanded(child: Text(l10n.design, style: HankoTextStyles.pageTitle)),
        IconButton(
          onPressed: onOpenSettings ?? () {},
          tooltip: l10n.settings,
          icon: const Icon(Icons.settings_outlined),
          color: HankoColors.gold,
          iconSize: 34,
          padding: EdgeInsets.zero,
          constraints: const BoxConstraints.tightFor(width: 48, height: 48),
        ),
      ],
    );
  }
}

class _HeroDesignCard extends StatelessWidget {
  const _HeroDesignCard({required this.height, required this.onStartDesign});

  final double height;
  final VoidCallback? onStartDesign;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;

    return HankoSurfaceCard(
      height: height,
      child: Stack(
        clipBehavior: Clip.hardEdge,
        children: [
          Positioned(
            top: 28,
            right: -4,
            width: 198,
            child: _FadedAssetImage(
              assetPath: 'assets/design/com003_hero_seal.png',
            ),
          ),
          Positioned(
            left: 26,
            top: 67,
            width: 246,
            child: Text(
              l10n.createCustomSeal,
              softWrap: false,
              style: HankoTextStyles.heroTitle,
            ),
          ),
          const Positioned(left: 26, top: 174, child: _DividerMark()),
          Positioned(
            left: 27,
            top: 205,
            width: 215,
            child: Text(
              l10n.customSealDescription,
              style: HankoTextStyles.body,
            ),
          ),
          Positioned(
            left: 26,
            top: 282,
            width: 172,
            height: 52,
            child: HankoPrimaryButton(
              label: l10n.startDesigning,
              onPressed: onStartDesign,
            ),
          ),
        ],
      ),
    );
  }
}

class _DividerMark extends StatelessWidget {
  const _DividerMark();

  @override
  Widget build(BuildContext context) {
    return const Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        _GoldLine(),
        SizedBox(width: 10),
        _OutlinedDiamond(size: 10),
        SizedBox(width: 10),
        _GoldLine(),
      ],
    );
  }
}

class _GoldLine extends StatelessWidget {
  const _GoldLine();

  @override
  Widget build(BuildContext context) {
    return Container(width: 58, height: 1, color: HankoColors.gold);
  }
}

class _FeatureCard extends StatelessWidget {
  const _FeatureCard({
    required this.title,
    required this.body,
    required this.assetPath,
    required this.icon,
    required this.imageTop,
    required this.imageRight,
    required this.imageWidth,
    required this.height,
    this.onTap,
  });

  final String title;
  final String body;
  final String assetPath;
  final _FeatureIcon icon;
  final double imageTop;
  final double imageRight;
  final double imageWidth;
  final double height;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      button: onTap != null,
      child: GestureDetector(
        behavior: HitTestBehavior.opaque,
        onTap: onTap,
        child: HankoSurfaceCard(
          height: height,
          radius: HankoRadii.md,
          child: Stack(
            clipBehavior: Clip.hardEdge,
            children: [
              Positioned(
                top: imageTop,
                right: imageRight,
                width: imageWidth,
                child: _FadedAssetImage(assetPath: assetPath),
              ),
              Positioned(
                left: 28,
                top: 29,
                width: 64,
                height: 64,
                child: _IconMedallion(icon: icon),
              ),
              Positioned(
                left: 20,
                right: 13,
                top: 141,
                child: Text(
                  title,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: HankoTextStyles.cardTitle,
                ),
              ),
              Positioned(
                left: 20,
                right: 17,
                top: 177,
                child: Text(body, style: HankoTextStyles.compactBody),
              ),
              const Positioned(
                right: 24,
                bottom: 25,
                child: Icon(
                  Icons.chevron_right,
                  size: 31,
                  color: HankoColors.gold,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _FadedAssetImage extends StatelessWidget {
  const _FadedAssetImage({required this.assetPath});

  final String assetPath;

  @override
  Widget build(BuildContext context) {
    return ShaderMask(
      blendMode: BlendMode.dstIn,
      shaderCallback: (bounds) {
        return LinearGradient(
          begin: Alignment.centerLeft,
          end: Alignment.centerRight,
          stops: const [0, 0.16, 1],
          colors: const [Colors.transparent, Colors.white, Colors.white],
        ).createShader(bounds);
      },
      child: Image.asset(assetPath, fit: BoxFit.contain),
    );
  }
}

class _IconMedallion extends StatelessWidget {
  const _IconMedallion({required this.icon});

  final _FeatureIcon icon;

  @override
  Widget build(BuildContext context) {
    return DecoratedBox(
      decoration: const BoxDecoration(
        color: HankoColors.medallion,
        shape: BoxShape.circle,
      ),
      child: Center(
        child: CustomPaint(
          size: const Size.square(34),
          painter: _FeatureIconPainter(icon),
        ),
      ),
    );
  }
}

class _OutlinedDiamond extends StatelessWidget {
  const _OutlinedDiamond({required this.size});

  final double size;

  @override
  Widget build(BuildContext context) {
    return Transform.rotate(
      angle: 0.7853981634,
      child: Container(
        width: size,
        height: size,
        decoration: BoxDecoration(
          border: Border.all(color: HankoColors.gold, width: 1.5),
        ),
      ),
    );
  }
}

class _FeatureIconPainter extends CustomPainter {
  const _FeatureIconPainter(this.icon);

  final _FeatureIcon icon;

  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()
      ..color = HankoColors.gold
      ..style = PaintingStyle.stroke
      ..strokeWidth = 2
      ..strokeCap = StrokeCap.round
      ..strokeJoin = StrokeJoin.round;

    switch (icon) {
      case _FeatureIcon.saved:
        final book = RRect.fromRectAndRadius(
          Rect.fromLTWH(size.width * 0.13, size.height * 0.13, 22, 26),
          const Radius.circular(2.5),
        );
        canvas.drawRRect(book, paint);
        canvas.drawLine(
          Offset(size.width * 0.26, size.height * 0.33),
          Offset(size.width * 0.13, size.height * 0.33),
          paint,
        );
        final bookmark = Path()
          ..moveTo(size.width * 0.70, size.height * 0.48)
          ..lineTo(size.width * 0.89, size.height * 0.48)
          ..lineTo(size.width * 0.89, size.height * 0.90)
          ..lineTo(size.width * 0.79, size.height * 0.82)
          ..lineTo(size.width * 0.70, size.height * 0.90)
          ..close();
        canvas.drawPath(bookmark, paint);
        break;
      case _FeatureIcon.diamond:
        final path = Path()
          ..moveTo(size.width * 0.50, size.height * 0.91)
          ..lineTo(size.width * 0.08, size.height * 0.35)
          ..lineTo(size.width * 0.25, size.height * 0.12)
          ..lineTo(size.width * 0.75, size.height * 0.12)
          ..lineTo(size.width * 0.92, size.height * 0.35)
          ..close();
        canvas.drawPath(path, paint);
        canvas.drawLine(
          Offset(size.width * 0.08, size.height * 0.35),
          Offset(size.width * 0.92, size.height * 0.35),
          paint,
        );
        canvas.drawLine(
          Offset(size.width * 0.25, size.height * 0.12),
          Offset(size.width * 0.50, size.height * 0.91),
          paint,
        );
        canvas.drawLine(
          Offset(size.width * 0.75, size.height * 0.12),
          Offset(size.width * 0.50, size.height * 0.91),
          paint,
        );
        canvas.drawLine(
          Offset(size.width * 0.39, size.height * 0.12),
          Offset(size.width * 0.30, size.height * 0.35),
          paint,
        );
        canvas.drawLine(
          Offset(size.width * 0.61, size.height * 0.12),
          Offset(size.width * 0.70, size.height * 0.35),
          paint,
        );
        break;
    }
  }

  @override
  bool shouldRepaint(covariant _FeatureIconPainter oldDelegate) {
    return oldDelegate.icon != icon;
  }
}

enum _FeatureIcon { saved, diamond }
