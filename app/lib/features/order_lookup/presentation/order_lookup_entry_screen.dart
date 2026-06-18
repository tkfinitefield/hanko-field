import 'package:flutter/material.dart';

import '../../../app/localization/app_localization.dart';
import '../../../app/theme/app_theme.dart';
import '../../../core/api/core_api.dart';
import '../../../core/domain/money.dart';
import '../../../core/widgets/core_widgets.dart';
import '../domain/order_lookup_models.dart';

enum _OrderLookupStep { input, loading, notFound, error }

class OrderLookupEntryScreen extends StatefulWidget {
  const OrderLookupEntryScreen({
    super.key,
    this.initialOrderNo,
    this.initialEmail,
    this.onLookup,
    this.lookupOrder,
    this.onLookupResult,
    this.onBack,
  });

  final String? initialOrderNo;
  final String? initialEmail;
  final ValueChanged<OrderLookupRequest>? onLookup;
  final OrderLookupFetcher? lookupOrder;
  final ValueChanged<OrderStatus>? onLookupResult;
  final VoidCallback? onBack;

  @override
  State<OrderLookupEntryScreen> createState() => _OrderLookupEntryScreenState();
}

class _OrderLookupEntryScreenState extends State<OrderLookupEntryScreen> {
  late final TextEditingController _orderNoController;
  late final TextEditingController _emailController;
  var _step = _OrderLookupStep.input;
  var _lookupRequestId = 0;
  OrderLookupRequest? _lastRequest;
  OrderStatus? _lookupResult;
  String? _lookupEmail;
  Object? _lookupError;

  @override
  void initState() {
    super.initState();
    _orderNoController = TextEditingController(
      text: widget.initialOrderNo?.trim() ?? '',
    );
    _emailController = TextEditingController(
      text: widget.initialEmail?.trim() ?? '',
    );
    _orderNoController.addListener(_handleInputChanged);
    _emailController.addListener(_handleInputChanged);
  }

  @override
  void didUpdateWidget(covariant OrderLookupEntryScreen oldWidget) {
    super.didUpdateWidget(oldWidget);
    _orderNoController.removeListener(_handleInputChanged);
    _emailController.removeListener(_handleInputChanged);
    var didChangeInitialInput = false;
    if (widget.initialOrderNo != oldWidget.initialOrderNo &&
        widget.initialOrderNo != null) {
      _orderNoController.text = widget.initialOrderNo!.trim();
      didChangeInitialInput = true;
    }
    if (widget.initialEmail != oldWidget.initialEmail &&
        widget.initialEmail != null) {
      _emailController.text = widget.initialEmail!.trim();
      didChangeInitialInput = true;
    }
    if (didChangeInitialInput) {
      _lookupRequestId++;
      _lookupResult = null;
      _lookupEmail = null;
      _lookupError = null;
      _step = _OrderLookupStep.input;
    }
    _orderNoController.addListener(_handleInputChanged);
    _emailController.addListener(_handleInputChanged);
  }

  @override
  void dispose() {
    _orderNoController.removeListener(_handleInputChanged);
    _emailController.removeListener(_handleInputChanged);
    _orderNoController.dispose();
    _emailController.dispose();
    super.dispose();
  }

  bool get _canLookup {
    return _step != _OrderLookupStep.loading &&
        (widget.onLookup != null || widget.lookupOrder != null) &&
        _orderNoController.text.trim().isNotEmpty &&
        _emailController.text.trim().isNotEmpty;
  }

  bool get _isLoading => _step == _OrderLookupStep.loading;

  void _handleInputChanged() {
    if (_lookupResult != null) {
      _lookupResult = null;
      _lookupEmail = null;
    }
    if (_lookupError != null) {
      _lookupError = null;
    }
    if (_step != _OrderLookupStep.input) {
      _step = _OrderLookupStep.input;
    }
    setState(() {});
  }

  Future<void> _submitLookup() async {
    if (!_canLookup) {
      return;
    }
    FocusScope.of(context).unfocus();
    final request = OrderLookupRequest(
      orderNo: _orderNoController.text.trim(),
      email: _emailController.text.trim(),
    );
    _lastRequest = request;
    widget.onLookup?.call(request);

    final lookupOrder = widget.lookupOrder;
    if (lookupOrder == null) {
      return;
    }

    final requestId = ++_lookupRequestId;
    setState(() {
      _lookupResult = null;
      _lookupEmail = null;
      _lookupError = null;
      _step = _OrderLookupStep.loading;
    });
    try {
      final result = await lookupOrder(request);
      if (!mounted || requestId != _lookupRequestId) {
        return;
      }
      widget.onLookupResult?.call(result);
      if (!mounted || requestId != _lookupRequestId) {
        return;
      }
      setState(() {
        _lookupResult = result;
        _lookupEmail = request.email;
        _lookupError = null;
        _step = _OrderLookupStep.input;
      });
    } on HankoApiException catch (error) {
      if (!mounted || requestId != _lookupRequestId) {
        return;
      }
      setState(() {
        _lookupResult = null;
        _lookupEmail = null;
        if (_isLookupNotFoundError(error)) {
          _lookupError = null;
          _step = _OrderLookupStep.notFound;
        } else {
          _lookupError = error;
          _step = _OrderLookupStep.error;
        }
      });
    } catch (error) {
      if (!mounted || requestId != _lookupRequestId) {
        return;
      }
      setState(() {
        _lookupResult = null;
        _lookupEmail = null;
        _lookupError = error;
        _step = _OrderLookupStep.error;
      });
    }
  }

  void _retryLookup() {
    final request = _lastRequest;
    if (request == null || _isLoading) {
      return;
    }
    _orderNoController.text = request.orderNo;
    _emailController.text = request.email;
    _submitLookup();
  }

  void _lookupAnotherOrder() {
    _lookupRequestId++;
    setState(() {
      _lookupResult = null;
      _lookupEmail = null;
      _lookupError = null;
      _step = _OrderLookupStep.input;
    });
  }

  @override
  Widget build(BuildContext context) {
    final result = _lookupResult;
    if (result != null) {
      return OrderLookupResultScreen(
        status: result,
        email: _lookupEmail,
        onBack: widget.onBack,
        onLookupAnother: _lookupAnotherOrder,
      );
    }

    final l10n = context.l10n;

    return HankoFeaturePage(
      title: l10n.orderLookup,
      children: [
        if (widget.onBack != null)
          Align(
            alignment: Alignment.centerLeft,
            child: IconButton(
              onPressed: widget.onBack,
              tooltip: MaterialLocalizations.of(context).backButtonTooltip,
              icon: const Icon(Icons.arrow_back),
            ),
          ),
        HankoSurfaceCard(
          padding: const EdgeInsets.all(24),
          radius: 15,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              HankoTextField(
                label: l10n.orderNo,
                hintText: l10n.orderNoHint,
                controller: _orderNoController,
                enabled: !_isLoading,
                textInputAction: TextInputAction.next,
              ),
              const SizedBox(height: 16),
              HankoTextField(
                label: l10n.email,
                hintText: l10n.emailHint,
                controller: _emailController,
                enabled: !_isLoading,
                keyboardType: TextInputType.emailAddress,
                textInputAction: TextInputAction.search,
                onFieldSubmitted: (_) => _submitLookup(),
              ),
              const SizedBox(height: 24),
              HankoPrimaryButton(
                label: l10n.lookupOrder,
                icon: Icons.search,
                onPressed: _canLookup ? _submitLookup : null,
              ),
            ],
          ),
        ),
        if (_step != _OrderLookupStep.input) ...[
          const SizedBox(height: 16),
          _OrderLookupStateCard(
            step: _step,
            error: _lookupError,
            onRetry: _retryLookup,
          ),
        ],
      ],
    );
  }
}

bool _isLookupNotFoundError(HankoApiException error) {
  return switch (error.code.trim().toLowerCase()) {
    'order_not_found' || 'not_found' || 'validation_error' => true,
    _ => error.statusCode == 404,
  };
}

class OrderLookupResultScreen extends StatelessWidget {
  const OrderLookupResultScreen({
    super.key,
    required this.status,
    this.email,
    this.onBack,
    this.onLookupAnother,
  });

  final OrderStatus status;
  final String? email;
  final VoidCallback? onBack;
  final VoidCallback? onLookupAnother;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final createdAt = status.createdAt;
    final trackingNumber = _trimmedOrNull(status.trackingNumber);
    final carrier = _trimmedOrNull(status.fulfillmentCarrier);
    final sealText = _trimmedOrNull(status.sealText);
    final listingTitle =
        _trimmedOrNull(status.listingTitle) ?? _trimmedOrNull(status.listingId);
    final emailValue = _trimmedOrNull(email);

    return HankoFeaturePage(
      title: l10n.orderLookupResultTitle,
      children: [
        if (onBack != null)
          Align(
            alignment: Alignment.centerLeft,
            child: IconButton(
              onPressed: onBack,
              tooltip: MaterialLocalizations.of(context).backButtonTooltip,
              icon: const Icon(Icons.arrow_back),
            ),
          ),
        HankoSurfaceCard(
          radius: HankoRadii.sm,
          padding: const EdgeInsets.fromLTRB(18, 20, 18, 20),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const Icon(
                Icons.receipt_long_outlined,
                color: HankoColors.gold,
                size: 38,
              ),
              const SizedBox(height: HankoSpacing.md),
              Text(l10n.orderLookupResultMessage, style: HankoTextStyles.body),
              const SizedBox(height: HankoSpacing.md),
              _LookupDetailLine(label: l10n.orderNo, value: status.orderNo),
              if (createdAt != null)
                _LookupDetailLine(
                  label: l10n.orderLookupOrderDateLabel,
                  value: _formatDateTime(createdAt),
                ),
              _LookupDetailLine(
                label: l10n.orderLookupOrderStatusLabel,
                value: _statusLabel(l10n, status.orderStatus),
              ),
              if (emailValue != null)
                _LookupDetailLine(label: l10n.email, value: emailValue),
            ],
          ),
        ),
        const SizedBox(height: HankoSpacing.md),
        HankoEmailSentNotice(
          title: l10n.emailSentNoticeTitle,
          message: l10n.emailSentNoticeMessage,
          orderNoLabel: l10n.orderNo,
          emailLabel: l10n.email,
        ),
        const SizedBox(height: HankoSpacing.md),
        _LookupSectionCard(
          icon: Icons.timeline_outlined,
          title: l10n.orderLookupProgressTitle,
          children: [
            _LookupDetailLine(
              label: l10n.paymentStatusTitle,
              value: _statusLabel(l10n, status.paymentStatus),
            ),
            _LookupDetailLine(
              label: l10n.orderLookupProductionStatusLabel,
              value: _statusLabel(l10n, status.productionStatus),
            ),
            _LookupDetailLine(
              label: l10n.orderLookupShippingStatusLabel,
              value: _statusLabel(l10n, status.shippingStatus),
            ),
            _LookupDetailLine(
              label: l10n.orderLookupFulfillmentStatusLabel,
              value: _statusLabel(l10n, status.fulfillmentStatus),
            ),
          ],
        ),
        const SizedBox(height: HankoSpacing.md),
        _LookupSectionCard(
          icon: Icons.inventory_2_outlined,
          title: l10n.orderLookupContentTitle,
          children: [
            _LookupDetailLine(
              label: l10n.orderLookupSelectedSealLabel,
              value: sealText ?? l10n.orderLookupNoTrackingValue,
            ),
            _LookupDetailLine(
              label: l10n.orderLookupGemstoneLabel,
              value: listingTitle ?? l10n.orderLookupNoTrackingValue,
            ),
            _LookupDetailLine(
              label: l10n.orderTotalLabel,
              value: _formatMoney(status.pricing),
            ),
          ],
        ),
        const SizedBox(height: HankoSpacing.md),
        _LookupSectionCard(
          icon: Icons.local_shipping_outlined,
          title: l10n.orderLookupTrackingDetailsTitle,
          children: [
            _LookupDetailLine(
              label: l10n.orderLookupShippingStatusLabel,
              value: _statusLabel(l10n, status.shippingStatus),
            ),
            if (carrier != null)
              _LookupDetailLine(
                label: l10n.orderLookupCarrierLabel,
                value: carrier,
              ),
            _LookupDetailLine(
              label: l10n.orderLookupTrackingNumberLabel,
              value: trackingNumber ?? l10n.orderLookupNoTrackingValue,
            ),
            if (status.shippedAt != null)
              _LookupDetailLine(
                label: l10n.orderLookupShippedAtLabel,
                value: _formatDateTime(status.shippedAt!),
              ),
            if (status.updatedAt != null)
              _LookupDetailLine(
                label: l10n.orderLookupUpdatedAtLabel,
                value: _formatDateTime(status.updatedAt!),
              ),
          ],
        ),
        if (onLookupAnother != null) ...[
          const SizedBox(height: HankoSpacing.md),
          HankoPrimaryButton(
            label: l10n.orderLookupLookupAnotherAction,
            icon: Icons.search,
            onPressed: onLookupAnother,
          ),
        ],
      ],
    );
  }
}

class _LookupSectionCard extends StatelessWidget {
  const _LookupSectionCard({
    required this.icon,
    required this.title,
    required this.children,
  });

  final IconData icon;
  final String title;
  final List<Widget> children;

  @override
  Widget build(BuildContext context) {
    return HankoSurfaceCard(
      radius: HankoRadii.sm,
      padding: const EdgeInsets.fromLTRB(18, 18, 18, 6),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Row(
            children: [
              Icon(icon, color: HankoColors.gold, size: 22),
              const SizedBox(width: HankoSpacing.sm),
              Expanded(child: Text(title, style: HankoTextStyles.cardTitle)),
            ],
          ),
          const SizedBox(height: HankoSpacing.sm),
          ...children,
        ],
      ),
    );
  }
}

class _LookupDetailLine extends StatelessWidget {
  const _LookupDetailLine({required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: HankoSpacing.sm),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Expanded(
            flex: 3,
            child: Text(label, style: HankoTextStyles.compactBody),
          ),
          const SizedBox(width: HankoSpacing.sm),
          Expanded(
            flex: 4,
            child: Text(
              value,
              textAlign: TextAlign.right,
              style: HankoTextStyles.label,
            ),
          ),
        ],
      ),
    );
  }
}

class _OrderLookupStateCard extends StatelessWidget {
  const _OrderLookupStateCard({
    required this.step,
    required this.error,
    required this.onRetry,
  });

  final _OrderLookupStep step;
  final Object? error;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    return switch (step) {
      _OrderLookupStep.loading => HankoStateView.loading(
        title: l10n.orderLookupLoadingTitle,
        message: l10n.orderLookupLoadingMessage,
      ),
      _OrderLookupStep.notFound => HankoStateView.empty(
        title: l10n.orderLookupNotFoundTitle,
        message: l10n.orderLookupNotFoundMessage,
        actionLabel: l10n.tryAgain,
        onAction: onRetry,
      ),
      _OrderLookupStep.error => HankoErrorStateView(
        error: error,
        actionLabel: l10n.tryAgain,
        onAction: onRetry,
      ),
      _OrderLookupStep.input => const SizedBox.shrink(),
    };
  }
}

String? _trimmedOrNull(String? value) {
  final trimmed = value?.trim();
  if (trimmed == null || trimmed.isEmpty) {
    return null;
  }
  return trimmed;
}

String _formatDateTime(DateTime value) {
  final local = value.toLocal();
  final date = [
    local.year.toString().padLeft(4, '0'),
    local.month.toString().padLeft(2, '0'),
    local.day.toString().padLeft(2, '0'),
  ].join('-');
  final time = [
    local.hour.toString().padLeft(2, '0'),
    local.minute.toString().padLeft(2, '0'),
  ].join(':');
  return '$date $time';
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

String _statusLabel(HankoLocalizations l10n, String value) {
  final normalized = value.trim().toLowerCase();
  return switch (normalized) {
    'paid' => l10n.orderStatusPaid,
    'unpaid' => l10n.orderStatusUnpaid,
    'pending' => l10n.orderStatusPending,
    'pending_payment' => l10n.orderStatusPendingPayment,
    'failed' => l10n.orderStatusFailed,
    'cancelled' || 'canceled' => l10n.orderStatusCanceled,
    'not_started' => l10n.orderStatusNotStarted,
    'in_production' => l10n.orderStatusInProduction,
    'completed' => l10n.orderStatusCompleted,
    'preparing_shipment' => l10n.orderStatusPreparingShipment,
    'not_shipped' => l10n.orderStatusNotShipped,
    'shipped' => l10n.orderStatusShipped,
    'fulfilled' => l10n.orderStatusFulfilled,
    '' => l10n.orderStatusEmpty,
    _ => _labelFromToken(value),
  };
}

String _labelFromToken(String value) {
  final normalized = value.trim().replaceAll(RegExp(r'[_-]+'), ' ');
  if (normalized.isEmpty) {
    return '-';
  }
  return normalized
      .split(RegExp(r'\s+'))
      .map((word) {
        if (word.isEmpty) {
          return word;
        }
        return word[0].toUpperCase() + word.substring(1);
      })
      .join(' ');
}
