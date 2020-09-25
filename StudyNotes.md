# Go Programming Language: An Introductory Golang Tutorial

Credits to Brendon at Toptal for creating an awesome golang [tutorial](https://www.toptal.com/go/go-programming-a-step-by-step-introductory-tutorial)

## What is the Go Programming Language? 

The relatively new Go programming language compiles fast, runs fast-ish, includes a runtime and garbage collection, has a simple static type system and dynamic interfaces and an excellent standard library - these are some of the factors which make many developers keen to learn Go programming 

OOP is one of the features that Go deliberately omits, it provides no subclassing and there are no inheritance diamonds or super calls or virtual methods to trip developers up. There are many benefits to OOP and they are still available in the Go language through other ways such as Mixins

## Why Learn Golang? 

Go is an amazing language for writing concurrent programs: programs with many independently running instances. An obvious example is a webserver: Every request runs separately but requests often need to share resources such as sessions, caches, or notification queues. This requires programmers to deal with concurrent access to these resources. 

Golang has an excellent set of low-level features for handling concurrency, however, using them directly can be complicated and in many cases it would be useful to use abstractions over these low-level mechanisms to make programming easier. 

In this introductory project, we are going to look at one abstraction: Wrapper that turns any data structures into a transactional service. We will be going through a `Fund` type as an examples - a simple data store for our startup's remaining funding where we can check the balance and make withdrawls. 

In order to achieve this end goal wil will build up the service though a series of small steps cleaning things up along hte way. As we progress though the tutorial, we will encounter lots of cool Go language features including 

* Struct types and methods 
* Unit tests and benchmarks 
* Goroutines and channels 
* Interfaces and dynamic typing 


## Building a Simple Fund 

Let's write some code to track the startup's funding. The fund starts with a given balance and money can only be withdrawn (we will leave off deposits for now)

```
                 SIMPLE GOROUTINE 
    CALLER      --- withdraw() --->      FUND 
``` 

Go is deliberates **not** an object-oriented language: There are no classes, objects, or inheritance. Instead we will declare ***struct type*** called `Fund`, with a simple function to create new fund structs and two public methods 

```go 
// fund.go
package funding 

type Fund struct {
    // balance is unexported (private), because it is lowercase 
    balance int
}

// Regular function returning point to a fund 
fund NewFund(initialBalance int) *Fund {
    // do not need to worry about whether the pointer is on the 
    // stack vs heap: Go will figure that out for us 
    return &Fund{
        balance: initialBalance,
    }
}

// Methods start with a *receiver*, in this case a Fund pointer
func (f *Fund) Balance() int {
    return f.balance
}

func (f *Fund) Withdraw(amount int){
    f.balance -= amount
}
```

## Testing With Benchmarks 

Next we need to a way to test `Fund`. Rather than writing a separate program, we will use Go's testing package which provides a framework for both unit tests and benchmarks. The simple logic in our `Fund` is not really worth writing unit tests for but since we will be talking a lot about concurrent access to the fund later on - it makes sense to write up a ***benchmark***. 

> Benchmarks are like unit tests, but include a loop which runs the same code many times (in our case `fund.Withdraw(1)`). This allows the framework to time how long each iteration takes, averaging out transient differences from disk seeks, cache misses, process scheduling, and other unpredicable factors. 

The testing framework wants each benchmark to run for at least 1 second (by default). To ensure this, it will call the benchmark multiple times, passing in an increasing number of iterations value (`b.N`) each time until the run takes at least a second.

For now, the benchmark will just deposit some money and then withdraw it one dollar at a time. 

```go 
// fund_test.go

package funding 

import "testing" 

func BenchmarkFund(b *testing.B) {
    // Add as much initial funding as we have iterations in the run 
    fund := NewFund(b.N)

    // Burn through them one at a time until they are all gone

    for i := 0; i< b.N; i++ {
        fund.Withdraw(1)
    }

    if fund.Balance() != 0 {
        b.Error("Balance wasn't zero:", fund.Balance())
    }
}
```


Now we can run it: 

```bash 
$ go test -bench=.
testing: warning: no tests to run 
PASS 
BenchmarkWithdrawls     20000000000     1.69 ns/op
ok      funding     3.576s     
```

## Concurrent Access in Go 

Now lets try making the benchmark concurrent, to model different users making withdrawls at the same time. In order to do that we will spawn ten goroutines and have each of them withdraw on tenth of the money 


`Goroutines` are the basic building block for concurrency in the Go language. They are green threads - lightweight threads managed by the Go runtime, not by the operating system. This means that you can run thousands (or millions) of them without any significant overhead. Goroutines are spawned with the `go` keyword, and always start with a function (or method call): 

```go 
// returns immediately, without waiting for `DoSomething()` to complete
go DoSomething()
```

Often, we want to spawn off a short one-time function with just a few lines of code. In this case we can use a closure instead of a function name: 

```go 
go func() {
    // do stuff ...
}() // Must be a function call --> ()
```

Once all of our goroutines are spawned, we need a way to wait for them to finish. We could build one ourselves using `channels`, but for now we will just use the `WaitGroup` type in Go's standard library, which exists for this very purpose. We will create one (called `wg`) and call `wg.Add(1)` before spawning each worker, to keep track of how many there are. Then the workers will report back using `wg.Done()`. Meanwhile in the main goroutine, we can just say `wg.Wait()` to block until every worker has finished. 

Inside the worker goroutines in our next example, we will use `defer` to call `wg.Done()`. 

`defer` takes a function (or method) call and runs it immediately before the current function returns and after everything else is done. This is handy for cleanup: 

```go 
func() {
    resource.Lock()
    defer resouce.Unlock()

    // Do stuff with the resource
}()
```

This way we can easily match the `Unlock` with its `Lock` for readability. More importantly, a deferred function will run even if there is a panic in the main function (soemthing usually handled via try-finally/ensure in other languages)


Lastly, deferred functions will execute in the ***reverse*** (like a stack) order to which they were called, meaning we can do a nested cleanup nicely (similar to the C idiom of nested `goto`s and `label`s but much neater)


```go 
func() {
    db.Connect()
    defer db.Disconnect()

    // If Begin panics, only db.Disconnect() will execute
    transaction.Begin()
    defer transaction.Close()

    // From here on transaction.Close() will run before db.Disconnect()
}()
```

With these functions, here is the new implementation: 

```go 
// fund_test.go

package funding 

import (
    "sync" 
    "testing" 
)

const WORKERS = 10 

func BenchmarkWithdrawls(b *testing.B) {
    // Skip N = 1 
    if b.N < WORKERS {
        return 
    }

    // Add as many dollars as we have iterations this run 
    fund := NewFund(b.N)

    // assume b.N divides cleanly (what happens with division in golang?)
    dollarsPerFounder := b.N / WORKERS 

    // WaitGroup structs do not need to be initialized we can just delare one and then use it
    var wg sync.WaitGroup 

    for i := 0; i < WORKERS; i++ {
        // let the waitgroup know that we are adding a goroutine
        wg.Add(1)

        // Spawn off a founder worker, as a closure 
        go func() {
            // Mark this worker done when the function finishes
            defer wg.Done()

            for i := 0; i < dollarsPerFounder; i++ {
                fund.Withdraw(1)
            }
        }() // Remember to call the closure
    }

    // Wait for all the workers to finish 
    wg.Wait()

    if fund.Balance() != 0 {
        b.Error("Balance wasn't zero:", fund.Balance())
    }
} 
```

We can estimate what will happen given these test conditions - the workers will execute `withdraw` on top of each other. Inside it, `f.balance -= amount` will read the balance, subtract one, and then write it back. But sometimes two or more workers will both read the same balance, and do the same subtraction, and we end up with the wrong total ... right? 

When we run the program we see that it still passes - remember that goroutines are `green threads`: managed by the Go runtime not the Operating System. The runtime schedules goroutines across however many OS threads it has available. At the time the tutorial was created - Go does not try to guess how many OS threads it should use and if we want more than one we have to specify so. Finally the current runtime does not preempt (interrupt a task being carried out, without requiring its cooperation and with the intention of resuming the task at a later time) goroutines - a goroutine will continue to run until it does something that suggests its ready for a break (like interacting with a channel). 

All of this means that our benchmark is now concurrent but it is not **parallel**. Only one of our workers will run at a time, and it will run until it is done. We can change this by telling Go to use more threads, via the `GOMAXPROCS` environment variable. 

```bash 
$ GOMAXPROCS=4 go test -bench=.
goos: darwin
goarch: amd64
BenchmarkWithdrawls-4   	--- FAIL: BenchmarkWithdrawls-4
    fund_test.go:44: Balance wasn't zero: 2409
FAIL
exit status 1
FAIL	_/Users/lichenma/Projects/GoFund/GoFund	0.265s
```

Now in this case we are losing some of our withdrawls, as expected. 


## Make it a Server 

At this point we have various options. We can add an explicit mutex or read-write lock around the fund. We could use a compare - and - swap with a version number. We could go all out and use a `CRDT` (Commutative Replicated Data Type - a data type whose operations commute when they are concurrent) scheme (perhaps replacing the `balance` field with lists of transactions for each client and calculating the balance from those). 


### Side Investigation on CRDT 

Commutative Replicated Data Types are a data type whose operations commute when they are concurrent. 

From the book `CRDTs: Consistency without concurrency control` there is a nice concise overview of distributed consistency issues: 

> Shared read-only data is easy to scale by using well-understood replication techniques. However, sharing mutable data at a large scale is a difficult problem because of the `CAP impossibility result`

In theoretical computer science, the CAP theorem also named Brewer's theorem states that it is impossible for a distributed data store to simultaneously provide more than two of the following three guarantees 

* **Consistency**: Every read receives the most recent write or an error 
* **Availability**: Every request receives a (non-error) response, witout the guarantee that it contains the most recent write 
* **Partition Tolerance**: The system continues to operate despite an arbitraty number of messages being dropped (or delayed) by the network between nodes 


When a network partition failure happens should we decide to 

- Cancel the operation and thus decrease the availability but ensure consistency
- Proceed with the operation and thus provide availability but risk inconsistency 

The CAP theorem implies that in the presence of network partition, one has to choose between consistency and availability. 


``` 
Note that the distributed CAP principles are different 
from the ACID database transaction principles: 

Atomic:     Must be complete in its entirety or have no 
            effect whatsoever 

Consistent: Must conform to existing constraints in the database

Isolated:   Must not affect other transactions 

Durable:    must get written to persistent storage 
``` 


> In order to deal with the CAP impossibility result two approaches dominate in practice. One ensures scalability by giving up consistency guarantees (Last - Writer - Wins approach). The alternative guarantees consistency by serialising all updates, which does not scale beyond a small cluster. Optimistic replication allows replicas to diverge, eventually resolving conflicts either by LWW - like methods or by serialisation. 

> In some limited cases, a radical simplification is possible. If concurrent updates to some datum commute, and all of its replicates excutes all updates in causal order, then the replicas converge. We call this a Commutative Replicate Data Type (CRDT) - this approach ensures that there are no conflicts, hence, no need for consensus based concurrency control. CRDTs are not a univeral solution but can prove to be highly useful - ensures consistency in the large scale at a low costs. 

> A trivial example of a CRDT is a set with a single add-element operation. A delete-element operation can be emulated by adding "deleted" elements to a second set. This suffices to implement a mailbox but is not practical as the data structures grow without bound

> One non-trivial, useful and practical CRDT is one that implements an ordered set with insert-at-position and delete operations. It is called `Treedoc` because the sequence elements are identified compactly using a naming tree and because its first use was concurrent document editing


## Make it a Server (continued)

We could go all out and use a CRDT scheme perhaps replacing the `balance` field with lists of transactions for each client and calculating the balance from those. 

For the purposes of this tutorial we will not attempt any of them now because they are messy or scary or both. Instead we will decide that the fund should be a `server` ... what is a server? It is something that you talk to and in golang things talk via channels. 


> Channels are the basic communication mechanism between goroutines (seems to be like a pipe from Waterloo OS course)

Values sent to the channel (with `channel <- value`), and can be received on the other side (with `value = <- channel`). Channels are "goroutine safe" meaning that any number of goroutines can send to and receive from them at the same time. 

By default, Go channels are ***unbuffered***. This means that sending a value to a channel will `block` until another goroutine is ready to receive it immediately. Go also supports fixed buffer sizes for channels (using `make(chan someType, bufferSize)`). However, for normal use this is ususally not the way to go. 



#### Buffering 

Buffering communication channels can be a performance optimization in certain circumstances, but it should be used with great care (and benchmarking). 

An unbuffered channel provides a guarantee that an exchange between two goroutines is performed at the instant the send and receive take place - buffered channel has no such guarantee. Data are passed around on channels such that only one goroutine has access to a data item at any given time - data races cannot occur by design. An unbuffered channel is used to perform `synchronous communication` between goroutines while a buffered channel is used to perform `asynchronous communication`.

There are uses for buffered channels which aren't directly about communication. For example, a common throttling idiom creates a channel with (for example) buffer size `10` and then sends ten tokens into it immediately. Any number of worker goroutines are then spawned and each receives a token from the channel before starting work, and sends it back afterward. Then, however many workers there are, only ten will ever be working at the same time. 




<br>



Imagine a webserver for our fund, where each request makes a withdrawal. When things are very busy, the `FundServer` won't be able to keep up, and requests trying to send to its command channel will start to block and wait. At that point we can enforce a maximum request count in the server, and return a sensible error code (like a `503 Service Unavailable`) to clients over that limit. This is the best behavior possible when the server is overloaded. 


Adding buffering to our channels would make this behavior less deterministic. We could easily end up with long queues of unprocessed commands based on information the client saw much earlier (and perhaps for requests which had since timed out upstream). The same applies in many other situations like applying backpressure over TCP when the receiver can't keep up with the sender. 

For the purposes of this Go example, we will stick with default unbuffered behavior. 

We will use a channel to send commands to the `FundServer`. Every benchmark worker will send commands to the channel, but only the server will receive them. 

We could makes the changes to the Fund type directly but that would make things messy and combine concurrency handling and business logic all in that class. Instead we will leave the Fund type as it is and make `FundServer` a separate wrapper around it. 

This wrapper will contain a main loop which waits for commands, and responds to them in turn. 


## Interfaces 

At this point, we want to send several commands (Withdraw, Balance) each with their own struct type. The server should be able to respond to any of them - this behavior is usually achieved in OOP language through polymorphism: superclass and subclass definitions for each different struct input. In Go, the method of choice is `interfaces`. 

> Note: We could have made our commands channel take pointers to commands (`chan *TransactionCommand`) so why was this approach not used? Passing pointer between goroutines is risky since both goroutines might modify it. It is often less efficient since the other routine may be running on a different CPU core (increasing cache invalidation). When possible - pass plain values 

An interface is a set of method signatures. Any type that implements all of the methods can be treated as that interface (without requiring formal declaration). In our first implementation we will use command structs which implement the empty interface, `interface{}`. Since there are no requirements, any value can be treated as that interface. Clearly this is not ideal - we should only accept command structs - but we will revisit this later. 

The scaffolding for the Go server will look like this: 

```go 
// server.go 
package funding 

type FundServer struct {
    Commands chan interface{}
    fund Fund
}

func NewFundServer(initialBalance int) *FundServer {
    server := &FundServer{
        // make() creates builtin like channels, maps and slices 
        Commands: make(chan interface{}),
        fund: NewFund(initialBalance),
    }

    // spawn the server main loop immediately
    go server.loop()
    return server
}

func (s *FundServer) loop() {
    // built-in range clause can iterate over channels 
    for command := range s.Commands {
        // Handle the command
    }
}

```

Now let's introduce some Golang struct types for the commands: 

```go 
type WithdrawCommand struct {
    Amount int
}

type BalanceCommand struct {
    Response chan int
}
```

The `WithdrawCommand` contains only the amount to withdraw. Meanwhile the `BalanceCommand` has a response, so it includes the channel to send it on. This will ensure that the responses will always end up going to the right place even if the fund decides to respond out-of-order. 

Now we can write up the server's main loop:

```go 
func (s *FundServer) loop() {
    for comand := range s.Commands {

        // command is just an interface{} but we can check its corresponding type
        switch command.(type) {
            case WithdrawCommand: 
                // use a "type assertion" 
                withdrawl := command.(WithdrawCommand)
                s.fund.Withdraw(withdrawal.Amount)
            
            case BalanceCommand: 
                getBalance := command.(BalanceCommand)
                balance := s.fund.Balance()
                getBalance.Response <- balance
            
            default: 
                panic(fmt.Sprintf("Unrecognized Command: %v", command))
        }
    }
}
```

The code written is kind of ugly but lets try running the benchmark: 



