package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"

	conf "github.com/aayush0325/keyforge/internal/config"
	"github.com/aayush0325/keyforge/internal/rdb"
)

// Handshake performs the three way handshake with the master and applies the
// dataset it sends as part of the full resynchronization.
//
// It returns the connection to the master plus the reader that is positioned
// right after the RDB payload, so that the caller can keep reading the
// replication stream from there.
func Handshake(masterHost string, masterPort int, port int) (conn net.Conn, reader *bufio.Reader) {
	addr := net.JoinHostPort(masterHost, strconv.Itoa(masterPort))
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Printf("Failed to connect to master at %s: %v", addr, err)
		return nil, nil
	}

	writer := bufio.NewWriter(conn)
	reader = bufio.NewReader(conn)

	// The replies of the handshake are read right away, including the one of
	// PSYNC, which carries the replication offset the master expects us to start
	// from.
	reply := sendCommand(writer, reader, "PING", "*1\r\n$4\r\nPING\r\n")
	logRawBytes("PING", reply.raw)

	reply = sendCommand(writer, reader, "REPLCONF listening-port",
		fmt.Sprintf("*3\r\n$8\r\nREPLCONF\r\n$14\r\nlistening-port\r\n$4\r\n%d\r\n", port))
	logRawBytes("REPLCONF listening-port", reply.raw)

	reply = sendCommand(writer, reader, "REPLCONF capa", "*3\r\n$8\r\nREPLCONF\r\n$4\r\ncapa\r\n$6\r\npsync2\r\n")
	logRawBytes("REPLCONF capa", reply.raw)

	reply = sendCommand(writer, reader, "PSYNC", "*3\r\n$5\r\nPSYNC\r\n$1\r\n?\r\n$2\r\n-1\r\n")
	logRawBytes("PSYNC", reply.raw)

	// +FULLRESYNC <replid> <offset> (or +CONTINUE <replid> when the master has a
	// backlog we could resume from, which keyforge does not keep around)
	offset, err := parseFullResync(reply.body)
	if err != nil {
		log.Printf("Replica: %v", err)
		conn.Close()
		return nil, nil
	}

	// The dataset follows as a bulk string, load it before serving any client so
	// that a replica never answers with an incomplete copy.
	payload, err := receiveRdb(reader)
	if err != nil {
		log.Printf("Replica: %v", err)
		conn.Close()
		return nil, nil
	}

	if err := loadDataset(payload); err != nil {
		log.Printf("Replica: failed loading the dataset sent by the master: %v", err)
		conn.Close()
		return nil, nil
	}

	// The offset reported by the master already accounts for the payload, so
	// from here on both sides count the same bytes.
	conf.SetOffset(offset)
	log.Printf("Replica: loaded %d bytes from master, replication offset is now %d", len(payload), conf.GetOffset())

	log.Printf("Replica: Waiting for commands from master")

	return conn, reader
}

// handshakeReply is what a replica got back for one handshake command.
type handshakeReply struct {
	operation string
	raw       []byte // the bytes that were sent
	body      string // the reply line, without the trailing CRLF
}

func (r handshakeReply) String() string { return strings.TrimSpace(r.body) }

// sendCommand writes a command of the handshake and returns both what was sent
// and the reply the master answered with.
func sendCommand(writer *bufio.Writer, reader *bufio.Reader, operation, raw string) handshakeReply {
	writer.WriteString(raw)
	writer.Flush()

	reply := handshakeReply{operation: operation, raw: []byte(raw)}

	line, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("Replica sent %s, no reply: %v", operation, err)
		return reply
	}

	reply.body = strings.TrimRight(line, "\r\n")
	log.Printf("Replica sent %s, received: %s", operation, reply)

	return reply
}

// parseFullResync parses the reply to PSYNC and returns the replication offset
// the replica has to continue from.
func parseFullResync(line string) (uint64, error) {
	log.Printf("Replica: PSYNC reply %q", line)

	line = strings.TrimRight(line, "\r\n")
	if !strings.HasPrefix(line, "+") {
		return 0, fmt.Errorf("unexpected reply to PSYNC: %q", line)
	}

	fields := strings.Fields(line[1:])
	if len(fields) != 3 {
		return 0, fmt.Errorf("malformed reply to PSYNC: %q", line)
	}

	switch fields[0] {
	case "FULLRESYNC", "CONTINUE":
	default:
		return 0, fmt.Errorf("unsupported PSYNC reply: %q", line)
	}

	offset, err := strconv.ParseUint(fields[2], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("malformed replication offset in %q: %w", line, err)
	}

	conf.ReplID = fields[1]

	return offset, nil
}

// receiveRdb reads the bulk string holding the RDB payload sent by the master.
func receiveRdb(reader *bufio.Reader) ([]byte, error) {
	log.Printf("Replica: Waiting for RDB from master")

	header, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("error reading the RDB header: %w", err)
	}

	header = strings.TrimRight(header, "\r\n")
	log.Printf("Replica: RDB header (raw): %q", header)

	if !strings.HasPrefix(header, "$") {
		return nil, fmt.Errorf("expected a bulk string holding the RDB, got %q", header)
	}

	length, err := strconv.Atoi(header[1:])
	if err != nil {
		return nil, fmt.Errorf("malformed RDB bulk length %q: %w", header, err)
	}

	if length <= 0 {
		return nil, errors.New("master sent an empty RDB payload")
	}

	payload := make([]byte, length)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return nil, fmt.Errorf("error reading the RDB payload: %w", err)
	}

	// Trailing CRLF of the bulk string
	if _, err := reader.Discard(2); err != nil {
		return nil, fmt.Errorf("error reading the RDB trailer: %w", err)
	}

	log.Printf("Replica: RDB data received: %d bytes", len(payload))

	return payload, nil
}

// loadDataset applies a snapshot to the stores while no write command can run.
func loadDataset(payload []byte) error {
	rdb.LockDataset()
	defer rdb.UnlockDataset()

	return rdb.Load(payload)
}

func logRawBytes(operation string, rawBytes []byte) {
	if rawBytes != nil {
		log.Printf("[RAW_BYTES] %s: %q", operation, string(rawBytes))
	}
}
