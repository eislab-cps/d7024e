# Introduction
This guide explains how to implement a mock network layer in Golang to implement
distributed algorithms. The mock network layer simulates network communication between nodes
in a distributed system, allowing for testing and development without the need for actual network infrastructure.

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

Below is an example of a simple mock network layer interface in Golang. Save the following code in a file named `pkg/network/network.go`:
```go
package mock

type Address struct {
    IP   string
    Port int // 1-65535
}

type Network interface {
    Listen(addr Address) (Socket, error)
    Dial(addr Address) (Socket, error)

    // Network partition simulation
    Partition(group1, group2 []Address)
    Heal()
}

type Socket interface {
    Send(msg Message) error
    Recv() (Message, error)
    Close() error
}

type Message struct {
    From    Address
    To      Address
    Payload []byte
}
```

The `Partition` method simulates a network partition between two groups of nodes, while the `Heal` method restores connectivity between all nodes. This allows us to test how distributed algorithms handle network partitions and recoveries. Next, we can create a mock implementation of the `Network` interface. This implementation will use channels to simulate message passing between nodes.

Save the following code in a file named `pkg/network/mock_network.go`:

```go
package network
import (
    "errors"
    "sync"
)

type mockNetwork struct {
    mu        sync.RWMutex
    listeners map[Address]chan Message
    partitions map[Address]bool // true if the address is partitioned
}

func NewMockNetwork() Network {
    return &mockNetwork{
        listeners: make(map[Address]chan Message),
        partitions: make(map[Address]bool),
    }
}

func (n *mockNetwork) Listen(addr Address) (Socket, error) {
    n.mu.Lock()
    defer n.mu.Unlock()
    if _, exists := n.listeners[addr]; exists {
        return nil, errors.New("address already in use")
    }
    ch := make(chan Message, 100) // buffered channel
    n.listeners[addr] = ch
    return &mockSocket{addr: addr, network: n, recvCh: ch}, nil
}

func (n *mockNetwork) Dial(addr Address) (Socket, error) {
    n.mu.RLock()
    defer n.mu.RUnlock()
    if _, exists := n.listeners[addr]; !exists {
        return nil, errors.New("address not found")
    }
    return &mockSocket{addr: addr, network: n}, nil
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

type mockSocket struct {
    addr    Address
    network *mockNetwork
    recvCh  chan Message
}

func (s *mockSocket) Send(msg Message) error {
    s.network.mu.RLock()
    defer s.network.mu.RUnlock()
    if s.network.partitions[s.addr] || s.network.partitions[msg.To] {
        return errors.New("network partitioned")
    }
    if ch, exists := s.network.listeners[msg.To]; exists {
        ch <- msg
        return nil
    }
    return errors.New("destination address not found")
}

func (s *mockSocket) Recv() (Message, error) {
    if s.recvCh == nil {
        return Message{}, errors.New("socket not listening")
    }
    msg := <-s.recvCh
    return msg, nil
}

func (s *mockSocket) Close() error {
    s.network.mu.Lock()
    defer s.network.mu.Unlock()
    if s.recvCh != nil {
        close(s.recvCh)
        delete(s.network.listeners, s.addr)
        s.recvCh = nil
    }
    return nil
}
```

This mock network implementation allows nodes to listen for incoming messages and dial other nodes to send messages. It also supports simulating network partitions and healing the network.

To reduce boilerplate code, let's also introduce a client and a server abstraction.
Save the following code in a file named `pkg/network/client_server.go`:

```go
package network 

import (
    "fmt"
    "log"
)

// Server provides a high-level abstraction for receiving messages
type Server struct {
    addr     Address
    network  Network
    socket   Socket
    handlers map[string]MessageHandler
}

// MessageHandler is a function that processes incoming messages
type MessageHandler func(msg Message) error

// NewServer creates a new server that listens on the given address
func NewServer(network Network, addr Address) (*Server, error) {
    socket, err := network.Listen(addr)
    if err != nil {
        return nil, fmt.Errorf("failed to create server: %v", err)
    }
    
    return &Server{
        addr:     addr,
        network:  network,
        socket:   socket,
        handlers: make(map[string]MessageHandler),
    }, nil
}

// Handle registers a message handler for a specific message type
func (s *Server) Handle(msgType string, handler MessageHandler) {
    s.handlers[msgType] = handler
}

// Start begins listening for incoming messages
func (s *Server) Start() {
    go func() {
        for {
            msg, err := s.socket.Recv()
            if err != nil {
                log.Printf("Server %s failed to receive message: %v", s.addr.String(), err)
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
            
            if handler, exists := s.handlers[msgType]; exists {
                if err := handler(msg); err != nil {
                    log.Printf("Handler error: %v", err)
                }
            } else if defaultHandler, exists := s.handlers["default"]; exists {
                if err := defaultHandler(msg); err != nil {
                    log.Printf("Default handler error: %v", err)
                }
            }
        }
    }()
}

// Close shuts down the server
func (s *Server) Close() error {
    return s.socket.Close()
}

// Address returns the server's address
func (s *Server) Address() Address {
    return s.addr
}

// Client provides a high-level abstraction for sending messages
type Client struct {
    addr    Address
    network Network
}

// NewClient creates a new client
func NewClient(network Network, addr Address) *Client {
    return &Client{
        addr:    addr,
        network: network,
    }
}

// Send sends a message to the target address
func (c *Client) Send(to Address, msgType string, data []byte) error {
    socket, err := c.network.Dial(to)
    if err != nil {
        return fmt.Errorf("failed to dial %s: %v", to.String(), err)
    }
    defer socket.Close()
    
    // Format payload as "msgType:data"
    var payload []byte
    if msgType != "" {
        payload = append([]byte(msgType+":"), data...)
    } else {
        payload = data
    }
    
    msg := Message{
        From:    c.addr,
        To:      to,
        Payload: payload,
    }
    
    return socket.Send(msg)
}

// SendString is a convenience method for sending string messages
func (c *Client) SendString(to Address, msgType, data string) error {
    return c.Send(to, msgType, []byte(data))
}

// Address returns the client's address
func (c *Client) Address() Address {
    return c.addr
}
```


To test the mock network layer, we can create a simple test case that sets up two nodes, has one node send a message to the other, and verifies that the message is received correctly. We can also test the network partitioning functionality.

Save the following code in a file named `pkg/network/mock_network_test.go`:

```go
package network_test
import (
    "testing"
    "time"
    "mock"
)

func TestSending
