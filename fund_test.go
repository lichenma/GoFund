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

    server := NewFundServer(b.N)

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
                server.Commands <- WithdrawCommand{ Amount: 1 }
            }
        }() // Remember to call the closure
    }

    // Wait for all the workers to finish 
    wg.Wait()

    balanceResponseChan := make(chan int)
    server.Commands <- BalanceCommand{ Response: balanceResponseChan }
    balance := <- balanceResponseChan

    if balance != 0 {
        b.Error("Balance wasn't zero:", balance)
    }
} 