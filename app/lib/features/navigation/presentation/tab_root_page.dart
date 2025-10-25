import 'package:app/core/routing/app_route_configuration.dart';
import 'package:app/core/routing/app_state_notifier.dart';
import 'package:app/core/routing/app_tab.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// 各タブのルート画面
class AppTabRootPage extends ConsumerWidget {
  const AppTabRootPage({required this.tab, super.key});

  final AppTab tab;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Scaffold(
      appBar: AppBar(
        title: Text(tab.label),
        actions: [
          IconButton(
            icon: const Icon(Icons.notifications_none),
            onPressed: () {},
            tooltip: 'お知らせ',
          ),
          IconButton(
            icon: const Icon(Icons.search),
            onPressed: () {},
            tooltip: '検索',
          ),
          IconButton(
            icon: const Icon(Icons.help_outline),
            onPressed: () {},
            tooltip: 'ヘルプ',
          ),
        ],
      ),
      body: _TabBody(tab: tab, ref: ref),
    );
  }
}

class _TabBody extends StatelessWidget {
  const _TabBody({required this.tab, required this.ref});

  final AppTab tab;
  final WidgetRef ref;

  @override
  Widget build(BuildContext context) {
    switch (tab) {
      case AppTab.creation:
        final stages = [
          ['new'],
          ['input'],
          ['style'],
          ['editor'],
          ['preview'],
        ];
        return _buildList(
          context,
          title: tab.headline,
          subtitle: 'ディープリンクから該当ステップに遷移できます',
          children: [
            for (final stage in stages)
              ListTile(
                leading: const Icon(Icons.tune),
                title: Text('ステップ / ${stage.join(' / ')}'),
                onTap: () => _push(ref, CreationStageRoute(stage)),
              ),
          ],
        );
      case AppTab.shop:
        return _buildList(
          context,
          title: tab.headline,
          subtitle: '素材・商品詳細を Tab スタックで保持',
          children: [
            ListTile(
              leading: const Icon(Icons.category_outlined),
              title: const Text('素材 #onyx'),
              onTap: () => _push(
                ref,
                ShopDetailRoute(entity: 'materials', identifier: 'onyx'),
              ),
            ),
            ListTile(
              leading: const Icon(Icons.inventory_outlined),
              title: const Text('商品 #seal-001 → addons'),
              onTap: () => _push(
                ref,
                ShopDetailRoute(
                  entity: 'products',
                  identifier: 'seal-001',
                  trailingSegments: const ['addons'],
                ),
              ),
            ),
          ],
        );
      case AppTab.orders:
        final orders = ['HF-202401', 'HF-202402', 'HF-202403'];
        return _buildList(
          context,
          title: tab.headline,
          subtitle: '同じタブ内で注文詳細をスタック',
          children: [
            for (final orderId in orders)
              ListTile(
                leading: const Icon(Icons.receipt_long_outlined),
                title: Text('注文 $orderId'),
                onTap: () => _push(ref, OrderDetailsRoute(orderId: orderId)),
                trailing: IconButton(
                  icon: const Icon(Icons.timeline_outlined),
                  onPressed: () => _push(
                    ref,
                    OrderDetailsRoute(
                      orderId: orderId,
                      trailing: const ['production'],
                    ),
                  ),
                ),
              ),
          ],
        );
      case AppTab.library:
        final designs = ['JP-INK-01', 'JP-INK-02'];
        return _buildList(
          context,
          title: tab.headline,
          subtitle: '保存済み印鑑の詳細',
          children: [
            for (final designId in designs)
              ListTile(
                leading: const Icon(Icons.collections_bookmark_outlined),
                title: Text('デザイン $designId'),
                onTap: () => _push(ref, LibraryEntryRoute(designId: designId)),
                trailing: IconButton(
                  icon: const Icon(Icons.logout_outlined),
                  tooltip: 'エクスポート',
                  onPressed: () => _push(
                    ref,
                    LibraryEntryRoute(
                      designId: designId,
                      trailing: const ['export'],
                    ),
                  ),
                ),
              ),
          ],
        );
      case AppTab.profile:
        final sections = [
          ['addresses'],
          ['payments'],
          ['notifications'],
          ['support'],
        ];
        return _buildList(
          context,
          title: tab.headline,
          subtitle: '設定ページも Deep Link へ対応',
          children: [
            for (final section in sections)
              ListTile(
                leading: const Icon(Icons.settings_outlined),
                title: Text(section.join(' / ')),
                onTap: () => _push(ref, ProfileSectionRoute(section)),
              ),
          ],
        );
    }
  }

  Widget _buildList(
    BuildContext context, {
    required String title,
    required String subtitle,
    required List<Widget> children,
  }) {
    final theme = Theme.of(context);
    return ListView(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      children: [
        ListTile(
          title: Text(title, style: theme.textTheme.titleLarge),
          subtitle: Text(subtitle),
        ),
        ...children,
      ],
    );
  }

  void _push(WidgetRef ref, IndependentRoute route) {
    ref.read(appStateProvider.notifier).push(route);
  }
}
