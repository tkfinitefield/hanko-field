import 'package:flutter/material.dart';

import '../../../app/localization/app_localization.dart';
import '../../../app/theme/app_theme.dart';
import '../../../core/domain/money.dart';
import '../../../core/widgets/core_widgets.dart';
import '../domain/stone_listing.dart';
import 'stone_selection_confirmation.dart';

class StonesHomeScreen extends StatefulWidget {
  const StonesHomeScreen({
    super.key,
    this.result,
    this.isLoading = false,
    this.loadError,
    this.onRetry,
    this.onOpenStoneDetail,
    this.selectedStoneId,
    this.onSelectStone,
  });

  final StoneListingsResult? result;
  final bool isLoading;
  final Object? loadError;
  final VoidCallback? onRetry;
  final ValueChanged<StoneListing>? onOpenStoneDetail;
  final String? selectedStoneId;
  final ValueChanged<StoneListing>? onSelectStone;

  @override
  State<StonesHomeScreen> createState() => _StonesHomeScreenState();
}

class _StonesHomeScreenState extends State<StonesHomeScreen> {
  var _filters = const _StoneFilters();
  var _sortOrder = _StoneSortOrder.recommended;

  @override
  void didUpdateWidget(covariant StonesHomeScreen oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.result != widget.result) {
      _filters = _filters.constrainTo(
        widget.result?.listings ?? const <StoneListing>[],
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final listings = widget.result?.listings ?? const <StoneListing>[];
    final filteredListings = _filters.apply(listings);
    final sortedListings = _sortOrder.apply(filteredListings);

    return HankoFeaturePage(
      title: l10n.stones,
      children: [
        Text(l10n.browseStonesDescription, style: HankoTextStyles.body),
        const SizedBox(height: HankoSpacing.sm),
        if (widget.isLoading)
          HankoStateView.loading(
            title: l10n.stonesLoadingTitle,
            message: l10n.stonesLoadingMessage,
          )
        else if (widget.loadError != null)
          HankoErrorStateView(
            error: widget.loadError,
            actionLabel: l10n.tryAgain,
            onAction: widget.onRetry,
          )
        else if (listings.isEmpty)
          HankoStateView.empty(
            title: l10n.noStonesLoaded,
            message: l10n.noStonesLoadedMessage,
          )
        else ...[
          _StoneSortControl(sortOrder: _sortOrder, onPressed: _showSortSheet),
          const SizedBox(height: HankoSpacing.md),
          _StoneFiltersPanel(
            listings: listings,
            filters: _filters,
            onChanged: (filters) => setState(() => _filters = filters),
          ),
          const SizedBox(height: HankoSpacing.md),
          if (sortedListings.isEmpty)
            HankoStateView.empty(
              title: l10n.noStonesMatchFilters,
              message: l10n.noStonesMatchFiltersMessage,
            )
          else
            for (var index = 0; index < sortedListings.length; index++) ...[
              _StoneListingCard(
                listing: sortedListings[index],
                onOpenStoneDetail: widget.onOpenStoneDetail,
                isSelectedForOrder:
                    sortedListings[index].isOrderable &&
                    widget.selectedStoneId == sortedListings[index].id,
                onSelectStone: widget.onSelectStone,
              ),
              if (index < sortedListings.length - 1)
                const SizedBox(height: HankoSpacing.md),
            ],
        ],
      ],
    );
  }

  Future<void> _showSortSheet() async {
    final selected = await showModalBottomSheet<_StoneSortOrder>(
      context: context,
      backgroundColor: HankoColors.surface,
      showDragHandle: true,
      builder: (context) => _StoneSortSheet(selected: _sortOrder),
    );
    if (!mounted || selected == null || selected == _sortOrder) {
      return;
    }
    setState(() => _sortOrder = selected);
  }
}

enum _StoneSortOrder {
  recommended,
  newest,
  priceLowToHigh,
  priceHighToLow;

  List<StoneListing> apply(List<StoneListing> listings) {
    if (this == _StoneSortOrder.recommended) {
      return List.of(listings, growable: false);
    }

    final sorted = [
      for (var index = 0; index < listings.length; index++)
        (index: index, listing: listings[index]),
    ];
    switch (this) {
      case _StoneSortOrder.recommended:
        break;
      case _StoneSortOrder.newest:
        sorted.sort((left, right) {
          final order = right.listing.sortOrder.compareTo(
            left.listing.sortOrder,
          );
          return order == 0 ? left.index.compareTo(right.index) : order;
        });
      case _StoneSortOrder.priceLowToHigh:
        sorted.sort((left, right) {
          final price = left.listing.price.amount.compareTo(
            right.listing.price.amount,
          );
          return price == 0 ? left.index.compareTo(right.index) : price;
        });
      case _StoneSortOrder.priceHighToLow:
        sorted.sort((left, right) {
          final price = right.listing.price.amount.compareTo(
            left.listing.price.amount,
          );
          return price == 0 ? left.index.compareTo(right.index) : price;
        });
    }
    return sorted.map((entry) => entry.listing).toList(growable: false);
  }
}

String _sortLabel(GeneratedHankoLocalizations l10n, _StoneSortOrder sortOrder) {
  return switch (sortOrder) {
    _StoneSortOrder.recommended => l10n.stoneSortRecommended,
    _StoneSortOrder.newest => l10n.stoneSortNewest,
    _StoneSortOrder.priceLowToHigh => l10n.stoneSortPriceLowToHigh,
    _StoneSortOrder.priceHighToLow => l10n.stoneSortPriceHighToLow,
  };
}

String _sortKey(_StoneSortOrder sortOrder) {
  return switch (sortOrder) {
    _StoneSortOrder.recommended => 'recommended',
    _StoneSortOrder.newest => 'newest',
    _StoneSortOrder.priceLowToHigh => 'price-low-to-high',
    _StoneSortOrder.priceHighToLow => 'price-high-to-low',
  };
}

class _StoneSortControl extends StatelessWidget {
  const _StoneSortControl({required this.sortOrder, required this.onPressed});

  final _StoneSortOrder sortOrder;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    return HankoSurfaceCard(
      radius: HankoRadii.sm,
      padding: const EdgeInsets.fromLTRB(14, 12, 14, 12),
      child: Row(
        children: [
          const Icon(Icons.sort, color: HankoColors.gold, size: 19),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              _sortLabel(l10n, sortOrder),
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: HankoTextStyles.label,
            ),
          ),
          TextButton.icon(
            key: const Key('stone-sort-open'),
            onPressed: onPressed,
            icon: const Icon(Icons.swap_vert, size: 18),
            label: Text(l10n.stoneSortAction),
          ),
        ],
      ),
    );
  }
}

class _StoneSortSheet extends StatelessWidget {
  const _StoneSortSheet({required this.selected});

  final _StoneSortOrder selected;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final options = _StoneSortOrder.values;
    return SafeArea(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(24, 4, 24, 24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(
              l10n.stoneSortTitle,
              textAlign: TextAlign.center,
              style: HankoTextStyles.sectionTitle,
            ),
            const SizedBox(height: HankoSpacing.md),
            for (final option in options)
              _StoneSortOptionTile(
                key: Key('stone-sort-${_sortKey(option)}'),
                label: _sortLabel(l10n, option),
                selected: option == selected,
                onTap: () => Navigator.of(context).pop(option),
              ),
          ],
        ),
      ),
    );
  }
}

class _StoneSortOptionTile extends StatelessWidget {
  const _StoneSortOptionTile({
    super.key,
    required this.label,
    required this.selected,
    required this.onTap,
  });

  final String label;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return InkWell(
      borderRadius: BorderRadius.circular(HankoRadii.sm),
      onTap: onTap,
      child: Container(
        constraints: const BoxConstraints(minHeight: 54),
        decoration: const BoxDecoration(
          border: Border(
            bottom: BorderSide(color: HankoColors.surfaceBorder, width: 0.7),
          ),
        ),
        child: Row(
          children: [
            Expanded(
              child: Text(
                label,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: HankoTextStyles.label,
              ),
            ),
            Icon(
              selected ? Icons.radio_button_checked : Icons.radio_button_off,
              color: selected ? HankoColors.gold : HankoColors.body,
              size: 22,
            ),
          ],
        ),
      ),
    );
  }
}

class _StoneFilters {
  const _StoneFilters({
    this.materialKey,
    this.colorFamily,
    this.patternPrimary,
    this.availability,
  });

  final String? materialKey;
  final String? colorFamily;
  final String? patternPrimary;
  final String? availability;

  bool get hasActive =>
      materialKey != null ||
      colorFamily != null ||
      patternPrimary != null ||
      availability != null;

  _StoneFilters withMaterialKey(String? value) {
    return _StoneFilters(
      materialKey: _normalizeFilterValue(value),
      colorFamily: colorFamily,
      patternPrimary: patternPrimary,
      availability: availability,
    );
  }

  _StoneFilters withColorFamily(String? value) {
    return _StoneFilters(
      materialKey: materialKey,
      colorFamily: _normalizeFilterValue(value),
      patternPrimary: patternPrimary,
      availability: availability,
    );
  }

  _StoneFilters withPatternPrimary(String? value) {
    return _StoneFilters(
      materialKey: materialKey,
      colorFamily: colorFamily,
      patternPrimary: _normalizeFilterValue(value),
      availability: availability,
    );
  }

  _StoneFilters withAvailability(String? value) {
    return _StoneFilters(
      materialKey: materialKey,
      colorFamily: colorFamily,
      patternPrimary: patternPrimary,
      availability: _normalizeFilterValue(value),
    );
  }

  _StoneFilters constrainTo(List<StoneListing> listings) {
    final materialKeys = listings.map((listing) => listing.materialKey).toSet();
    final colorFamilies = listings
        .map((listing) => listing.facets.colorFamily)
        .toSet();
    final patternPrimaries = listings
        .map((listing) => listing.facets.patternPrimary)
        .toSet();

    return _StoneFilters(
      materialKey: materialKeys.contains(materialKey) ? materialKey : null,
      colorFamily: colorFamilies.contains(colorFamily) ? colorFamily : null,
      patternPrimary: patternPrimaries.contains(patternPrimary)
          ? patternPrimary
          : null,
      availability: availability,
    );
  }

  List<StoneListing> apply(List<StoneListing> listings) {
    return listings
        .where((listing) {
          if (materialKey != null && listing.materialKey != materialKey) {
            return false;
          }
          if (colorFamily != null &&
              listing.facets.colorFamily != colorFamily) {
            return false;
          }
          if (patternPrimary != null &&
              listing.facets.patternPrimary != patternPrimary) {
            return false;
          }
          if (availability == _availableFilterValue && !listing.isOrderable) {
            return false;
          }
          if (availability == _unavailableFilterValue && listing.isOrderable) {
            return false;
          }
          return true;
        })
        .toList(growable: false);
  }
}

class _StoneFiltersPanel extends StatelessWidget {
  const _StoneFiltersPanel({
    required this.listings,
    required this.filters,
    required this.onChanged,
  });

  final List<StoneListing> listings;
  final _StoneFilters filters;
  final ValueChanged<_StoneFilters> onChanged;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    return HankoSurfaceCard(
      radius: HankoRadii.sm,
      padding: const EdgeInsets.fromLTRB(14, 14, 14, 16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Row(
            children: [
              const Icon(Icons.tune, color: HankoColors.gold, size: 19),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  l10n.stoneFiltersTitle,
                  style: HankoTextStyles.label,
                ),
              ),
              if (filters.hasActive)
                TextButton.icon(
                  key: const Key('stone-filters-reset'),
                  onPressed: () => onChanged(const _StoneFilters()),
                  icon: const Icon(Icons.close, size: 16),
                  label: Text(l10n.stoneFilterReset),
                ),
            ],
          ),
          const SizedBox(height: HankoSpacing.xs),
          _StoneFilterGroup(
            keyPrefix: 'material',
            label: l10n.stoneFilterMaterial,
            allLabel: l10n.stoneFilterAll,
            selectedValue: filters.materialKey,
            options: _materialOptions(listings),
            onChanged: (value) => onChanged(filters.withMaterialKey(value)),
          ),
          const SizedBox(height: HankoSpacing.sm),
          _StoneFilterGroup(
            keyPrefix: 'color',
            label: l10n.stoneFilterColor,
            allLabel: l10n.stoneFilterAll,
            selectedValue: filters.colorFamily,
            options: _tokenOptions(
              listings.map((listing) => listing.facets.colorFamily),
            ),
            onChanged: (value) => onChanged(filters.withColorFamily(value)),
          ),
          const SizedBox(height: HankoSpacing.sm),
          _StoneFilterGroup(
            keyPrefix: 'pattern',
            label: l10n.stoneFilterPattern,
            allLabel: l10n.stoneFilterAll,
            selectedValue: filters.patternPrimary,
            options: _tokenOptions(
              listings.map((listing) => listing.facets.patternPrimary),
            ),
            onChanged: (value) => onChanged(filters.withPatternPrimary(value)),
          ),
          const SizedBox(height: HankoSpacing.sm),
          _StoneFilterGroup(
            keyPrefix: 'availability',
            label: l10n.stoneFilterAvailability,
            allLabel: l10n.stoneFilterAll,
            selectedValue: filters.availability,
            options: [
              _StoneFilterOption(
                value: _availableFilterValue,
                label: l10n.stoneAvailable,
              ),
              _StoneFilterOption(
                value: _unavailableFilterValue,
                label: l10n.stoneUnavailable,
              ),
            ],
            onChanged: (value) => onChanged(filters.withAvailability(value)),
          ),
        ],
      ),
    );
  }

  List<_StoneFilterOption> _materialOptions(List<StoneListing> listings) {
    final labelsByKey = <String, String>{};
    for (final listing in listings) {
      final key = listing.materialKey.trim();
      if (key.isEmpty || labelsByKey.containsKey(key)) {
        continue;
      }
      labelsByKey[key] = _materialLabel(listing);
    }
    final options = labelsByKey.entries
        .map(
          (entry) => _StoneFilterOption(value: entry.key, label: entry.value),
        )
        .toList(growable: false);
    options.sort((left, right) => left.label.compareTo(right.label));
    return options;
  }

  List<_StoneFilterOption> _tokenOptions(Iterable<String> tokens) {
    final seen = <String>{};
    final options = <_StoneFilterOption>[];
    for (final token in tokens) {
      final value = token.trim();
      if (value.isEmpty || !seen.add(value)) {
        continue;
      }
      options.add(
        _StoneFilterOption(value: value, label: _labelFromToken(value)),
      );
    }
    options.sort((left, right) => left.label.compareTo(right.label));
    return options;
  }
}

class _StoneFilterGroup extends StatelessWidget {
  const _StoneFilterGroup({
    required this.keyPrefix,
    required this.label,
    required this.allLabel,
    required this.selectedValue,
    required this.options,
    required this.onChanged,
  });

  final String keyPrefix;
  final String label;
  final String allLabel;
  final String? selectedValue;
  final List<_StoneFilterOption> options;
  final ValueChanged<String?> onChanged;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(label, style: HankoTextStyles.compactBody),
        const SizedBox(height: 6),
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: [
            _StoneChoiceChip(
              key: Key('stone-filter-$keyPrefix-all'),
              label: allLabel,
              selected: selectedValue == null,
              onSelected: () => onChanged(null),
            ),
            for (final option in options)
              _StoneChoiceChip(
                key: Key(
                  'stone-filter-$keyPrefix-${_filterKeyToken(option.value)}',
                ),
                label: option.label,
                selected: selectedValue == option.value,
                onSelected: () => onChanged(option.value),
              ),
          ],
        ),
      ],
    );
  }
}

class _StoneChoiceChip extends StatelessWidget {
  const _StoneChoiceChip({
    super.key,
    required this.label,
    required this.selected,
    required this.onSelected,
  });

  final String label;
  final bool selected;
  final VoidCallback onSelected;

  @override
  Widget build(BuildContext context) {
    return ChoiceChip(
      label: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 220),
        child: Text(label, maxLines: 1, overflow: TextOverflow.ellipsis),
      ),
      selected: selected,
      showCheckmark: false,
      avatar: selected
          ? const Icon(Icons.check, size: 16, color: HankoColors.red)
          : null,
      selectedColor: HankoColors.medallion,
      backgroundColor: HankoColors.surface,
      side: BorderSide(
        color: selected ? HankoColors.gold : HankoColors.surfaceBorder,
      ),
      labelStyle: HankoTextStyles.compactBody.copyWith(
        color: selected ? HankoColors.ink : HankoColors.body,
        fontWeight: selected ? FontWeight.w700 : FontWeight.w500,
      ),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(HankoRadii.sm),
      ),
      onSelected: (_) => onSelected(),
    );
  }
}

class _StoneFilterOption {
  const _StoneFilterOption({required this.value, required this.label});

  final String value;
  final String label;
}

const _availableFilterValue = 'available';
const _unavailableFilterValue = 'unavailable';

String? _normalizeFilterValue(String? value) {
  final trimmed = value?.trim();
  if (trimmed == null || trimmed.isEmpty) {
    return null;
  }
  return trimmed;
}

String _materialLabel(StoneListing listing) {
  final label = listing.materialLabel.trim();
  if (label.isNotEmpty) {
    return label;
  }
  return _labelFromToken(listing.materialKey);
}

String _filterKeyToken(String value) {
  return value.toLowerCase().replaceAll(RegExp(r'[^a-z0-9_-]+'), '-');
}

class _StoneListingCard extends StatelessWidget {
  const _StoneListingCard({
    required this.listing,
    required this.onOpenStoneDetail,
    required this.isSelectedForOrder,
    required this.onSelectStone,
  });

  final StoneListing listing;
  final ValueChanged<StoneListing>? onOpenStoneDetail;
  final bool isSelectedForOrder;
  final ValueChanged<StoneListing>? onSelectStone;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;

    return HankoSurfaceCard(
      padding: const EdgeInsets.fromLTRB(18, 18, 18, 18),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          _StoneListingImage(listing: listing),
          const SizedBox(height: HankoSpacing.md),
          Text(
            _materialLabel(listing),
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: HankoTextStyles.compactBody,
          ),
          const SizedBox(height: HankoSpacing.xs),
          Text(
            listing.title,
            maxLines: 2,
            overflow: TextOverflow.ellipsis,
            style: HankoTextStyles.cardTitle,
          ),
          const SizedBox(height: HankoSpacing.sm),
          Text(_formatMoney(listing.price), style: HankoTextStyles.label),
          const SizedBox(height: HankoSpacing.md),
          Wrap(
            spacing: 10,
            runSpacing: 8,
            children: [
              _StoneAttributeChip(
                icon: Icons.palette_outlined,
                label: _labelFromToken(listing.facets.colorFamily),
              ),
              _StoneAttributeChip(
                icon: Icons.grain,
                label: _labelFromToken(listing.facets.patternPrimary),
              ),
              _StoneAttributeChip(
                icon: Icons.straighten,
                label: listing.sizeLabel,
              ),
              _StoneAvailabilityChip(listing: listing),
            ],
          ),
          const SizedBox(height: HankoSpacing.md),
          OutlinedButton.icon(
            onPressed: onOpenStoneDetail == null
                ? null
                : () => onOpenStoneDetail?.call(listing),
            style: OutlinedButton.styleFrom(
              foregroundColor: HankoColors.ink,
              minimumSize: const Size.fromHeight(48),
              side: const BorderSide(color: HankoColors.surfaceBorder),
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(HankoRadii.sm),
              ),
            ),
            icon: const Icon(Icons.info_outline, size: 20),
            label: Text(
              l10n.viewStoneDetails,
              style: HankoTextStyles.label.copyWith(color: HankoColors.ink),
            ),
          ),
          const SizedBox(height: HankoSpacing.sm),
          HankoPrimaryButton(
            label: isSelectedForOrder
                ? l10n.stoneSelectedForOrderAction
                : l10n.selectStone,
            icon: isSelectedForOrder ? Icons.check : Icons.arrow_forward,
            onPressed:
                listing.isOrderable &&
                    onSelectStone != null &&
                    !isSelectedForOrder
                ? () async {
                    final confirmed = await confirmStoneSelection(
                      context,
                      listing,
                    );
                    if (confirmed) {
                      onSelectStone?.call(listing);
                    }
                  }
                : null,
          ),
        ],
      ),
    );
  }
}

class _StoneListingImage extends StatelessWidget {
  const _StoneListingImage({required this.listing});

  final StoneListing listing;

  @override
  Widget build(BuildContext context) {
    final primaryPhoto = _primaryPhoto(listing.photos);
    final fallback = _StoneImageFallback(title: listing.title);

    return AspectRatio(
      aspectRatio: 16 / 10,
      child: ClipRRect(
        borderRadius: BorderRadius.circular(HankoRadii.sm),
        child: primaryPhoto == null || primaryPhoto.assetUrl.trim().isEmpty
            ? fallback
            : Image.network(
                primaryPhoto.assetUrl,
                fit: BoxFit.cover,
                semanticLabel: primaryPhoto.alt,
                errorBuilder: (context, error, stackTrace) => fallback,
                loadingBuilder: (context, child, loadingProgress) {
                  if (loadingProgress == null) {
                    return child;
                  }
                  return fallback;
                },
              ),
      ),
    );
  }

  StoneListingPhoto? _primaryPhoto(List<StoneListingPhoto> photos) {
    if (photos.isEmpty) {
      return null;
    }
    for (final photo in photos) {
      if (photo.isPrimary) {
        return photo;
      }
    }
    final sorted = [...photos]
      ..sort((left, right) => left.sortOrder.compareTo(right.sortOrder));
    return sorted.first;
  }
}

class _StoneImageFallback extends StatelessWidget {
  const _StoneImageFallback({required this.title});

  final String title;

  @override
  Widget build(BuildContext context) {
    return DecoratedBox(
      decoration: const BoxDecoration(color: HankoColors.medallion),
      child: Center(
        child: Icon(
          Icons.diamond_outlined,
          color: HankoColors.gold,
          size: 54,
          semanticLabel: title,
        ),
      ),
    );
  }
}

class _StoneAvailabilityChip extends StatelessWidget {
  const _StoneAvailabilityChip({required this.listing});

  final StoneListing listing;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final isAvailable = listing.isOrderable;
    return _StoneAttributeChip(
      icon: isAvailable ? Icons.check_circle_outline : Icons.block,
      label: isAvailable ? l10n.stoneAvailable : l10n.stoneUnavailable,
      color: isAvailable ? HankoColors.gold : HankoColors.error,
    );
  }
}

class _StoneAttributeChip extends StatelessWidget {
  const _StoneAttributeChip({
    required this.icon,
    required this.label,
    this.color = HankoColors.body,
  });

  final IconData icon;
  final String label;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(icon, color: color, size: 17),
        const SizedBox(width: 6),
        Text(label, style: HankoTextStyles.compactBody),
      ],
    );
  }
}

String _formatMoney(Money money) {
  final display = money.display?.trim();
  if (display != null && display.isNotEmpty) {
    return display;
  }
  final formattedAmount = _formatWholeNumber(money.amount);
  return switch (money.currency.toUpperCase()) {
    'JPY' => '¥$formattedAmount',
    'USD' => '\$$formattedAmount',
    _ => '${money.currency.toUpperCase()} $formattedAmount',
  };
}

String _formatWholeNumber(int value) {
  final sign = value < 0 ? '-' : '';
  final digits = value.abs().toString();
  final buffer = StringBuffer(sign);
  for (var index = 0; index < digits.length; index++) {
    if (index > 0 && (digits.length - index) % 3 == 0) {
      buffer.write(',');
    }
    buffer.write(digits[index]);
  }
  return buffer.toString();
}

String _labelFromToken(String token) {
  final words = token
      .split(RegExp(r'[_\-\s]+'))
      .where((word) => word.isNotEmpty)
      .toList(growable: false);
  if (words.isEmpty) {
    return token;
  }
  return words
      .map((word) {
        if (word.length == 1) {
          return word.toUpperCase();
        }
        return '${word[0].toUpperCase()}${word.substring(1).toLowerCase()}';
      })
      .join(' ');
}
