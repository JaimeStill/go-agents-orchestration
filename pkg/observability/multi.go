package observability

import "context"

type MultiObserver struct {
	observers []Observer
}

func NewMultiObserver(observers ...Observer) *MultiObserver {
	filtered := make([]Observer, 0, len(observers))
	for _, obs := range observers {
		if obs != nil {
			filtered = append(filtered, obs)
		}
	}
	return &MultiObserver{observers: filtered}
}

func (m *MultiObserver) OnEvent(ctx context.Context, event Event) {
	for _, obs := range m.observers {
		obs.OnEvent(ctx, event)
	}
}
