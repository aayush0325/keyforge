package rdb

import (
	"bytes"
	"testing"
	"time"

	"github.com/aayush0325/keyforge/internal/db"
	"github.com/aayush0325/keyforge/internal/streams"
	"github.com/aayush0325/keyforge/internal/utils"
)

func TestMain(m *testing.M) {
	// The stores have to be up before anything is written into them
	utils.GlobalInitFunction()
	m.Run()
}

func initStores(t *testing.T) {
	t.Helper()

	rdb := NewDataset()
	Apply(rdb)
}

// TestLiveDatasetRoundTrip dumps the content of the running stores, loads the
// payload back and makes sure the second dump is byte identical to the first
// one, which proves nothing was lost or reordered on the way.
func TestLiveDatasetRoundTrip(t *testing.T) {
	initStores(t)

	expiry := time.Now().Add(time.Hour).Truncate(time.Millisecond)

	db.SetEntry("plain", []byte("value"), time.Time{})
	db.SetEntry("with-ttl", []byte("temporary"), expiry)
	db.SetEntry("binary", []byte{0x00, 0xFF, 0x10, 0x0D, 0x0A}, time.Time{})

	db.ResetList("list", []string{"a", "b", "c"})
	db.ResetList("empty-list-never-written", nil)

	streams.RestoreStream("stream", []*streams.StreamEntry{
		{ID: &streams.StreamID{Ms: 1000, Seq: 0}, Entry: map[string]string{"name": "alice"}},
		{ID: &streams.StreamID{Ms: 1000, Seq: 1}, Entry: map[string]string{"name": "bob", "age": "30"}},
		{ID: &streams.StreamID{Ms: 2000, Seq: 0}, Entry: map[string]string{"name": "carol"}},
	})

	first, err := Serialize()
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}

	// Loading replaces the content of the stores with the payload ...
	if err := Load(first); err != nil {
		t.Fatalf("Load: %v", err)
	}

	// ... and dumping it again has to yield the very same bytes.
	second, err := Serialize()
	if err != nil {
		t.Fatalf("Serialize after Load: %v", err)
	}

	if !bytes.Equal(first, second) {
		t.Fatalf("round trip through the stores changed the payload:\nfirst:  %x\nsecond: %x", first, second)
	}

	dataset, err := Parse(second)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if got := string(dataset.Strings["plain"].Value); got != "value" {
		t.Errorf("plain = %q, want %q", got, "value")
	}

	if got := dataset.Strings["with-ttl"].ExpiresAt; !got.Equal(expiry) {
		t.Errorf("with-ttl expiry = %v, want %v", got, expiry)
	}

	if got := len(dataset.Lists["list"]); got != 3 {
		t.Errorf("list has %d elements, want 3", got)
	}

	entries := dataset.Streams["stream"]
	if len(entries) != 3 {
		t.Fatalf("stream has %d entries, want 3", len(entries))
	}
	if entries[2].Entry["name"] != "carol" || entries[2].ID.Ms != 2000 {
		t.Errorf("last stream entry = %+v, want 2000-0 carol", entries[2])
	}

	initStores(t)
}

func TestLoadReplacesPreviousContent(t *testing.T) {
	db.SetEntry("stale", []byte("value"), time.Time{})
	db.ResetList("stale-list", []string{"x"})

	dataset := NewDataset()
	dataset.Strings["fresh"] = StringEntry{Value: []byte("value")}

	Load(SerializeDatasetOrFail(t, dataset))

	snapshot := db.SnapshotEntries()
	if _, ok := snapshot["stale"]; ok {
		t.Error("stale string key survived the load")
	}
	if _, ok := snapshot["fresh"]; !ok {
		t.Error("fresh string key is missing after the load")
	}

	if lists := db.SnapshotLists(); len(lists["stale-list"]) != 0 {
		t.Error("stale list survived the load")
	}

	initStores(t)
}

func TestLoadSkipsExpiredKeys(t *testing.T) {
	dataset := NewDataset()
	dataset.Strings["dead"] = StringEntry{
		Value:     []byte("value"),
		ExpiresAt: time.Now().Add(-time.Hour),
	}
	dataset.Strings["alive"] = StringEntry{Value: []byte("value")}

	Load(SerializeDatasetOrFail(t, dataset))

	snapshot := db.SnapshotEntries()
	if _, ok := snapshot["dead"]; ok {
		t.Error("an already expired key was loaded")
	}
	if _, ok := snapshot["alive"]; !ok {
		t.Error("a live key was dropped")
	}

	initStores(t)
}

// SerializeDatasetOrFail is a small helper keeping the tests readable.
func SerializeDatasetOrFail(t *testing.T, dataset *Dataset) []byte {
	t.Helper()

	payload, err := SerializeDataset(dataset)
	if err != nil {
		t.Fatalf("SerializeDataset: %v", err)
	}
	return payload
}
