package parser

import (
	"bufio"
	"fmt"

	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func handleBoolean(r *bufio.Reader) (resp.Message, error) {
	b, err := r.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	if len(b) != 3 || b[1] != '\r' || b[2] != '\n' || (b[0] != 't' && b[0] != 'f') {
		return nil, fmt.Errorf("Invalid message for boolean format")
	}
	if b[0] == 't' {
		return &resp.Boolean{Val: true}, nil
	}
	return &resp.Boolean{Val: false}, nil
}
