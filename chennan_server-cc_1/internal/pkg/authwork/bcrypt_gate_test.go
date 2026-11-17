package authwork

import (
	"errors"
	"testing"
)

func TestBcryptCapacityIsPartitionedBetweenPublicAndPrivateWork(t *testing.T) {
	if cap(publicBcryptGate) < 1 || cap(privateBcryptGate) < 1 {
		t.Fatalf("invalid gate capacities: public=%d private=%d", cap(publicBcryptGate), cap(privateBcryptGate))
	}
	if got := cap(publicBcryptGate) + cap(privateBcryptGate); got != bcryptTotalConcurrency {
		t.Fatalf("partition total=%d, want %d", got, bcryptTotalConcurrency)
	}
	if cap(privateBcryptGate) > 2 {
		t.Fatalf("private capacity=%d, want at most 2", cap(privateBcryptGate))
	}
}

func TestPrivateGateFailsFastWithoutConsumingPublicCapacity(t *testing.T) {
	for i := 0; i < cap(privateBcryptGate); i++ {
		privateBcryptGate <- struct{}{}
	}
	t.Cleanup(func() {
		for len(privateBcryptGate) > 0 {
			<-privateBcryptGate
		}
	})

	if err := ComparePrivateHashAndPassword([]byte("unused"), []byte("unused")); !errors.Is(err, ErrBusy) {
		t.Fatalf("private full error=%v, want ErrBusy", err)
	}
	if got := len(publicBcryptGate); got != 0 {
		t.Fatalf("private saturation consumed %d public slots", got)
	}
}
