package concurrent

import "context"

func RegisterContextKeyWhenDetach(key any) {
	if key == nil {
		return
	}
	internal.contextKeys.Store(key, struct{}{})
}

func copyContextValues(to, from context.Context) {
	internal.contextKeys.Range(func(key, value any) bool {
		if v := from.Value(key); v != nil {
			to.Value(key)
		}
		return true
	})
}
