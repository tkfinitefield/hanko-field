import 'package:flutter_test/flutter_test.dart';
import 'package:hankofield/features/settings/presentation/settings_content.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('loads English settings content from JSON assets', () async {
    final content = await SettingsContentBundle.forLanguage('en');

    expect(content.about.heading, 'Your seal, made from gemstone');
    expect(content.howItWorks.steps, hasLength(4));
    expect(content.faq.items, hasLength(6));
    expect(content.privacy.officialUrl, 'https://finitefield.org/en/privacy/');
    expect(content.contact.options[1].value, 'dev@finitefield.org');
  });

  test('loads Japanese settings content from JSON assets', () async {
    final content = await SettingsContentBundle.forLanguage('ja');

    expect(content.about.heading, '宝石でつくる、あなたの印鑑');
    expect(content.howItWorks.steps, hasLength(4));
    expect(content.faq.items, hasLength(6));
    expect(content.terms.officialUrl, 'https://finitefield.org/ja/terms');
    expect(content.contact.options[1].value, 'dev@finitefield.org');
  });

  test(
    'falls back to English settings content for unsupported app locale',
    () async {
      final content = await SettingsContentBundle.forLanguage('fr');

      expect(content.about.heading, 'Your seal, made from gemstone');
    },
  );

  test('parses settings content from explicit JSON shape', () {
    final content = SettingsContentBundle.fromJson({
      'about': {
        'heading': 'About heading',
        'body': 'About body',
        'points': [
          {'title': 'Point title', 'body': 'Point body'},
        ],
        'tagline': 'Tagline',
      },
      'howItWorks': {
        'heading': 'How heading',
        'intro': 'How intro',
        'steps': [
          {'title': 'Step title', 'body': 'Step body'},
        ],
        'summaryTitle': 'Summary title',
        'summaryBody': 'Summary body',
      },
      'faq': {
        'heading': 'FAQ heading',
        'items': [
          {'question': 'Question', 'answer': 'Answer'},
        ],
      },
      'privacy': {
        'updated': 'Updated',
        'intro': 'Privacy intro',
        'officialLinkLabel': 'Privacy link',
        'officialUrl': 'https://example.com/privacy',
        'sections': [
          {'title': 'Privacy title', 'body': 'Privacy body'},
        ],
      },
      'terms': {
        'updated': 'Updated',
        'intro': 'Terms intro',
        'officialLinkLabel': 'Terms link',
        'officialUrl': 'https://example.com/terms',
        'sections': [
          {'title': 'Terms title', 'body': 'Terms body'},
        ],
      },
      'contact': {
        'heading': 'Contact heading',
        'intro': 'Contact intro',
        'options': [
          {'title': 'Contact title', 'body': 'Contact body', 'value': 'email'},
        ],
        'replyNote': 'Reply note',
      },
    });

    expect(content.about.points.single.title, 'Point title');
    expect(content.howItWorks.steps.single.body, 'Step body');
    expect(content.faq.items.single.answer, 'Answer');
    expect(content.privacy.sections.single.title, 'Privacy title');
    expect(content.terms.officialLinkLabel, 'Terms link');
    expect(content.contact.options.single.value, 'email');
  });
}
