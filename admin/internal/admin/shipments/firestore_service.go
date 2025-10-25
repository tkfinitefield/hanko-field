package shipments

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	syntheticAlertExceptionLabel       = "配送例外が検出されました"
	syntheticAlertExceptionDesc        = "%d 件が要対応ステータスです。優先的に確認してください。"
	syntheticAlertExceptionActionLabel = "例外を確認"
	syntheticAlertDelayLabel           = "SLA遅延リスク"
	syntheticAlertDelayDesc            = "%d 件がSLA警告ゾーンに入っています。"
	syntheticAlertDelayActionLabel     = "遅延を確認"
	defaultAlertActionLabel            = "詳細を見る"
)

// FirestoreConfig tunes Firestore-backed shipment tracking.
type FirestoreConfig struct {
	TrackingCollection     string
	AlertsCollection       string
	MetadataDocPath        string
	FetchLimit             int
	AlertsLimit            int
	CacheTTL               time.Duration
	DefaultRefreshInterval time.Duration
	Now                    func() time.Time
	BatchService           Service
}

// FirestoreService hydrates tracking data from Firestore views fed by carrier webhooks.
type FirestoreService struct {
	client             *firestore.Client
	trackingCollection string
	alertsCollection   string
	metadataRef        *firestore.DocumentRef
	fetchLimit         int
	alertsLimit        int
	cacheTTL           time.Duration
	defaultRefresh     time.Duration
	now                func() time.Time
	batches            Service
	mu                 sync.RWMutex
	dataset            trackingDataset
	alertsMu           sync.RWMutex
	alertsCache        alertsCache
	datasetFetchMu     sync.Mutex
}

type trackingDataset struct {
	shipments       []TrackingShipment
	filters         TrackingFilters
	version         string
	lastUpdated     time.Time
	refreshInterval time.Duration
	generated       time.Time
	expires         time.Time
}

type alertsCache struct {
	alerts  []TrackingAlert
	expires time.Time
}

type metadataSnapshot struct {
	version         string
	lastUpdated     time.Time
	refreshInterval time.Duration
}

type trackingDocument struct {
	ShipmentID         string    `firestore:"shipmentId"`
	OrderID            string    `firestore:"orderId"`
	OrderNumber        string    `firestore:"orderNumber"`
	CustomerName       string    `firestore:"customerName"`
	Carrier            string    `firestore:"carrier"`
	CarrierLabel       string    `firestore:"carrierLabel"`
	Status             string    `firestore:"status"`
	StatusLabel        string    `firestore:"statusLabel"`
	StatusTone         string    `firestore:"statusTone"`
	TrackingNumber     string    `firestore:"trackingNumber"`
	ServiceLevel       string    `firestore:"serviceLevel"`
	Destination        string    `firestore:"destination"`
	Region             string    `firestore:"region"`
	Lane               string    `firestore:"lane"`
	LastEvent          string    `firestore:"lastEvent"`
	LastEventAt        time.Time `firestore:"lastEventAt"`
	ETA                time.Time `firestore:"eta"`
	DelayMinutes       int       `firestore:"delayMinutes"`
	SLAStatus          string    `firestore:"slaStatus"`
	SLATone            string    `firestore:"slaTone"`
	ExceptionLabel     string    `firestore:"exceptionLabel"`
	ExceptionTone      string    `firestore:"exceptionTone"`
	AlertIcon          string    `firestore:"alertIcon"`
	OrderURL           string    `firestore:"orderUrl"`
	DestinationRegion  string    `firestore:"destinationRegion"`
	DestinationPref    string    `firestore:"destinationPref"`
	DestinationCountry string    `firestore:"destinationCountry"`
	UpdatedAt          time.Time `firestore:"updatedAt"`
}

type alertDocument struct {
	Label       string    `firestore:"label"`
	Description string    `firestore:"description"`
	Tone        string    `firestore:"tone"`
	ActionLabel string    `firestore:"actionLabel"`
	ActionURL   string    `firestore:"actionUrl"`
	Priority    int       `firestore:"priority"`
	ExpiresAt   time.Time `firestore:"expiresAt"`
}

// NewFirestoreService constructs a Firestore-backed tracking service.
func NewFirestoreService(client *firestore.Client, cfg FirestoreConfig) *FirestoreService {
	if client == nil {
		panic("shipments: firestore client is required")
	}
	if cfg.TrackingCollection == "" {
		cfg.TrackingCollection = "ops_tracking_shipments"
	}
	if cfg.AlertsCollection == "" {
		cfg.AlertsCollection = "ops_tracking_alerts"
	}
	if cfg.FetchLimit <= 0 {
		cfg.FetchLimit = 500
	}
	if cfg.AlertsLimit <= 0 {
		cfg.AlertsLimit = 5
	}
	if cfg.CacheTTL <= 0 {
		cfg.CacheTTL = 15 * time.Second
	}
	if cfg.DefaultRefreshInterval <= 0 {
		cfg.DefaultRefreshInterval = 30 * time.Second
	}
	nowFn := cfg.Now
	if nowFn == nil {
		nowFn = time.Now
	}

	var metadataRef *firestore.DocumentRef
	if trimmed := strings.Trim(cfg.MetadataDocPath, "/"); trimmed != "" {
		if ref := documentRefFromPath(client, trimmed); ref != nil {
			metadataRef = ref
		} else {
			log.Printf("shipments: invalid metadata doc path %q; metadata cache invalidation disabled", trimmed)
		}
	}

	batchSvc := cfg.BatchService
	if batchSvc == nil {
		batchSvc = NewStaticService()
	}

	return &FirestoreService{
		client:             client,
		trackingCollection: cfg.TrackingCollection,
		alertsCollection:   cfg.AlertsCollection,
		metadataRef:        metadataRef,
		fetchLimit:         cfg.FetchLimit,
		alertsLimit:        cfg.AlertsLimit,
		cacheTTL:           cfg.CacheTTL,
		defaultRefresh:     cfg.DefaultRefreshInterval,
		now:                nowFn,
		batches:            batchSvc,
	}
}

// ListBatches delegates to the configured batch service (static placeholder for now).
func (s *FirestoreService) ListBatches(ctx context.Context, token string, query ListQuery) (ListResult, error) {
	if s.batches == nil {
		return ListResult{}, fmt.Errorf("shipments: batch service not configured")
	}
	return s.batches.ListBatches(ctx, token, query)
}

// BatchDetail delegates batch detail lookups to the configured service.
func (s *FirestoreService) BatchDetail(ctx context.Context, token, batchID string) (BatchDetail, error) {
	if s.batches == nil {
		return BatchDetail{}, fmt.Errorf("shipments: batch service not configured")
	}
	return s.batches.BatchDetail(ctx, token, batchID)
}

// ListTracking reads shipment tracking rows from Firestore-backed views.
func (s *FirestoreService) ListTracking(ctx context.Context, _ string, query TrackingQuery) (TrackingResult, error) {
	dataset, err := s.loadDataset(ctx)
	if err != nil {
		return TrackingResult{}, err
	}

	filtered := filterTrackingShipments(dataset.shipments, query)
	rows, pagination := paginateTrackingShipments(filtered, query.Page, query.PageSize)

	lastRefresh := dataset.lastUpdated
	if lastRefresh.IsZero() {
		lastRefresh = dataset.generated
	}
	summary := trackingSummary(dataset.shipments, lastRefresh, dataset.refreshInterval)

	alerts := s.resolveAlerts(ctx, summary)

	return TrackingResult{
		Summary:    summary,
		Shipments:  rows,
		Filters:    cloneTrackingFilters(dataset.filters),
		Pagination: pagination,
		Generated:  dataset.generated,
		Alerts:     alerts,
	}, nil
}

func (s *FirestoreService) loadDataset(ctx context.Context) (trackingDataset, error) {
	meta, err := s.loadMetadata(ctx)
	if err != nil {
		return trackingDataset{}, err
	}
	now := s.now()

	s.mu.RLock()
	if s.dataset.isValid(now, meta.version) {
		cached := s.dataset.withMetadata(meta, now, s.cacheTTL, s.defaultRefresh)
		s.mu.RUnlock()
		return cached, nil
	}
	s.mu.RUnlock()

	s.datasetFetchMu.Lock()
	defer s.datasetFetchMu.Unlock()

	meta, err = s.loadMetadata(ctx)
	if err != nil {
		return trackingDataset{}, err
	}
	now = s.now()

	s.mu.RLock()
	if s.dataset.isValid(now, meta.version) {
		cached := s.dataset.withMetadata(meta, now, s.cacheTTL, s.defaultRefresh)
		s.mu.RUnlock()
		return cached, nil
	}
	s.mu.RUnlock()

	shipments, err := s.fetchShipments(ctx)
	if err != nil {
		return trackingDataset{}, err
	}

	newDataset := trackingDataset{
		shipments: shipments,
		filters: TrackingFilters{
			StatusOptions:  buildTrackingStatusOptions(shipments),
			CarrierOptions: buildTrackingCarrierOptions(shipments),
			LaneOptions:    buildTrackingLaneOptions(shipments),
			RegionOptions:  buildTrackingRegionOptions(shipments),
		},
		version:         meta.version,
		lastUpdated:     meta.lastUpdated,
		refreshInterval: meta.refreshIntervalOr(s.defaultRefresh),
		generated:       now,
		expires:         now.Add(s.cacheTTL),
	}

	s.mu.Lock()
	s.dataset = newDataset
	s.mu.Unlock()

	return newDataset, nil
}

func (s *FirestoreService) fetchShipments(ctx context.Context) ([]TrackingShipment, error) {
	iter := s.client.Collection(s.trackingCollection).
		OrderBy("lastEventAt", firestore.Desc).
		Limit(s.fetchLimit).
		Documents(ctx)
	defer iter.Stop()

	var shipments []TrackingShipment
	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("shipments: load tracking dataset failed: %w", err)
		}
		sh, err := decodeTrackingShipment(snap)
		if err != nil {
			log.Printf("shipments: skip tracking doc %s: %v", snap.Ref.Path, err)
			continue
		}
		shipments = append(shipments, sh)
	}
	return shipments, nil
}

func (s *FirestoreService) loadMetadata(ctx context.Context) (metadataSnapshot, error) {
	if s.metadataRef == nil {
		return metadataSnapshot{}, nil
	}

	snap, err := s.metadataRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return metadataSnapshot{}, nil
		}
		return metadataSnapshot{}, fmt.Errorf("shipments: fetch tracking metadata: %w", err)
	}

	var doc struct {
		Version                string    `firestore:"version"`
		UpdatedAt              time.Time `firestore:"updatedAt"`
		RefreshIntervalSeconds int       `firestore:"refreshIntervalSeconds"`
	}
	if err := snap.DataTo(&doc); err != nil {
		return metadataSnapshot{}, fmt.Errorf("shipments: parse metadata doc: %w", err)
	}

	meta := metadataSnapshot{
		version:     doc.Version,
		lastUpdated: doc.UpdatedAt,
	}
	if meta.version == "" {
		meta.version = snap.UpdateTime.String()
	}
	if meta.lastUpdated.IsZero() {
		meta.lastUpdated = snap.UpdateTime
	}
	if doc.RefreshIntervalSeconds > 0 {
		meta.refreshInterval = time.Duration(doc.RefreshIntervalSeconds) * time.Second
	}

	return meta, nil
}

func (s *FirestoreService) resolveAlerts(ctx context.Context, summary TrackingSummary) []TrackingAlert {
	alerts, err := s.loadAlerts(ctx)
	if err != nil {
		log.Printf("shipments: tracking alerts fallback: %v", err)
		return syntheticTrackingAlerts(summary)
	}
	if len(alerts) == 0 {
		return syntheticTrackingAlerts(summary)
	}
	return alerts
}

func (s *FirestoreService) loadAlerts(ctx context.Context) ([]TrackingAlert, error) {
	if s.alertsCollection == "" {
		return nil, nil
	}

	now := s.now()
	s.alertsMu.RLock()
	if len(s.alertsCache.alerts) > 0 && s.alertsCache.expires.After(now) {
		alerts := append([]TrackingAlert(nil), s.alertsCache.alerts...)
		s.alertsMu.RUnlock()
		return alerts, nil
	}
	s.alertsMu.RUnlock()

	alerts, err := s.fetchAlerts(ctx, now)
	if err != nil {
		return nil, err
	}

	s.alertsMu.Lock()
	defer s.alertsMu.Unlock()
	if len(s.alertsCache.alerts) > 0 && s.alertsCache.expires.After(now) {
		return append([]TrackingAlert(nil), s.alertsCache.alerts...), nil
	}

	s.alertsCache = alertsCache{
		alerts:  alerts,
		expires: now.Add(s.cacheTTL),
	}
	return append([]TrackingAlert(nil), alerts...), nil
}

func (s *FirestoreService) fetchAlerts(ctx context.Context, now time.Time) ([]TrackingAlert, error) {
	iter := s.client.Collection(s.alertsCollection).
		OrderBy("priority", firestore.Asc).
		OrderBy("updatedAt", firestore.Desc).
		Limit(s.alertsLimit).
		Documents(ctx)
	defer iter.Stop()

	var alerts []TrackingAlert
	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("shipments: load tracking alerts failed: %w", err)
		}
		var doc alertDocument
		if err := snap.DataTo(&doc); err != nil {
			log.Printf("shipments: skip alert doc %s: %v", snap.Ref.Path, err)
			continue
		}
		if !doc.ExpiresAt.IsZero() && doc.ExpiresAt.Before(now) {
			continue
		}
		alerts = append(alerts, TrackingAlert{
			Label:       doc.Label,
			Description: doc.Description,
			Tone:        defaultTone(doc.Tone),
			ActionLabel: firstNonEmpty(doc.ActionLabel, defaultAlertActionLabel),
			ActionURL:   firstNonEmpty(doc.ActionURL, "/admin/shipments/tracking"),
		})
	}
	return alerts, nil
}

func decodeTrackingShipment(snap *firestore.DocumentSnapshot) (TrackingShipment, error) {
	var doc trackingDocument
	if err := snap.DataTo(&doc); err != nil {
		return TrackingShipment{}, fmt.Errorf("shipments: parse tracking doc: %w", err)
	}

	sh := TrackingShipment{
		ID:             firstNonEmpty(doc.ShipmentID, snap.Ref.ID, doc.TrackingNumber),
		OrderID:        doc.OrderID,
		OrderNumber:    doc.OrderNumber,
		CustomerName:   doc.CustomerName,
		Carrier:        doc.Carrier,
		CarrierLabel:   doc.CarrierLabel,
		Status:         toTrackingStatus(doc.Status),
		StatusLabel:    doc.StatusLabel,
		StatusTone:     doc.StatusTone,
		TrackingNumber: doc.TrackingNumber,
		ServiceLevel:   doc.ServiceLevel,
		Destination:    firstNonEmpty(doc.Destination, doc.DestinationPref, doc.DestinationRegion),
		Region:         firstNonEmpty(doc.Region, doc.DestinationRegion),
		Lane:           doc.Lane,
		LastEvent:      doc.LastEvent,
		DelayMinutes:   doc.DelayMinutes,
		SLAStatus:      doc.SLAStatus,
		SLATone:        doc.SLATone,
		ExceptionLabel: doc.ExceptionLabel,
		ExceptionTone:  doc.ExceptionTone,
		AlertIcon:      doc.AlertIcon,
		OrderURL:       doc.OrderURL,
	}

	if !doc.LastEventAt.IsZero() {
		sh.LastEventAt = doc.LastEventAt
	} else if !doc.UpdatedAt.IsZero() {
		sh.LastEventAt = doc.UpdatedAt
	} else {
		sh.LastEventAt = snap.UpdateTime
	}
	if !doc.ETA.IsZero() {
		eta := doc.ETA
		sh.EstimatedArrival = &eta
	}

	return normalizeTrackingShipment(sh), nil
}

func toTrackingStatus(value string) TrackingStatus {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "label_created":
		return TrackingStatusLabelCreated
	case "out_for_delivery":
		return TrackingStatusOutForDelivery
	case "delivered":
		return TrackingStatusDelivered
	case "exception":
		return TrackingStatusException
	default:
		return TrackingStatusInTransit
	}
}

func documentRefFromPath(client *firestore.Client, path string) *firestore.DocumentRef {
	parts := strings.Split(path, "/")
	if len(parts) < 2 || len(parts)%2 != 0 {
		return nil
	}
	var ref *firestore.DocumentRef
	for i := 0; i < len(parts); i += 2 {
		collectionID := strings.TrimSpace(parts[i])
		docID := strings.TrimSpace(parts[i+1])
		if collectionID == "" || docID == "" {
			return nil
		}
		if ref == nil {
			ref = client.Collection(collectionID).Doc(docID)
		} else {
			ref = ref.Collection(collectionID).Doc(docID)
		}
	}
	return ref
}

func (d trackingDataset) isValid(now time.Time, version string) bool {
	if len(d.shipments) == 0 {
		return false
	}
	if version != "" && version != d.version {
		return false
	}
	return now.Before(d.expires)
}

func (d trackingDataset) withMetadata(meta metadataSnapshot, now time.Time, ttl, refreshFallback time.Duration) trackingDataset {
	clone := trackingDataset{
		shipments:       d.shipments,
		filters:         d.filters,
		version:         d.version,
		lastUpdated:     meta.lastUpdated,
		refreshInterval: meta.refreshIntervalOr(refreshFallback),
		generated:       d.generated,
		expires:         now.Add(ttl),
	}
	if clone.refreshInterval <= 0 {
		clone.refreshInterval = refreshFallback
	}
	if clone.lastUpdated.IsZero() {
		clone.lastUpdated = d.lastUpdated
	}
	if clone.version == "" {
		clone.version = meta.version
	}
	return clone
}

func (m metadataSnapshot) refreshIntervalOr(fallback time.Duration) time.Duration {
	if m.refreshInterval > 0 {
		return m.refreshInterval
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func defaultTone(tone string) string {
	switch strings.TrimSpace(tone) {
	case "danger", "warning", "success", "info", "slate":
		return tone
	default:
		return "info"
	}
}

func syntheticTrackingAlerts(summary TrackingSummary) []TrackingAlert {
	var alerts []TrackingAlert
	if summary.Exceptions > 0 {
		alerts = append(alerts, TrackingAlert{
			Label:       syntheticAlertExceptionLabel,
			Description: fmt.Sprintf(syntheticAlertExceptionDesc, summary.Exceptions),
			Tone:        "danger",
			ActionLabel: syntheticAlertExceptionActionLabel,
			ActionURL:   "/admin/shipments/tracking?status=exception",
		})
	}
	if summary.Delayed > 0 {
		alerts = append(alerts, TrackingAlert{
			Label:       syntheticAlertDelayLabel,
			Description: fmt.Sprintf(syntheticAlertDelayDesc, summary.Delayed),
			Tone:        "warning",
			ActionLabel: syntheticAlertDelayActionLabel,
			ActionURL:   "/admin/shipments/tracking?delay=delayed",
		})
	}
	return alerts
}
