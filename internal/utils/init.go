package utils

import (
	"github.com/aayush0325/keyforge/internal/db"
	"github.com/aayush0325/keyforge/internal/pubsub"
	"github.com/aayush0325/keyforge/internal/streams"
)

func GlobalInitFunction() {
	db.InitKVStore()
	pubsub.InitPubSub()
	streams.InitStreamGlobalInstance()
}
