/*

  Copyright 2017 Loopring Project Ltd (Loopring Foundation).

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.

*/

package ordermanager_test

import (
	"github.com/Loopring/relay/test"
	"github.com/ethereum/go-ethereum/common"
	"testing"
)

func TestOrderManagerImpl_MinerOrders(t *testing.T) {
	entity := test.Entity()

	om := test.GenerateOrderManager()
	protocol := test.Protocol()
	tokenS := entity.Tokens["LRC"]
	tokenB := entity.Tokens["WETH"]

	states := om.MinerOrders(protocol, tokenS, tokenB, 10, 10, 20, nil)
	for k, v := range states {
		t.Logf("list number %d, order.hash %s", k, v.RawOrder.Hash.Hex())
		t.Logf("list number %d, order.tokenS %s", k, v.RawOrder.TokenS.Hex())
		t.Logf("list number %d, order.price %s", k, v.RawOrder.Price.String())
	}
}

func TestOrderManagerImpl_GetOrderByHash(t *testing.T) {
	om := test.GenerateOrderManager()
	states, _ := om.GetOrderByHash(common.HexToHash("0xaaa99b5c64fe1f6ae594994d1f6c252dc49c2d0db6bb185df99f5ffa8de64fdb"))

	t.Logf("order.hash %s", states.RawOrder.Hash.Hex())
	t.Logf("order.tokenS %s", states.RawOrder.TokenS.Hex())
}

func TestOrderManagerImpl_GetOrderBook(t *testing.T) {
	om := test.GenerateOrderManager()
	protocol := common.HexToAddress("0x03E0F73A93993E5101362656Af1162eD80FB54F2")
	tokenS := common.HexToAddress("0x2956356cD2a2bf3202F771F50D3D14A367b48070")
	tokenB := common.HexToAddress("0x86Fa049857E0209aa7D9e616F7eb3b3B78ECfdb0")
	list, err := om.GetOrderBook(protocol, tokenS, tokenB, 100)
	if err != nil {
		panic(err)
	}

	for _, v := range list {
		t.Logf("orderhash", v.RawOrder.Hash.Hex())
	}
}
