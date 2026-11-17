// Package authwork bounds CPU-expensive authentication work without holding
// permits during database, Turnstile, Tencent IM, or other network operations.
package authwork

import (
	"errors"
	"runtime"

	"golang.org/x/crypto/bcrypt"
)

var ErrBusy = errors.New("authentication cpu capacity exhausted")

const DefaultCost = bcrypt.DefaultCost

var bcryptTotalConcurrency = bcryptConcurrency()

var (
	publicBcryptGate  = make(chan struct{}, publicBcryptConcurrency(bcryptTotalConcurrency))
	privateBcryptGate = make(chan struct{}, privateBcryptConcurrency(bcryptTotalConcurrency))
)

func bcryptConcurrency() int {
	capacity := runtime.GOMAXPROCS(0)
	if capacity < 2 {
		return 2
	}
	if capacity > 8 {
		return 8
	}
	return capacity
}

func privateBcryptConcurrency(total int) int {
	if total >= 8 {
		return 2
	}
	return 1
}

func publicBcryptConcurrency(total int) int {
	capacity := total - privateBcryptConcurrency(total)
	if capacity < 1 {
		return 1
	}
	return capacity
}

func acquire(gate chan struct{}) (func(), error) {
	select {
	case gate <- struct{}{}:
		return func() { <-gate }, nil
	default:
		return nil, ErrBusy
	}
}

func CompareHashAndPassword(hashedPassword, password []byte) error {
	release, err := acquire(publicBcryptGate)
	if err != nil {
		return err
	}
	defer release()
	return bcrypt.CompareHashAndPassword(hashedPassword, password)
}

func GenerateFromPassword(password []byte, cost int) ([]byte, error) {
	release, err := acquire(publicBcryptGate)
	if err != nil {
		return nil, err
	}
	defer release()
	return bcrypt.GenerateFromPassword(password, cost)
}

// ComparePrivateHashAndPassword reserves a small, separate capacity slice for
// authenticated password-management endpoints. A valid user/admin token cannot
// consume the capacity reserved for public login and recovery.
func ComparePrivateHashAndPassword(hashedPassword, password []byte) error {
	release, err := acquire(privateBcryptGate)
	if err != nil {
		return err
	}
	defer release()
	return bcrypt.CompareHashAndPassword(hashedPassword, password)
}

// GeneratePrivateFromPassword is the authenticated counterpart of
// GenerateFromPassword and uses only the private capacity slice.
func GeneratePrivateFromPassword(password []byte, cost int) ([]byte, error) {
	release, err := acquire(privateBcryptGate)
	if err != nil {
		return nil, err
	}
	defer release()
	return bcrypt.GenerateFromPassword(password, cost)
}
