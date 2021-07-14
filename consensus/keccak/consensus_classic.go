// Copyright 2019 The multi-geth Authors
// This file is part of the multi-geth library.
//
// The multi-geth library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The multi-geth library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the multi-geth library. If not, see <http://www.gnu.org/licenses/>.
package keccak

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params/types/ctypes"
	"github.com/ethereum/go-ethereum/params/vars"
)

func ecip1017BlockReward(config ctypes.ChainConfigurator, header *types.Header, uncles []*types.Header) (*big.Int, []*big.Int) {
	blockReward := vars.FrontierBlockReward

	// Ensure value 'era' is configured.
	eraLen := config.GetEthashECIP1017EraRounds()
	era := GetBlockEra(header.Number, new(big.Int).SetUint64(*eraLen))
	wr := GetBlockWinnerRewardByEra(era, blockReward)                    // wr "winner reward". 5, 4, 3.2, 2.56, ...
	wurs := GetBlockWinnerRewardForUnclesByEra(era, uncles, blockReward) // wurs "winner uncle rewards"
	wr.Add(wr, wurs)

	// Reward uncle miners.
	uncleRewards := make([]*big.Int, len(uncles))
	for i, uncle := range uncles {
		ur := GetBlockUncleRewardByEra(era, header, uncle, blockReward)
		uncleRewards[i] = ur
	}

	return wr, uncleRewards
}

func ecip1010Explosion(config ctypes.ChainConfigurator, next *big.Int, exPeriodRef *big.Int) {
	// https://github.com/ethereumproject/ECIPs/blob/master/ECIPs/ECIP-1010.md

	if next.Uint64() < *config.GetEthashECIP1010ContinueTransition() {
		exPeriodRef.SetUint64(*config.GetEthashECIP1010PauseTransition())
	} else {
		length := new(big.Int).SetUint64(*config.GetEthashECIP1010ContinueTransition() - *config.GetEthashECIP1010PauseTransition())
		exPeriodRef.Sub(exPeriodRef, length)
	}
}

// GetBlockEra gets which "Era" a given block is within, given an era length (ecip-1017 has era=5,000,000 blocks)
// Returns a zero-index era number, so "Era 1": 0, "Era 2": 1, "Era 3": 2 ...
func GetBlockEra(blockNum, eraLength *big.Int) *big.Int {
	// If genesis block or impossible negative-numbered block, return zero-val.
	if blockNum.Sign() < 1 {
		return new(big.Int)
	}

	remainder := big.NewInt(0).Mod(big.NewInt(0).Sub(blockNum, big.NewInt(1)), eraLength)
	base := big.NewInt(0).Sub(blockNum, remainder)

	d := big.NewInt(0).Div(base, eraLength)
	dremainder := big.NewInt(0).Mod(d, big.NewInt(1))

	return new(big.Int).Sub(d, dremainder)
}
