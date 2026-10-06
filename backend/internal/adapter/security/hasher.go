// Package security contains the concrete implementations of the cryptographic
// and identity ports. Nothing here is referenced by the service layer except
// through port.PasswordHasher / port.TokenIssuer / port.IDGenerator.
package security

import (
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/holihur/openshop/internal/port"
)

// BcryptHasher implements port.PasswordHasher using bcrypt.
type BcryptHasher struct {
	cost int
}

func NewBcryptHasher() *BcryptHasher { return &BcryptHasher{cost: bcrypt.DefaultCost} }

func (h *BcryptHasher) Hash(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), h.cost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (h *BcryptHasher) Compare(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// UUIDGenerator implements port.IDGenerator with random v4 UUIDs.
type UUIDGenerator struct{}

func NewUUIDGenerator() *UUIDGenerator { return &UUIDGenerator{} }

func (UUIDGenerator) NewID() string { return uuid.NewString() }

var (
	_ port.PasswordHasher = (*BcryptHasher)(nil)
	_ port.IDGenerator    = UUIDGenerator{}
)
