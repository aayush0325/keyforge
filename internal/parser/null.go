package parser

import (
	"bufio"
	"fmt"

	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func handleNull(r *bufio.Reader) (resp.Message, error) {
	crlf, err := r.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	if len(crlf) != 2 || crlf[0] != '\r' || crlf[1] != '\n' {
		return nil, fmt.Errorf("Invalid CRLF in null type")
	}

	return &resp.Null{}, nil
}
