package task1

import (
	"sync"
	"testing"
)

// Looks fine: usually passes, even with -race
func TestWithdrawNormal(t *testing.T) {
	a := NewAccount(100)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.Withdraw(100)
		}()
	}
	wg.Wait()
	if a.Balance() < 0 {
		t.Fatalf("negative balance: %d", a.Balance())
	}
}

// Forces the bad schedule: both goroutines pass the check first
func TestWithdrawForcedInterleaving(t *testing.T) {
	a := NewAccount(100)

	var barrier sync.WaitGroup
	barrier.Add(2)
	a.hook = func() {
		barrier.Done()
		barrier.Wait() // both wait here until both have checked
	}

	var wg sync.WaitGroup
	results := make([]bool, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i] = a.Withdraw(100)
		}(i)
	}
	wg.Wait()

	if results[0] && results[1] {
		t.Errorf("both withdrawals succeeded on a balance of 100")
	}
	if a.Balance() < 0 {
		t.Errorf("balance went negative: %d", a.Balance())
	}
}