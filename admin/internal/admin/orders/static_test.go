package orders

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStaticServiceStatusModal(t *testing.T) {
	t.Parallel()

	svc := NewStaticService()
	ctx := context.Background()

	modal, err := svc.StatusModal(ctx, "", "order-1052")
	require.NoError(t, err)
	require.Equal(t, "order-1052", modal.Order.ID)

	var selected int
	for _, option := range modal.Choices {
		if option.Selected {
			require.True(t, option.Disabled)
			selected++
		}
		if option.Disabled && !option.Selected {
			require.NotEmpty(t, option.DisabledReason)
		}
		if !option.Disabled {
			require.NotEmpty(t, option.Description)
		}
	}
	require.Equal(t, 1, selected, "exactly one status should be selected")
	require.NotEmpty(t, modal.LatestTimeline)
}

func TestStaticServiceUpdateStatusSuccess(t *testing.T) {
	t.Parallel()

	svc := NewStaticService()
	ctx := context.Background()

	initialCount := len(svc.timelines["order-1052"])
	require.NotZero(t, initialCount)

	result, err := svc.UpdateStatus(ctx, "", "order-1052", StatusUpdateRequest{
		Status:         StatusReadyToShip,
		Note:           "包装確認済み",
		NotifyCustomer: true,
		ActorEmail:     "ops@example.com",
	})
	require.NoError(t, err)
	require.Equal(t, StatusReadyToShip, result.Order.Status)
	require.Equal(t, "出荷待ち", result.Order.StatusLabel)
	require.True(t, len(result.Order.Notes) > 0)
	require.Contains(t, result.Order.Notes[0], "包装確認済み")
	require.Equal(t, initialCount+1, len(svc.timelines["order-1052"]))

	modal, err := svc.StatusModal(ctx, "", "order-1052")
	require.NoError(t, err)
	require.Equal(t, StatusReadyToShip, modal.Order.Status)
}

func TestStaticServiceUpdateStatusInvalid(t *testing.T) {
	t.Parallel()

	svc := NewStaticService()
	ctx := context.Background()

	_, err := svc.UpdateStatus(ctx, "", "order-1052", StatusUpdateRequest{
		Status: StatusPendingPayment,
	})
	require.Error(t, err)
	var transitionErr *StatusTransitionError
	require.ErrorAs(t, err, &transitionErr)
	require.Equal(t, StatusInProduction, transitionErr.From)
	require.Equal(t, StatusPendingPayment, transitionErr.To)

	modal, err := svc.StatusModal(ctx, "", "order-1052")
	require.NoError(t, err)
	require.Equal(t, StatusInProduction, modal.Order.Status)
}
