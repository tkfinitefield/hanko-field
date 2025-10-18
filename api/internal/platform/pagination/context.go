package pagination

import "context"

type contextKey string

const paramsContextKey contextKey = "github.com/hanko-field/api/internal/platform/pagination/params"

// WithParams stores the parsed pagination parameters on the context.
func WithParams(ctx context.Context, params Params) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, paramsContextKey, params)
}

// FromContext retrieves pagination parameters when they were previously attached via WithParams.
func FromContext(ctx context.Context) (Params, bool) {
	if ctx == nil {
		return Params{}, false
	}
	params, ok := ctx.Value(paramsContextKey).(Params)
	return params, ok
}

// FromContextOrDefault fetches pagination parameters or returns defaults when absent.
func FromContextOrDefault(ctx context.Context) Params {
	params, ok := FromContext(ctx)
	if !ok {
		return Params{PageSize: DefaultPageSize}
	}
	if params.PageSize <= 0 {
		params.PageSize = DefaultPageSize
	}
	return params
}
