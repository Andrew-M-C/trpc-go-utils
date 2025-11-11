package concurrent

import "sync"

var internal = struct {
	contextKeys sync.Map
}{
	contextKeys: sync.Map{},
}
