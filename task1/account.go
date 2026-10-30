package task1

import "sync"

type Account struct {
	mu      sync.Mutex
	balance int
	hook    func() // test-only: runs between check and update
}

func NewAccount(balance int) *Account {
	return &Account{
		balance: balance,
	}
}

func (a *Account) Withdraw(amount int) bool {
	a.mu.Lock()

	if a.balance < amount {
		a.mu.Unlock()
		return false
	}

	a.mu.Unlock()

	// BUG:
	// The check and update are not atomic.

	if a.hook != nil {
		a.hook()
	}


	a.mu.Lock()
	a.balance -= amount
	a.mu.Unlock()

	return true
}

func (a *Account) Balance() int {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.balance
}