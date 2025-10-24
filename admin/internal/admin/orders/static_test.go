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

func TestStaticServiceStartBulkExportCSV(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc := NewStaticService()

	job, err := svc.StartBulkExport(ctx, "", BulkExportRequest{
		Format:   ExportFormatCSV,
		OrderIDs: []string{"order-1052", "order-1051"},
	})
	require.NoError(t, err)
	require.Equal(t, ExportFormatCSV, job.Format)
	require.NotEmpty(t, job.ID)
	require.Equal(t, 2, job.TotalOrders)
	require.Zero(t, job.ProcessedOrders)
	require.NotEmpty(t, job.Fields)
	require.Contains(t, job.Fields, "order_number")
	require.NotContains(t, job.Fields, "internal_notes")

	jobs, err := svc.ListExportJobs(ctx, "")
	require.NoError(t, err)
	require.NotEmpty(t, jobs)

	var found bool
	for _, listed := range jobs {
		if listed.ID == job.ID {
			found = true
			break
		}
	}
	require.True(t, found, "started export job should be present in listings")
}

func TestStaticServiceExportJobLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc := NewStaticService()

	job, err := svc.StartBulkExport(ctx, "", BulkExportRequest{
		Format: ExportFormatPDF,
	})
	require.NoError(t, err)

	status1, err := svc.ExportJobStatus(ctx, "", job.ID)
	require.NoError(t, err)
	require.Equal(t, job.ID, status1.Job.ID)
	require.Equal(t, ExportFormatPDF, status1.Job.Format)

	if status1.Done {
		// Some datasets may complete immediately when only a single order matches.
		require.Equal(t, 100, status1.Job.Progress)
		require.NotEmpty(t, status1.Job.DownloadURL)
		return
	}

	require.False(t, status1.Done)
	require.Greater(t, status1.Job.Progress, 0)
	require.Empty(t, status1.Job.DownloadURL)

	status2, err := svc.ExportJobStatus(ctx, "", job.ID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, status2.Job.Progress, status1.Job.Progress)

	status3, err := svc.ExportJobStatus(ctx, "", job.ID)
	require.NoError(t, err)
	require.True(t, status3.Done)
	require.Equal(t, 100, status3.Job.Progress)
	require.Equal(t, status3.Job.TotalOrders, status3.Job.ProcessedOrders)
	require.NotEmpty(t, status3.Job.DownloadURL)
}

func TestStaticServiceStartBulkExportValidations(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc := NewStaticService()

	job, err := svc.StartBulkExport(ctx, "", BulkExportRequest{
		Format: ExportFormat("xlsx"),
	})
	require.NoError(t, err)
	require.Equal(t, ExportFormatCSV, job.Format)

	_, err = svc.StartBulkExport(ctx, "", BulkExportRequest{
		Format:   ExportFormatCSV,
		OrderIDs: []string{"non-existent"},
	})
	require.ErrorIs(t, err, ErrExportNoOrders)
}
