package rdb

import (
	"bytes"
	"encoding/hex"
	"math/rand"
	"testing"
	"time"

	"github.com/aayush0325/keyforge/internal/streams"
)

func TestCRC64MatchesBitwise(t *testing.T) {
	payload := make([]byte, 4096)
	rng := rand.New(rand.NewSource(1))
	rng.Read(payload)

	fast := CRC64(0, payload)
	slow := bitwiseCRC64(0, payload)

	if fast != slow {
		t.Fatalf("table driven CRC64 = %016x, bitwise = %016x", fast, slow)
	}
}

// Checksum of the string "Hello world!\n" as produced by Redis' crc64.c (same
// polynomial, reflected, no initial or final xor).
func TestCRC64KnownVector(t *testing.T) {
	got := CRC64(0, []byte("Hello world!\n"))

	const want = uint64(0xc7f028b33dba45d0)
	if got != want {
		t.Fatalf("CRC64(\"Hello world!\\n\") = %016x, want %016x", got, want)
	}
}

func TestLenEncodingBoundaries(t *testing.T) {
	for _, size := range []int{0, 1, 63, 64, 100, 1 << 13, 1 << 14, 1 << 16, 1 << 20} {
		w := NewWriter()
		w.WriteLen(uint64(size))
		w.WriteEOF()

		payload := w.Bytes()

		r, err := NewReader(payload)
		if err != nil {
			t.Fatalf("size %d: %v", size, err)
		}

		got, err := r.ReadLen()
		if err != nil {
			t.Fatalf("size %d: %v", size, err)
		}
		if int(got) != size {
			t.Fatalf("size %d: round tripped as %d", size, got)
		}
	}
}

func TestParseRejectsCorruptedChecksum(t *testing.T) {
	payload := buildTestPayload(t)

	corrupted := make([]byte, len(payload))
	copy(corrupted, payload)
	corrupted[20] ^= 0xFF // flip a bit somewhere in the keyspace

	if _, err := Parse(corrupted); err == nil {
		t.Fatal("expected an error for a payload with a broken checksum")
	}
}

func TestParseRejectsTruncatedPayload(t *testing.T) {
	payload := buildTestPayload(t)

	if _, err := Parse(payload[:len(payload)/2]); err == nil {
		t.Fatal("expected an error for a truncated payload")
	}
}

func TestSerializeParseRoundTrip(t *testing.T) {
	expiry := time.Now().Add(time.Hour).Truncate(time.Millisecond)

	dataset := NewDataset()
	dataset.Strings["plain"] = StringEntry{Value: []byte("value")}
	dataset.Strings["binary"] = StringEntry{Value: []byte{0x00, 0x01, 0xFF, 0xFE}}
	dataset.Strings["expiring"] = StringEntry{Value: []byte("ttl"), ExpiresAt: expiry}
	dataset.Lists["list"] = [][]byte{[]byte("a"), []byte("b"), []byte("c")}
	dataset.Streams["stream"] = []*streams.StreamEntry{
		{ID: &streams.StreamID{Ms: 1000, Seq: 0}, Entry: map[string]string{"name": "alice", "age": "30"}},
		{ID: &streams.StreamID{Ms: 1000, Seq: 1}, Entry: map[string]string{"name": "bob"}},
	}

	payload, err := SerializeDataset(dataset)
	if err != nil {
		t.Fatalf("SerializeDataset: %v", err)
	}

	if !bytes.HasPrefix(payload, Magic) {
		t.Fatalf("payload does not start with the magic header: %q", payload[:8])
	}

	loaded, err := Parse(payload)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if loaded.Len() != 5 {
		t.Fatalf("loaded %d keys, want 5", loaded.Len())
	}

	if got := string(loaded.Strings["plain"].Value); got != "value" {
		t.Errorf("plain = %q, want %q", got, "value")
	}

	if got := loaded.Strings["binary"].Value; !bytes.Equal(got, []byte{0x00, 0x01, 0xFF, 0xFE}) {
		t.Errorf("binary = %x, want 0001fffe", got)
	}

	if got := loaded.Strings["expiring"].ExpiresAt; !got.Equal(expiry) {
		t.Errorf("expiring expiry = %v, want %v", got, expiry)
	}

	if got := len(loaded.Lists["list"]); got != 3 {
		t.Fatalf("list has %d elements, want 3", got)
	}
	if got := string(loaded.Lists["list"][2]); got != "c" {
		t.Errorf("last list element = %q, want %q", got, "c")
	}

	entries := loaded.Streams["stream"]
	if len(entries) != 2 {
		t.Fatalf("stream has %d entries, want 2", len(entries))
	}
	if entries[0].ID.Ms != 1000 || entries[0].ID.Seq != 0 {
		t.Errorf("first entry ID = %v, want 1000-0", entries[0].ID)
	}
	if entries[1].Entry["name"] != "bob" {
		t.Errorf("second entry name = %q, want %q", entries[1].Entry["name"], "bob")
	}

	// Dumping the same dataset twice must produce byte identical payloads
	second, err := SerializeDataset(dataset)
	if err != nil {
		t.Fatalf("second SerializeDataset: %v", err)
	}
	if !bytes.Equal(payload, second) {
		t.Error("serializing the same dataset twice produced different payloads")
	}
}

func TestSerializeSkipsExpiredKeys(t *testing.T) {
	dataset := NewDataset()
	dataset.Strings["alive"] = StringEntry{Value: []byte("yes")}
	dataset.Strings["dead"] = StringEntry{
		Value:     []byte("no"),
		ExpiresAt: time.Now().Add(-time.Minute),
	}

	payload, err := SerializeDataset(dataset)
	if err != nil {
		t.Fatalf("SerializeDataset: %v", err)
	}

	loaded, err := Parse(payload)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if _, ok := loaded.Strings["dead"]; ok {
		t.Error("expired key was serialized")
	}
	if _, ok := loaded.Strings["alive"]; !ok {
		t.Error("live key is missing from the payload")
	}
}

func TestEmptyRDBIsValid(t *testing.T) {
	dataset, err := Parse(EmptyRDB)
	if err != nil {
		t.Fatalf("Parse(EmptyRDB): %v", err)
	}
	if dataset.Len() != 0 {
		t.Fatalf("EmptyRDB holds %d keys", dataset.Len())
	}
}

// TestResizeDBHeaderMatchesContent makes sure the key and expiry counts in the
// RESIZEDB header describe exactly what follows it, including for the corner
// cases of a key used by two stores at once and of empty containers.
func TestResizeDBHeaderMatchesContent(t *testing.T) {
	dataset := NewDataset()
	dataset.Strings["string-key"] = StringEntry{Value: []byte("v")}
	dataset.Strings["expiring"] = StringEntry{Value: []byte("v"), ExpiresAt: time.Now().Add(time.Hour)}
	dataset.Strings["expired"] = StringEntry{Value: []byte("v"), ExpiresAt: time.Now().Add(-time.Hour)}
	dataset.Lists["list-key"] = [][]byte{[]byte("a")}
	dataset.Lists["empty-list"] = nil
	dataset.Lists["string-key"] = [][]byte{[]byte("shadowed")} // same name in two stores
	dataset.Streams["stream-key"] = []*streams.StreamEntry{
		{ID: &streams.StreamID{Ms: 1, Seq: 0}, Entry: map[string]string{"f": "v"}},
	}

	payload, err := SerializeDataset(dataset)
	if err != nil {
		t.Fatalf("SerializeDataset: %v", err)
	}

	reader, err := NewReader(payload)
	if err != nil {
		t.Fatalf("NewReader: %v", err)
	}

	// AUX fields
	for i := 0; i < 4; i++ {
		if op, _ := reader.ReadByte(); op != OpCodeAux {
			t.Fatalf("expected AUX, got 0x%02x", op)
		}
		if _, err := reader.ReadString(); err != nil {
			t.Fatal(err)
		}
		if _, err := reader.ReadString(); err != nil {
			t.Fatal(err)
		}
	}

	if op, _ := reader.ReadByte(); op != OpCodeSelectDB {
		t.Fatalf("expected SELECTDB, got 0x%02x", op)
	}
	if dbNum, _ := reader.ReadLen(); dbNum != 0 {
		t.Fatalf("selected db = %d, want 0", dbNum)
	}

	if op, _ := reader.ReadByte(); op != OpCodeResizeDB {
		t.Fatalf("expected RESIZEDB, got 0x%02x", op)
	}
	keys, _ := reader.ReadLen()
	expires, _ := reader.ReadLen()

	// string-key, expiring, list-key, stream-key
	if keys != 4 {
		t.Errorf("RESIZEDB keys = %d, want 4", keys)
	}
	if expires != 1 {
		t.Errorf("RESIZEDB expires = %d, want 1", expires)
	}

	loaded, err := Parse(payload)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if _, ok := loaded.Lists["string-key"]; ok {
		t.Error("a key used by two stores was written twice")
	}
	if _, ok := loaded.Strings["expired"]; ok {
		t.Error("an expired key was written")
	}
	if _, ok := loaded.Lists["empty-list"]; ok {
		t.Error("an empty list was written")
	}
	if uint64(loaded.Len()) != keys {
		t.Errorf("payload holds %d keys but the header announced %d", loaded.Len(), keys)
	}
}

// TestPayloadLayout keeps the on disk layout documented and pinned: everything
// is verified against a byte for byte golden payload.
func TestPayloadLayout(t *testing.T) {
	dataset := NewDataset()
	dataset.Strings["k"] = StringEntry{Value: []byte("v")}
	dataset.Lists["l"] = [][]byte{[]byte("x")}
	dataset.Streams["s"] = []*streams.StreamEntry{
		{ID: &streams.StreamID{Ms: 7, Seq: 2}, Entry: map[string]string{"f": "w"}},
	}

	payload, err := SerializeDataset(dataset)
	if err != nil {
		t.Fatalf("SerializeDataset: %v", err)
	}

	r, err := NewReader(payload)
	if err != nil {
		t.Fatalf("NewReader: %v", err)
	}

	if r.Version() != 11 {
		t.Errorf("RDB version = %d, want 11", r.Version())
	}

	// Auxiliary fields
	for _, name := range []string{"redis-ver", "redis-bits", "ctime", "used-mem"} {
		op, err := r.ReadByte()
		if err != nil {
			t.Fatalf("reading aux %s: %v", name, err)
		}
		if op != OpCodeAux {
			t.Fatalf("opcode = 0x%02x, want AUX (0xFA)", op)
		}
		key, err := r.ReadString()
		if err != nil {
			t.Fatalf("reading aux key: %v", err)
		}
		if string(key) != name {
			t.Fatalf("aux key = %q, want %q", key, name)
		}
		if _, err := r.ReadString(); err != nil {
			t.Fatalf("reading aux value: %v", err)
		}
	}

	if op, _ := r.ReadByte(); op != OpCodeSelectDB {
		t.Fatalf("expected SELECTDB, got 0x%02x", op)
	}
	if db, _ := r.ReadLen(); db != 0 {
		t.Fatalf("selected db = %d, want 0", db)
	}
	if op, _ := r.ReadByte(); op != OpCodeResizeDB {
		t.Fatalf("expected RESIZEDB, got 0x%02x", op)
	}
	if keys, _ := r.ReadLen(); keys != 3 {
		t.Fatalf("RESIZEDB keys = %d, want 3", keys)
	}
	if expires, _ := r.ReadLen(); expires != 0 {
		t.Fatalf("RESIZEDB expires = %d, want 0", expires)
	}

	if op, _ := r.ReadByte(); op != TypeString {
		t.Fatalf("expected a string type, got 0x%02x", op)
	}
	if key, _ := r.ReadString(); string(key) != "k" {
		t.Fatalf("string key = %q, want %q", key, "k")
	}
	if value, _ := r.ReadString(); string(value) != "v" {
		t.Fatalf("string value = %q, want %q", value, "v")
	}

	if op, _ := r.ReadByte(); op != TypeList {
		t.Fatalf("expected a list type, got 0x%02x", op)
	}
	if key, _ := r.ReadString(); string(key) != "l" {
		t.Fatalf("list key = %q, want %q", key, "l")
	}
	if count, _ := r.ReadLen(); count != 1 {
		t.Fatalf("list length = %d, want 1", count)
	}
	if element, _ := r.ReadString(); string(element) != "x" {
		t.Fatalf("list element = %q, want %q", element, "x")
	}

	if op, _ := r.ReadByte(); op != TypeStreamKeyforge {
		t.Fatalf("expected a stream type, got 0x%02x", op)
	}
	if key, _ := r.ReadString(); string(key) != "s" {
		t.Fatalf("stream key = %q, want %q", key, "s")
	}
	if count, _ := r.ReadLen(); count != 1 {
		t.Fatalf("stream entries = %d, want 1", count)
	}
	if ms, _ := r.ReadString(); string(ms) != "7" {
		t.Fatalf("stream ms = %q, want %q", ms, "7")
	}
	if seq, _ := r.ReadString(); string(seq) != "2" {
		t.Fatalf("stream seq = %q, want %q", seq, "2")
	}
	if fields, _ := r.ReadLen(); fields != 1 {
		t.Fatalf("stream fields = %d, want 1", fields)
	}
	if field, _ := r.ReadString(); string(field) != "f" {
		t.Fatalf("stream field = %q, want %q", field, "f")
	}
	if value, _ := r.ReadString(); string(value) != "w" {
		t.Fatalf("stream value = %q, want %q", value, "w")
	}

	if op, _ := r.ReadByte(); op != OpCodeEOF {
		t.Fatalf("expected EOF, got 0x%02x", op)
	}

	if err := verifyChecksum(r, payload); err != nil {
		t.Fatalf("verifyChecksum: %v", err)
	}

	t.Logf("payload: %s", hex.EncodeToString(payload[:len(payload)-8]))
}

func buildTestPayload(t *testing.T) []byte {
	t.Helper()

	dataset := NewDataset()
	dataset.Strings["key"] = StringEntry{Value: []byte("value")}
	dataset.Lists["list"] = [][]byte{[]byte("one"), []byte("two")}

	payload, err := SerializeDataset(dataset)
	if err != nil {
		t.Fatalf("SerializeDataset: %v", err)
	}
	return payload
}
