package utils

import (
	"github.com/codecrafters-io/redis-starter-go/internal/db"
	"github.com/codecrafters-io/redis-starter-go/internal/pubsub"
	"github.com/codecrafters-io/redis-starter-go/internal/streams"
)

func GlobalInitFunction() {
	db.InitKVStore()
	pubsub.InitPubSub()
	streams.InitStreamGlobalInstance()
}
