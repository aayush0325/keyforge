# Keyforge - A Redis Implementation in Go

Keyforge is a feature-rich Redis server implementation written in Go. It provides a subset of Redis commands optimized for performance and simplicity while maintaining protocol compatibility with Redis clients.

## Features

- **RESP Protocol Support**: Full Redis Serialization Protocol (RESP) implementation for client-server communication
- **Data Structures**: Support for Strings, Lists, and Streams
- **Pub/Sub Messaging**: Publish-Subscribe pattern implementation for real-time messaging
- **Persistence**: In-memory data storage with TTL (Time-To-Live) support
- **Connection Handling**: Multi-threaded concurrent connection handling
- **Debug Mode**: Optional debug logging for command execution

## Getting Started

### Prerequisites

- Go 1.25.0 or later

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd keyforge
```

2. Build the server:
```bash
go build -o keyforge ./app
```

3. Run the server:
```bash
./keyforge
```

The server will start and listen on `localhost:6379` (default Redis port).

### Debug Mode

Run with debug logging enabled:
```bash
./keyforge -debug
```

This will log all incoming commands to the console for debugging purposes.

## Supported Commands

### String Commands

#### SET
Set a key-value pair with optional TTL (time-to-live).

**Syntax:**
```
SET key value [EX seconds] [PX milliseconds] [NX | XX]
```

**Examples:**
```
SET mykey "Hello"
SET mykey "World" EX 10
SET mykey "Value" NX  # Only set if key doesn't exist
SET mykey "Value" XX  # Only set if key exists
```

**Return:** Simple string "OK"

---

#### SETNX
Set a key-value pair only if the key does not already exist.

**Syntax:**
```
SETNX key value
```

**Examples:**
```
SETNX mykey "Hello"
```

**Return:** Integer (1 if set, 0 if not set)

---

#### GET
Retrieve the value of a key.

**Syntax:**
```
GET key
```

**Examples:**
```
GET mykey
```

**Return:** Bulk string with the value, or null if key doesn't exist

---

### List Commands

#### LPUSH
Insert one or more values at the head of a list.

**Syntax:**
```
LPUSH key value [value ...]
```

**Examples:**
```
LPUSH mylist "world"
LPUSH mylist "hello"
```

**Return:** Integer representing the length of the list after the operation

---

#### RPUSH
Insert one or more values at the tail of a list.

**Syntax:**
```
RPUSH key value [value ...]
```

**Examples:**
```
RPUSH mylist "one"
RPUSH mylist "two"
```

**Return:** Integer representing the length of the list after the operation

---

#### LPOP
Remove and return the first element of a list.

**Syntax:**
```
LPOP key [count]
```

**Examples:**
```
LPOP mylist
```

**Return:** Bulk string with the popped value, or null if list is empty

---

#### BLPOP
Blocking version of LPOP. Waits for an element to be available.

**Syntax:**
```
BLPOP key [key ...] timeout
```

**Examples:**
```
BLPOP mylist 0
```

**Return:** Array containing the key and the value, or null if timeout expires

---

#### LLEN
Get the length of a list.

**Syntax:**
```
LLEN key
```

**Examples:**
```
LLEN mylist
```

**Return:** Integer representing the number of elements in the list

---

#### LRANGE
Get a range of elements from a list.

**Syntax:**
```
LRANGE key start stop
```

**Examples:**
```
LRANGE mylist 0 -1  # Get all elements
LRANGE mylist 0 2   # Get first 3 elements
```

**Return:** Array of bulk strings representing the elements

---

### Stream Commands

#### XADD
Add an entry to a stream.

**Syntax:**
```
XADD key ID field value [field value ...]
```

**Examples:**
```
XADD mystream * name "Alice" age "30"
XADD mystream 1000-0 field1 "value1"
```

**Return:** Bulk string with the ID of the added entry

---

#### XRANGE
Get a range of entries from a stream.

**Syntax:**
```
XRANGE key start end [COUNT count]
```

**Examples:**
```
XRANGE mystream - +                    # Get all entries
XRANGE mystream 1000-0 2000-0          # Get entries within ID range
XRANGE mystream - + COUNT 10           # Get first 10 entries
```

**Return:** Array of entries (each entry is a pair of [ID, [field, value, ...]])

---

#### XREAD
Read from one or more streams.

**Syntax:**
```
XREAD [COUNT count] [BLOCK milliseconds] STREAMS key [key ...] id [id ...]
```

**Examples:**
```
XREAD STREAMS mystream 0            # Read all entries from mystream
XREAD COUNT 2 STREAMS mystream 0    # Read 2 entries
XREAD BLOCK 1000 STREAMS mystream $ # Block for 1000ms, read new entries
```

**Return:** Array of streams with their entries

---

### Key Commands

#### DEL
Delete one or more keys.

**Syntax:**
```
DEL key [key ...]
```

**Examples:**
```
DEL mykey
DEL key1 key2 key3
```

**Return:** Integer representing the number of keys deleted

---

#### EXISTS
Check if one or more keys exist.

**Syntax:**
```
EXISTS key [key ...]
```

**Examples:**
```
EXISTS mykey
EXISTS key1 key2 key3
```

**Return:** Integer representing the number of existing keys

---

#### TYPE
Get the type of a key.

**Syntax:**
```
TYPE key
```

**Examples:**
```
TYPE mystring   # Returns "string"
TYPE mylist     # Returns "list"
TYPE mystream   # Returns "stream"
```

**Return:** Simple string representing the type (string, list, stream, none)

---

### Pub/Sub Commands

#### PUBLISH
Publish a message to a channel.

**Syntax:**
```
PUBLISH channel message
```

**Examples:**
```
PUBLISH mychannel "Hello, World!"
```

**Return:** Integer representing the number of subscribers that received the message

---

#### SUBSCRIBE
Subscribe to one or more channels.

**Syntax:**
```
SUBSCRIBE channel [channel ...]
```

**Examples:**
```
SUBSCRIBE mychannel
SUBSCRIBE channel1 channel2
```

**Return:** Array messages containing the subscription confirmations and messages

---

#### UNSUBSCRIBE
Unsubscribe from one or more channels.

**Syntax:**
```
UNSUBSCRIBE [channel [channel ...]]
```

**Examples:**
```
UNSUBSCRIBE mychannel
UNSUBSCRIBE  # Unsubscribe from all channels
```

**Return:** Array messages containing the unsubscribe confirmations

---

### Connection Commands

#### PING
Test the connection to the server.

**Syntax:**
```
PING [message]
```

**Examples:**
```
PING
PING "Hello"
```

**Return:** Simple string "PONG" or the provided message

---

#### ECHO
Echo the provided message.

**Syntax:**
```
ECHO message
```

**Examples:**
```
ECHO "Hello, Redis!"
```

**Return:** Bulk string with the echoed message

---

#### HELLO
Greeting command (Redis 6.0+ compatibility).

**Syntax:**
```
HELLO [protover]
```

**Examples:**
```
HELLO
HELLO 3
```

**Return:** Array with server information

---

#### CLIENT
Manage client connections.

**Syntax:**
```
CLIENT subcommand [args]
```

**Note:** Partial implementation for Redis-CLI compatibility

---

#### CONFIG
Get or set configuration parameters.

**Syntax:**
```
CONFIG GET parameter
CONFIG SET parameter value
```

**Examples:**
```
CONFIG GET port
CONFIG SET maxmemory 1000000
```

**Return:** Array with configuration values or status

---

#### COMMAND
Get information about Redis commands (Redis-CLI compatibility).

**Syntax:**
```
COMMAND
```

**Return:** Array of command information

---

## Data Types

### Strings
Keyforge stores simple key-value pairs where both keys and values are binary-safe strings. Supports TTL (Time-To-Live) for automatic key expiration.

### Lists
Doubly-linked list implementation supporting LPUSH, RPUSH, LPOP, BLPOP, LLEN, and LRANGE operations. Lists are created implicitly when the first element is added.

### Streams
Time-series data structure with entries identified by their timestamp (ID). Each entry contains a set of field-value pairs. Supports efficient range queries and blocking reads.

## Internal Architecture

### Components

**Core Modules:**
- **Parser** (`internal/parser/`): RESP protocol parser for client requests
- **Commands** (`internal/commands/`): Command execution handlers
- **Database** (`internal/db/`): In-memory data storage with shard-based concurrency
- **Pub/Sub** (`internal/pubsub/`): Message broker for publish-subscribe functionality
- **Streams** (`internal/streams/`): Stream data structure implementation with Radix tree support
- **Utils** (`internal/utils/`): Helper utilities and data structures
- **RESP** (`internal/resp/`): Redis Serialization Protocol implementation
- **Deque** (`internal/ds/`): Double-ended queue data structure

### Threading Model

Keyforge uses a multi-threaded architecture:
- Each client connection is handled in a separate goroutine
- Database operations use sharded locks for high concurrency
- Pub/Sub uses global state with connection locks for message delivery
- All operations are thread-safe

## Limitations and Future Work

Currently Unsupported:
- Persistence (AOF, RDB snapshots)
- Cluster mode
- Transactions (MULTI/EXEC)
- Lua scripting
- Sorted sets and hash data types
- Key expiration background cleanup (keys expire but aren't cleaned up actively)
- Authentication (AUTH command)
- Connection timeouts and keepalive

## Testing

Run tests with:
```bash
go test ./...
```

Or use the provided test script:
```bash
bash tests/verify_xread_block.sh
```

## Project Structure

```
keyforge/
├── app/
│   └── main.go              # Application entry point
├── internal/
│   ├── commands/            # Command implementations
│   ├── db/                  # Database storage layer
│   ├── ds/                  # Data structures (deque)
│   ├── parser/              # RESP parser
│   ├── pubsub/              # Pub/Sub implementation
│   ├── resp/                # RESP protocol types
│   ├── streams/             # Stream data structure
│   └── utils/               # Utility functions
├── tests/                   # Integration tests
├── go.mod                   # Go module definition
└── README.md                # This file
```
