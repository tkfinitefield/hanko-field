import 'package:flutter/material.dart';

/// ルートタブ
enum AppTab { creation, shop, orders, library, profile }

/// デフォルトタブ（アプリ起動/ディープリンク不一致時）
const AppTab kDefaultAppTab = AppTab.creation;

extension AppTabX on AppTab {
  /// ボトムバー表示名（日本語）
  String get label => switch (this) {
    AppTab.creation => '作成',
    AppTab.shop => 'ショップ',
    AppTab.orders => '注文',
    AppTab.library => 'マイ印鑑',
    AppTab.profile => 'プロフィール',
  };

  /// シェル内で使うアイコン
  IconData get icon => switch (this) {
    AppTab.creation => Icons.edit_outlined,
    AppTab.shop => Icons.store_mall_directory_outlined,
    AppTab.orders => Icons.receipt_long_outlined,
    AppTab.library => Icons.inventory_2_outlined,
    AppTab.profile => Icons.person_outline,
  };

  /// 深いリンクのパスセグメント
  String get pathSegment => switch (this) {
    AppTab.creation => 'design',
    AppTab.shop => 'shop',
    AppTab.orders => 'orders',
    AppTab.library => 'library',
    AppTab.profile => 'profile',
  };

  /// サマリ・見出し用タイトル
  String get headline => switch (this) {
    AppTab.creation => '印影を作成',
    AppTab.shop => '素材と商品を探す',
    AppTab.orders => '制作・注文を追跡',
    AppTab.library => '保存した印影',
    AppTab.profile => 'プロフィールと設定',
  };
}
