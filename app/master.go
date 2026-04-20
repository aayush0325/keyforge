package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/codecrafters-io/redis-starter-go/internal/commands"
	"github.com/codecrafters-io/redis-starter-go/internal/parser"
	"github.com/codecrafters-io/redis-starter-go/internal/pubsub"
)

func handleClientConn(c net.Conn, reader *bufio.Reader, isMaster bool) {
	defer c.Close()

	writer := bufio.NewWriter(c)
	if reader == nil {
		reader = bufio.NewReader(c)
	}
	defer writer.Flush()

	conn := pubsub.Connection{
		W:                    writer,
		R:                    reader,
		Channels:             make(map[string]struct{}),
		IsTransactionQueued:  false,
		TransactionCommands:  nil,
		IsTransactionRunning: false,
		TransactionResponse:  nil,
		IsMaster:             isMaster, // Client connection is not a replica
	}

	for {
		msg, err := parser.Parse(reader)
		if err != nil {
			if err == io.EOF {
				return
			}
			log.Printf("Error parsing client command: %v", err)
			fmt.Fprintf(writer, "-ERR %s\r\n", err.Error())
			writer.Flush()
			return
		}

		log.Printf("IsMaster: %t, Received from client: %s", isMaster, msg.ToBytes())

		commands.ExecuteCommands(msg, &conn)

		writer.Flush()
	}
}
