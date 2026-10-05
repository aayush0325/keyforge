package commands

import (
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	conf "github.com/aayush0325/keyforge/internal/config"
	"github.com/aayush0325/keyforge/internal/pubsub"
	"github.com/aayush0325/keyforge/internal/rdb"
	"github.com/aayush0325/keyforge/internal/resp"
)

// lastSave holds the unix timestamp of the last successful SAVE/BGSAVE.
var lastSave atomic.Int64

// RDBPath returns the location of the RDB file, dir and dbfilename as
// configured through the flags or CONFIG SET.
func RDBPath() string {
	return filepath.Join(conf.Dir, conf.DBFilename)
}

// withDatasetLock runs fn while no write command can interleave, which is what
// every whole dataset operation (SAVE, BGSAVE, DEBUG RELOAD, PSYNC) needs.
//
// The lock is not reentrant and EXEC of a transaction already holds it, so
// inside a transaction fn runs without taking it again.
func withDatasetLock(conn *pubsub.Connection, fn func() error) error {
	if conn != nil && conn.IsTransactionRunning {
		return fn()
	}

	rdb.LockDataset()
	defer rdb.UnlockDataset()

	return fn()
}

// save writes a snapshot of the dataset to the configured RDB file.
func save() error {
	return rdb.DumpToDisk(RDBPath())
}

// SaveOnShutdown writes a final snapshot, used by the signal handler of main.
func SaveOnShutdown() error {
	rdb.LockDataset()
	defer rdb.UnlockDataset()

	return save()
}

// saveCommand implements the SAVE command, it blocks until the snapshot hit the
// disk.
func saveCommand(args *resp.Array, conn *pubsub.Connection) {
	if !requireExactArgs(args, 1, "save", conn) {
		return
	}

	if err := withDatasetLock(conn, save); err != nil {
		log.Printf("save: failed dumping the dataset: %v", err)
		conn.Write(&resp.SimpleError{Val: []byte("ERR " + err.Error())})
		return
	}

	lastSave.Store(time.Now().Unix())

	conn.Write(&resp.SimpleString{Val: []byte("OK")})
}

// bgsave implements the BGSAVE command, the snapshot is taken in the
// background while the server keeps serving clients.
func bgsave(args *resp.Array, conn *pubsub.Connection) {
	if !requireExactArgs(args, 1, "bgsave", conn) {
		return
	}

	conn.Write(&resp.SimpleString{Val: []byte("Background saving started")})

	go func() {
		// Give the reply a chance to reach the client first, Redis reports the
		// background save through a separate message.
		time.Sleep(10 * time.Millisecond)

		// The lock is taken here rather than passed in: a transaction may have
		// released it already, and waiting for it here keeps the client free.
		if err := SaveOnShutdown(); err != nil {
			log.Printf("bgsave: failed dumping the dataset: %v", err)
			return
		}

		lastSave.Store(time.Now().Unix())
		log.Printf("bgsave: background save finished")
	}()
}

// lastsaveCommand implements the LASTSAVE command.
func lastsaveCommand(args *resp.Array, conn *pubsub.Connection) {
	if !requireExactArgs(args, 1, "lastsave", conn) {
		return
	}

	conn.Write(&resp.Integer{Val: lastSave.Load()})
}

// debug implements DEBUG, keyforge only supports DEBUG RELOAD which reloads the
// dataset from the RDB file on disk. It is handy to verify that a dump written
// by SAVE can be read back.
func debug(args *resp.Array, conn *pubsub.Connection) {
	if !requireMinArgs(args, 2, "debug", conn) {
		return
	}

	subCmd, ok := getBulkArg(args, 1, "debug", conn)
	if !ok {
		return
	}

	if string(subCmd.Str) != "reload" {
		conn.Write(&resp.SimpleError{Val: []byte("ERR Unknown DEBUG subcommand or wrong number of arguments for '" + string(subCmd.Str) + "'")})
		return
	}

	payload, err := os.ReadFile(RDBPath())
	if err != nil {
		conn.Write(&resp.SimpleError{Val: []byte("ERR " + err.Error())})
		return
	}

	// Reloading replaces the whole dataset, so it needs the same lock a dump does
	if err := withDatasetLock(conn, func() error { return rdb.Load(payload) }); err != nil {
		conn.Write(&resp.SimpleError{Val: []byte("ERR " + err.Error())})
		return
	}

	conn.Write(&resp.SimpleString{Val: []byte("OK")})
}
