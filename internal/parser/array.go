package parser

import (
	"bufio"
	"fmt"
	"strconv"

	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func handleArray(r *bufio.Reader) (resp.Message, error) {
	num, err := r.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	// validate CRLF
	if len(num) < 3 || num[len(num)-1] != '\n' || num[len(num)-2] != '\r' {
		return nil, fmt.Errorf("invalid format of giving number of elements for an array")
	}

	// strip CRLF
	num = num[:len(num)-2]

	numElements, err := strconv.ParseInt((string)(num), 10, 64)

	if err != nil {
		return nil, err
	}

	if numElements == -1 {
		return &resp.Array{
			Val: nil,
		}, nil
	}

	if numElements < 0 {
		return nil, fmt.Errorf("Number of elements in an array cannot be negative")
	}

	resultArray := &resp.Array{
		Val: make([]resp.Message, numElements),
	}

	for i := int64(0); i < numElements; i++ {
		el, err := Parse(r)
		if err != nil {
			return nil, err
		}
		resultArray.Val[i] = el
	}
	return resultArray, nil
}
