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

package miner

import (
	"errors"
	"github.com/Loopring/relay/log"
	"math"
	"math/big"

	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/marketcap"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
)

type Evaluator struct {
	marketCapProvider         marketcap.MarketCapProvider
	rateRatioCVSThreshold     int64
	gasUsedWithLength         map[int]*big.Int
	realCostRate, walletSplit *big.Rat
}

func (e *Evaluator) ComputeRing(ringState *types.Ring) error {

	productAmountS := big.NewRat(int64(1), int64(1))
	productAmountB := big.NewRat(int64(1), int64(1))

	//compute price
	for _, order := range ringState.Orders {
		amountS := new(big.Rat).SetInt(order.OrderState.RawOrder.AmountS)
		amountB := new(big.Rat).SetInt(order.OrderState.RawOrder.AmountB)

		productAmountS.Mul(productAmountS, amountS)
		productAmountB.Mul(productAmountB, amountB)

		order.SPrice = new(big.Rat)
		order.SPrice.Quo(amountS, amountB)

		order.BPrice = new(big.Rat)
		order.BPrice.Quo(amountB, amountS)
	}

	productPrice := new(big.Rat)
	productPrice.Quo(productAmountS, productAmountB)
	//todo:change pow to big.Int
	priceOfFloat, _ := productPrice.Float64()
	rootOfRing := math.Pow(priceOfFloat, 1/float64(len(ringState.Orders)))
	rate := new(big.Rat).SetFloat64(rootOfRing)
	ringState.ReducedRate = new(big.Rat)
	ringState.ReducedRate.Inv(rate)
	log.Debugf("Miner,rate:%s, priceFloat:%f , len:%d, rootOfRing:%f, reducedRate:%s ", rate.String(), priceOfFloat, len(ringState.Orders), rootOfRing, ringState.ReducedRate.RatString())

	//todo:get the fee for select the ring of mix income
	//LRC等比例下降，首先需要计算fillAmountS
	//分润的fee，首先需要计算fillAmountS，fillAmountS取决于整个环路上的完全匹配的订单
	//如何计算最小成交量的订单，计算下一次订单的卖出或买入，然后根据比例替换
	minVolumeIdx := 0

	for idx, filledOrder := range ringState.Orders {
		filledOrder.SPrice.Mul(filledOrder.SPrice, ringState.ReducedRate)

		filledOrder.BPrice.Inv(filledOrder.SPrice)

		amountS := new(big.Rat).SetInt(filledOrder.OrderState.RawOrder.AmountS)
		//amountB := new(big.Rat).SetInt(filledOrder.OrderState.RawOrder.AmountB)

		//根据用户设置，判断是以卖还是买为基准
		//买入不超过amountB
		filledOrder.RateAmountS = new(big.Rat).Set(amountS)
		filledOrder.RateAmountS.Mul(amountS, ringState.ReducedRate)
		//if BuyNoMoreThanAmountB , AvailableAmountS need to be reduced by the ratePrice
		//if filledOrder.OrderState.RawOrder.BuyNoMoreThanAmountB {
		//	availbleAmountB := new(big.Rat).Set(filledOrder.AvailableAmountB)
		//	availableAmountS := new(big.Rat).Mul(filledOrder.RateAmountS, availbleAmountB)
		//	availableAmountS.Quo(availableAmountS, amountB)
		//	if filledOrder.AvailableAmountB.Cmp(new(big.Rat).SetInt(filledOrder.OrderState.RawOrder.AmountB)) < 0 {
		//		filledOrder.AvailableAmountS.Set(availableAmountS)
		//	}
		//}

		//与上一订单的买入进行比较
		var lastOrder *types.FilledOrder
		if idx > 0 {
			lastOrder = ringState.Orders[idx-1]
		}

		filledOrder.FillAmountS = new(big.Rat)
		if lastOrder != nil && lastOrder.FillAmountB.Cmp(filledOrder.AvailableAmountS) >= 0 {
			//当前订单为最小订单
			filledOrder.FillAmountS.Set(filledOrder.AvailableAmountS)
			minVolumeIdx = idx
			//根据minVolumeIdx进行最小交易量的计算,两个方向进行
		} else if lastOrder == nil {
			filledOrder.FillAmountS.Set(filledOrder.AvailableAmountS)
		} else {
			//上一订单为最小订单需要对remainAmountS进行折扣计算
			filledOrder.FillAmountS.Set(lastOrder.FillAmountB)
		}
		filledOrder.FillAmountB = new(big.Rat).Mul(filledOrder.FillAmountS, filledOrder.BPrice)
	}

	//compute the volume of the ring by the min volume
	//todo:the first and the last
	//if (ring.RawRing.Orders[len(ring.RawRing.Orders) - 1].FillAmountB.Cmp(ring.RawRing.Orders[0].FillAmountS) < 0) {
	//	minVolumeIdx = len(ring.RawRing.Orders) - 1
	//	for i := minVolumeIdx-1; i >= 0; i-- {
	//		//按照前面的，同步减少交易量
	//		order := ring.RawRing.Orders[i]
	//		var nextOrder *types.FilledOrder
	//		nextOrder = ring.RawRing.Orders[i + 1]
	//		order.FillAmountB = nextOrder.FillAmountS
	//		order.FillAmountS.Mul(order.FillAmountB, order.EnlargedSPrice)
	//	}
	//}

	for i := minVolumeIdx - 1; i >= 0; i-- {
		//按照前面的，同步减少交易量
		order := ringState.Orders[i]
		var nextOrder *types.FilledOrder
		nextOrder = ringState.Orders[i+1]
		order.FillAmountB = nextOrder.FillAmountS
		order.FillAmountS.Mul(order.FillAmountB, order.SPrice)
	}

	for i := minVolumeIdx + 1; i < len(ringState.Orders); i++ {
		order := ringState.Orders[i]
		var lastOrder *types.FilledOrder
		lastOrder = ringState.Orders[i-1]
		order.FillAmountS = lastOrder.FillAmountB
		order.FillAmountB.Mul(order.FillAmountS, order.BPrice)
	}

	//compute the fee of this ring and orders, and set the feeSelection
	e.computeFeeOfRingAndOrder(ringState)

	//cvs
	cvs, err := PriceRateCVSquare(ringState)
	if nil != err {
		return err
	} else {
		if cvs.Int64() <= e.rateRatioCVSThreshold {
			return nil
		} else {
			return errors.New("Miner,cvs must less than RateRatioCVSThreshold")
		}
	}
}

func (e *Evaluator) computeFeeOfRingAndOrder(ringState *types.Ring) {

	for _, filledOrder := range ringState.Orders {
		var lrcAddress common.Address
		if implAddress, exists := ethaccessor.ProtocolAddresses()[filledOrder.OrderState.RawOrder.Protocol]; exists {
			lrcAddress = implAddress.LrcTokenAddress
		}

		//todo:成本节约
		legalAmountOfSaving := new(big.Rat)
		if filledOrder.OrderState.RawOrder.BuyNoMoreThanAmountB {
			amountS := new(big.Rat).SetInt(filledOrder.OrderState.RawOrder.AmountS)
			amountB := new(big.Rat).SetInt(filledOrder.OrderState.RawOrder.AmountB)
			sPrice := new(big.Rat)
			sPrice.Quo(amountS, amountB)
			savingAmount := new(big.Rat)
			savingAmount.Mul(filledOrder.FillAmountB, sPrice)
			savingAmount.Sub(savingAmount, filledOrder.FillAmountS)
			filledOrder.FeeS = savingAmount
			legalAmountOfSaving = e.getLegalCurrency(filledOrder.OrderState.RawOrder.TokenS, filledOrder.FeeS)
		} else {
			savingAmount := new(big.Rat).Set(filledOrder.FillAmountB)
			savingAmount.Mul(savingAmount, ringState.ReducedRate)
			savingAmount.Sub(filledOrder.FillAmountB, savingAmount)
			filledOrder.FeeS = savingAmount
			legalAmountOfSaving = e.getLegalCurrency(filledOrder.OrderState.RawOrder.TokenB, filledOrder.FeeS)
		}

		//compute lrcFee
		rate := new(big.Rat).Quo(filledOrder.FillAmountS, new(big.Rat).SetInt(filledOrder.OrderState.RawOrder.AmountS))
		filledOrder.LrcFee = new(big.Rat).SetInt(filledOrder.OrderState.RawOrder.LrcFee)
		filledOrder.LrcFee.Mul(filledOrder.LrcFee, rate)

		if filledOrder.AvailableLrcBalance.Cmp(filledOrder.LrcFee) <= 0 {
			filledOrder.LrcFee = filledOrder.AvailableLrcBalance
		}

		legalAmountOfLrc := e.getLegalCurrency(lrcAddress, filledOrder.LrcFee)
		log.Debugf("raw.lrc:%s, AvailableLrcBalance:%s, legalAmountOfLrc:%s saving:%s", filledOrder.OrderState.RawOrder.LrcFee.String(), filledOrder.AvailableLrcBalance.FloatString(0), legalAmountOfLrc.String(), legalAmountOfSaving.String())
		filledOrder.LegalLrcFee = legalAmountOfLrc

		splitPer := new(big.Rat).SetInt64(int64(filledOrder.OrderState.RawOrder.MarginSplitPercentage))
		legalAmountOfSaving.Mul(legalAmountOfSaving, splitPer)
		filledOrder.LegalFeeS = legalAmountOfSaving
	}

}

//成环之后才可计算能否成交，否则不需计算，判断是否能够成交，不能使用除法计算
func PriceValid(a2BOrder *types.OrderState, b2AOrder *types.OrderState) bool {
	amountS := new(big.Int).Mul(a2BOrder.RawOrder.AmountS, b2AOrder.RawOrder.AmountS)
	amountB := new(big.Int).Mul(a2BOrder.RawOrder.AmountB, b2AOrder.RawOrder.AmountB)
	return amountS.Cmp(amountB) >= 0
}

func PriceRateCVSquare(ringState *types.Ring) (*big.Int, error) {
	rateRatios := []*big.Int{}
	scale, _ := new(big.Int).SetString("10000", 0)
	for _, filledOrder := range ringState.Orders {
		rawOrder := filledOrder.OrderState.RawOrder
		s1b0, _ := new(big.Int).SetString(filledOrder.RateAmountS.FloatString(0), 10)
		//s1b0 = s1b0.Mul(s1b0, rawOrder.AmountB)

		s0b1 := new(big.Int).SetBytes(rawOrder.AmountS.Bytes())
		//s0b1 = s0b1.Mul(s0b1, rawOrder.AmountB)
		if s1b0.Cmp(s0b1) > 0 {
			return nil, errors.New("Miner,rateAmountS must less than amountS")
		}
		ratio := new(big.Int).Set(scale)
		ratio.Mul(ratio, s1b0).Div(ratio, s0b1)
		rateRatios = append(rateRatios, ratio)
	}
	return CVSquare(rateRatios, scale), nil
}

func CVSquare(rateRatios []*big.Int, scale *big.Int) *big.Int {
	avg := big.NewInt(0)
	length := big.NewInt(int64(len(rateRatios)))
	length1 := big.NewInt(int64(len(rateRatios) - 1))
	for _, ratio := range rateRatios {
		avg.Add(avg, ratio)
	}
	avg = avg.Div(avg, length)

	cvs := big.NewInt(0)
	for _, ratio := range rateRatios {
		sub := big.NewInt(0)
		sub.Sub(ratio, avg)

		subSquare := new(big.Int).Mul(sub, sub)
		cvs.Add(cvs, subSquare)
	}
	//todo:avg may be zero??
	return cvs.Mul(cvs, scale).Div(cvs, avg).Mul(cvs, scale).Div(cvs, avg).Div(cvs, length1)
}

func (e *Evaluator) getLegalCurrency(tokenAddress common.Address, amount *big.Rat) *big.Rat {
	c, _ := e.marketCapProvider.LegalCurrencyValue(tokenAddress, amount)
	return c
}

func (e *Evaluator) EvaluateReceived(ringState *types.Ring) (gas, gasPrice *big.Int, costLegal, received *big.Rat) {
	gasPrice = ethaccessor.EstimateGasPrice()
	gas = e.gasUsedWithLength[len(ringState.Orders)]
	protocolCost := new(big.Int).Mul(gas, gasPrice)

	costEth := new(big.Rat).SetInt(protocolCost)
	costLegal, _ = e.marketCapProvider.LegalCurrencyValueOfEth(costEth)
	received = new(big.Rat)
	legalFee := new(big.Rat)
	for _, order := range ringState.Orders {
		if order.LegalLrcFee.Cmp(order.LegalFeeS) < 0 {
			legalFee.Add(legalFee, order.LegalFeeS)
		} else {
			legalFee.Add(legalFee, order.LegalLrcFee)
		}
	}

	costLegal.Mul(costLegal, e.realCostRate)
	received.Sub(legalFee, costLegal)
	received.Mul(received, e.walletSplit)
	return
}

func NewEvaluator(marketCapProvider marketcap.MarketCapProvider, rateRatioCVSThreshold int64, subsidy, walletSplit float64) *Evaluator {
	gasUsedMap := make(map[int]*big.Int)
	gasUsedMap[2] = big.NewInt(400000)
	//todo:confirm this value
	gasUsedMap[3] = big.NewInt(400000)
	gasUsedMap[4] = big.NewInt(400000)
	e := &Evaluator{marketCapProvider: marketCapProvider, rateRatioCVSThreshold: rateRatioCVSThreshold, gasUsedWithLength: gasUsedMap}
	e.realCostRate = new(big.Rat)
	e.realCostRate.SetFloat64(float64(1.0) - subsidy)
	e.walletSplit = new(big.Rat)
	e.walletSplit.SetFloat64(walletSplit)
	return e
}
