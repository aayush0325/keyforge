package parser

import (
	"bufio"
	"fmt"

	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func handleSimpleError(r *bufio.Reader) (resp.Message, error) {
	str, err := r.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	// validate CRLF
	if len(str) < 2 || str[len(str)-1] != '\n' || str[len(str)-2] != '\r' {
		return nil, fmt.Errorf("CRLF invalid for simple string")
	}
	// strip CRLF from the string
	str = str[:len(str)-2]
	return &resp.SimpleError{Val: str}, nil
}
