// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package keccak

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

// Unit test for keccakHasher function.
func TestKeccakHasher(t *testing.T) {

	// Create a block to verify
	hash := hexutil.MustDecode("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	nonce := uint64(12345678)

	wantDigest := hexutil.MustDecode("0xeffd292d6666dba4d6a4c221dd6d4b34b4ec3972a4cb0d944a8a8936cceca713")
	wantResult := hexutil.MustDecode("0xeffd292d6666dba4d6a4c221dd6d4b34b4ec3972a4cb0d944a8a8936cceca713")

	digest, result := keccakHasher(hash, nonce)
	if !bytes.Equal(digest, wantDigest) {
		t.Errorf("keccak digest mismatch: have %x, want %x", digest, wantDigest)
	}
	if !bytes.Equal(result, wantResult) {
		t.Errorf("keccak result mismatch: have %x, want %x", result, wantResult)
	}
}
