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

package ctypes

import (
	"math/big"

	"github.com/ethereum/go-ethereum/params/vars"
)

func KeccakBlockReward(c ChainConfigurator, n *big.Int) *big.Int {
	// Select the correct block reward based on chain progression
	blockReward := vars.FrontierBlockReward
	if c == nil || n == nil {
		return blockReward
	}

	if c.IsEnabled(c.GetEthashEIP1234Transition, n) {
		return vars.EIP1234FBlockReward
	} else if c.IsEnabled(c.GetEthashEIP649Transition, n) {
		return vars.EIP649FBlockReward
	} else if len(c.GetEthashBlockRewardSchedule()) > 0 {
		// Because the map is not necessarily sorted low-high, we
		// have to ensure that we're walking upwards only.
		var lastActivation uint64
		for activation, reward := range c.GetEthashBlockRewardSchedule() {
			if activation <= n.Uint64() { // Is forked
				if activation >= lastActivation {
					lastActivation = activation
					blockReward = reward
				}
			}
		}
	}

	return blockReward
}
