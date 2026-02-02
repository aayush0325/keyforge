package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/codecrafters-io/redis-starter-go/internal/commands"
	"github.com/codecrafters-io/redis-starter-go/internal/parser"
	"github.com/codecrafters-io/redis-starter-go/internal/pubsub"
	"github.com/codecrafters-io/redis-starter-go/internal/utils"
)

var debug = flag.Bool("debug", false, "Enable debug mode to log all commands")

func handleConn(c net.Conn) {
	defer c.Close()

	reader := bufio.NewReader(c)
	writer := bufio.NewWriter(c)
	defer writer.Flush()

	Conn := pubsub.Connection{
		W:        writer,
		Channels: make(map[string]struct{}),
	}

	for {
		msg, err := parser.Parse(reader)
		if err != nil {
			if err == io.EOF {
				return
			}
			writer.Write([]byte(fmt.Sprintf("-ERR %s\r\n", err.Error())))
			writer.Flush()
			return
		}

		commands.ExecuteCommands(msg, &Conn)
		writer.Flush()
	}
}

func main() {
	flag.Parse()
	commands.DebugMode = *debug

	go func() {
		log.Println("pprof listening on http://localhost:6060")
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	log.Printf("STARTING REDIS SERVER")
	if commands.DebugMode {
		log.Printf("DEBUG MODE ENABLED")
	}

	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		log.Println("Failed to bind to port 6379")
		os.Exit(1)
	}
	defer l.Close()

	utils.GlobalInitFunction()

	for {
		conn, err := l.Accept()
		log.Printf("RECEIVED A CONNECTION")
		if err != nil {
			log.Println("Error accepting connection:", err)
			os.Exit(1)
		}

		go handleConn(conn)
	}
}
