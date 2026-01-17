package parser

import (
	"bufio"
	"fmt"

	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func Parse(r *bufio.Reader) (resp.Message, error) {
	first, err := r.ReadByte()

	if err != nil {
		return nil, err
	}

	switch first {
	case '$':
		return handleBulkString(r)
	case '+':
		return handleSimpleString(r)
	case '-':
		return handleSimpleError(r)
	case ':':
		return handleInteger(r)
	case '_':
		return handleNull(r)
	case '#':
		return handleBoolean(r)
	case '*':
		return handleArray(r)
	default:
		// Check if this might be an inline command (plain text format)
		// Inline commands start with printable ASCII characters
		if first >= 0x21 && first <= 0x7E {
			return handleInlineCommand(first, r)
		}
		return nil, fmt.Errorf("Not a valid RESP message (got byte: %q / 0x%02x)", first, first)
	}
}
