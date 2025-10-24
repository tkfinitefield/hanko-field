package repositories

import "errors"

var (
	// ErrAssetNotReady indicates the asset has not completed post-processing and cannot be downloaded.
	ErrAssetNotReady = errors.New("asset repository: asset not ready")
	// ErrAssetSoftDeleted indicates the asset has been soft deleted and is no longer accessible.
	ErrAssetSoftDeleted = errors.New("asset repository: asset soft deleted")
)
