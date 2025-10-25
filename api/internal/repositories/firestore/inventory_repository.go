package firestore

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	domain "github.com/hanko-field/api/internal/domain"
	pfirestore "github.com/hanko-field/api/internal/platform/firestore"
	"github.com/hanko-field/api/internal/repositories"
)

const (
	inventoryCollection         = "inventory"
	stockReservationsCollection = "stockReservations"

	reservationStatusReserved  = "reserved"
	reservationStatusCommitted = "committed"
	reservationStatusReleased  = "released"
)

type InventoryRepository struct {
	provider     *pfirestore.Provider
	stocks       *pfirestore.BaseRepository[stockDocument]
	reservations *pfirestore.BaseRepository[reservationDocument]
}

func NewInventoryRepository(provider *pfirestore.Provider) (*InventoryRepository, error) {
	if provider == nil {
		return nil, errors.New("inventory repository requires firestore provider")
	}
	stocks := pfirestore.NewBaseRepository[stockDocument](provider, inventoryCollection, nil, nil)
	reservations := pfirestore.NewBaseRepository[reservationDocument](provider, stockReservationsCollection, nil, nil)
	return &InventoryRepository{provider: provider, stocks: stocks, reservations: reservations}, nil
}

func (r *InventoryRepository) Reserve(ctx context.Context, req repositories.InventoryReserveRequest) (repositories.InventoryReserveResult, error) {
	if r == nil || r.provider == nil {
		return repositories.InventoryReserveResult{}, errors.New("inventory repository not initialised")
	}
	if req.Reservation.ID == "" {
		return repositories.InventoryReserveResult{}, errors.New("inventory reserve: reservation id is required")
	}
	if len(req.Reservation.Lines) == 0 {
		return repositories.InventoryReserveResult{}, errors.New("inventory reserve: at least one line is required")
	}

	now := req.Now.UTC()
	reservation := req.Reservation
	reservation.Status = reservationStatusReserved
	reservation.CreatedAt = reservation.CreatedAt.UTC()
	if reservation.CreatedAt.IsZero() {
		reservation.CreatedAt = now
	}
	reservation.UpdatedAt = now
	reservation.ExpiresAt = reservation.ExpiresAt.UTC()

	var result repositories.InventoryReserveResult
	err := r.provider.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		resRef, err := r.reservations.DocumentRef(ctx, reservation.ID)
		if err != nil {
			return err
		}

		if _, err := tx.Get(resRef); err == nil {
			return repositories.NewInventoryError(repositories.InventoryErrorInvalidReservationState, fmt.Sprintf("reservation %s already exists", reservation.ID), nil)
		} else if status.Code(err) != codes.NotFound {
			return err
		}

		stocks := make(map[string]domain.InventoryStock)
		for _, line := range reservation.Lines {
			sku := strings.TrimSpace(line.SKU)
			if sku == "" {
				return repositories.NewInventoryError(repositories.InventoryErrorStockNotFound, "inventory reserve: sku is required", nil)
			}
			if line.Quantity <= 0 {
				return repositories.NewInventoryError(repositories.InventoryErrorUnknown, fmt.Sprintf("inventory reserve: quantity for %s must be > 0", sku), nil)
			}

			stockRef, err := r.stocks.DocumentRef(ctx, sku)
			if err != nil {
				return err
			}
			snap, err := tx.Get(stockRef)
			if err != nil {
				if status.Code(err) == codes.NotFound {
					return repositories.NewInventoryError(repositories.InventoryErrorStockNotFound, fmt.Sprintf("stock %s not found", sku), err)
				}
				return err
			}
			var stockDoc stockDocument
			if err := snap.DataTo(&stockDoc); err != nil {
				return fmt.Errorf("decode inventory stock %s: %w", sku, err)
			}
			available := stockDoc.OnHand - stockDoc.Reserved
			if available < line.Quantity {
				return repositories.NewInventoryError(repositories.InventoryErrorInsufficientStock, fmt.Sprintf("insufficient stock for %s", sku), nil)
			}
			stockDoc.Reserved += line.Quantity
			stockDoc.UpdatedAt = now
			stockDoc.recalculate()
			if err := tx.Set(stockRef, stockDoc); err != nil {
				return err
			}
			stocks[sku] = stockDoc.toDomain(sku)
		}

		resDoc := newReservationDocument(reservation)
		resDoc.UpdatedAt = now
		if resDoc.CreatedAt.IsZero() {
			resDoc.CreatedAt = now
		}
		resDoc.Status = reservationStatusReserved

		if err := tx.Create(resRef, resDoc); err != nil {
			if status.Code(err) == codes.AlreadyExists {
				return repositories.NewInventoryError(repositories.InventoryErrorInvalidReservationState, fmt.Sprintf("reservation %s already exists", reservation.ID), err)
			}
			return err
		}

		result = repositories.InventoryReserveResult{
			Reservation: resDoc.toDomain(reservation.ID),
			Stocks:      stocks,
		}
		return nil
	})
	if err != nil {
		return repositories.InventoryReserveResult{}, wrapInventoryError("inventory.reserve", err)
	}
	return result, nil
}

func (r *InventoryRepository) Commit(ctx context.Context, req repositories.InventoryCommitRequest) (repositories.InventoryCommitResult, error) {
	if r == nil || r.provider == nil {
		return repositories.InventoryCommitResult{}, errors.New("inventory repository not initialised")
	}
	if strings.TrimSpace(req.ReservationID) == "" {
		return repositories.InventoryCommitResult{}, errors.New("inventory commit: reservation id is required")
	}

	now := req.Now.UTC()
	var result repositories.InventoryCommitResult

	err := r.provider.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		resRef, err := r.reservations.DocumentRef(ctx, req.ReservationID)
		if err != nil {
			return err
		}
		resSnap, err := tx.Get(resRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return repositories.NewInventoryError(repositories.InventoryErrorReservationNotFound, fmt.Sprintf("reservation %s not found", req.ReservationID), err)
			}
			return err
		}
		resDoc, err := decodeReservation(resSnap)
		if err != nil {
			return err
		}
		if resDoc.Status != reservationStatusReserved {
			return repositories.NewInventoryError(repositories.InventoryErrorInvalidReservationState, fmt.Sprintf("reservation %s is not in reserved status", req.ReservationID), nil)
		}
		if req.OrderRef != "" && !strings.EqualFold(resDoc.OrderRef, req.OrderRef) {
			return repositories.NewInventoryError(repositories.InventoryErrorInvalidReservationState, fmt.Sprintf("reservation %s order mismatch", req.ReservationID), nil)
		}

		stocks := make(map[string]domain.InventoryStock)
		for _, line := range resDoc.Lines {
			sku := strings.TrimSpace(line.SKU)
			stockRef, err := r.stocks.DocumentRef(ctx, sku)
			if err != nil {
				return err
			}
			snap, err := tx.Get(stockRef)
			if err != nil {
				if status.Code(err) == codes.NotFound {
					return repositories.NewInventoryError(repositories.InventoryErrorStockNotFound, fmt.Sprintf("stock %s not found", sku), err)
				}
				return err
			}
			var stockDoc stockDocument
			if err := snap.DataTo(&stockDoc); err != nil {
				return fmt.Errorf("decode inventory stock %s: %w", sku, err)
			}
			if stockDoc.Reserved < line.Quantity {
				return repositories.NewInventoryError(repositories.InventoryErrorInvalidReservationState, fmt.Sprintf("reserved quantity for %s is insufficient", sku), nil)
			}
			if stockDoc.OnHand < line.Quantity {
				return repositories.NewInventoryError(repositories.InventoryErrorInvalidReservationState, fmt.Sprintf("onHand for %s cannot drop below zero", sku), nil)
			}
			stockDoc.Reserved -= line.Quantity
			stockDoc.OnHand -= line.Quantity
			stockDoc.UpdatedAt = now
			stockDoc.recalculate()
			if err := tx.Set(stockRef, stockDoc); err != nil {
				return err
			}
			stocks[sku] = stockDoc.toDomain(sku)
		}

		resDoc.Status = reservationStatusCommitted
		resDoc.UpdatedAt = now
		resDoc.CommittedAt = &now
		if err := tx.Set(resRef, resDoc); err != nil {
			return err
		}

		result = repositories.InventoryCommitResult{
			Reservation: resDoc.toDomain(req.ReservationID),
			Stocks:      stocks,
		}
		return nil
	})
	if err != nil {
		return repositories.InventoryCommitResult{}, wrapInventoryError("inventory.commit", err)
	}
	return result, nil
}

func (r *InventoryRepository) Release(ctx context.Context, req repositories.InventoryReleaseRequest) (repositories.InventoryReleaseResult, error) {
	if r == nil || r.provider == nil {
		return repositories.InventoryReleaseResult{}, errors.New("inventory repository not initialised")
	}
	if strings.TrimSpace(req.ReservationID) == "" {
		return repositories.InventoryReleaseResult{}, errors.New("inventory release: reservation id is required")
	}

	now := req.Now.UTC()
	var result repositories.InventoryReleaseResult

	err := r.provider.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		resRef, err := r.reservations.DocumentRef(ctx, req.ReservationID)
		if err != nil {
			return err
		}
		resSnap, err := tx.Get(resRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return repositories.NewInventoryError(repositories.InventoryErrorReservationNotFound, fmt.Sprintf("reservation %s not found", req.ReservationID), err)
			}
			return err
		}
		resDoc, err := decodeReservation(resSnap)
		if err != nil {
			return err
		}
		if resDoc.Status != reservationStatusReserved {
			return repositories.NewInventoryError(repositories.InventoryErrorInvalidReservationState, fmt.Sprintf("reservation %s not in reserved status", req.ReservationID), nil)
		}

		stocks := make(map[string]domain.InventoryStock)
		for _, line := range resDoc.Lines {
			sku := strings.TrimSpace(line.SKU)
			stockRef, err := r.stocks.DocumentRef(ctx, sku)
			if err != nil {
				return err
			}
			snap, err := tx.Get(stockRef)
			if err != nil {
				if status.Code(err) == codes.NotFound {
					return repositories.NewInventoryError(repositories.InventoryErrorStockNotFound, fmt.Sprintf("stock %s not found", sku), err)
				}
				return err
			}
			var stockDoc stockDocument
			if err := snap.DataTo(&stockDoc); err != nil {
				return fmt.Errorf("decode inventory stock %s: %w", sku, err)
			}
			if stockDoc.Reserved < line.Quantity {
				return repositories.NewInventoryError(repositories.InventoryErrorInvalidReservationState, fmt.Sprintf("reserved quantity for %s is insufficient", sku), nil)
			}
			stockDoc.Reserved -= line.Quantity
			stockDoc.UpdatedAt = now
			stockDoc.recalculate()
			if err := tx.Set(stockRef, stockDoc); err != nil {
				return err
			}
			stocks[sku] = stockDoc.toDomain(sku)
		}

		resDoc.Status = reservationStatusReleased
		resDoc.UpdatedAt = now
		resDoc.ReleasedAt = &now
		if req.Reason != "" {
			resDoc.Reason = strings.TrimSpace(req.Reason)
		}
		if err := tx.Set(resRef, resDoc); err != nil {
			return err
		}

		result = repositories.InventoryReleaseResult{
			Reservation: resDoc.toDomain(req.ReservationID),
			Stocks:      stocks,
		}
		return nil
	})
	if err != nil {
		return repositories.InventoryReleaseResult{}, wrapInventoryError("inventory.release", err)
	}
	return result, nil
}

func (r *InventoryRepository) GetReservation(ctx context.Context, reservationID string) (domain.InventoryReservation, error) {
	if r == nil || r.reservations == nil {
		return domain.InventoryReservation{}, errors.New("inventory repository not initialised")
	}
	reservationID = strings.TrimSpace(reservationID)
	if reservationID == "" {
		return domain.InventoryReservation{}, errors.New("inventory get reservation: id is required")
	}

	doc, err := r.reservations.Get(ctx, reservationID)
	if err != nil {
		if repoErr, ok := err.(*pfirestore.Error); ok && repoErr.IsNotFound() {
			return domain.InventoryReservation{}, repositories.NewInventoryError(repositories.InventoryErrorReservationNotFound, fmt.Sprintf("reservation %s not found", reservationID), err)
		}
		return domain.InventoryReservation{}, wrapInventoryError("inventory.getReservation", err)
	}

	return doc.Data.toDomain(doc.ID), nil
}

func (r *InventoryRepository) ListLowStock(ctx context.Context, query repositories.InventoryLowStockQuery) (domain.CursorPage[domain.InventoryStock], error) {
	if r == nil || r.stocks == nil {
		return domain.CursorPage[domain.InventoryStock]{}, errors.New("inventory repository not initialised")
	}

	pageSize := query.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}

	client, err := r.provider.Client(ctx)
	if err != nil {
		return domain.CursorPage[domain.InventoryStock]{}, wrapInventoryError("inventory.lowStock", err)
	}

	firestoreQuery := client.Collection(inventoryCollection).Query
	if query.Threshold > 0 {
		firestoreQuery = firestoreQuery.Where("available", "<=", query.Threshold).OrderBy("available", firestore.Asc)
	} else {
		firestoreQuery = firestoreQuery.Where("safetyDelta", "<", 0).OrderBy("safetyDelta", firestore.Asc)
	}
	firestoreQuery = firestoreQuery.OrderBy("sku", firestore.Asc).Limit(pageSize + 1)

	var decodedToken *inventoryPageToken
	if token := strings.TrimSpace(query.PageToken); token != "" {
		tok, err := decodeInventoryPageToken(token)
		if err != nil {
			return domain.CursorPage[domain.InventoryStock]{}, wrapInventoryError("inventory.lowStock", err)
		}
		decodedToken = tok
	}
	if decodedToken != nil {
		if query.Threshold > 0 {
			firestoreQuery = firestoreQuery.StartAfter(decodedToken.Available, decodedToken.SKU)
		} else {
			firestoreQuery = firestoreQuery.StartAfter(decodedToken.SafetyDelta, decodedToken.SKU)
		}
	}

	iter := firestoreQuery.Documents(ctx)
	defer iter.Stop()

	var stocks []domain.InventoryStock
	for {
		snap, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return domain.CursorPage[domain.InventoryStock]{}, wrapInventoryError("inventory.lowStock", err)
		}
		var doc stockDocument
		if err := snap.DataTo(&doc); err != nil {
			return domain.CursorPage[domain.InventoryStock]{}, fmt.Errorf("decode inventory stock %s: %w", snap.Ref.ID, err)
		}
		stocks = append(stocks, doc.toDomain(snap.Ref.ID))
	}

	hasMore := len(stocks) > pageSize
	if hasMore {
		stocks = stocks[:pageSize]
	}
	var nextToken string
	if hasMore && len(stocks) > 0 {
		last := stocks[len(stocks)-1]
		encoded, err := encodeInventoryPageToken(inventoryPageToken{SKU: last.SKU, Available: last.Available, SafetyDelta: last.SafetyDelta})
		if err != nil {
			return domain.CursorPage[domain.InventoryStock]{}, wrapInventoryError("inventory.lowStock", err)
		}
		nextToken = encoded
	}

	return domain.CursorPage[domain.InventoryStock]{
		Items:         stocks,
		NextPageToken: nextToken,
	}, nil
}

func (r *InventoryRepository) ConfigureSafetyStock(ctx context.Context, cfg repositories.InventorySafetyStockConfig) (domain.InventoryStock, error) {
	if r == nil || r.stocks == nil {
		return domain.InventoryStock{}, errors.New("inventory repository not initialised")
	}
	sku := strings.TrimSpace(cfg.SKU)
	if sku == "" {
		return domain.InventoryStock{}, repositories.NewInventoryError(repositories.InventoryErrorUnknown, "inventory configure safety: sku is required", nil)
	}
	if cfg.SafetyStock < 0 {
		return domain.InventoryStock{}, repositories.NewInventoryError(repositories.InventoryErrorUnknown, "inventory configure safety: safety stock must be >= 0", nil)
	}
	if cfg.InitialOnHand != nil && *cfg.InitialOnHand < 0 {
		return domain.InventoryStock{}, repositories.NewInventoryError(repositories.InventoryErrorUnknown, "inventory configure safety: initial stock must be >= 0", nil)
	}
	productRef := strings.TrimSpace(cfg.ProductRef)
	now := cfg.Now.UTC()
	var updated domain.InventoryStock
	err := r.provider.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		stockRef, err := r.stocks.DocumentRef(ctx, sku)
		if err != nil {
			return err
		}
		var doc stockDocument
		snap, err := tx.Get(stockRef)
		if err != nil {
			if status.Code(err) != codes.NotFound {
				return err
			}
			doc = stockDocument{}
		} else if err := snap.DataTo(&doc); err != nil {
			return fmt.Errorf("decode inventory stock %s: %w", sku, err)
		}
		if productRef != "" {
			doc.ProductRef = productRef
		}
		doc.SKU = sku
		doc.SafetyStock = cfg.SafetyStock
		if cfg.InitialOnHand != nil {
			doc.OnHand = *cfg.InitialOnHand
		}
		doc.UpdatedAt = now
		doc.recalculate()
		if err := tx.Set(stockRef, doc); err != nil {
			return err
		}
		updated = doc.toDomain(sku)
		return nil
	})
	if err != nil {
		return domain.InventoryStock{}, wrapInventoryError("inventory.configureSafety", err)
	}
	return updated, nil
}

// Helper structures ---------------------------------------------------------

type stockDocument struct {
	SKU         string    `firestore:"sku"`
	ProductRef  string    `firestore:"productRef"`
	OnHand      int       `firestore:"onHand"`
	Reserved    int       `firestore:"reserved"`
	Available   int       `firestore:"available"`
	SafetyStock int       `firestore:"safetyStock"`
	SafetyDelta int       `firestore:"safetyDelta"`
	UpdatedAt   time.Time `firestore:"updatedAt"`
}

func (s *stockDocument) recalculate() {
	s.Available = s.OnHand - s.Reserved
	s.SafetyDelta = s.Available - s.SafetyStock
}

func (s stockDocument) toDomain(id string) domain.InventoryStock {
	return domain.InventoryStock{
		SKU:         id,
		ProductRef:  strings.TrimSpace(s.ProductRef),
		OnHand:      s.OnHand,
		Reserved:    s.Reserved,
		Available:   s.Available,
		SafetyStock: s.SafetyStock,
		SafetyDelta: s.SafetyDelta,
		UpdatedAt:   s.UpdatedAt,
	}
}

type reservationDocument struct {
	OrderRef       string                    `firestore:"orderRef"`
	UserRef        string                    `firestore:"userRef"`
	Status         string                    `firestore:"status"`
	Lines          []reservationLineDocument `firestore:"lines"`
	IdempotencyKey string                    `firestore:"idempotencyKey,omitempty"`
	Reason         string                    `firestore:"reason,omitempty"`
	ExpiresAt      time.Time                 `firestore:"expiresAt"`
	ReleasedAt     *time.Time                `firestore:"releasedAt,omitempty"`
	CommittedAt    *time.Time                `firestore:"committedAt,omitempty"`
	CreatedAt      time.Time                 `firestore:"createdAt"`
	UpdatedAt      time.Time                 `firestore:"updatedAt"`
}

type reservationLineDocument struct {
	ProductRef string `firestore:"productRef"`
	SKU        string `firestore:"sku"`
	Quantity   int    `firestore:"qty"`
}

func newReservationDocument(res domain.InventoryReservation) reservationDocument {
	lines := make([]reservationLineDocument, len(res.Lines))
	for i, line := range res.Lines {
		lines[i] = reservationLineDocument{
			ProductRef: strings.TrimSpace(line.ProductRef),
			SKU:        strings.TrimSpace(line.SKU),
			Quantity:   line.Quantity,
		}
	}
	return reservationDocument{
		OrderRef:       strings.TrimSpace(res.OrderRef),
		UserRef:        strings.TrimSpace(res.UserRef),
		Status:         strings.TrimSpace(res.Status),
		Lines:          lines,
		IdempotencyKey: strings.TrimSpace(res.IdempotencyKey),
		Reason:         strings.TrimSpace(res.Reason),
		ExpiresAt:      res.ExpiresAt.UTC(),
		ReleasedAt:     res.ReleasedAt,
		CommittedAt:    res.CommittedAt,
		CreatedAt:      res.CreatedAt.UTC(),
		UpdatedAt:      res.UpdatedAt.UTC(),
	}
}

type inventoryPageToken struct {
	SKU         string
	Available   int
	SafetyDelta int
}

func encodeInventoryPageToken(token inventoryPageToken) (string, error) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	if err := enc.Encode(token); err != nil {
		return "", fmt.Errorf("encode inventory page token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes.TrimSpace(buf.Bytes())), nil
}

func decodeInventoryPageToken(encoded string) (*inventoryPageToken, error) {
	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode inventory page token: %w", err)
	}
	var token inventoryPageToken
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("decode inventory page token json: %w", err)
	}
	return &token, nil
}

func (d reservationDocument) toDomain(id string) domain.InventoryReservation {
	lines := make([]domain.InventoryReservationLine, len(d.Lines))
	for i, line := range d.Lines {
		lines[i] = domain.InventoryReservationLine{
			ProductRef: strings.TrimSpace(line.ProductRef),
			SKU:        strings.TrimSpace(line.SKU),
			Quantity:   line.Quantity,
		}
	}
	return domain.InventoryReservation{
		ID:             id,
		OrderRef:       strings.TrimSpace(d.OrderRef),
		UserRef:        strings.TrimSpace(d.UserRef),
		Status:         strings.TrimSpace(d.Status),
		Lines:          lines,
		IdempotencyKey: strings.TrimSpace(d.IdempotencyKey),
		Reason:         strings.TrimSpace(d.Reason),
		ExpiresAt:      d.ExpiresAt,
		ReleasedAt:     d.ReleasedAt,
		CommittedAt:    d.CommittedAt,
		CreatedAt:      d.CreatedAt,
		UpdatedAt:      d.UpdatedAt,
	}
}

func decodeReservation(snap *firestore.DocumentSnapshot) (reservationDocument, error) {
	var doc reservationDocument
	if err := snap.DataTo(&doc); err != nil {
		return reservationDocument{}, fmt.Errorf("decode reservation %s: %w", snap.Ref.ID, err)
	}
	return doc, nil
}

func wrapInventoryError(op string, err error) error {
	if err == nil {
		return nil
	}
	var invErr *repositories.InventoryError
	if errors.As(err, &invErr) {
		if invErr.Op == "" {
			invErr.Op = op
		}
		return invErr
	}
	return pfirestore.WrapError(op, err)
}
