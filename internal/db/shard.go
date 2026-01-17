package db

const (
	Shards     = 16
	shardsMask = Shards - 1
)

// shardForKey returns the shard index for a given key using FNV-1a hash
// Uses zero-allocation string iteration for performance
func shardForKey(key string) int {
	const (
		offset32 = 2166136261
		prime32  = 16777619
	)

	hash := uint32(offset32)
	for i := 0; i < len(key); i++ {
		hash ^= uint32(key[i])
		hash *= prime32
	}

	return int(hash & shardsMask)
}
