package parser

import (
	"bufio"
	"fmt"
	"io"
	"strconv"

	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func handleBulkString(r *bufio.Reader) (resp.Message, error) {
	num, err := r.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	if len(num) < 2 || num[len(num)-2] != '\r' {
		return nil, fmt.Errorf("Too few bytes for a bulk string")
	}

	num = num[:len(num)-2] // skip over the CRLF in the end
	numBytes, err := strconv.Atoi(string(num))

	if err != nil {
		return nil, err
	}

	if numBytes == -1 {
		return &resp.BulkString{ // NULL bulk string
			Size: -1,
			Str:  nil,
		}, nil
	}

	if numBytes < 0 {
		return nil, fmt.Errorf("Invalid size for a bulk string")
	}

	str := make([]byte, numBytes)
	_, err = io.ReadFull(r, str)
	if err != nil {
		return nil, err
	}

	// validate trailing CRLF
	crlf := make([]byte, 2)
	if _, err := io.ReadFull(r, crlf); err != nil {
		return nil, err
	}

	if crlf[0] != '\r' || crlf[1] != '\n' {
		return nil, fmt.Errorf("invalid bulk string termination")
	}

	return &resp.BulkString{
		Str:  str,
		Size: numBytes,
	}, nil
}
