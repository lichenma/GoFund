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

> Benchmarks are like unit tests, but include a loop which runs the same code many times (in our case `fund.Withdraw(1)`). This allows the framework to time how long each iteration takes, 