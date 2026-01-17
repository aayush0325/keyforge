package parser

import (
	"bufio"
	"fmt"
	"strconv"

	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func handleInteger(r *bufio.Reader) (resp.Message, error) {
	num, err := r.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	// Validate CRLF
	if len(num) < 2 || num[len(num)-1] != '\n' || num[len(num)-2] != '\r' {
		return nil, fmt.Errorf("Invalid terminating CRLF")
	}

	// Strip CRLF
	num = num[:len(num)-2]

	// Parse string as int (parseInt handles the optional signs)
	i, err := strconv.ParseInt((string)(num), 10, 64)

	if err != nil {
		return nil, err
	}

	return &resp.Integer{
		Val: i,
	}, nil
}
