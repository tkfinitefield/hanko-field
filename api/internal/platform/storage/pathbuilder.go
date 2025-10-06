package storage

import (
	"fmt"
	"strings"
	"sync"
)

// AssetPurpose captures high-level intent for storage layout decisions.
type AssetPurpose string

const (
	PurposeDesignMaster AssetPurpose = "design-master"
	PurposePreview      AssetPurpose = "preview"
	PurposeReceipt      AssetPurpose = "receipt"
)

// PathParams provide required identifiers to compose storage object keys.
type PathParams struct {
	DesignID      string
	UploadID      string
	VersionID     string
	OrderID       string
	InvoiceNumber string
	FileName      string
}

// PathBuilder composes the object path for a given asset purpose.
type PathBuilder func(PathParams) (string, error)

var (
	pathBuilders = map[AssetPurpose]PathBuilder{
		PurposeDesignMaster: buildDesignMasterPath,
		PurposePreview:      buildPreviewPath,
		PurposeReceipt:      buildReceiptPath,
	}
	pathBuildersMu sync.RWMutex
)

// RegisterPathBuilder overrides or registers a builder for a specific purpose.
func RegisterPathBuilder(purpose AssetPurpose, builder PathBuilder) {
	pathBuildersMu.Lock()
	defer pathBuildersMu.Unlock()
	if builder == nil {
		delete(pathBuilders, purpose)
		return
	}
	pathBuilders[purpose] = builder
}

// BuildObjectPath resolves the storage object path for the given purpose.
func BuildObjectPath(purpose AssetPurpose, params PathParams) (string, error) {
	pathBuildersMu.RLock()
	builder, ok := pathBuilders[purpose]
	pathBuildersMu.RUnlock()
	if !ok {
		return "", fmt.Errorf("storage: unsupported asset purpose %q", purpose)
	}
	return builder(params)
}

func buildDesignMasterPath(params PathParams) (string, error) {
	designID, err := validateSegment("designID", params.DesignID)
	if err != nil {
		return "", err
	}
	uploadID, err := validateSegment("uploadID", params.UploadID)
	if err != nil {
		return "", err
	}
	fileName, err := validateFileName(params.FileName)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("assets/designs/%s/sources/%s/%s", designID, uploadID, fileName), nil
}

func buildPreviewPath(params PathParams) (string, error) {
	designID, err := validateSegment("designID", params.DesignID)
	if err != nil {
		return "", err
	}
	versionID, err := validateSegment("versionID", params.VersionID)
	if err != nil {
		return "", err
	}
	fileName, err := validateFileName(params.FileName)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("assets/designs/%s/previews/%s/%s", designID, versionID, fileName), nil
}

func buildReceiptPath(params PathParams) (string, error) {
	orderID, err := validateSegment("orderID", params.OrderID)
	if err != nil {
		return "", err
	}
	name := strings.TrimSpace(params.FileName)
	if name == "" && params.InvoiceNumber != "" {
		name = fmt.Sprintf("%s.pdf", strings.TrimSpace(params.InvoiceNumber))
	}
	fileName, err := validateFileName(name)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("assets/orders/%s/invoices/%s", orderID, fileName), nil
}

func validateSegment(name, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("storage: %s is required", name)
	}
	if strings.ContainsAny(value, "/\\") {
		return "", fmt.Errorf("storage: %s contains invalid path characters", name)
	}
	if strings.Contains(value, "..") {
		return "", fmt.Errorf("storage: %s contains invalid traversal sequence", name)
	}
	return value, nil
}

func validateFileName(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("storage: fileName is required")
	}
	if strings.ContainsAny(value, "/\\") {
		return "", fmt.Errorf("storage: fileName contains invalid path characters")
	}
	if strings.Contains(value, "..") {
		return "", fmt.Errorf("storage: fileName contains invalid traversal sequence")
	}
	return value, nil
}
