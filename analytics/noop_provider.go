package analytics

import "context"

// NoopProvider is a no-operation implementation of the Analytics interface.
// It's used when analytics is disabled or as a placeholder.
type NoopProvider struct{}

// NewNoopProvider creates a new no-op analytics provider.
func NewNoopProvider() *NoopProvider {
	return &NoopProvider{}
}

// TrackEvent implements the Analytics interface but does nothing.
func (n *NoopProvider) TrackEvent(ctx context.Context, event Event) error {
	return nil
}

// Flush implements the Analytics interface but does nothing.
func (n *NoopProvider) Flush(ctx context.Context) error {
	return nil
}

// Close implements the Analytics interface but does nothing.
func (n *NoopProvider) Close() error {
	return nil
}
