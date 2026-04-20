package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/internal/config"
	"github.com/codecrafters-io/redis-starter-go/internal/utils"
)

var debug = flag.Bool("debug", false, "Enable debug mode to log all commands")
var port = flag.Int("port", 6379, "Port to listen on")
var replicaof = flag.String("replicaof", "", " act as a replica of <host> <port>")

// TODO: add handling for broken replicas
// TODO: add parsing for RDB files
// TODO: add polling replicas for getack periodically

func main() {
	flag.Parse() // Parse flags
	config.DebugMode = *debug
	config.Offset = 0
	b := make([]byte, 20)
	rand.Read(b)
	config.ReplID = hex.EncodeToString(b)

	// Start the profiling server
	go func() {
		log.Println("pprof listening on http://localhost:6060")
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	// Initialize
	utils.GlobalInitFunction()

	// Parse flags
	if !config.DebugMode {
		config.DebugMode = true
		log.Printf("DEBUG MODE ENABLED")
	}

	// If this instance is a replica we'll try to do a 3 way handshake and then
	// allocate a thread to listen to the primary instance
	if *replicaof != "" {
		config.IsReplica = true
		parts := strings.Split(*replicaof, " ")
		if len(parts) != 2 {
			log.Fatalf("Invalid replicaof format. Expected '<host> <port>'")
		}
		masterHost := parts[0]
		masterPort := 0
		fmt.Sscanf(parts[1], "%d", &masterPort)

		// The handshake function returns a connection with the primary replica
		// or nil if the handshake failed
		conn, reader := Handshake(masterHost, masterPort, *port)
		if conn == nil {
			log.Fatal("Couldn't establish connection with primary replica")
		}

		log.Printf("started handleClientConn for mastrer")

		// Allocate a thread to listen to the primary replica
		go handleClientConn(conn, reader, true) // IsMaster = true
	}

	// Continue flow for a replica as it's normal to support readers
	addr := fmt.Sprintf("0.0.0.0:%d", *port)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("Failed to bind to port %d", *port)
		os.Exit(1)
	}
	defer l.Close()

	log.Printf("STARTING REDIS SERVER on port %d", *port)

	for {
		conn, err := l.Accept()
		log.Printf("RECEIVED A CONNECTION from %s", conn.RemoteAddr())
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}

		go handleClientConn(conn, nil, false) // IsMaster = false
	}
}
