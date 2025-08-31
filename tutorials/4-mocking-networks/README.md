# Introduction
This guide explains how to implement a mock network layer in Golang to make it easier to implement
distributed algorithms. The mock network layer simulates network communication between nodes
in a distributed system, allowing for testing and development without the need for actual network infrastructure.

Mock networks provide several critical advantages for distributed systems development:

- **Deterministic behavior**: Unlike real networks, mock networks provide predictable, reproducible behavior. 
- **Controlled failure simulation**: Easily test network partitions, message loss, and latency without complex network setup.
- **Correctness verification**: Focus on algorithm logic rather than network implementation details.
- **No external dependencies**: Tests run without requiring network infrastructure, ports, or external services.
- **Isolation**: Test individual components without interference from network issues.
- **Fast execution**: In-memory message passing is orders of magnitude faster than real network I/O.
- **Rapid prototyping**: Implement and test distributed algorithms without network complexity.
- **Debugging**: Full visibility into message flow and timing without network packet analysis.
- **CI/CD**: Tests run reliably in any environment without network configuration.
- **Edge case testing**: Easily create scenarios like split-brain, network partitions, and message reordering.
- **Performance analysis**: Measure algorithm behavior without network latency noise.

Interfaces define contracts that multiple implementations can fulfill, making code more flexible and testable. This becomes especially valuable when you need different network implementations:

```go
// Same interface, different implementations
var network Network

// For testing - fast, deterministic, controllable
network = NewMockNetwork()

// For production - real TCP/UDP networking
network = NewTCPNetwork()

// For simulation - with realistic latency and packet loss
network = NewSimulatedNetwork()
```

By defining an abstract Network interface that decouples implementation from specification it becomes possible to:

- **Test with mock**, **deploy with real networks**.
- **Deploy with real networks** Switch to actual TCP/UDP for production.
- **Benchmark with simulation** that models realistic network conditions.
- **Switch implementations** without changing algorithm code.

# Understanding Golang interfaces
Golang interfaces are a way to define behavior in your code. An interface is a type that specifies a set of method signatures. Any type that implements those methods satisfies the interface.

Below is a simple example of a Golang interface:
```go
package main
import "fmt"
type Animal interface {
    Speak() string
}
type Dog struct{}
func (d Dog) Speak() string {
    return "Woof!"
}
type Cat struct{}
func (c Cat) Speak() string {
    return "Meow!"
}
func main() {
    var a Animal
    a = Dog{}
    fmt.Println(a.Speak()) // Output: Woof!
    a = Cat{}
    fmt.Println(a.Speak()) // Output: Meow!
}
```

```console
$ go run main.go
Woof!
Meow!
```

Golang is not a traditional object-oriented programming language. While it does support some object-oriented concepts like encapsulation and polymorphism through interfaces, it does not have classes or inheritance. Instead, Golang uses structs to define data types and interfaces to define behavior. This approach encourages composition over inheritance, promoting more modular and maintainable code.

In object-oriented programming languages like Java or C++, you typically have to explicitly declare that a class implements an interface. For example, in Java, you would use the `implements` keyword. Golang interfaces on the other hand are implicit. This means that you don't need to explicitly declare that a type implements an interface. If a type has the methods that the interface requires, it automatically satisfies the interface. This implicit implementation allows for more flexibility and decoupling in your code. You can create new types that satisfy an interface without modifying the interface itself or the types that already implement it.

# Implementing a mock network layer in Golang
To implement a mock network layer in Golang, we need to define an interface for the network layer and then create a mock implementation of that interface. The mock implementation will simulate network communication between nodes. What functionality we want to include in our mock network layer will depend on the specific requirements of the distributed algorithms we are implementing. However, a basic mock network layer might include methods for sending and receiving messages between nodes.

```go
package main

import "fmt"

type Address struct {
	IP   string
	Port int // 1-65535
}

type Network interface {
	Listen(addr Address) (Connection, error)
	Dial(addr Address) (Connection, error)

	// Network partition simulation
	Partition(group1, group2 []Address)
	Heal()
}

type Connection interface {
	Send(msg Message) error
	Recv() (Message, error)
	Close() error
}

type Message struct {
	From    Address
	To      Address
	Payload []byte
	network Network // Reference to network for replies
}
```

To implement a mock network we need to implement the ``Network`` and the ``Connection`` interface. Instead of sending data over an IP network, we'll simulate network communication using Go channels.

The mockNetwork acts as a central message router that:
- **Maps addresses to channels**: Each network address corresponds to a Go channel that receives messages.
- **Routes messages between nodes**: When you send to an address, it forwards the message to that address's channel.
- **Simulates network failures**: Can partition nodes or make them unreachable.

```go
package main

import (
	"errors"
	"sync"
)

type mockNetwork struct {
	mu         sync.RWMutex
	listeners  map[Address]chan Message
	partitions map[Address]bool // true if the address is partitioned
}

func NewMockNetwork() Network {
	return &mockNetwork{
		listeners:  make(map[Address]chan Message),
		partitions: make(map[Address]bool),
	}
}

func (n *mockNetwork) Listen(addr Address) (Connection, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if _, exists := n.listeners[addr]; exists {
		return nil, errors.New("address already in use")
	}
	ch := make(chan Message, 100) // buffered channel
	n.listeners[addr] = ch
	return &mockConnection{addr: addr, network: n, recvCh: ch}, nil
}

func (n *mockNetwork) Dial(addr Address) (Connection, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if _, exists := n.listeners[addr]; !exists {
		return nil, errors.New("address not found")
	}
	return &mockConnection{addr: addr, network: n}, nil
}

func (n *mockNetwork) Partition(group1, group2 []Address) {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, addr := range group1 {
		n.partitions[addr] = true
	}
	for _, addr := range group2 {
		n.partitions[addr] = true
	}
}

func (n *mockNetwork) Heal() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.partitions = make(map[Address]bool)
}

type mockConnection struct {
	addr    Address
	network *mockNetwork
	recvCh  chan Message
	mu      sync.RWMutex
	closed  bool
}

func (c *mockConnection) Send(msg Message) error {
	c.network.mu.RLock()
	
	if c.network.partitions[c.addr] || c.network.partitions[msg.To] {
		c.network.mu.RUnlock()
		return errors.New("network partitioned")
	}
	
	ch, exists := c.network.listeners[msg.To]
	if !exists {
		c.network.mu.RUnlock()
		return errors.New("destination address not found")
	}
	
	// Add network reference to message for replies
	msg.network = c.network
	
	// Keep the lock while sending to prevent the channel from being closed
	select {
	case ch <- msg:
		c.network.mu.RUnlock()
		return nil
	default:
		c.network.mu.RUnlock()
		return errors.New("message queue full")
	}
}

func (c *mockConnection) Recv() (Message, error) {
	c.mu.RLock()
	if c.closed || c.recvCh == nil {
		c.mu.RUnlock()
		return Message{}, errors.New("connection not listening")
	}
	ch := c.recvCh
	c.mu.RUnlock()
	
	msg, ok := <-ch
	if !ok {
		return Message{}, errors.New("connection closed")
	}
	return msg, nil
}

func (c *mockConnection) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil // Already closed
	}
	c.closed = true
	c.mu.Unlock()
	
	c.network.mu.Lock()
	defer c.network.mu.Unlock()
	
	if c.recvCh != nil {
		close(c.recvCh)
		delete(c.network.listeners, c.addr)
		c.recvCh = nil
	}
	return nil
}
```

## Node Implementation
While the low-level ``Network`` and ``Connection`` interfaces provide the foundation of our mock network implementation, using them directly requires repetitive boilerplate code. Let's implement a unified ``Node`` abstraction that simplifies sending and receiving messages.

- **Network layer**: Handles low-level message transport, partitioning, connection management.
- **Node**: Unified abstraction that handles both incoming and outgoing message patterns.
- **Application**: Focuses on distributed algorithm logic, not network plumbing.

```go
package main

import (
	"fmt"
	"log"
	"sync"
)

// Node provides a unified abstraction for both sending and receiving messages
type Node struct {
	addr       Address
	network    Network
	connection Connection
	handlers   map[string]MessageHandler
	mu         sync.RWMutex
	closed     bool
	closeMu    sync.RWMutex
}

// MessageHandler is a function that processes incoming messages
type MessageHandler func(msg Message) error

// NewNode creates a new node that can both send and receive messages
func NewNode(network Network, addr Address) (*Node, error) {
	connection, err := network.Listen(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create node: %v", err)
	}

	return &Node{
		addr:       addr,
		network:    network,
		connection: connection,
		handlers:   make(map[string]MessageHandler),
	}, nil
}

// Handle registers a message handler for a specific message type
func (n *Node) Handle(msgType string, handler MessageHandler) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.handlers[msgType] = handler
}

// Start begins listening for incoming messages
func (n *Node) Start() {
	go func() {
		for {
			n.closeMu.RLock()
			if n.closed {
				n.closeMu.RUnlock()
				return
			}
			n.closeMu.RUnlock()

			msg, err := n.connection.Recv()
			if err != nil {
				n.closeMu.RLock()
				if !n.closed {
					log.Printf("Node %s failed to receive message: %v", n.addr.String(), err)
				}
				n.closeMu.RUnlock()
				return
			}

			// Extract message type from payload (first part before ':')
			msgType := "default"
			payload := string(msg.Payload)
			if len(payload) > 0 {
				for i, char := range payload {
					if char == ':' {
						msgType = payload[:i]
						break
					}
				}
			}

			n.mu.RLock()
			handler, exists := n.handlers[msgType]
			if !exists {
				handler, exists = n.handlers["default"]
			}
			n.mu.RUnlock()
			
			if exists && handler != nil {
				if err := handler(msg); err != nil {
					log.Printf("Handler error: %v", err)
				}
			}
		}
	}()
}

// Send sends a message to the target address
func (n *Node) Send(to Address, msgType string, data []byte) error {
	connection, err := n.network.Dial(to)
	if err != nil {
		return fmt.Errorf("failed to dial %s: %v", to.String(), err)
	}
	defer connection.Close()

	// Format payload as "msgType:data"
	var payload []byte
	if msgType != "" {
		payload = append([]byte(msgType+":"), data...)
	} else {
		payload = data
	}

	msg := Message{
		From:    n.addr,
		To:      to,
		Payload: payload,
	}

	return connection.Send(msg)
}

// SendString is a convenience method for sending string messages
func (n *Node) SendString(to Address, msgType, data string) error {
	return n.Send(to, msgType, []byte(data))
}

// Close shuts down the node
func (n *Node) Close() error {
	n.closeMu.Lock()
	n.closed = true
	n.closeMu.Unlock()
	return n.connection.Close()
}

// Address returns the node's address
func (n *Node) Address() Address {
	return n.addr
}
```

# Examples 
The ```node_test.go```` file contains test code to experiment with the mock network implementation. When working with Go, consider organizing your experimental code and examples as test functions in ```_test.go``` files rather than using ```main()```` functions. Since a Go package can only have one ```main()```  function but unlimited test functions, this approach allows you to:

- Keep multiple code examples in a single project without conflicts.
- Run specific examples individually using ```go test -run TestName```.
- Re-execute and modify examples easily.
- Build a collection of working snippets for reference.
- Follow Go community conventions.

#### 1. **Helloworld** example
```go
func TestHelloworld(t *testing.T) {
	// Create network and nodes
	network := NewMockNetwork()
	alice, _ := NewNode(network, Address{IP: "127.0.0.1", Port: 8080})
	bob, _ := NewNode(network, Address{IP: "127.0.0.1", Port: 8081})

	// Channel for synchronization
	done := make(chan struct{})

	// Alice says hello when she receives a message
	alice.Handle("hello", func(msg Message) error {
		fmt.Printf("Alice: Hello %s!\n", msg.From.IP)
		return msg.ReplyString("reply", "Nice to meet you!")
	})

	// Bob prints replies
	bob.Handle("reply", func(msg Message) error {
		fmt.Printf("Bob: %s\n", string(msg.Payload)[6:]) // Skip "reply:" prefix
		done <- struct{}{}
		return nil
	})

	// Start nodes and send message
	alice.Start()
	bob.Start()
	bob.SendString(alice.Address(), "hello", "Hi Alice!")

	// Wait for completion and cleanup
	<-done
	alice.Close()
	bob.Close()
	fmt.Println("Done!")
}
```

```console
go test *.go -v -test.run TestHelloworld
=== RUN   TestHelloworld
Alice: Hello 127.0.0.1!
Bob: Nice to meet you!
Done!
--- PASS: TestHelloworld (0.00s)
PASS
ok      command-line-arguments  0.325s
```

# Thread Safety and Concurrency
The mock network implementation uses Go's `sync.RWMutex` (Read-Write Mutex) to ensure thread safety when multiple goroutines access shared data structures concurrently.

## Understanding RWMutex vs Mutex
- **`sync.Mutex`**: Provides exclusive access - only one goroutine can hold the lock at a time.
- **`sync.RWMutex`**: Provides shared/exclusive access:
- Multiple goroutines can hold read locks simultaneously (`RLock()`)
- Only one goroutine can hold a write lock (`Lock()`)
- Write locks are exclusive - no other read or write locks can be held

## Network-Level Synchronization
The `mockNetwork` struct uses `sync.RWMutex` to protect shared maps:

```go
type mockNetwork struct {
    mu         sync.RWMutex
    listeners  map[Address]chan Message  // Protected by mu
    partitions map[Address]bool          // Protected by mu
}
```

### Read Operations (RLock)
Used when only reading from shared data structures:

```go
func (n *mockNetwork) Dial(addr Address) (Connection, error) {
    n.mu.RLock()              // Multiple goroutines can read simultaneously
    defer n.mu.RUnlock()      // Always unlock when function returns
    
    if _, exists := n.listeners[addr]; !exists {
        return nil, errors.New("address not found")
    }
    return &mockConnection{addr: addr, network: n}, nil
}
```

**Why RLock here?** `Dial()` only needs to check if an address exists in the listeners map. Since it's not modifying the map, multiple goroutines can safely read from it concurrently using `RLock()`.

### Write Operations (Lock)
Used when modifying shared data structures:

```go
func (n *mockNetwork) Listen(addr Address) (Connection, error) {
    n.mu.Lock()               // Exclusive access - blocks all other operations
    defer n.mu.Unlock()      // Always unlock when function returns
    
    if _, exists := n.listeners[addr]; exists {
        return nil, errors.New("address already in use")
    }
    ch := make(chan Message, 100)
    n.listeners[addr] = ch    // Modifying the map requires exclusive access
    return &mockConnection{addr: addr, network: n, recvCh: ch}, nil
}
```

**Why Lock here?** `Listen()` modifies the listeners map by adding a new entry. This requires exclusive access to prevent race conditions.

## Connection-Level Synchronization
Each `mockConnection` has its own mutex to protect connection-specific state:

```go
type mockConnection struct {
    addr    Address
    network *mockNetwork
    recvCh  chan Message
    mu      sync.RWMutex     // Protects connection-specific state
    closed  bool             // Protected by mu
}
```

### Locking in Send()
The `Send()` method also requires a locking mechanism to prevent race conditions:

```go
func (c *mockConnection) Send(msg Message) error {
    c.network.mu.RLock()     // Read-lock network state
    
    // Check partition status (read operation)
    if c.network.partitions[c.addr] || c.network.partitions[msg.To] {
        c.network.mu.RUnlock()
        return errors.New("network partitioned")
    }
    
    // Get destination channel (read operation)
    ch, exists := c.network.listeners[msg.To]
    if !exists {
        c.network.mu.RUnlock()
        return errors.New("destination address not found")
    }
    
    // Send message while holding lock to prevent channel closure
    select {
    case ch <- msg:
        c.network.mu.RUnlock()
        return nil
    default:
        c.network.mu.RUnlock()
        return errors.New("message queue full")
    }
}
```

The network lock is held during the entire send operation, including the channel send. This prevents the destination connection from being closed while we're sending to it, avoiding panics from sending to closed channels.

The `Close()` method uses a two-level locking strategy. Connection-level lock prevents double-close race conditions. Network-level lock ensures atomic removal from the network's listener map.

```go
func (c *mockConnection) Close() error {
    // First, mark connection as closed with connection-level lock
    c.mu.Lock()
    if c.closed {
        c.mu.Unlock()
        return nil // Already closed - prevent double-close
    }
    c.closed = true
    c.mu.Unlock()
    
    // Then modify network state with network-level lock
    c.network.mu.Lock()
    defer c.network.mu.Unlock()
    
    if c.recvCh != nil {
        close(c.recvCh)                          // Close the channel
        delete(c.network.listeners, c.addr)      // Remove from network
        c.recvCh = nil
    }
    return nil
}
```

The `Recv()` method safely accesses the receive channel:

```go
func (c *mockConnection) Recv() (Message, error) {
    c.mu.RLock()                    // Read-lock connection state
    if c.closed || c.recvCh == nil {
        c.mu.RUnlock()
        return Message{}, errors.New("connection not listening")
    }
    ch := c.recvCh                  // Get channel reference while locked
    c.mu.RUnlock()                  // Release lock before blocking operation
    
    msg, ok := <-ch                 // This may block, so lock is released
    if !ok {
        return Message{}, errors.New("connection closed")
    }
    return msg, nil
}
```

Get the channel reference under lock, then release the lock before the potentially blocking channel receive operation.

The `Node` struct protects its message handlers map:

```go
// Registering handlers (write operation)
func (n *Node) Handle(msgType string, handler MessageHandler) {
    n.mu.Lock()                     // Exclusive access for modification
    defer n.mu.Unlock()
    n.handlers[msgType] = handler   // Modify handlers map
}

// Processing messages (read operation)
n.mu.RLock()                        // Shared access for reading
handler, exists := n.handlers[msgType]
if !exists {
    handler, exists = n.handlers["default"]
}
n.mu.RUnlock()                      // Release before calling handler

if exists && handler != nil {
    handler(msg)                    // Call handler without holding lock
}
```

The lock is released before calling the message handler to prevent holding the lock during potentially long-running user code.
