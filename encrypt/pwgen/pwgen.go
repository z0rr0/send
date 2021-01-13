package pwgen

// Package pwgen is a helper to generate readable random passwords.

import (
	crand "crypto/rand"
	"encoding/binary"
	"math/rand"
)

// Alphabet is all allowed for password generation symbols
const Alphabet = `#%&+-3479@CFHJKLMNPRTVWXbcdfghjkmnpqrstvwxz`

// CryptoRandSource represents a source of uniformly-distributed random int64 values in the range [0, 1<<63).
type CryptoRandSource struct{}

// Int63 returns a non-negative random 63-bit integer as an int64 from CryptoRandSource.
func (CryptoRandSource) Int63() int64 {
	var b [8]byte
	_, err := crand.Read(b[:])
	if err != nil {
		// fail - can't continue
		// there is no possibility to return an error
		panic(err)
	}
	return int64(binary.LittleEndian.Uint64(b[:]) & (1<<63 - 1))
}

// Seed is fake CryptoRandSource Seed implementation for Source interface.
func (CryptoRandSource) Seed(int64) {}

// New returns a new random string with length `n`.
func New(n int) string {
	source := &CryptoRandSource{}
	random := rand.New(source)

	lenAlphabet := len(Alphabet)
	container := make([]byte, n)

	for i := range container {
		container[i] = Alphabet[random.Intn(lenAlphabet)]
	}
	return string(container)
}
