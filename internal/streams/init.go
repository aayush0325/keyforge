package streams

import "sync"

var (
	Global GlobalInstance
	once   sync.Once
)

func InitStreamGlobalInstance() {
	once.Do(func() {
		Global = GlobalInstance{
			KV: make(map[string]*Stream),
		}
	})
}
