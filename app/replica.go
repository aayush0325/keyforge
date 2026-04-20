package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
)

func Handshake(masterHost string, masterPort int, port int) (conn net.Conn, reader *bufio.Reader) {
	addr := fmt.Sprintf("%s:%d", masterHost, masterPort)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Printf("Failed to connect to master at %s: %v", addr, err)
		return nil, nil
	}

	writer := bufio.NewWriter(conn)
	reader = bufio.NewReader(conn)

	rawBytes := sendPing(writer, reader, port)
	logRawBytes("PING", rawBytes)

	rawBytes = sendReplconfListeningPort(writer, reader, port)
	logRawBytes("REPLCONF listening-port", rawBytes)

	rawBytes = sendReplconfCapa(writer, reader)
	logRawBytes("REPLCONF capa", rawBytes)

	rawBytes = sendPsync(writer, reader)
	logRawBytes("PSYNC", rawBytes)

	rdbLen, rawBytes := receiveRdbHeader(reader)
	logRawBytes("RDB header", rawBytes)

	if rawBytes != nil {
		receiveRdbData(reader, rdbLen)
	}

	log.Printf("Replica: Waiting for commands from master")

	return conn, reader
}

func sendPing(writer *bufio.Writer, reader *bufio.Reader, port int) []byte {
	rawBytes := []byte("*1\r\n$4\r\nPING\r\n")
	writer.Write(rawBytes)
	writer.Flush()

	r, _ := reader.ReadString('\n')
	log.Printf("Replica sent PING, received: %s", strings.TrimSpace(r))
	return rawBytes
}

func sendReplconfListeningPort(writer *bufio.Writer, reader *bufio.Reader, port int) []byte {
	rawBytes := []byte(fmt.Sprintf("*3\r\n$8\r\nREPLCONF\r\n$14\r\nlistening-port\r\n$4\r\n%d\r\n", port))
	writer.Write(rawBytes)
	writer.Flush()

	r, _ := reader.ReadString('\n')
	log.Printf("Replica sent REPLCONF listening-port, received: %s", strings.TrimSpace(r))
	return rawBytes
}

func sendReplconfCapa(writer *bufio.Writer, reader *bufio.Reader) []byte {
	rawBytes := []byte("*3\r\n$8\r\nREPLCONF\r\n$4\r\ncapa\r\n$6\r\npsync2\r\n")
	writer.Write(rawBytes)
	writer.Flush()

	r, _ := reader.ReadString('\n')
	log.Printf("Replica sent REPLCONF capa, received: %s", strings.TrimSpace(r))
	return rawBytes
}

func sendPsync(writer *bufio.Writer, reader *bufio.Reader) []byte {
	rawBytes := []byte("*3\r\n$5\r\nPSYNC\r\n$1\r\n?\r\n$2\r\n-1\r\n")
	writer.Write(rawBytes)
	writer.Flush()

	r, _ := reader.ReadString('\n')
	log.Printf("Replica sent PSYNC, received: %s", strings.TrimSpace(r))
	return rawBytes
}

func receiveRdbHeader(reader *bufio.Reader) (int, []byte) {
	log.Printf("Replica: Waiting for RDB from master")

	rdbHeader, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("Replica: Error reading RDB header: %v", err)
		return 0, nil
	}

	rdbHeader = strings.TrimSpace(rdbHeader)
	log.Printf("Replica: RDB header (raw): %q", rdbHeader)

	headerBytes := []byte(rdbHeader + "\r\n")

	rdbLen := 0
	fmt.Sscanf(rdbHeader, "$%d", &rdbLen)
	log.Printf("Replica: Expecting RDB length: %d", rdbLen)

	return rdbLen, headerBytes
}

func receiveRdbData(reader *bufio.Reader, rdbLen int) {
	// read exact RDB bytes
	n, err := io.CopyN(io.Discard, reader, int64(rdbLen))
	if err != nil {
		log.Printf("Replica: Error reading RDB data: %v", err)
		return
	}

	log.Printf("Replica: RDB data consumed: %d bytes (+CRLF)", n)
}

func logRawBytes(operation string, rawBytes []byte) {
	if rawBytes != nil {
		log.Printf("[RAW_BYTES] %s: %q", operation, string(rawBytes))
	}
}
