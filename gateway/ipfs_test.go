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

package gateway_test

import (
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/crypto"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/market/util"
	"github.com/Loopring/relay/test"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ipfs/go-ipfs-api"
	"math"
	"math/big"
	"testing"
	"time"
)

const (
	suffix       = "0000000000000000" //0.01
	TOKEN_SYMBOL = "LRC"
	WETH         = "WETH"
)

func TestSingleOrder(t *testing.T) {
	c := test.Cfg()
	entity := test.Entity()

	// get keystore and unlock account
	tokenAddressA := util.AllTokens[TOKEN_SYMBOL].Protocol
	tokenAddressB := util.AllTokens[WETH].Protocol
	testAcc := entity.Accounts[0]
	walletId := big.NewInt(1)
	privkey := entity.PrivateKey

	ks := keystore.NewKeyStore(c.Keystore.Keydir, keystore.StandardScryptN, keystore.StandardScryptP)
	account := accounts.Account{Address: testAcc.Address}
	ks.Unlock(account, testAcc.Passphrase)
	cyp := crypto.NewKSCrypto(true, ks)
	crypto.Initialize(cyp)

	// set order and marshal to json
	protocol := common.HexToAddress(c.Common.ProtocolImpl.Address[test.Version])

	amountS1, _ := new(big.Int).SetString("1"+suffix, 0)
	amountB1, _ := new(big.Int).SetString("10"+suffix, 0)
	lrcFee1 := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(20))
	order := test.CreateOrder(
		privkey,
		walletId,
		tokenAddressA,
		tokenAddressB,
		protocol,
		account.Address,
		amountS1,
		amountB1,
		lrcFee1,
	)
	bs, _ := order.MarshalJSON()

	// get ipfs shell and sub order
	sh := shell.NewShell(c.Ipfs.Url())
	pubMessage(sh, string(bs))
}

func TestRing(t *testing.T) {
	const orderPairNumber = 1

	c := test.Cfg()
	entity := test.Entity()

	// get ipfs shell and sub order
	sh := shell.NewShell(c.Ipfs.Url())
	lrc := util.SupportTokens[TOKEN_SYMBOL].Protocol

	eth := util.SupportMarkets[WETH].Protocol

	account1 := entity.Accounts[0]
	account2 := entity.Accounts[1]

	privkey := entity.PrivateKey

	// set order and marshal to json
	protocol := common.HexToAddress(c.Common.ProtocolImpl.Address[test.Version])
	for i := 0; i < orderPairNumber; i++ {
		walletId := big.NewInt(int64(i))

		// 卖出0.1个eth， 买入300个lrc,lrcFee为20个lrc
		amountS1, _ := new(big.Int).SetString("1"+suffix, 0)
		amountB1, _ := new(big.Int).SetString("1000"+suffix, 0)
		lrcFee1 := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(10)) // 20个lrc
		order1 := test.CreateOrder(
			privkey,
			walletId,
			eth,
			lrc,
			protocol,
			account1.Address,
			amountS1,
			amountB1,
			lrcFee1,
		)
		bs1, _ := order1.MarshalJSON()

		// 卖出1000个lrc,买入0.1个eth,lrcFee为20个lrc
		amountS2, _ := new(big.Int).SetString("2000"+suffix, 0)
		amountB2, _ := new(big.Int).SetString("1"+suffix, 0)
		lrcFee2 := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(5))
		order2 := test.CreateOrder(
			privkey,
			walletId,
			lrc,
			eth,
			protocol,
			account2.Address,
			amountS2,
			amountB2,
			lrcFee2,
		)
		bs2, _ := order2.MarshalJSON()

		pubMessage(sh, string(bs1))
		pubMessage(sh, string(bs2))
	}
}

func TestBatchRing(t *testing.T) {
	c := test.Cfg()
	entity := test.Entity()

	lrc := util.SupportTokens[TOKEN_SYMBOL].Protocol
	eth := util.SupportMarkets[WETH].Protocol

	account1 := entity.Accounts[0]
	account2 := entity.Accounts[1]

	walletId := big.NewInt(1)
	privkey := entity.PrivateKey

	// set order and marshal to json
	protocol := common.HexToAddress(c.Common.ProtocolImpl.Address[test.Version])

	// 卖出0.1个eth， 买入300个lrc,lrcFee为20个lrc
	amountS1, _ := new(big.Int).SetString("10"+suffix, 0)
	amountB1, _ := new(big.Int).SetString("30000"+suffix, 0)
	lrcFee1 := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(5)) // 20个lrc
	order1 := test.CreateOrder(
		privkey,
		walletId,
		eth,
		lrc,
		protocol,
		account1.Address,
		amountS1,
		amountB1,
		lrcFee1,
	)
	bs1, _ := order1.MarshalJSON()

	// 卖出0.1个eth， 买入200个lrc,lrcFee为20个lrc
	amountS3, _ := new(big.Int).SetString("10"+suffix, 0)
	amountB3, _ := new(big.Int).SetString("20000"+suffix, 0)
	lrcFee3 := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(5)) // 20个lrc
	order3 := test.CreateOrder(
		privkey,
		walletId,
		eth,
		lrc,
		protocol,
		account1.Address,
		amountS3,
		amountB3,
		lrcFee3,
	)
	bs3, _ := order3.MarshalJSON()

	// 卖出1000个lrc,买入0.1个eth,lrcFee为20个lrc
	amountS2, _ := new(big.Int).SetString("100000"+suffix, 0)
	amountB2, _ := new(big.Int).SetString("10"+suffix, 0)
	lrcFee2 := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(5))
	order2 := test.CreateOrder(
		privkey,
		walletId,
		lrc,
		eth,
		protocol,
		account2.Address,
		amountS2,
		amountB2,
		lrcFee2,
	)
	bs2, _ := order2.MarshalJSON()

	// get ipfs shell and sub order
	sh := shell.NewShell(c.Ipfs.Url())
	pubMessage(sh, string(bs1))
	pubMessage(sh, string(bs3))
	pubMessage(sh, string(bs2))
}

func TestPrepareProtocol(t *testing.T) {
	test.PrepareTestData()
}

func TestPrepareAccount(t *testing.T) {
	test.SetTokenBalances()
}

func TestAllowance(t *testing.T) {
	test.AllowanceToLoopring(nil, nil)
}

func pubMessage(sh *shell.Shell, data string) {
	c := test.Cfg()
	topic := c.Ipfs.BroadcastTopics[0]
	err := sh.PubSubPublish(topic, data)
	if err != nil {
		panic(err.Error())
	}
}

func MatchTestPrepare() (*config.GlobalConfig, *test.TestEntity) {
	c := test.Cfg()
	entity := test.Entity()
	testAcc1 := entity.Accounts[0]
	testAcc2 := entity.Accounts[1]
	password1 := entity.Accounts[0].Passphrase
	password2 := entity.Accounts[1].Passphrase

	// get keystore and unlock account
	ks := keystore.NewKeyStore(entity.KeystoreDir, keystore.StandardScryptN, keystore.StandardScryptP)
	acc1 := accounts.Account{Address: testAcc1.Address}
	acc2 := accounts.Account{Address: testAcc2.Address}

	ks.Unlock(acc1, password1)
	ks.Unlock(acc2, password2)

	cyp := crypto.NewKSCrypto(true, ks)
	crypto.Initialize(cyp)
	return c, entity
}

//test the amount and discount
func TestMatcher_Case1(t *testing.T) {
	c, entity := MatchTestPrepare()

	tokenAddressA := util.SupportTokens["LRC"].Protocol
	tokenAddressB := util.SupportMarkets["WETH"].Protocol

	tokenCallMethodA := ethaccessor.ContractCallMethod(ethaccessor.Erc20Abi(), tokenAddressA)
	tokenCallMethodB := ethaccessor.ContractCallMethod(ethaccessor.Erc20Abi(), tokenAddressB)
	var tokenAmountA types.Big
	var tokenAmountB types.Big
	tokenCallMethodA(&tokenAmountA, "balanceOf", "latest", entity.Accounts[0].Address)
	tokenCallMethodB(&tokenAmountB, "balanceOf", "latest", entity.Accounts[0].Address)
	t.Logf("before match, addressA:%s -> tokenA:%s, amount:%s", entity.Accounts[0].Address.Hex(), tokenAddressA.Hex(), tokenAmountA.BigInt().String())
	t.Logf("before match, addressA:%s -> tokenB:%s, amount:%s", entity.Accounts[0].Address.Hex(), tokenAddressB.Hex(), tokenAmountB.BigInt().String())

	// set order and marshal to json
	protocol := common.HexToAddress(c.Common.ProtocolImpl.Address[test.Version])

	amountS1, _ := new(big.Int).SetString("1"+suffix, 0)
	amountB1, _ := new(big.Int).SetString("10"+suffix, 0)
	lrcFee1 := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(20))
	walletId := big.NewInt(1)
	privkey := entity.PrivateKey

	order1 := test.CreateOrder(
		privkey,
		walletId,
		tokenAddressA,
		tokenAddressB,
		protocol,
		entity.Accounts[0].Address,
		amountS1,
		amountB1,
		lrcFee1,
	)
	bs1, _ := order1.MarshalJSON()

	amountS2, _ := new(big.Int).SetString("10"+suffix, 0)
	amountB2, _ := new(big.Int).SetString("1"+suffix, 0)
	lrcFee2 := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(20))
	order2 := test.CreateOrder(
		privkey,
		walletId,
		tokenAddressB,
		tokenAddressA,
		protocol,
		entity.Accounts[1].Address,
		amountS2,
		amountB2,
		lrcFee2,
	)
	bs2, _ := order2.MarshalJSON()

	// get ipfs shell and sub order
	sh := shell.NewShell(c.Ipfs.Url())
	pubMessage(sh, string(bs1))
	pubMessage(sh, string(bs2))

	//waiting for the result of match and submit ring,

	time.Sleep(time.Minute)
	var tokenAmountAAfterMatch types.Big
	var tokenAmountBAfterMatch types.Big
	tokenCallMethodA(&tokenAmountAAfterMatch, "balanceOf", "latest", entity.Accounts[0].Address)
	tokenCallMethodB(&tokenAmountBAfterMatch, "balanceOf", "latest", entity.Accounts[0].Address)
	t.Logf("before match, addressA:%s -> tokenA:%s, amount:%s", entity.Accounts[0].Address.Hex(), tokenAddressA.Hex(), tokenAmountAAfterMatch.BigInt().String())
	t.Logf("before match, addressA:%s -> tokenB:%s, amount:%s", entity.Accounts[0].Address.Hex(), tokenAddressB.Hex(), tokenAmountBAfterMatch.BigInt().String())

}

//test account lrcFee insufficient
func TestMatcher_Case2(t *testing.T) {
	c, entity := MatchTestPrepare()

	tokenAddressA := util.SupportTokens["EOS"].Protocol
	tokenAddressB := util.SupportMarkets["WETH"].Protocol

	tokenCallMethodA := ethaccessor.ContractCallMethod(ethaccessor.Erc20Abi(), tokenAddressA)
	tokenCallMethodB := ethaccessor.ContractCallMethod(ethaccessor.Erc20Abi(), tokenAddressB)
	var tokenAmountA types.Big
	var tokenAmountB types.Big
	tokenCallMethodA(&tokenAmountA, "balanceOf", "latest", entity.Accounts[0].Address)
	tokenCallMethodB(&tokenAmountB, "balanceOf", "latest", entity.Accounts[0].Address)
	t.Logf("before match, addressA:%s -> tokenA:%s, amount:%s", entity.Accounts[0].Address.Hex(), tokenAddressA.Hex(), tokenAmountA.BigInt().String())
	t.Logf("before match, addressA:%s -> tokenB:%s, amount:%s", entity.Accounts[0].Address.Hex(), tokenAddressB.Hex(), tokenAmountB.BigInt().String())

	// set order and marshal to json
	protocol := common.HexToAddress(c.Common.ProtocolImpl.Address[test.Version])

	amountS1, _ := new(big.Int).SetString("10"+suffix, 0)
	amountB1, _ := new(big.Int).SetString("100"+suffix, 0)
	lrcFee1 := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(20))
	walletId := big.NewInt(1)
	privkey := entity.PrivateKey

	order1 := test.CreateOrder(
		privkey,
		walletId,
		tokenAddressA,
		tokenAddressB,
		protocol,
		entity.Accounts[0].Address,
		amountS1,
		amountB1,
		lrcFee1,
	)
	bs1, _ := order1.MarshalJSON()
	println(string(bs1))

	amountS2, _ := new(big.Int).SetString("50"+suffix, 0)
	amountB2, _ := new(big.Int).SetString("5"+suffix, 0)
	lrcFee2 := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(20))
	order2 := test.CreateOrder(
		privkey,
		walletId,
		tokenAddressB,
		tokenAddressA,
		protocol,
		entity.Accounts[1].Address,
		amountS2,
		amountB2,
		lrcFee2,
	)
	bs2, _ := order2.MarshalJSON()
	//bs3, _ := order2.MarshalJSON()

	// get ipfs shell and sub order
	sh := shell.NewShell(c.Ipfs.Url())
	//pubMessage(sh, string(bs1))
	pubMessage(sh, string(bs2))

	//waiting for the result of match and submit ring,

	var tokenAmountAAfterMatch types.Big
	var tokenAmountBAfterMatch types.Big
	tokenCallMethodA(&tokenAmountAAfterMatch, "balanceOf", "latest", entity.Accounts[0].Address)
	tokenCallMethodB(&tokenAmountBAfterMatch, "balanceOf", "latest", entity.Accounts[0].Address)
	t.Logf("before match, addressA:%s -> tokenA:%s, amount:%s", entity.Accounts[0].Address.Hex(), tokenAddressA.Hex(), tokenAmountAAfterMatch.BigInt().String())
	t.Logf("before match, addressA:%s -> tokenB:%s, amount:%s", entity.Accounts[0].Address.Hex(), tokenAddressB.Hex(), tokenAmountBAfterMatch.BigInt().String())
}

//test account balance insufficient
func TestMatcher_Case3(t *testing.T) {
	amountS, _ := new(big.Int).SetString("10000000000", 0)
	amountB, _ := new(big.Int).SetString("70000000000000008", 0)

	//price := 1428.5714285714284

	tokenSAddress := common.HexToAddress("0x419D0d8BdD9aF5e606Ae2232ed285Aff190E711b")
	tokenBAddress := common.HexToAddress("0x2956356cD2a2bf3202F771F50D3D14A367b48070")

	tokenS, _ := util.AddressToToken(tokenSAddress)
	tokenB, _ := util.AddressToToken(tokenBAddress)

	price := new(big.Rat).Mul(
		new(big.Rat).SetFrac(amountS, amountB),
		new(big.Rat).SetFrac(tokenB.Decimals, tokenS.Decimals),
	)

	t.Logf(price.FloatString(6))
}

//test multi orders
func TestMatcher_Case4(t *testing.T) {

}

//test multi round
func TestMatcher_Case5(t *testing.T) {
	num := int64(math.Pow(10.0, 18.0))
	ret := new(big.Rat).SetInt64(num).FloatString(0)
	t.Log(ret)

	//
	v := new(big.Rat).SetFrac(big.NewInt(1), big.NewInt(1))
	v.Quo(big.NewRat(1, 1), v)
	t.Log(v.FloatString(18))
}
