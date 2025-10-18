package storage

import "testing"

func TestBuildDesignMasterPath(t *testing.T) {
	path, err := BuildObjectPath(PurposeDesignMaster, PathParams{
		DesignID: "design123",
		UploadID: "upload789",
		FileName: "source.png",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "assets/designs/design123/sources/upload789/source.png"
	if path != expected {
		t.Fatalf("expected %s, got %s", expected, path)
	}
}

func TestBuildReceiptPathUsesInvoiceNumber(t *testing.T) {
	path, err := BuildObjectPath(PurposeReceipt, PathParams{
		OrderID:       "order123",
		InvoiceNumber: "INV-2025-001",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "assets/orders/order123/invoices/INV-2025-001.pdf"
	if path != expected {
		t.Fatalf("expected %s, got %s", expected, path)
	}
}

func TestBuildObjectPathRejectsInvalidSegment(t *testing.T) {
	_, err := BuildObjectPath(PurposeDesignMaster, PathParams{
		DesignID: "../bad",
		UploadID: "upload",
		FileName: "file.png",
	})
	if err == nil {
		t.Fatalf("expected error for invalid segment")
	}
}
