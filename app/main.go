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
	"os/signal"
	"strings"
	"syscall"

	"github.com/aayush0325/keyforge/internal/commands"
	"github.com/aayush0325/keyforge/internal/config"
	"github.com/aayush0325/keyforge/internal/rdb"
	"github.com/aayush0325/keyforge/internal/utils"
)

var debug = flag.Bool("debug", false, "Enable debug mode to log all commands")
var port = flag.Int("port", 6379, "Port to listen on")
var replicaof = flag.String("replicaof", "", " act as a replica of <host> <port>")
var dir = flag.String("dir", ".", "Directory holding the RDB file")
var dbfilename = flag.String("dbfilename", "dump.rdb", "Name of the RDB file inside -dir")
var noSave = flag.Bool("nosave", false, "Do not write an RDB file on shutdown")

// TODO: add handling for broken replicas
// TODO: add polling replicas for getack periodically

func main() {
	flag.Parse() // Parse flags
	config.DebugMode = *debug
	config.Offset = 0
	b := make([]byte, 20)
	rand.Read(b)
	config.ReplID = hex.EncodeToString(b)

	config.Dir = *dir
	config.DBFilename = *dbfilename

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

	// Restore the dataset from the RDB file left behind by a previous run
	if err := rdb.LoadFromDisk(commands.RDBPath()); err != nil {
		log.Printf("Could not load %s: %v", commands.RDBPath(), err)
		os.Exit(1)
	}

	// A snapshot is written on shutdown so that a restart comes back with the
	// same data even when no SAVE was issued explicitly.
	installShutdownHandler()

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

		// The handshake performs the full resynchronization, applying the dataset
		// the master sends, and then returns a connection positioned right after
		// the replication stream
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
		if err != nil {
			if !config.ShuttingDown.Load() {
				log.Println("Error accepting connection:", err)
			}
			continue
		}

		log.Printf("RECEIVED A CONNECTION from %s", conn.RemoteAddr())

		go handleClientConn(conn, nil, false) // IsMaster = false
	}
}

// installShutdownHandler writes a final snapshot when the process is asked to
// terminate, mirroring what Redis does on SIGTERM.
func installShutdownHandler() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-signals
		log.Printf("Received signal %v, shutting down", sig)

		config.ShuttingDown.Store(true)

		if !*noSave {
			if err := commands.SaveOnShutdown(); err != nil {
				log.Printf("Could not save the dataset: %v", err)
			}
		}

		os.Exit(0)
	}()
}
