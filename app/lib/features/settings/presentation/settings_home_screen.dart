import 'package:declarative_nav/declarative_nav.dart';
import 'package:flutter/material.dart';

import '../../../app/localization/app_localization.dart';
import '../../../app/localization/language_registry.dart';
import '../../../app/theme/app_theme.dart';
import '../../../core/widgets/core_widgets.dart';
import 'settings_content.dart';

class SettingsHomeScreen extends StatelessWidget {
  const SettingsHomeScreen({super.key, this.onLocaleSelected});

  final ValueChanged<Locale>? onLocaleSelected;

  @override
  Widget build(BuildContext context) {
    return SettingsScreen(onLocaleSelected: onLocaleSelected);
  }
}

enum SettingsInitialDestination { contact }

class SettingsScreen extends StatefulWidget {
  const SettingsScreen({
    super.key,
    this.onClose,
    this.initialDestination,
    this.onLocaleSelected,
  });

  final VoidCallback? onClose;
  final SettingsInitialDestination? initialDestination;
  final ValueChanged<Locale>? onLocaleSelected;

  @override
  State<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends State<SettingsScreen> {
  static const _rootPage = PageEntry(
    key: 'COM-004-settings-root',
    name: '/settings',
  );

  var _pages = const <PageEntry>[_rootPage];

  @override
  void initState() {
    super.initState();
    _pages = _pagesForInitialDestination(widget.initialDestination);
  }

  @override
  void didUpdateWidget(covariant SettingsScreen oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.initialDestination != widget.initialDestination) {
      _pages = _pagesForInitialDestination(widget.initialDestination);
    }
  }

  void _openDestination(_SettingsDestination destination) {
    setState(() {
      _pages = [_rootPage, _pageForDestination(destination)];
    });
  }

  void _popDestination() {
    if (_pages.length <= 1) {
      return;
    }
    setState(() => _pages = const [_rootPage]);
  }

  @override
  Widget build(BuildContext context) {
    return PopScope(
      canPop: false,
      onPopInvokedWithResult: (didPop, result) {
        if (didPop) {
          return;
        }
        if (_pages.length > 1) {
          _popDestination();
          return;
        }
        widget.onClose?.call();
      },
      child: DeclarativePagesNavigator(
        pages: _pages,
        buildPage: (context, page) {
          final destination = page.data;
          if (destination is _SettingsDestination) {
            return _SettingsDetailPage(
              destination: destination,
              onBack: _popDestination,
              onLocaleSelected: widget.onLocaleSelected,
            );
          }

          return _SettingsMenuPage(
            onOpenDestination: _openDestination,
            onClose: widget.onClose,
          );
        },
        onPopTop: _popDestination,
      ),
    );
  }
}

List<PageEntry> _pagesForInitialDestination(
  SettingsInitialDestination? initialDestination,
) {
  final destination = switch (initialDestination) {
    SettingsInitialDestination.contact => _SettingsDestination.contact,
    null => null,
  };
  if (destination == null) {
    return const [_SettingsScreenState._rootPage];
  }
  return [_SettingsScreenState._rootPage, _pageForDestination(destination)];
}

PageEntry _pageForDestination(_SettingsDestination destination) {
  return PageEntry(
    key: destination.pageKey,
    name: destination.routeName,
    data: destination,
  );
}

class _SettingsMenuPage extends StatelessWidget {
  const _SettingsMenuPage({
    required this.onOpenDestination,
    required this.onClose,
  });

  final ValueChanged<_SettingsDestination> onOpenDestination;
  final VoidCallback? onClose;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;

    return _SettingsPageFrame(
      title: l10n.settings,
      trailing: onClose == null
          ? null
          : IconButton(
              onPressed: onClose,
              tooltip: MaterialLocalizations.of(context).closeButtonTooltip,
              icon: const Icon(Icons.close),
              color: HankoColors.gold,
            ),
      children: [
        HankoSurfaceCard(
          padding: const EdgeInsets.symmetric(vertical: 8),
          radius: HankoRadii.md,
          child: Column(
            children: [
              _SettingsRow(
                destination: _SettingsDestination.language,
                onTap: onOpenDestination,
              ),
              _SettingsRow(
                destination: _SettingsDestination.about,
                onTap: onOpenDestination,
              ),
              _SettingsRow(
                destination: _SettingsDestination.howItWorks,
                onTap: onOpenDestination,
              ),
              _SettingsRow(
                destination: _SettingsDestination.faq,
                onTap: onOpenDestination,
              ),
              _SettingsRow(
                destination: _SettingsDestination.privacy,
                onTap: onOpenDestination,
              ),
              _SettingsRow(
                destination: _SettingsDestination.terms,
                onTap: onOpenDestination,
              ),
              _SettingsRow(
                destination: _SettingsDestination.contact,
                onTap: onOpenDestination,
              ),
              _SettingsRow(
                destination: _SettingsDestination.version,
                onTap: onOpenDestination,
              ),
            ],
          ),
        ),
      ],
    );
  }
}

class _SettingsRow extends StatelessWidget {
  const _SettingsRow({required this.destination, required this.onTap});

  final _SettingsDestination destination;
  final ValueChanged<_SettingsDestination> onTap;

  @override
  Widget build(BuildContext context) {
    final label = destination.title(context.l10n);

    return Semantics(
      button: true,
      child: SizedBox(
        height: 52,
        child: InkWell(
          onTap: () => onTap(destination),
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 18),
            child: Row(
              children: [
                Icon(destination.icon, color: HankoColors.gold, size: 22),
                const SizedBox(width: 14),
                Expanded(child: Text(label, style: HankoTextStyles.label)),
                const Icon(
                  Icons.chevron_right,
                  color: HankoColors.gold,
                  size: 22,
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _SettingsDetailPage extends StatelessWidget {
  const _SettingsDetailPage({
    required this.destination,
    required this.onBack,
    required this.onLocaleSelected,
  });

  final _SettingsDestination destination;
  final VoidCallback onBack;
  final ValueChanged<Locale>? onLocaleSelected;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;

    return _SettingsPageFrame(
      title: destination.title(l10n),
      leading: IconButton(
        onPressed: onBack,
        tooltip: MaterialLocalizations.of(context).backButtonTooltip,
        icon: const Icon(Icons.arrow_back_ios_new),
        color: HankoColors.gold,
      ),
      children: [
        switch (destination) {
          _SettingsDestination.language => _LanguageSettingsContent(
            currentLocale: l10n.locale,
            onLocaleSelected: onLocaleSelected,
          ),
          _SettingsDestination.about ||
          _SettingsDestination.howItWorks ||
          _SettingsDestination.faq ||
          _SettingsDestination.privacy ||
          _SettingsDestination.terms ||
          _SettingsDestination.contact => _SettingsContentLoader(
            routeCode: fallbackRouteCodeForLocale(l10n.locale),
            destination: destination,
          ),
          _SettingsDestination.version => const _VersionSettingsContent(),
        },
      ],
    );
  }
}

class _SettingsContentLoader extends StatefulWidget {
  const _SettingsContentLoader({
    required this.routeCode,
    required this.destination,
  });

  final String routeCode;
  final _SettingsDestination destination;

  @override
  State<_SettingsContentLoader> createState() => _SettingsContentLoaderState();
}

class _SettingsContentLoaderState extends State<_SettingsContentLoader> {
  Object? _bundle;
  Future<SettingsContentBundle>? _content;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final bundle = DefaultAssetBundle.of(context);
    if (!identical(_bundle, bundle)) {
      _bundle = bundle;
      _content = SettingsContentBundle.forLanguage(
        widget.routeCode,
        bundle: bundle,
      );
    }
  }

  @override
  void didUpdateWidget(covariant _SettingsContentLoader oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.routeCode != widget.routeCode) {
      _content = SettingsContentBundle.forLanguage(
        widget.routeCode,
        bundle: DefaultAssetBundle.of(context),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    return FutureBuilder<SettingsContentBundle>(
      future: _content,
      builder: (context, snapshot) {
        if (snapshot.hasError) {
          return HankoStateView.error(
            title: l10n.commonGenericErrorTitle,
            message: l10n.commonGenericErrorMessage,
          );
        }

        final content = snapshot.data;
        if (content == null) {
          return HankoStateView.loading(
            title: widget.destination.title(l10n),
            message: l10n.splashPreparing,
          );
        }

        return switch (widget.destination) {
          _SettingsDestination.about => _AboutSettingsContent(
            content: content.about,
          ),
          _SettingsDestination.howItWorks => _HowItWorksSettingsContent(
            content: content.howItWorks,
          ),
          _SettingsDestination.faq => _FaqSettingsContent(content: content.faq),
          _SettingsDestination.privacy => _LegalSettingsContent(
            content: content.privacy,
            icon: Icons.privacy_tip_outlined,
          ),
          _SettingsDestination.terms => _LegalSettingsContent(
            content: content.terms,
            icon: Icons.description_outlined,
          ),
          _SettingsDestination.contact => _ContactSettingsContent(
            content: content.contact,
          ),
          _SettingsDestination.language ||
          _SettingsDestination.version => const SizedBox.shrink(),
        };
      },
    );
  }
}

class _LanguageSettingsContent extends StatelessWidget {
  const _LanguageSettingsContent({
    required this.currentLocale,
    required this.onLocaleSelected,
  });

  final Locale currentLocale;
  final ValueChanged<Locale>? onLocaleSelected;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _ContentIntroCard(
          icon: Icons.language,
          title: l10n.settingsLanguageTitle,
          body: l10n.settingsLanguageMessage,
        ),
        const SizedBox(height: HankoSpacing.md),
        _LanguageRegistryRows(
          currentLocale: currentLocale,
          onLocaleSelected: onLocaleSelected,
        ),
      ],
    );
  }
}

class _LanguageRegistryRows extends StatefulWidget {
  const _LanguageRegistryRows({
    required this.currentLocale,
    required this.onLocaleSelected,
  });

  final Locale currentLocale;
  final ValueChanged<Locale>? onLocaleSelected;

  @override
  State<_LanguageRegistryRows> createState() => _LanguageRegistryRowsState();
}

class _LanguageRegistryRowsState extends State<_LanguageRegistryRows> {
  Object? _bundle;
  Future<AppLanguageRegistry>? _registry;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final bundle = DefaultAssetBundle.of(context);
    if (!identical(_bundle, bundle)) {
      _bundle = bundle;
      _registry = AppLanguageRegistry.load(bundle: bundle);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    return FutureBuilder<AppLanguageRegistry>(
      future: _registry,
      builder: (context, snapshot) {
        if (snapshot.hasError) {
          return HankoStateView.error(
            title: l10n.commonGenericErrorTitle,
            message: l10n.commonGenericErrorMessage,
          );
        }

        final languages = snapshot.data?.selectableLanguages;
        if (languages == null) {
          return HankoStateView.loading(
            title: l10n.settingsLanguageTitle,
            message: l10n.splashPreparing,
          );
        }

        return HankoSurfaceCard(
          padding: const EdgeInsets.symmetric(vertical: HankoSpacing.xs),
          radius: HankoRadii.md,
          child: Column(
            children: [
              for (final language in languages)
                _LanguageOptionRow(
                  language: language,
                  isSelected: language.matchesLocale(widget.currentLocale),
                  onTap: widget.onLocaleSelected == null
                      ? null
                      : () => widget.onLocaleSelected!(language.locale),
                ),
            ],
          ),
        );
      },
    );
  }
}

class _LanguageOptionRow extends StatelessWidget {
  const _LanguageOptionRow({
    required this.language,
    required this.isSelected,
    required this.onTap,
  });

  final AppLanguageOption language;
  final bool isSelected;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      button: true,
      selected: isSelected,
      child: SizedBox(
        height: 56,
        child: InkWell(
          onTap: onTap,
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 18),
            child: Row(
              children: [
                const Icon(Icons.translate, color: HankoColors.gold, size: 20),
                const SizedBox(width: 14),
                Expanded(
                  child: Column(
                    mainAxisAlignment: MainAxisAlignment.center,
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(language.nativeName, style: HankoTextStyles.label),
                      if (language.englishNameLabel case final label?)
                        Text(label, style: HankoTextStyles.compactBody),
                    ],
                  ),
                ),
                Icon(
                  isSelected
                      ? Icons.radio_button_checked
                      : Icons.radio_button_unchecked,
                  color: isSelected ? HankoColors.gold : HankoColors.body,
                  size: 24,
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _VersionSettingsContent extends StatelessWidget {
  const _VersionSettingsContent();

  static const _displayVersion = String.fromEnvironment(
    'HANKO_APP_VERSION',
    defaultValue: '1.0.4+10',
  );

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;

    return HankoSurfaceCard(
      padding: const EdgeInsets.all(HankoSpacing.lg),
      radius: HankoRadii.md,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Icon(Icons.tag_outlined, color: HankoColors.gold, size: 32),
          const SizedBox(height: HankoSpacing.md),
          Text(l10n.settingsVersionTitle, style: HankoTextStyles.sectionTitle),
          const SizedBox(height: HankoSpacing.sm),
          Text(
            l10n.settingsVersionMessage(_displayVersion),
            style: HankoTextStyles.body,
          ),
        ],
      ),
    );
  }
}

class _AboutSettingsContent extends StatelessWidget {
  const _AboutSettingsContent({required this.content});

  final SettingsAboutContent content;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _ContentIntroCard(
          icon: Icons.auto_awesome,
          title: content.heading,
          body: content.body,
        ),
        for (final point in content.points) ...[
          const SizedBox(height: HankoSpacing.md),
          _SettingsTextCard(
            icon: Icons.diamond_outlined,
            title: point.title,
            body: point.body,
          ),
        ],
        const SizedBox(height: HankoSpacing.md),
        _TaglineCard(text: content.tagline),
      ],
    );
  }
}

class _FaqSettingsContent extends StatelessWidget {
  const _FaqSettingsContent({required this.content});

  final SettingsFaqContent content;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _ContentIntroCard(
          icon: Icons.help_outline,
          title: content.heading,
          body: context.l10n.settingsFaqIntro,
        ),
        for (final item in content.items) ...[
          const SizedBox(height: HankoSpacing.md),
          _FaqCard(item: item),
        ],
      ],
    );
  }
}

class _HowItWorksSettingsContent extends StatelessWidget {
  const _HowItWorksSettingsContent({required this.content});

  final SettingsHowItWorksContent content;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _ContentIntroCard(
          icon: Icons.route_outlined,
          title: content.heading,
          body: content.intro,
        ),
        for (var index = 0; index < content.steps.length; index++) ...[
          const SizedBox(height: HankoSpacing.md),
          _NumberedStepCard(
            number: index + 1,
            title: content.steps[index].title,
            body: content.steps[index].body,
          ),
        ],
        const SizedBox(height: HankoSpacing.md),
        _ContentIntroCard(
          icon: Icons.volunteer_activism_outlined,
          title: content.summaryTitle,
          body: content.summaryBody,
        ),
      ],
    );
  }
}

class _LegalSettingsContent extends StatelessWidget {
  const _LegalSettingsContent({required this.content, required this.icon});

  final SettingsLegalContent content;
  final IconData icon;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _ContentIntroCard(
          icon: icon,
          title: content.updated,
          body: content.intro,
        ),
        const SizedBox(height: HankoSpacing.md),
        _LegalLinkCard(
          label: content.officialLinkLabel,
          url: content.officialUrl,
        ),
        for (final section in content.sections) ...[
          const SizedBox(height: HankoSpacing.md),
          _SettingsTextCard(
            icon: Icons.article_outlined,
            title: section.title,
            body: section.body,
          ),
        ],
      ],
    );
  }
}

class _ContactSettingsContent extends StatelessWidget {
  const _ContactSettingsContent({required this.content});

  final SettingsContactContent content;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _ContentIntroCard(
          icon: Icons.support_agent_outlined,
          title: content.heading,
          body: content.intro,
        ),
        for (final option in content.options) ...[
          const SizedBox(height: HankoSpacing.md),
          _ContactOptionCard(option: option),
        ],
        const SizedBox(height: HankoSpacing.md),
        _TaglineCard(text: content.replyNote),
      ],
    );
  }
}

class _ContentIntroCard extends StatelessWidget {
  const _ContentIntroCard({
    required this.icon,
    required this.title,
    required this.body,
  });

  final IconData icon;
  final String title;
  final String body;

  @override
  Widget build(BuildContext context) {
    return HankoSurfaceCard(
      padding: const EdgeInsets.all(HankoSpacing.lg),
      radius: HankoRadii.md,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(icon, color: HankoColors.gold, size: 32),
          const SizedBox(height: HankoSpacing.md),
          Text(title, style: HankoTextStyles.sectionTitle),
          const SizedBox(height: HankoSpacing.sm),
          Text(body, style: HankoTextStyles.body),
        ],
      ),
    );
  }
}

class _NumberedStepCard extends StatelessWidget {
  const _NumberedStepCard({
    required this.number,
    required this.title,
    required this.body,
  });

  final int number;
  final String title;
  final String body;

  @override
  Widget build(BuildContext context) {
    return HankoSurfaceCard(
      padding: const EdgeInsets.all(18),
      radius: HankoRadii.md,
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _StepBadge(number: number),
          const SizedBox(width: HankoSpacing.md),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(title, style: HankoTextStyles.cardTitle),
                const SizedBox(height: HankoSpacing.sm),
                Text(body, style: HankoTextStyles.body),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _StepBadge extends StatelessWidget {
  const _StepBadge({required this.number});

  final int number;

  @override
  Widget build(BuildContext context) {
    return SizedBox.square(
      dimension: 32,
      child: DecoratedBox(
        decoration: BoxDecoration(
          color: HankoColors.medallion,
          borderRadius: BorderRadius.circular(16),
          border: Border.all(color: HankoColors.surfaceBorder),
        ),
        child: Center(
          child: Text(
            '$number',
            style: HankoTextStyles.label.copyWith(color: HankoColors.gold),
          ),
        ),
      ),
    );
  }
}

class _SettingsTextCard extends StatelessWidget {
  const _SettingsTextCard({
    required this.icon,
    required this.title,
    required this.body,
  });

  final IconData icon;
  final String title;
  final String body;

  @override
  Widget build(BuildContext context) {
    return HankoSurfaceCard(
      padding: const EdgeInsets.all(18),
      radius: HankoRadii.md,
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(icon, color: HankoColors.gold, size: 24),
          const SizedBox(width: HankoSpacing.md),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(title, style: HankoTextStyles.cardTitle),
                const SizedBox(height: HankoSpacing.sm),
                Text(body, style: HankoTextStyles.body),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _ContactOptionCard extends StatelessWidget {
  const _ContactOptionCard({required this.option});

  final SettingsContactOption option;

  @override
  Widget build(BuildContext context) {
    return HankoSurfaceCard(
      padding: const EdgeInsets.all(18),
      radius: HankoRadii.md,
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Icon(Icons.mail_outline, color: HankoColors.gold, size: 24),
          const SizedBox(width: HankoSpacing.md),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(option.title, style: HankoTextStyles.cardTitle),
                const SizedBox(height: HankoSpacing.sm),
                Text(option.body, style: HankoTextStyles.body),
                const SizedBox(height: HankoSpacing.sm),
                SelectableText(option.value, style: HankoTextStyles.label),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _FaqCard extends StatelessWidget {
  const _FaqCard({required this.item});

  final SettingsFaqItem item;

  @override
  Widget build(BuildContext context) {
    return HankoSurfaceCard(
      padding: const EdgeInsets.all(18),
      radius: HankoRadii.md,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Icon(
                Icons.question_answer_outlined,
                color: HankoColors.gold,
                size: 24,
              ),
              const SizedBox(width: HankoSpacing.md),
              Expanded(
                child: Text(item.question, style: HankoTextStyles.cardTitle),
              ),
            ],
          ),
          const SizedBox(height: HankoSpacing.sm),
          Text(item.answer, style: HankoTextStyles.body),
        ],
      ),
    );
  }
}

class _LegalLinkCard extends StatelessWidget {
  const _LegalLinkCard({required this.label, required this.url});

  final String label;
  final String url;

  @override
  Widget build(BuildContext context) {
    return HankoSurfaceCard(
      padding: const EdgeInsets.all(18),
      radius: HankoRadii.md,
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Icon(Icons.open_in_new, color: HankoColors.gold, size: 24),
          const SizedBox(width: HankoSpacing.md),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(label, style: HankoTextStyles.cardTitle),
                const SizedBox(height: HankoSpacing.sm),
                SelectableText(url, style: HankoTextStyles.compactBody),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _TaglineCard extends StatelessWidget {
  const _TaglineCard({required this.text});

  final String text;

  @override
  Widget build(BuildContext context) {
    return HankoSurfaceCard(
      padding: const EdgeInsets.all(18),
      radius: HankoRadii.md,
      child: Center(
        child: Text(
          text,
          style: HankoTextStyles.label.copyWith(color: HankoColors.gold),
          textAlign: TextAlign.center,
        ),
      ),
    );
  }
}

class _SettingsPageFrame extends StatelessWidget {
  const _SettingsPageFrame({
    required this.title,
    required this.children,
    this.leading,
    this.trailing,
  });

  final String title;
  final List<Widget> children;
  final Widget? leading;
  final Widget? trailing;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: HankoColors.background,
      child: SingleChildScrollView(
        physics: const BouncingScrollPhysics(),
        child: Padding(
          padding: const EdgeInsets.fromLTRB(26, 48, 26, HankoSpacing.xl),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Row(
                children: [
                  if (leading != null) ...[
                    SizedBox.square(dimension: 48, child: leading),
                    const SizedBox(width: HankoSpacing.sm),
                  ],
                  Expanded(
                    child: Text(title, style: HankoTextStyles.pageTitle),
                  ),
                  if (trailing != null) ...[
                    const SizedBox(width: HankoSpacing.sm),
                    SizedBox.square(dimension: 48, child: trailing),
                  ],
                ],
              ),
              if (children.isNotEmpty) const SizedBox(height: HankoSpacing.lg),
              ...children,
            ],
          ),
        ),
      ),
    );
  }
}

enum _SettingsDestination {
  language,
  about,
  howItWorks,
  faq,
  privacy,
  terms,
  contact,
  version;

  String get pageKey {
    return switch (this) {
      _SettingsDestination.language => 'COM-004-language',
      _SettingsDestination.about => 'COM-005-about',
      _SettingsDestination.howItWorks => 'COM-006-how-it-works',
      _SettingsDestination.faq => 'COM-007-faq',
      _SettingsDestination.privacy => 'COM-009-privacy',
      _SettingsDestination.terms => 'COM-010-terms',
      _SettingsDestination.contact => 'COM-008-contact',
      _SettingsDestination.version => 'COM-004-version',
    };
  }

  String get routeName {
    return switch (this) {
      _SettingsDestination.language => '/settings/language',
      _SettingsDestination.about => '/settings/about',
      _SettingsDestination.howItWorks => '/settings/how-it-works',
      _SettingsDestination.faq => '/settings/faq',
      _SettingsDestination.privacy => '/settings/privacy',
      _SettingsDestination.terms => '/settings/terms',
      _SettingsDestination.contact => '/settings/contact',
      _SettingsDestination.version => '/settings/version',
    };
  }

  IconData get icon {
    return switch (this) {
      _SettingsDestination.language => Icons.language,
      _SettingsDestination.about => Icons.info_outline,
      _SettingsDestination.howItWorks => Icons.route_outlined,
      _SettingsDestination.faq => Icons.help_outline,
      _SettingsDestination.privacy => Icons.privacy_tip_outlined,
      _SettingsDestination.terms => Icons.description_outlined,
      _SettingsDestination.contact => Icons.mail_outline,
      _SettingsDestination.version => Icons.tag_outlined,
    };
  }

  String title(GeneratedHankoLocalizations l10n) {
    return switch (this) {
      _SettingsDestination.language => l10n.language,
      _SettingsDestination.about => l10n.about,
      _SettingsDestination.howItWorks => l10n.howItWorks,
      _SettingsDestination.faq => l10n.faq,
      _SettingsDestination.privacy => l10n.privacy,
      _SettingsDestination.terms => l10n.terms,
      _SettingsDestination.contact => l10n.contact,
      _SettingsDestination.version => l10n.version,
    };
  }
}
