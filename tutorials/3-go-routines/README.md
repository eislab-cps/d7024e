# Tutorial 
This tutorial explains how to use and test goroutines. Goroutines are lightweight, cooperatively scheduled functions (aka coroutines) that can pause and resume execution while maintaining their state. 

The key difference from traditional threads is in their memory footprint and management overhead. Threads are heavyweight, requiring MB of memory each and expensive system calls for creation and switching, while goroutines only require KB of memory and switch with minimal overhead. Threads provide true parallelism but require complex synchronization, whereas coroutines traditionally provide concurrency through cooperative multitasking on a single thread.

Goroutines enhance the coroutine model by combining cooperative scheduling with runtime management. The Go runtime multiplexes thousands of goroutines across a small number of OS threads, giving you the simplicity and lightweight nature of coroutines while still achieving parallelism across CPU cores. This hybrid approach makes concurrent programming much easier to use and more efficient than traditional threading models.

# Creating goroutines
Goroutines are created using the `go` keyword followed by a function call. This starts the function in a new goroutine, allowing it to run concurrently with other goroutines.

```go
package main

func sayHello() {
    println("Hello, World!")
}

func main() {
	go sayHello()
}
```

In this example, the `sayHello` function is executed in a new goroutine when `go sayHello()` is called. 

Alternatively, it is possible to inline the function definition:

```go
package main

import "fmt"

func main() {
    go func() {
        fmt.Println("Hello, World!")
    }()
}
````

However, in both these examples, the main function *may* exit before `sayHello` has a chance to run, so the program may terminate without printing anything. We can solve this by adding a simple sleep in the main function to give the goroutine time to execute.

```go
package main

import (
	"fmt"
	"time"
)

func sayHello() {
	fmt.Println("Hello, World!")
}

func main() {
	go sayHello()
	time.Sleep(1 * time.Second)
}
```

A much better approach is to use a `sync.WaitGroup` to wait for the goroutine to finish. 

```go
package main
import (
    "fmt"
    "sync"
)
func sayHello(wg *sync.WaitGroup) {
    defer wg.Done() // Mark this goroutine as done when it finishes
    fmt.Println("Hello, World!")
}

func main() {
    var wg sync.WaitGroup
    wg.Add(1) // We have one goroutine to wait for
    go sayHello(&wg)
    wg.Wait() // Wait for all goroutines to finish
}
```

Note that ´defer´ is called when the function exits, whether by reaching the end of the function or by a return statement. This is useful for ensuring that resources are cleaned up properly, even if an error occurs. 

# Synchronization and channels
Like other concurrent programming models, goroutines need a way to synchronize their actions. A standard way to do this is by using mutexes. A mutex is a mutual exclusion lock that can be used to protect shared data from being accessed by multiple goroutines at the same time. Another way to synchronize goroutines is by using channels. Channels are a way to communicate between goroutines and can be used to send and receive values. 

## Race conditions and mutexes
Let first create a race condition by incrementing a shared counter from multiple goroutines without synchronization: A race condition occurs when two or more goroutines access shared data and try to change it at the same time. If one goroutine is in the middle of changing the data, and another goroutine reads or writes to the same data, it can lead to inconsistent or unexpected results.

```go
package main

import "sync"

var counter int
var wg sync.WaitGroup

func increment() {
	defer wg.Done()
	for i := 0; i < 1000; i++ {
		counter++
	}
}

func main() {
	wg.Add(2)
	go increment()
	go increment()
	wg.Wait()
	
    fmt.Println("Final Counter:", counter)
}
```

Since, the counter variable accessed simultaneously by two goroutines, the final value of counter may be less than 2000 due to lost updates. Go has built-in support for detecting race conditions, by using the `-race` flag when running or building the program: 

```console
go run -race main.go
==================
WARNING: DATA RACE
Read at 0x000102f6caa8 by goroutine 5:
  main.increment()
      /Users/johan/Development/github/eislab-cps/d7024e/tutorials/3-go-routines/example1/main.go:11 +0x8c

Previous write at 0x000102f6caa8 by goroutine 6:
  main.increment()
      /Users/johan/Development/github/eislab-cps/d7024e/tutorials/3-go-routines/example1/main.go:11 +0xa4

Goroutine 5 (running) created at:
  main.main()
      /Users/johan/Development/github/eislab-cps/d7024e/tutorials/3-go-routines/example1/main.go:17 +0x38
§
Goroutine 6 (finished) created at:
  main.main()
      /Users/johan/Development/github/eislab-cps/d7024e/tutorials/3-go-routines/example1/main.go:18 +0x44
==================
==================
WARNING: DATA RACE
Write at 0x000102f6caa8 by goroutine 5:
  main.increment()
      /Users/johan/Development/github/eislab-cps/d7024e/tutorials/3-go-routines/example1/main.go:11 +0xa4

Previous write at 0x000102f6caa8 by goroutine 6:
  main.increment()
      /Users/johan/Development/github/eislab-cps/d7024e/tutorials/3-go-routines/example1/main.go:11 +0xa4

Goroutine 5 (running) created at:
  main.main()
      /Users/johan/Development/github/eislab-cps/d7024e/tutorials/3-go-routines/example1/main.go:17 +0x38

Goroutine 6 (finished) created at:
  main.main()
      /Users/johan/Development/github/eislab-cps/d7024e/tutorials/3-go-routines/example1/main.go:18 +0x44
==================
Found 2 data race(s)
exit status 66
```

Let's first solve the race conditions using a mutex:

```go   
package main
import (
    "fmt"
    "sync"
)

var counter int
var mu sync.Mutex
var wg sync.WaitGroup

func increment() {
    defer wg.Done()
    for i := 0; i < 1000; i++ {
        mu.Lock()   // Lock the mutex before accessing the counter
        counter++
        mu.Unlock() // Unlock the mutex after accessing the counter
    }
}

func main() {
    wg.Add(2)
    go increment()
    go increment()
    wg.Wait()
	
    fmt.Println("Final Counter:", counter)
}
```

No race conditions detected this time. 

```console
go run -race  main.go
Final Counter: 2000
```

However, note that Go's race detector is not guaranteed to find all race conditions on every run. Race detection depends on the actual timing and scheduling of goroutines during execution. A race condition might exist in your code but not be detected if the goroutines don't happen to access shared memory simultaneously during that particular run. This is why you should run your tests multiple times and under different conditions to increase the chances of detecting race conditions.

## Channels
Channels provide a way for goroutines to communicate with each other and synchronize their execution. A channel is a typed pipe through which you can send and receive values with the built-in channel operator, `<-`. Think of a channel as connecting goroutines - they can send values into one end of the channel and receive them from the other end.

Let's create a simple example where two goroutines communicate using a channel. One goroutine (producer) will send integers to the channel, and another goroutine (consumer) will receive those integers and print them.

```go
package main

import (
    "fmt"
    "sync"
)

func producer(ch chan int, wg *sync.WaitGroup) {
    defer wg.Done()
    defer close(ch) // Close channel when done producing
    
    for i := 1; i <= 5; i++ {
        fmt.Printf("Sending: %d\n", i)
        ch <- i // Send value to channel
    }
}

func consumer(ch chan int, wg *sync.WaitGroup) {
    defer wg.Done()
   
      for {
        value, ok := <-ch // Receive value and check if channel is open
        if !ok {
            fmt.Println("Channel closed, stopping consumer")
            break
        }
        fmt.Printf("Received: %d\n", value)
    }
}

func main() {
    ch := make(chan int)
    var wg sync.WaitGroup
    
    wg.Add(2)
    go producer(ch, &wg)
    go consumer(ch, &wg)
    
    wg.Wait()
    fmt.Println("Communication complete!")
}
```

```console
go run main.go
Sending: 1
Sending: 2
Received: 1
Received: 2
Sending: 3
Sending: 4
Received: 3
Received: 4
Sending: 5
Received: 5
Channel closed, stopping consumer
Communication complete!
```

In this example, the `producer` goroutine sends integers 1 to 5 into the channel `ch`, and the `consumer` goroutine receives those integers from the channel and prints them. The channel is closed by the producer once it has finished sending all values, which signals the consumer to stop receiving.

### Buffered vs unbuffered channels
Channels can be buffered or unbuffered. By default, channels are **unbuffered** when created with `make(chan int)`, which is equivalent to `make(chan int, 0)` - meaning they have zero capacity for storing values.

**Unbuffered channels** (capacity 0) require both a sender and a receiver to be ready at the same time. The producer cannot send a value until there's a consumer ready to receive it at that exact moment. There's no storage - the value goes directly from sender to receiver. This creates synchronous communication where both goroutines must "meet" at the channel for the exchange to happen.

**Buffered channels** have capacity to store one or more values. The producer can send values even when no consumer is waiting, as long as there's space in the buffer. Once the buffer is full, the producer will block until a consumer receives a value, making space in the buffer.

```go
// Unbuffered channel (default) - zero capacity
ch1 := make(chan int)     // Same as make(chan int, 0)
ch2 := make(chan int, 0)  // Explicitly zero capacity

// Buffered channels with different capacities
ch3 := make(chan int, 1)  // Can store 1 value before blocking
ch4 := make(chan int, 3)  // Can store 3 values before blocking
```

**Important deadlock warning:**
```go
// Unbuffered - this would deadlock in a single goroutine
ch1 := make(chan int)
ch1 <- 42 // BLOCKS forever - no receiver ready

// Buffered with capacity 1 - this works fine
ch2 := make(chan int, 1) 
ch2 <- 42 // Doesn't block - value stored in buffer
value := <-ch2 // Gets the stored value
```

- **Unbuffered (capacity 0)**: Producer blocks until consumer is ready to receive (synchronous)
- **Buffered (capacity > 0)**: Producer only blocks when buffer is full (asynchronous until full)

This makes unbuffered channels perfect for synchronization and ensuring goroutines coordinate their actions, while buffered channels are useful for decoupling producer and consumer timing and handling bursts of data. 

Try changing the channel in the previous example to a buffered channel with capacity 5:

```console
go run main.go
Sending: 1
Sending: 2
Sending: 3
Sending: 4
Sending: 5
Received: 1
Received: 2
Received: 3
Received: 4
Received: 5
Channel closed, stopping consumer
Communication complete!
```

Why is the print out in different order? Because the producer can send all 5 values into the buffered channel without waiting for the consumer to receive them, allowing it to finish sending before the consumer starts receiving.

### Select statement
The `select` statement lets a goroutine wait on multiple communication operations. A `select` blocks until one of its cases can run, then it executes that case. It chooses randomly if multiple cases are ready simultaneously. Think of it like a `switch` statement but for channels.

**Basic example with multiple workers:**
```go
package main
import (
    "fmt"
    "math/rand"
    "sync"
    "time"
)

func worker(id int, ch chan string, wg *sync.WaitGroup) {
    defer wg.Done()
    time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
    ch <- fmt.Sprintf("Worker %d done", id)
}

func main() {
    // Use buffered channels to prevent deadlock
    ch1 := make(chan string, 1) // Buffer size 1
    ch2 := make(chan string, 1) // Buffer size 1
    var wg sync.WaitGroup
    
    wg.Add(2)
    go worker(1, ch1, &wg)
    go worker(2, ch2, &wg)
    
    // Wait for any worker to finish first
    select {
    case msg1 := <-ch1:
        fmt.Println("First response from ch1:", msg1)
    case msg2 := <-ch2:
        fmt.Println("First response from ch2:", msg2)
    }
    
    wg.Wait() // Wait for all workers to finish
    fmt.Println("All workers completed")
}
```

**Note:** With unbuffered channels, the `select` statement only receives from one channel, leaving the other worker blocked forever trying to send its message. By using buffered channels with capacity 1, each worker can send its message without blocking, even if no receiver is immediately ready. The worker that doesn't get selected by the `select` can still complete because its message gets stored in the buffer. If we instead used unbuffered channels, the program would deadlock because one worker would be blocked trying to send its message while the other worker's message is being received.

## Timeouts with select
Another common pattern is to use `select` with a timeout to avoid waiting indefinitely:

```go   
package main
import (
    "fmt"
    "time"
)

func main() {
    ch := make(chan string)
    
    go func() {
        time.Sleep(2 * time.Second) // Simulate work
        ch <- "Result from goroutine"
    }()
    
    select {
    case res := <-ch:
        fmt.Println("Received:", res)
    case <-time.After(1 * time.Second):
        fmt.Println("Timeout: no response within 1 second")
    }
}
```

```console
go run -race main.go
Timeout: no response within 1 second
```

This is crucial for operations like HTTP requests, database queries, or file I/O that might hang or take longer than expected. Without timeouts, a single slow operation could freeze your entire program. It's also essential in testing to prevent test suites from hanging indefinitely on broken code.

# Testing concurrent code
Testing concurrent code can be challenging due to its non-deterministic nature. Here are some strategies and tools to help you effectively test goroutines and concurrent programs in Go.

The race detector is your first line of defense against concurrency bugs. Always run your tests with the `-race` flag:

```bash
go test -race ./...
```

## Testing with Channels
Use channels to coordinate and verify behavior in tests. Let's create a simple function and test it with multiple workers:

Save this file as `./pkg/alg/alg.go`:
```go
package alg

// NumberDoubler takes numbers from an input channel,
// doubles each number, and sends the result to an output channel
func NumberDoubler(input <-chan int, output chan<- int) {
    // Process each number that comes in
    for number := range input {
        doubled := number * 2
        output <- doubled
    }
}
```

Save this file as `./pkg/alg/alg_test.go`
```go
package alg

import (
    "slices"
    "sort"
    "testing"
    "time"
)

func TestMultipleNumberDoublers(t *testing.T) {
    // Step 1: Create channels for communication
    inputChannel := make(chan int, 5)
    outputChannel := make(chan int, 5)
    
    // Step 2: Start 3 NumberDoublers working concurrently
    numWorkers := 3
    for i := 0; i < numWorkers; i++ {
        go NumberDoubler(inputChannel, outputChannel)
    }
    
    // Step 3: Send test numbers - workers will grab them unpredictably
    testNumbers := []int{1, 2, 3, 4, 5}
    for _, number := range testNumbers {
        inputChannel <- number
    }
    close(inputChannel) // Signal that we're done sending numbers
    
    // Step 4: Collect all results (order will be unpredictable)
    var actualResults []int
    expectedCount := len(testNumbers)
    
    for i := 0; i < expectedCount; i++ {
        select {
        case result := <-outputChannel:
            actualResults = append(actualResults, result)
        case <-time.After(1 * time.Second):
            t.Fatalf("Test timed out waiting for result %d", i+1)
        }
    }
    
    // Step 5: Sort both slices before comparing (since order is unpredictable)
    sort.Ints(actualResults)
    expectedResults := []int{2, 4, 6, 8, 10}
    sort.Ints(expectedResults) // Already sorted, but good practice
    
    if !slices.Equal(actualResults, expectedResults) {
        t.Errorf("Expected %v, got %v", expectedResults, actualResults)
    }
}
```

**Note about order:** With multiple workers processing the same input channel, results come back in unpredictable order because workers compete to grab numbers from the input channel. That's why we collect all results first, then sort both the actual and expected results before comparing them. This is a common pattern when testing concurrent code with non-deterministic ordering.

Another approach is to save results in a hashed structure like a map or set, which inherently ignores order, then compare the contents. Here's an alternative version using a map:

```go
func TestMultipleNumberDoublersWithMap(t *testing.T) {
    inputChannel := make(chan int, 5)
    outputChannel := make(chan int, 5)
    
    // Start 3 NumberDoublers
    numWorkers := 3
    for i := 0; i < numWorkers; i++ {
        go NumberDoubler(inputChannel, outputChannel)
    }
    
    // Send test numbers
    testNumbers := []int{1, 2, 3, 4, 5}
    for _, number := range testNumbers {
        inputChannel <- number
    }
    close(inputChannel)
    
    // Create map of expected results
    expectedResults := map[int]bool{2: true, 4: true, 6: true, 8: true, 10: true}
    
    // Check each result and mark as received
    for i := 0; i < len(testNumbers); i++ {
        result := <-outputChannel
        if !expectedResults[result] {
            t.Errorf("Unexpected result: %d", result)
        } else {
            delete(expectedResults, result) // Mark as received
        }
    }
    
    // Check that we got all expected results
    if len(expectedResults) > 0 {
        remaining := make([]int, 0, len(expectedResults))
        for result := range expectedResults {
            remaining = append(remaining, result)
        }
        t.Errorf("Missing expected results: %v", remaining)
    }
}
```

## Using Timeouts in Tests
Always use timeouts to prevent tests from hanging indefinitely:

```go
func TestWithTimeout(t *testing.T) {
    done := make(chan bool)
    
    go func() {
        // Simulate work that might hang
        time.Sleep(100 * time.Millisecond)
        done <- true
    }()
    
    select {
    case <-done:
        // Test passed
    case <-time.After(1 * time.Second):
        t.Fatal("Test timed out")
    }
}

## Best Practices for Testing Concurrent Code
1. **Always use `-race` flag** when running tests
2. **Use timeouts** to prevent hanging tests
3. **Test both success and failure scenarios**
4. **Use multiple runs** to catch intermittent issues
5. **Keep tests simple** - complex concurrent tests are hard to debug
6. **Use channels for coordination** rather than sleep statements
7. **Test goroutine cleanup** to prevent leaks
8. **Make tests as deterministic as possible** with proper synchronization

## Running Tests Multiple Times
Since concurrent bugs can be intermittent, always run tests multiple times:

```bash
go test -race -count=100 .
```

# Advanced Patterns
## Contexts for cancellation and timeouts
The `context` package provides a way to carry deadlines, cancellation signals, and other request-scoped values across API boundaries and between goroutines. It is commonly used in concurrent programming to manage the lifecycle of goroutines and ensure they can be cancelled or timed out appropriately.

Context can be used to coordinate graceful shutdown of multiple goroutines. When you cancel a context, all goroutines listening to that context receive the cancellation signal and can clean up properly. Here's a simple example demonstrating how to use `context` to manage goroutine cancellation:

```go   
package main
import (
    "context"
    "fmt"
    "sync"
    "time"
)

func worker(ctx context.Context, id int, wg *sync.WaitGroup) {
    defer wg.Done() // Signal completion when function exits
    
    for {
        select {
        case <-ctx.Done():
            fmt.Printf("Worker %d: stopping due to context cancellation\n", id)
            return
        default:
            fmt.Printf("Worker %d: working...\n", id)
            time.Sleep(500 * time.Millisecond) // Simulate work
        }
    }
}

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    var wg sync.WaitGroup
    
    // Start multiple workers
    for i := 1; i <= 3; i++ {
        wg.Add(1)
        go worker(ctx, i, &wg)
    }
    
    // Let them work for 2 seconds
    time.Sleep(2 * time.Second)
    
    // Cancel the context to stop all workers
    fmt.Println("Main: cancelling context")
    cancel()
    
    // Wait for all workers to properly finish
    wg.Wait()
    fmt.Println("Main: all workers have finished gracefully")
}
```

```console
go run main.go
Worker 3: working...
Worker 2: working...
Worker 1: working...
Worker 2: working...
Worker 1: working...
Worker 3: working...
Worker 1: working...
Worker 2: working...
Worker 3: working...
Worker 3: working...
Worker 2: working...
Worker 1: working...
Main: cancelling context
Worker 1: stopping due to context cancellation
Worker 3: stopping due to context cancellation
Worker 2: stopping due to context cancellation
Main: all workers have finished gracefully
```

Another use case is to set timeouts for operations. You can create a context with a timeout, and if the operation takes longer than the specified duration, it will be automatically cancelled:

```go
package main

import (
    "context"
    "fmt"
    "time"
)

func main() {
    // Create a context with a 1 second timeout
    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
    defer cancel() // Ensure resources are cleaned up
    
    // Simulate a long-running operation
    select {
    case <-time.After(2 * time.Second): // Simulate work taking 2 seconds
        fmt.Println("Operation completed")
    case <-ctx.Done(): // Context timeout or cancellation
        fmt.Println("Operation timed out:", ctx.Err())
    }
}
```

```console
go run main.go
Operation timed out: context deadline exceeded
```

Context are commonly used in network and database protocols, for example:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context() // Get request context
    
    // This will be cancelled if client disconnects
    select {
    case <-time.After(2 * time.Second):
        w.Write([]byte("Response"))
    case <-ctx.Done():
        // Client disconnected, stop processing
        return
    }
}
```

## Worker pools
A worker pool is a collection of goroutines that can execute tasks concurrently. This pattern is useful for limiting the number of concurrent operations, managing resource usage, and improving performance by reusing goroutines instead of creating new ones for each task.

Here's a simple implementation of a worker pool in Go:

```go
package main
import (
    "fmt"
    "sync"
    "time"
)   
type Task struct {
    id int
}
func worker(id int, tasks <-chan Task, wg *sync.WaitGroup) {
    defer wg.Done()
    for task := range tasks {
        fmt.Printf("Worker %d: processing task %d\n", id, task.id)
        time.Sleep(1 * time.Second) // Simulate work
    }
}

func main() {
    const numWorkers = 3
    const numTasks = 10

    tasks := make(chan Task, numTasks)
    var wg sync.WaitGroup

    // Start worker goroutines
    for i := 1; i <= numWorkers; i++ {
        wg.Add(1)
        go worker(i, tasks, &wg)
    }

    // Send tasks to the workers
    for i := 1; i <= numTasks; i++ {
        tasks <- Task{id: i}
    }
    close(tasks) // Close the channel to signal no more tasks

    wg.Wait() // Wait for all workers to finish
    fmt.Println("All tasks completed")
}
```

```console
go run main.go
Worker 3: processing task 1
Worker 2: processing task 3
Worker 1: processing task 2
Worker 1: processing task 4
Worker 2: processing task 6
Worker 3: processing task 5
Worker 3: processing task 7
Worker 1: processing task 8
Worker 2: processing task 9
Worker 1: processing task 10
All tasks completed
```

Worker pools are highly useful in Kademlia for managing the numerous concurrent network operations. For example, when handling incoming requests, you can use a worker pool to process each request concurrently while limiting the total number of active requests to avoid overwhelming the node.

```go
func (k *KademliaNode) handleIncomingRequests() {
    requests := make(chan IncomingRPC, 100)
    
    // Worker pool for processing incoming RPCs
    for i := 0; i < 10; i++ {
        go k.rpcWorker(requests)
    }
    
    // Network listener feeds requests to workers
    for {
        conn, err := k.listener.Accept()
        if err != nil {
            continue
        }
        
        rpc := parseIncomingRPC(conn)
        select {
        case requests <- rpc:
        default:
            // Drop request if workers are overwhelmed
            rpc.reject("server busy")
        }
    }
}

func (k *KademliaNode) rpcWorker(requests <-chan IncomingRPC) {
    for rpc := range requests {
        switch rpc.Type {
        case "FIND_NODE":
            k.handleFindNode(rpc)
        case "FIND_VALUE":
            k.handleFindValue(rpc)
        case "STORE":
            k.handleStore(rpc)
        case "PING":
            k.handlePing(rpc)
        }
    }
}
```

## Fan-out, fan-in
Fan-out, fan-in is a concurrency pattern where multiple goroutines (workers) process tasks concurrently (fan-out), and their results are collected into a single channel (fan-in). This pattern is useful for parallelizing work and aggregating results efficiently.

Here's an example demonstrating the fan-out, fan-in pattern:

```go   
package main
import (
    "fmt"
    "math/rand"
    "sync"
    "time"
)

func worker(id int, jobs <-chan int, results chan<- int, wg *sync.WaitGroup) {
    defer wg.Done()
    for job := range jobs {
        fmt.Printf("Worker %d: processing job %d\n", id, job)
        time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond) // Simulate variable work time
        results <- job * 2 // Send result to results channel
    }
}

func main() {
    const numWorkers = 3
    const numJobs = 5

    jobs := make(chan int, numJobs)
    results := make(chan int, numJobs)
    var wg sync.WaitGroup

    // Start worker goroutines (fan-out)
    for i := 1; i <= numWorkers; i++ {
        wg.Add(1)
        go worker(i, jobs, results, &wg)
    }

    // Send jobs to the workers
    for j := 1; j <= numJobs; j++ {
        jobs <- j
    }
    close(jobs) // Close the jobs channel to signal no more jobs

    // Wait for all workers to finish processing
    go func() {
        wg.Wait()
        close(results) // Close results channel when all workers are done
    }()

    // Collect results (fan-in)
    for result := range results {
        fmt.Printf("Received result: %d\n", result)
    }

    fmt.Println("All jobs processed")
}
```

```console
go run main.go
Worker 2: processing job 2
Worker 1: processing job 1
Worker 3: processing job 3
Worker 1: processing job 4
Received result: 2
Worker 2: processing job 5
Received result: 4
Received result: 6
Received result: 8
Received result: 10
All jobs processed
```

The fan-out/fan-in pattern is particularly valuable for implementing Kademlia DHT operations because many DHT operations require contacting multiple nodes simultaneously and aggregating their responses. For example, in Kademlia, when storing a key-value pair, the protocol requires replicating the *k* closest nodes to the key's hash to ensure availability and fault tolerance. Rather than storing sequentially, which would be slow and create timing windows where data could be lost, the fan-out pattern enables parallel storage operations. For example:

```go
func store(key string, value []byte, targetNodes []Node) error {
    results := make(chan StoreResult, len(targetNodes))
    
    // Fan-out: initiate storage on all target nodes simultaneously
    for _, node := range targetNodes {
        go func(n Node) {
            start := time.Now()
            err := n.Store(key, value) // Network RPC call
            results <- StoreResult{
                Node: n,
                Error: err,
                Duration: time.Since(start),
            }
        }(node)
    }
    
    // Fan-in: collect results and evaluate success criteria
    successCount := 0
    var failures []error
    totalLatency := time.Duration(0)
    
    for i := 0; i < len(targetNodes); i++ {
        result := <-results
        totalLatency += result.Duration
        
        if result.Error == nil {
            successCount++
            log.Printf("Successfully stored on node %s", result.Node.ID)
        } else {
            failures = append(failures, result.Error)
            log.Printf("Failed to store on node %s: %v", result.Node.ID, result.Error)
        }
    }
    
    // Evaluate success based on replication requirements
    minReplicas := len(targetNodes)/2 + 1 // Majority
    if successCount >= minReplicas {
        log.Printf("Store successful: %d/%d replicas created", successCount, len(targetNodes))
        return nil
    }
    
    return fmt.Errorf("insufficient replicas: %d/%d successful, errors: %v", 
        successCount, len(targetNodes), failures)
}
```
