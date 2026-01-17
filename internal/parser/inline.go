package parser

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

// handleInlineCommand parses Redis inline commands (plain text format)
// Used for telnet-style commands like: PING\r\n or SET key value\r\n
func handleInlineCommand(first byte, r *bufio.Reader) (resp.Message, error) {
	// Read the rest of the line
	line, err := r.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	// Combine first byte with the rest of the line
	fullLine := append([]byte{first}, line...)

	// Trim whitespace and split by spaces
	fullLine = bytes.TrimSpace(fullLine)
	parts := strings.Fields(string(fullLine))

	if len(parts) == 0 {
		return nil, nil
	}

	// Convert inline command to RESP Array format
	arr := &resp.Array{
		Val: make([]resp.Message, len(parts)),
	}

	for i, part := range parts {
		arr.Val[i] = &resp.BulkString{
			Str:  []byte(part),
			Size: len(part),
		}
	}

	return arr, nil
}
