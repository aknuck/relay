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

package gateway

import (
	"fmt"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/market"
	"github.com/Loopring/relay/market/util"
	"github.com/Loopring/relay/marketcap"
	"github.com/Loopring/relay/ordermanager"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"qiniupkg.com/x/errors.v7"
	"sort"
	"strconv"
	"strings"
	"time"
)

const DefaultContractVersion = "v1.2"
const DefaultCapCurrency = "CNY"

type Portfolio struct {
	Token      string `json:"token"`
	Amount     string `json:"amount"`
	Percentage string `json:"percentage"`
}

type PageResult struct {
	Data      []interface{} `json:"data"`
	PageIndex int           `json:"pageIndex"`
	PageSize  int           `json:"pageSize"`
	Total     int           `json:"total"`
}

type Depth struct {
	ContractVersion string `json:"contractVersion"`
	Market          string `json:"market"`
	Depth           AskBid `json:"depth"`
}

type AskBid struct {
	Buy  [][]string `json:"buy"`
	Sell [][]string `json:"sell"`
}

type DepthElement struct {
	Price  string   `json:"price"`
	Size   *big.Rat `json:"size"`
	Amount *big.Rat `json:"amount"`
}

type CommonTokenRequest struct {
	ContractVersion string `json:"contractVersion"`
	Owner           string `json:"owner"`
}

type SingleContractVersion struct {
	ContractVersion string `json:"contractVersion"`
}

type SingleMarket struct {
	Market string `json:"market"`
}

type TrendQuery struct {
	Market   string `json:"market"`
	Interval string `json:"interval"`
}

type SingleOwner struct {
	Owner string `json:"owner"`
}

type TxNotify struct {
	TxHash string `json:"txHash"`
}

type PriceQuoteQuery struct {
	Currency string `json:"currency"`
}

type CutoffRequest struct {
	Address         string `json:"address"`
	ContractVersion string `json:"contractVersion"`
	BlockNumber     string `json:"blockNumber"`
}

type EstimatedAllocatedAllowanceQuery struct {
	Owner string `json: "owner"`
	Token string `json: "token"`
}

type TransactionQuery struct {
	ThxHash   string   `json:"thxHash"`
	Owner     string   `json:"owner"`
	Symbol    string   `json: "symbol"`
	Status    string   `json: "status"`
	TxType    string   `json:"txType"`
	TrxHashes []string `json:"trxHashes"`
	PageIndex int      `json:"pageIndex"`
	PageSize  int      `json:"pageSize"`
}

type OrderQuery struct {
	Status          string `json:"status"`
	PageIndex       int    `json:"pageIndex"`
	PageSize        int    `json:"pageSize"`
	ContractVersion string `json:"contractVersion"`
	Owner           string `json:"owner"`
	Market          string `json:"market"`
	OrderHash       string `json:"orderHash"`
	Side            string `json:"side"`
}

type DepthQuery struct {
	Length          int    `json:"length"`
	ContractVersion string `json:"contractVersion"`
	Market          string `json:"market"`
}

type FillQuery struct {
	ContractVersion string `json:"contractVersion"`
	Market          string `json:"market"`
	Owner           string `json:"owner"`
	OrderHash       string `json:"orderHash"`
	RingHash        string `json:"ringHash"`
	PageIndex       int    `json:"pageIndex"`
	PageSize        int    `json:"pageSize"`
	Side            string `json:"side"`
}

type RingMinedQuery struct {
	ContractVersion string `json:"contractVersion"`
	RingHash        string `json:"ringHash"`
	PageIndex       int    `json:"pageIndex"`
	PageSize        int    `json:"pageSize"`
}

type RawOrderJsonResult struct {
	Protocol   string `json:"protocol"` // 智能合约地址
	Owner      string `json:"address"`
	Hash       string `json:"hash"`
	TokenS     string `json:"tokenS"`  // 卖出erc20代币智能合约地址
	TokenB     string `json:"tokenB"`  // 买入erc20代币智能合约地址
	AmountS    string `json:"amountS"` // 卖出erc20代币数量上限
	AmountB    string `json:"amountB"` // 买入erc20代币数量上限
	ValidSince string `json:"validSince"`
	ValidUntil string `json:"validUntil"` // 订单过期时间
	//Salt                  string `json:"salt"`
	LrcFee                string `json:"lrcFee"` // 交易总费用,部分成交的费用按该次撮合实际卖出代币额与比例计算
	BuyNoMoreThanAmountB  bool   `json:"buyNoMoreThanAmountB"`
	MarginSplitPercentage string `json:"marginSplitPercentage"` // 不为0时支付给交易所的分润比例，否则视为100%
	V                     string `json:"v"`
	R                     string `json:"r"`
	S                     string `json:"s"`
	WalletId              string `json:"walletId" gencodec:"required"`
	AuthAddr              string `json:"authAddr" gencodec:"required"`       //
	AuthPrivateKey        string `json:"authPrivateKey" gencodec:"required"` //
	Market                string `json:"market"`
	Side                  string `json:"side"`
	CreateTime            int64  `json:"createTime"`
}

type OrderJsonResult struct {
	RawOrder         RawOrderJsonResult `json:"originalOrder"`
	DealtAmountS     string             `json:"dealtAmountS"`
	DealtAmountB     string             `json:"dealtAmountB"`
	CancelledAmountS string             `json:"cancelledAmountS"`
	CancelledAmountB string             `json:"cancelledAmountB"`
	Status           string             `json:"status"`
}

type TransactionJsonResult struct {
	Protocol    common.Address     `json:"protocol"`
	Owner       common.Address     `json:"owner"`
	From        common.Address     `json:"from"`
	To          common.Address     `json:"to"`
	TxHash      common.Hash        `json:"txHash"`
	Symbol      string             `json:"symbol"`
	Content     TransactionContent `json:"content"`
	BlockNumber int64              `json:"blockNumber"`
	Value       string             `json:"value"`
	LogIndex    int64              `json:"logIndex"`
	Type        string             `json:"type"`
	Status      string             `json:"status"`
	CreateTime  int64              `json:"createTime"`
	UpdateTime  int64              `json:"updateTime"`
	Nonce       string             `json:"nonce"`
}

type TransactionContent struct {
	Market    string `json:"market"`
	OrderHash string `json:"orderHash"`
}

type PriceQuote struct {
	Currency string       `json:"currency"`
	Tokens   []TokenPrice `json:"tokens"`
}

type TokenPrice struct {
	Token string  `json:"symbol"`
	Price float64 `json:"price"`
}

type RingMinedDetail struct {
	RingInfo RingMinedInfo   `json:"ringInfo"`
	Fills    []dao.FillEvent `json:"fills"`
}

type RingMinedInfo struct {
	ID                 int                 `json:"id"`
	Protocol           string              `json:"protocol"`
	RingIndex          string              `json:"ringIndex"`
	RingHash           string              `json:"ringHash"`
	TxHash             string              `json:"txHash"`
	Miner              string              `json:"miner"`
	FeeRecipient       string              `json:"feeRecipient"`
	IsRinghashReserved bool                `json:"isRinghashReserved"`
	BlockNumber        int64               `json:"blockNumber"`
	TotalLrcFee        string              `json:"totalLrcFee"`
	TotalSplitFee      map[string]*big.Int `json:"totalSplitFee"`
	TradeAmount        int                 `json:"tradeAmount"`
	Time               int64               `json:"timestamp"`
}

type WalletServiceImpl struct {
	trendManager    market.TrendManager
	orderManager    ordermanager.OrderManager
	accountManager  market.AccountManager
	marketCap       marketcap.MarketCapProvider
	ethForwarder    *EthForwarder
	tickerCollector market.CollectorImpl
	rds             dao.RdsService
	oldWethAddress  string
}

func NewWalletService(trendManager market.TrendManager, orderManager ordermanager.OrderManager, accountManager market.AccountManager,
	capProvider marketcap.MarketCapProvider, ethForwarder *EthForwarder, collector market.CollectorImpl, rds dao.RdsService, oldWethAddress string) *WalletServiceImpl {
	w := &WalletServiceImpl{}
	w.trendManager = trendManager
	w.orderManager = orderManager
	w.accountManager = accountManager
	w.marketCap = capProvider
	w.ethForwarder = ethForwarder
	w.tickerCollector = collector
	w.rds = rds
	w.oldWethAddress = oldWethAddress
	return w
}
func (w *WalletServiceImpl) TestPing(input int) (resp []byte, err error) {

	var res string
	if input > 0 {
		res = "input is bigger than zero " + time.Now().String()
	} else if input == 0 {
		res = "input is equal zero " + time.Now().String()
	} else if input < 0 {
		res = "input is smaller than zero " + time.Now().String()
	}
	resp = []byte("{'abc' : '" + res + "'}")
	return
}

func (w *WalletServiceImpl) GetPortfolio(query SingleOwner) (res []Portfolio, err error) {
	res = make([]Portfolio, 0)
	if len(query.Owner) == 0 {
		return nil, errors.New("owner can't be nil")
	}

	account, _ := w.accountManager.GetBalance(DefaultContractVersion, query.Owner)
	balances := account.Balances
	if len(balances) == 0 {
		return
	}

	balancesCopy := make(map[string]market.Balance)

	for k, v := range balances {
		balancesCopy[k] = v
	}

	ethBalance := market.Balance{Token: "ETH", Balance: big.NewInt(0)}
	b, bErr := w.ethForwarder.GetBalance(query.Owner, "latest")
	if bErr == nil {
		ethBalance.Balance = types.HexToBigint(b)
		balancesCopy["ETH"] = ethBalance
	} else {
		return res, bErr
	}

	priceQuote, err := w.GetPriceQuote(PriceQuoteQuery{DefaultCapCurrency})
	if err != nil {
		return
	}

	priceQuoteMap := make(map[string]*big.Rat)
	for _, pq := range priceQuote.Tokens {
		priceQuoteMap[pq.Token] = new(big.Rat).SetFloat64(pq.Price)
	}

	totalAsset := big.NewRat(0, 1)
	for k, v := range balancesCopy {
		asset := new(big.Rat).Set(priceQuoteMap[k])
		asset = asset.Mul(asset, new(big.Rat).SetFrac(v.Balance, big.NewInt(1)))
		totalAsset = totalAsset.Add(totalAsset, asset)
	}

	for k, v := range balancesCopy {
		portfolio := Portfolio{Token: k, Amount: v.Balance.String()}
		asset := new(big.Rat).Set(priceQuoteMap[k])
		asset = asset.Mul(asset, new(big.Rat).SetFrac(v.Balance, big.NewInt(1)))
		totalAssetFloat, _ := totalAsset.Float64()
		var percentage float64
		if totalAssetFloat == 0 {
			percentage = 0
		} else {
			percentage, _ = asset.Quo(asset, totalAsset).Float64()
		}
		portfolio.Percentage = fmt.Sprintf("%.4f%%", 100*percentage)
		res = append(res, portfolio)
	}

	sort.Slice(res, func(i, j int) bool {
		percentStrLeft := strings.Replace(res[i].Percentage, "%", "", 1)
		percentStrRight := strings.Replace(res[j].Percentage, "%", "", 1)
		left, _ := strconv.ParseFloat(percentStrLeft, 64)
		right, _ := strconv.ParseFloat(percentStrRight, 64)
		return left > right
	})

	return
}

func (w *WalletServiceImpl) GetPriceQuote(query PriceQuoteQuery) (result PriceQuote, err error) {

	rst := PriceQuote{query.Currency, make([]TokenPrice, 0)}
	for k, v := range util.AllTokens {
		price, err := w.marketCap.GetMarketCapByCurrency(v.Protocol, query.Currency)
		if err != nil {
			return result, err
		}
		floatPrice, _ := price.Float64()
		rst.Tokens = append(rst.Tokens, TokenPrice{k, floatPrice})
		if k == "WETH" {
			rst.Tokens = append(rst.Tokens, TokenPrice{"ETH", floatPrice})
		}
	}

	return rst, nil
}

func (w *WalletServiceImpl) GetTickers(mkt SingleMarket) (result map[string]market.Ticker, err error) {
	result = make(map[string]market.Ticker)
	loopringTicker, err := w.trendManager.GetTickerByMarket(mkt.Market)
	if err == nil {
		result["loopr"] = loopringTicker
	} else {
		log.Info("get ticker from loopring error" + err.Error())
		return result, err
	}
	outTickers, err := w.tickerCollector.GetTickers(mkt.Market)
	if err == nil {
		for _, v := range outTickers {
			result[v.Exchange] = v
		}
	} else {
		log.Info("get other exchanges ticker error" + err.Error())
	}
	return result, nil
}

func (w *WalletServiceImpl) GetAllMarketTickers() (result []market.Ticker, err error) {
	return w.trendManager.GetTicker()
}

func (w *WalletServiceImpl) UnlockWallet(owner SingleOwner) (result string, err error) {
	if len(owner.Owner) == 0 {
		return "", errors.New("owner can't be null string")
	}

	unlockRst := w.accountManager.UnlockedWallet(owner.Owner)
	if unlockRst != nil {
		return "", unlockRst
	} else {
		return "unlock_notice_success", nil
	}
}

func (w *WalletServiceImpl) NotifyTransactionSubmitted(txNotify TxNotify) (result string, err error) {
	if len(txNotify.TxHash) == 0 {
		return "", errors.New("txHash can't be null string")
	}
	log.Info(">>>>>>>>>received tx notify " + txNotify.TxHash)
	tx := &ethaccessor.Transaction{}
	err = ethaccessor.GetTransactionByHash(tx, txNotify.TxHash, "pending")
	if err == nil {
		eventemitter.Emit(eventemitter.PendingTransactionEvent, tx)
		log.Info("emit transaction info " + tx.Hash)
	} else {
		log.Error(">>>>>>>>getTransaction error : " + err.Error())
	}
	return
}

func (w *WalletServiceImpl) GetOldVersionWethBalance(owner SingleOwner) (res string, err error) {
	b, err := ethaccessor.Erc20Balance(common.HexToAddress(w.oldWethAddress), common.HexToAddress(owner.Owner), "latest")
	if err != nil {
		return
	} else {
		return types.BigintToHex(b), nil
	}
}

func (w *WalletServiceImpl) SubmitOrder(order *types.OrderJsonRequest) (res string, err error) {
	err = HandleOrder(types.ToOrder(order))
	if err != nil {
		fmt.Println(err)
	}
	res = "SUBMIT_SUCCESS"
	return res, err
}

func (w *WalletServiceImpl) GetOrders(query *OrderQuery) (res PageResult, err error) {
	orderQuery, statusList, pi, ps := convertFromQuery(query)
	queryRst, err := w.orderManager.GetOrders(orderQuery, statusList, pi, ps)
	if err != nil {
		fmt.Println(err)
	}
	return buildOrderResult(queryRst), err
}

func (w *WalletServiceImpl) GetDepth(query DepthQuery) (res Depth, err error) {

	mkt := strings.ToUpper(query.Market)
	protocol := query.ContractVersion
	length := query.Length

	if mkt == "" || protocol == "" || util.ContractVersionConfig[protocol] == "" {
		err = errors.New("market and correct contract version must be applied")
		return
	}

	if length <= 0 || length > 20 {
		length = 20
	}

	a, b := util.UnWrap(mkt)

	_, err = util.WrapMarket(a, b)
	if err != nil {
		err = errors.New("unsupported market type")
		return
	}

	empty := make([][]string, 0)

	for i := range empty {
		empty[i] = make([]string, 0)
	}
	askBid := AskBid{Buy: empty, Sell: empty}
	depth := Depth{ContractVersion: util.ContractVersionConfig[protocol], Market: mkt, Depth: askBid}

	//(TODO) 考虑到需要聚合的情况，所以每次取2倍的数据，先聚合完了再cut, 不是完美方案，后续再优化
	asks, askErr := w.orderManager.GetOrderBook(
		common.HexToAddress(util.ContractVersionConfig[protocol]),
		util.AllTokens[a].Protocol,
		util.AllTokens[b].Protocol, length*2)

	if askErr != nil {
		err = errors.New("get depth error , please refresh again")
		return
	}

	depth.Depth.Sell = calculateDepth(asks, length, true, util.AllTokens[a].Decimals, util.AllTokens[b].Decimals)

	bids, bidErr := w.orderManager.GetOrderBook(
		common.HexToAddress(util.ContractVersionConfig[protocol]),
		util.AllTokens[b].Protocol,
		util.AllTokens[a].Protocol, length*2)

	if bidErr != nil {
		err = errors.New("get depth error , please refresh again")
		return
	}

	depth.Depth.Buy = calculateDepth(bids, length, false, util.AllTokens[b].Decimals, util.AllTokens[a].Decimals)

	return depth, err
}

func (w *WalletServiceImpl) GetFills(query FillQuery) (dao.PageResult, error) {
	res, err := w.orderManager.FillsPageQuery(fillQueryToMap(query))

	if err != nil {
		return dao.PageResult{}, nil
	}

	result := dao.PageResult{PageIndex: res.PageIndex, PageSize: res.PageSize, Total: res.Total, Data: make([]interface{}, 0)}

	for _, f := range res.Data {
		fill := f.(dao.FillEvent)
		if util.IsBuy(fill.TokenB) {
			fill.Side = "buy"
		} else {
			fill.Side = "sell"
		}
		fill.TokenS = util.AddressToAlias(fill.TokenS)
		fill.TokenB = util.AddressToAlias(fill.TokenB)

		result.Data = append(result.Data, fill)
	}
	return result, nil
}

func (w *WalletServiceImpl) GetTicker(query SingleContractVersion) (res []market.Ticker, err error) {
	res, err = w.trendManager.GetTicker()

	//for i, t := range res {
	//	w.fillBuyAndSell(&t, query.ContractVersion)
	//	res[i] = t
	//}
	return
}

func (w *WalletServiceImpl) GetTrend(query TrendQuery) (res []market.Trend, err error) {
	res, err = w.trendManager.GetTrends(query.Market, query.Interval)
	sort.Slice(res, func(i, j int) bool {
		return res[i].Start < res[j].Start
	})
	return
}

func (w *WalletServiceImpl) GetRingMined(query RingMinedQuery) (res dao.PageResult, err error) {
	return w.orderManager.RingMinedPageQuery(ringMinedQueryToMap(query))
}

func (w *WalletServiceImpl) GetRingMinedDetail(query RingMinedQuery) (res RingMinedDetail, err error) {
	if len(query.RingHash) == 0 {
		return res, errors.New("ring hash can't be null")
	}

	rings, err := w.orderManager.RingMinedPageQuery(ringMinedQueryToMap(query))

	if err != nil || rings.Total > 1 {
		log.Errorf("query ring error, %s, %d", err.Error(), rings.Total)
		return res, errors.New("query ring error occurs")
	}

	if rings.Total == 0 {
		return res, errors.New("no ring found by hash")
	}

	fills, err := w.orderManager.FindFillsByRingHash(common.HexToHash(query.RingHash))
	if err != nil {
		return res, err
	}
	return fillDetail(rings.Data[0].(dao.RingMinedEvent), fills)
}

func (w *WalletServiceImpl) GetBalance(balanceQuery CommonTokenRequest) (res market.AccountJson, err error) {
	if len(balanceQuery.Owner) == 0 {
		return res, errors.New("owner can't be null")
	}
	if len(balanceQuery.ContractVersion) == 0 {
		return res, errors.New("contract version can't be null")
	}
	account, _ := w.accountManager.GetBalance(balanceQuery.ContractVersion, balanceQuery.Owner)
	ethBalance := market.Balance{Token: "ETH", Balance: big.NewInt(0)}
	b, bErr := w.ethForwarder.GetBalance(balanceQuery.Owner, "latest")
	if bErr == nil {
		ethBalance.Balance = types.HexToBigint(b)
	}
	res = account.ToJsonObject(balanceQuery.ContractVersion, ethBalance)
	return
}

func (w *WalletServiceImpl) GetCutoff(query CutoffRequest) (result int64, err error) {
	cutoff, err := ethaccessor.GetCutoff(common.HexToAddress(util.ContractVersionConfig[query.ContractVersion]), common.HexToAddress(query.Address), query.BlockNumber)
	if err != nil {
		return 0, err
	}
	return cutoff.Int64(), nil
}

func (w *WalletServiceImpl) GetEstimatedAllocatedAllowance(query EstimatedAllocatedAllowanceQuery) (frozenAmount string, err error) {
	statusSet := make([]types.OrderStatus, 0)
	statusSet = append(statusSet, types.ORDER_NEW)
	statusSet = append(statusSet, types.ORDER_PARTIAL)

	token := query.Token
	owner := query.Owner

	tokenAddress := util.AliasToAddress(token)
	if tokenAddress.Hex() == "" {
		return "", errors.New("unsupported token alias " + token)
	}
	amount, err := w.orderManager.GetFrozenAmount(common.HexToAddress(owner), tokenAddress, statusSet)
	if err != nil {
		return "", err
	}

	return types.BigintToHex(amount), err
}

func (w *WalletServiceImpl) GetFrozenLRCFee(query SingleOwner) (frozenAmount string, err error) {
	statusSet := make([]types.OrderStatus, 0)
	statusSet = append(statusSet, types.ORDER_NEW)
	statusSet = append(statusSet, types.ORDER_PARTIAL)

	owner := query.Owner

	allLrcFee, err := w.orderManager.GetFrozenLRCFee(common.HexToAddress(owner), statusSet)
	if err != nil {
		return "", err
	}

	return types.BigintToHex(allLrcFee), err
}

func (w *WalletServiceImpl) GetSupportedMarket() (markets []string, err error) {
	return util.AllMarkets, err
}

func (w *WalletServiceImpl) GetSupportedTokens() (markets []types.Token, err error) {
	markets = make([]types.Token, 0)
	for _, v := range util.AllTokens {
		markets = append(markets, v)
	}
	return markets, err
}

func (w *WalletServiceImpl) GetTransactions(query TransactionQuery) (pr PageResult, err error) {

	trxQuery := make(map[string]interface{})

	if query.Symbol != "" {
		trxQuery["symbol"] = query.Symbol
	}

	if query.Owner != "" {
		trxQuery["owner"] = query.Owner
	}

	if query.ThxHash != "" {
		trxQuery["tx_hash"] = query.ThxHash
	}

	if txStatusToUint8(query.Status) > 0 {
		trxQuery["status"] = uint8(txStatusToUint8(query.Status))
	}

	if txTypeToUint8(query.TxType) > 0 {
		trxQuery["tx_type"] = uint8(txTypeToUint8(query.TxType))
	}

	pageIndex := query.PageIndex
	pageSize := query.PageSize

	daoPr, err := w.rds.TransactionPageQuery(trxQuery, pageIndex, pageSize)

	if err != nil {
		return pr, err
	}

	rst := PageResult{Total: daoPr.Total, PageIndex: daoPr.PageIndex, PageSize: daoPr.PageSize, Data: make([]interface{}, 0)}

	for _, d := range daoPr.Data {
		o := d.(dao.Transaction)
		tr := types.Transaction{}
		err = o.ConvertUp(&tr)
		rst.Data = append(rst.Data, toTxJsonResult(tr))
	}
	return rst, nil
}

func (w *WalletServiceImpl) GetTransactionsByHash(query TransactionQuery) (result []TransactionJsonResult, err error) {

	rst, err := w.rds.GetTrxByHashes(query.TrxHashes)

	if err != nil {
		return nil, err
	}

	result = make([]TransactionJsonResult, 0)
	for _, r := range rst {
		tr := types.Transaction{}
		err = r.ConvertUp(&tr)
		if err != nil {
			log.Error("convert error occurs..." + err.Error())
		}
		result = append(result, toTxJsonResult(tr))
	}

	return result, nil
}

func (w *WalletServiceImpl) GetPendingTransactions(query SingleOwner) (result []TransactionJsonResult, err error) {

	if len(query.Owner) == 0 {
		return nil, errors.New("owner can't be null")
	}

	txQuery := make(map[string]interface{})
	txQuery["owner"] = query.Owner
	txQuery["status"] = types.TX_STATUS_PENDING

	rst, err := w.rds.PendingTransactions(txQuery)

	if err != nil {
		return nil, err
	}

	result = make([]TransactionJsonResult, 0)
	for _, r := range rst {
		tr := types.Transaction{}
		err = r.ConvertUp(&tr)
		if err != nil {
			log.Error("convert error occurs..." + err.Error())
		}
		result = append(result, toTxJsonResult(tr))
	}

	return result, nil
}

func convertFromQuery(orderQuery *OrderQuery) (query map[string]interface{}, statusList []types.OrderStatus, pageIndex int, pageSize int) {

	query = make(map[string]interface{})
	statusList = convertStatus(orderQuery.Status)
	if orderQuery.Owner != "" {
		query["owner"] = orderQuery.Owner
	}
	if util.ContractVersionConfig[orderQuery.ContractVersion] != "" {
		query["protocol"] = util.ContractVersionConfig[orderQuery.ContractVersion]
	}

	if orderQuery.Market != "" {
		query["market"] = orderQuery.Market
	}

	if orderQuery.OrderHash != "" {
		query["order_hash"] = orderQuery.OrderHash
	}
	if strings.ToLower(orderQuery.Side) == "buy" {
		query["token_s"] = util.AllTokens["WETH"].Protocol.Hex()
	} else if strings.ToLower(orderQuery.Side) == "sell" {
		query["token_b"] = util.AllTokens["WETH"].Protocol.Hex()
	}

	pageIndex = orderQuery.PageIndex
	pageSize = orderQuery.PageSize
	return

}

func convertStatus(s string) []types.OrderStatus {
	switch s {
	case "ORDER_OPENED":
		return []types.OrderStatus{types.ORDER_NEW, types.ORDER_PARTIAL}
	case "ORDER_NEW":
		return []types.OrderStatus{types.ORDER_NEW}
	case "ORDER_PARTIAL":
		return []types.OrderStatus{types.ORDER_PARTIAL}
	case "ORDER_FINISHED":
		return []types.OrderStatus{types.ORDER_FINISHED}
	case "ORDER_CANCELLED":
		return []types.OrderStatus{types.ORDER_CANCEL, types.ORDER_CUTOFF}
	case "ORDER_CUTOFF":
		return []types.OrderStatus{types.ORDER_CUTOFF}
	case "ORDER_EXPIRE":
		return []types.OrderStatus{types.ORDER_EXPIRE}
	}
	return []types.OrderStatus{}
}

func getStringStatus(order types.OrderState) string {
	s := order.Status

	if order.IsExpired() {
		return "ORDER_EXPIRE"
	}

	if order.IsExpired() {
		return "ORDER_PENDING"
	}

	switch s {
	case types.ORDER_NEW:
		return "ORDER_OPENED"
	case types.ORDER_PARTIAL:
		return "ORDER_OPENED"
	case types.ORDER_FINISHED:
		return "ORDER_FINISHED"
	case types.ORDER_CANCEL:
		return "ORDER_CANCELLED"
	case types.ORDER_CUTOFF:
		return "ORDER_CUTOFF"
	case types.ORDER_PENDING:
		return "ORDER_PENDING"
	case types.ORDER_EXPIRE:
		return "ORDER_EXPIRE"
	}
	return "ORDER_UNKNOWN"
}

func calculateDepth(states []types.OrderState, length int, isAsk bool, tokenSDecimal, tokenBDecimal *big.Int) [][]string {

	if len(states) == 0 {
		return [][]string{}
	}

	depth := make([][]string, 0)
	for i := range depth {
		depth[i] = make([]string, 0)
	}

	depthMap := make(map[string]DepthElement)

	for _, s := range states {

		price := *s.RawOrder.Price
		amountS, amountB := s.RemainedAmount()
		amountS = amountS.Quo(amountS, new(big.Rat).SetFrac(tokenSDecimal, big.NewInt(1)))
		amountB = amountB.Quo(amountB, new(big.Rat).SetFrac(tokenBDecimal, big.NewInt(1)))

		if amountS.Cmp(new(big.Rat).SetFloat64(0)) == 0 {
			log.Debug("amount s is zero, skipped")
			continue
		}

		if amountB.Cmp(new(big.Rat).SetFloat64(0)) == 0 {
			log.Debug("amount b is zero, skipped")
			continue
		}

		if isAsk {
			price = *price.Inv(&price)
			priceFloatStr := price.FloatString(10)
			if v, ok := depthMap[priceFloatStr]; ok {
				amount := v.Amount
				size := v.Size
				amount = amount.Add(amount, amountS)
				size = size.Add(size, amountB)
				depthMap[priceFloatStr] = DepthElement{Price: v.Price, Amount: amount, Size: size}
			} else {
				depthMap[priceFloatStr] = DepthElement{Price: priceFloatStr, Amount: amountS, Size: amountB}
			}
		} else {
			priceFloatStr := price.FloatString(10)
			if v, ok := depthMap[priceFloatStr]; ok {
				amount := v.Amount
				size := v.Size
				amount = amount.Add(amount, amountB)
				size = size.Add(size, amountS)
				depthMap[priceFloatStr] = DepthElement{Price: v.Price, Amount: amount, Size: size}
			} else {
				depthMap[priceFloatStr] = DepthElement{Price: priceFloatStr, Amount: amountB, Size: amountS}
			}
		}
	}

	for k, v := range depthMap {
		amount, _ := v.Amount.Float64()
		size, _ := v.Size.Float64()
		depth = append(depth, []string{k, strconv.FormatFloat(amount, 'f', 10, 64), strconv.FormatFloat(size, 'f', 10, 64)})
	}

	sort.Slice(depth, func(i, j int) bool {
		cmpA, _ := strconv.ParseFloat(depth[i][0], 64)
		cmpB, _ := strconv.ParseFloat(depth[j][0], 64)
		if isAsk {
			return cmpA < cmpB
		} else {
			return cmpA > cmpB
		}

	})

	if length < len(depth) {
		return depth[:length]
	}
	return depth
}

func fillQueryToMap(q FillQuery) (map[string]interface{}, int, int) {
	rst := make(map[string]interface{})
	var pi, ps int
	if q.Market != "" {
		rst["market"] = q.Market
	}
	if q.PageIndex <= 0 {
		pi = 1
	} else {
		pi = q.PageIndex
	}
	if q.PageSize <= 0 || q.PageSize > 20 {
		ps = 20
	} else {
		ps = q.PageSize
	}
	if q.ContractVersion != "" {
		rst["contract_address"] = util.ContractVersionConfig[q.ContractVersion]
	}
	if q.Owner != "" {
		rst["owner"] = q.Owner
	}
	if q.OrderHash != "" {
		rst["order_hash"] = q.OrderHash
	}
	if q.RingHash != "" {
		rst["ring_hash"] = q.RingHash
	}

	if strings.ToLower(q.Side) == "buy" {
		rst["token_s"] = util.AllTokens["WETH"].Protocol.Hex()
	} else if strings.ToLower(q.Side) == "sell" {
		rst["token_b"] = util.AllTokens["WETH"].Protocol.Hex()
	}

	return rst, pi, ps
}

func ringMinedQueryToMap(q RingMinedQuery) (map[string]interface{}, int, int) {
	rst := make(map[string]interface{})
	var pi, ps int
	if q.PageIndex <= 0 {
		pi = 1
	} else {
		pi = q.PageIndex
	}
	if q.PageSize <= 0 || q.PageSize > 20 {
		ps = 20
	} else {
		ps = q.PageSize
	}
	if q.ContractVersion != "" {
		rst["contract_address"] = util.ContractVersionConfig[q.ContractVersion]
	}
	if q.RingHash != "" {
		rst["ring_hash"] = q.RingHash
	}

	return rst, pi, ps
}

func buildOrderResult(src dao.PageResult) PageResult {

	rst := PageResult{Total: src.Total, PageIndex: src.PageIndex, PageSize: src.PageSize, Data: make([]interface{}, 0)}

	for _, d := range src.Data {
		o := d.(types.OrderState)
		rst.Data = append(rst.Data, orderStateToJson(o))
	}
	return rst
}

func orderStateToJson(src types.OrderState) OrderJsonResult {

	rst := OrderJsonResult{}
	rst.DealtAmountB = types.BigintToHex(src.DealtAmountB)
	rst.DealtAmountS = types.BigintToHex(src.DealtAmountS)
	rst.CancelledAmountB = types.BigintToHex(src.CancelledAmountB)
	rst.CancelledAmountS = types.BigintToHex(src.CancelledAmountS)
	rst.Status = getStringStatus(src)
	rawOrder := RawOrderJsonResult{}
	rawOrder.Protocol = src.RawOrder.Protocol.String()
	rawOrder.Owner = src.RawOrder.Owner.String()
	rawOrder.Hash = src.RawOrder.Hash.String()
	rawOrder.TokenS = util.AddressToAlias(src.RawOrder.TokenS.String())
	rawOrder.TokenB = util.AddressToAlias(src.RawOrder.TokenB.String())
	rawOrder.AmountS = types.BigintToHex(src.RawOrder.AmountS)
	rawOrder.AmountB = types.BigintToHex(src.RawOrder.AmountB)
	rawOrder.ValidSince = types.BigintToHex(src.RawOrder.ValidSince)
	rawOrder.ValidUntil = types.BigintToHex(src.RawOrder.ValidUntil)
	rawOrder.LrcFee = types.BigintToHex(src.RawOrder.LrcFee)
	rawOrder.BuyNoMoreThanAmountB = src.RawOrder.BuyNoMoreThanAmountB
	rawOrder.MarginSplitPercentage = types.BigintToHex(big.NewInt(int64(src.RawOrder.MarginSplitPercentage)))
	rawOrder.V = types.BigintToHex(big.NewInt(int64(src.RawOrder.V)))
	rawOrder.R = src.RawOrder.R.Hex()
	rawOrder.S = src.RawOrder.S.Hex()
	rawOrder.WalletId = types.BigintToHex(src.RawOrder.WalletId)
	rawOrder.AuthAddr = src.RawOrder.AuthPrivateKey.Address().Hex()
	rawOrder.Market = src.RawOrder.Market
	if rawOrder.TokenB == "WETH" {
		rawOrder.Side = "sell"
	} else {
		rawOrder.Side = "buy"
	}
	auth, _ := src.RawOrder.AuthPrivateKey.MarshalText()
	rawOrder.AuthPrivateKey = string(auth)
	rawOrder.CreateTime = src.RawOrder.CreateTime
	rst.RawOrder = rawOrder
	return rst
}

func (w *WalletServiceImpl) fillBuyAndSell(ticker *market.Ticker, contractVersion string) {
	queryDepth := DepthQuery{1, contractVersion, ticker.Market}

	depth, err := w.GetDepth(queryDepth)
	if err != nil {
		log.Error("fill depth info failed")
	} else {
		if len(depth.Depth.Buy) > 0 {
			ticker.Buy = depth.Depth.Buy[0][0]
		}
		if len(depth.Depth.Sell) > 0 {
			ticker.Sell = depth.Depth.Sell[0][0]
		}
	}
}

func txStatusToUint8(txType string) int {
	switch txType {
	case "pending":
		return 1
	case "success":
		return 2
	case "failed":
		return 3
	default:
		return -1
	}
}

func txTypeToUint8(status string) int {
	switch status {
	case "approve":
		return 1
	case "send":
		return 2
	case "receive":
		return 3
	case "sell":
		return 4
	case "buy":
		return 5
	case "convert":
		return 7
	case "cancel_order":
		return 8
	case "cutoff":
		return 9
	case "cutoff_trading_pair":
		return 10
	default:
		return -1
	}
}

func toTxJsonResult(tx types.Transaction) TransactionJsonResult {
	dst := TransactionJsonResult{}
	dst.Protocol = tx.Protocol
	dst.Owner = tx.Owner
	dst.From = tx.From
	dst.To = tx.To
	dst.TxHash = tx.TxHash

	if tx.Type == types.TX_TYPE_CUTOFF_PAIR {
		ctx, err := tx.GetCutoffPairContent()
		if err == nil && ctx != nil {
			mkt, err := util.WrapMarketByAddress(ctx.Token1.Hex(), ctx.Token2.Hex())
			if err == nil {
				dst.Content = TransactionContent{Market: mkt}
			}
		}
	} else if tx.Type == types.TX_TYPE_CANCEL_ORDER {
		ctx, err := tx.GetCancelOrderHash()
		if err == nil && ctx != "" {
			dst.Content = TransactionContent{OrderHash: ctx}
		}
	}

	dst.BlockNumber = tx.BlockNumber.Int64()
	dst.LogIndex = tx.LogIndex
	if tx.Value == nil {
		dst.Value = "0"
	} else {
		dst.Value = tx.Value.String()
	}
	dst.Type = tx.TypeStr()
	dst.Status = tx.StatusStr()
	dst.CreateTime = tx.CreateTime
	dst.UpdateTime = tx.UpdateTime
	dst.Symbol = tx.Symbol
	dst.Nonce = tx.TxInfo.Nonce.String()
	return dst
}

func fillDetail(ring dao.RingMinedEvent, fills []dao.FillEvent) (rst RingMinedDetail, err error) {
	rst = RingMinedDetail{Fills: fills}
	ringInfo := RingMinedInfo{}
	ringInfo.ID = ring.ID
	ringInfo.RingHash = ring.RingHash
	ringInfo.BlockNumber = ring.BlockNumber
	ringInfo.Protocol = ring.Protocol
	ringInfo.TxHash = ring.TxHash
	ringInfo.Time = ring.Time
	ringInfo.RingIndex = ring.RingIndex
	ringInfo.Miner = ring.Miner
	ringInfo.FeeRecipient = ring.FeeRecipient
	ringInfo.IsRinghashReserved = ring.IsRinghashReserved
	ringInfo.TradeAmount = ring.TradeAmount
	ringInfo.TotalLrcFee = ring.TotalLrcFee
	ringInfo.TotalSplitFee = make(map[string]*big.Int)

	for _, f := range fills {
		if len(f.SplitS) > 0 && f.SplitS != "0" {
			symbol := util.AddressToAlias(f.TokenS)
			if len(symbol) > 0 {
				splitS, _ := new(big.Int).SetString(f.SplitS, 0)
				totalSplitS, ok := ringInfo.TotalSplitFee[symbol]
				if ok {
					ringInfo.TotalSplitFee[symbol] = totalSplitS.Add(splitS, totalSplitS)
				} else {
					ringInfo.TotalSplitFee[symbol] = splitS
				}
			}
		}
		if len(f.SplitB) > 0 && f.SplitB != "0" {
			symbol := util.AddressToAlias(f.TokenB)
			if len(symbol) > 0 {
				splitB, _ := new(big.Int).SetString(f.SplitB, 0)
				totalSplitB, ok := ringInfo.TotalSplitFee[symbol]
				if ok {
					ringInfo.TotalSplitFee[symbol] = totalSplitB.Add(splitB, totalSplitB)
				} else {
					ringInfo.TotalSplitFee[symbol] = splitB
				}
			}
		}
	}

	rst.RingInfo = ringInfo
	return rst, nil
}
