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
    // balanace is unexported (private), because it is lowercase 
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

Once all of our goroutines are spawned, we need a way to wait for them to finish. We could one ourselves using `channels`, but for now we will just use the `WaitGroup` type in Go's standard library, which exists for this very purpose. We will create one (called `wg`) and call `wg.Add(1)` before spawning each worker, to keep track of how many there are. Then the workers will report back using `wg.Done()`. Meanwhile in the main goroutine, we can just say `wg.Wait()` to block until every worker has finished. 

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

When we run the program we see that it still passes - remember that goroutines are `green threads`: managed by the Go runtime not the Operating System. The runtime schedules goroutines across however many OS threads it has available. At the time the tutorial was created - Go does not try to guess how many OS threads it should use and we want more than one we have to specify so. Finally the current runtime does not preempt (interrupt a task being carried out, without requiring its cooperation and with the intention of resuming the task at a later time) goroutines - a goroutine will continue to run until it does something that suggests its ready for a break (like interacting with a channel). 

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

At this point we have various options. We can add an explicit mutex or read-write lock around the fund. We could use a compare - and - swap with a version number. We could go all out and use a `CRDT` (Commutative Replicated Data Type - a data type whose operations commute when they are concurrent) scheme ()





