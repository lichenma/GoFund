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