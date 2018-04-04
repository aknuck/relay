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

package node

import (
	"sync"

	"fmt"
	"github.com/Loopring/relay/cache"
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/crypto"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/extractor"
	"github.com/Loopring/relay/gateway"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/market"
	"github.com/Loopring/relay/market/util"
	"github.com/Loopring/relay/marketcap"
	"github.com/Loopring/relay/miner"
	"github.com/Loopring/relay/miner/timing_matcher"
	"github.com/Loopring/relay/ordermanager"
	"github.com/Loopring/relay/txmanager"
	"github.com/Loopring/relay/types"
	"github.com/Loopring/relay/usermanager"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"go.uber.org/zap"
	"math/big"
)

const (
	MODEL_RELAY = "relay"
	MODEL_MINER = "miner"
)

type Node struct {
	globalConfig      *config.GlobalConfig
	rdsService        dao.RdsService
	ipfsSubService    gateway.IPFSSubService
	extractorService  extractor.ExtractorService
	orderManager      ordermanager.OrderManager
	userManager       usermanager.UserManager
	marketCapProvider marketcap.MarketCapProvider
	accountManager    market.AccountManager
	relayNode         *RelayNode
	mineNode          *MineNode

	stop   chan struct{}
	lock   sync.RWMutex
	logger *zap.Logger
}

type RelayNode struct {
	trendManager     market.TrendManager
	tickerCollector  market.CollectorImpl
	jsonRpcService   gateway.JsonrpcServiceImpl
	websocketService gateway.WebsocketServiceImpl
	socketIOService  gateway.SocketIOServiceImpl
	walletService    gateway.WalletServiceImpl
	txManager        txmanager.TransactionManager
}

func (n *RelayNode) Start() {
	n.txManager.Start()

	//gateway.NewJsonrpcService("8080").Start()
	fmt.Println("step in relay node start")
	n.tickerCollector.Start()
	go n.jsonRpcService.Start()
	//n.websocketService.Start()
	go n.socketIOService.Start()

}

func (n *RelayNode) Stop() {
	n.txManager.Stop()
}

type MineNode struct {
	miner *miner.Miner
}

func (n *MineNode) Start() {
	n.miner.Start()
}
func (n *MineNode) Stop() {
	n.miner.Stop()
}

func NewNode(logger *zap.Logger, globalConfig *config.GlobalConfig) *Node {
	n := &Node{}
	n.logger = logger
	n.globalConfig = globalConfig

	// register
	n.registerMysql()
	cache.NewCache(n.globalConfig.Redis)

	util.Initialize(n.globalConfig.Market, n.globalConfig.Common.ProtocolImpl.Address)
	n.registerMarketCap()
	n.registerAccessor()
	n.registerUserManager()
	n.registerIPFSSubService()
	n.registerOrderManager()
	n.registerExtractor()
	n.registerGateway()
	n.registerCrypto(nil)
	n.registerAccountManager()

	if "relay" == globalConfig.Mode {
		n.registerRelayNode()
	} else if "miner" == globalConfig.Mode {
		n.registerMineNode()
	} else {
		n.registerMineNode()
		n.registerRelayNode()
	}

	return n
}

func (n *Node) registerRelayNode() {
	n.relayNode = &RelayNode{}
	n.registerTransactionManager()
	n.registerTrendManager()
	n.registerTickerCollector()
	n.registerWalletService()
	n.registerJsonRpcService()
	n.registerWebsocketService()
	n.registerSocketIOService()
}

func (n *Node) registerMineNode() {
	n.mineNode = &MineNode{}
	ks := keystore.NewKeyStore(n.globalConfig.Keystore.Keydir, keystore.StandardScryptN, keystore.StandardScryptP)
	n.registerCrypto(ks)
	n.registerMiner()
}

func (n *Node) Start() {
	n.orderManager.Start()
	n.extractorService.Start()

	// todo delete after test
	txManager := txmanager.NewTxManager(n.rdsService, &n.accountManager)
	txManager.Start()

	ethaccessor.IncludeGasPriceEvaluator()

	extractorSyncWatcher := &eventemitter.Watcher{Concurrent: false, Handle: n.startAfterExtractorSync}
	eventemitter.On(eventemitter.SyncChainComplete, extractorSyncWatcher)

	chainForkWatcher := &eventemitter.Watcher{Concurrent: false, Handle: n.startAfterChainFork}
	eventemitter.On(eventemitter.ChainForkDetected, chainForkWatcher)
}

func (n *Node) startAfterExtractorSync(input eventemitter.EventData) error {
	n.ipfsSubService.Start()
	n.marketCapProvider.Start()

	if n.globalConfig.Mode == MODEL_RELAY {
		n.relayNode.Start()
	} else if n.globalConfig.Mode == MODEL_MINER {
		n.mineNode.Start()
	} else {
		n.relayNode.Start()
		n.mineNode.Start()
	}

	return nil
}

func (n *Node) startAfterChainFork(input eventemitter.EventData) error {
	// stop extractor
	if n.globalConfig.Mode == MODEL_MINER {
		n.mineNode.Stop()
	} else if n.globalConfig.Mode == MODEL_RELAY {
		//n.relayNode.Stop()
	} else {
		//n.relayNode.Stop()
		n.mineNode.Stop()
	}
	n.extractorService.Stop()

	// emit fork event,waiting for ordermanager and accountmanager finished procedure of process chain fork
	forkEvent := input.(*types.ForkedEvent)
	eventemitter.Emit(eventemitter.ChainForkProcess, forkEvent)

	// reset new block number and start extractor
	nextBlockNumber := new(big.Int).Add(forkEvent.ForkBlock, big.NewInt(1))
	n.extractorService.Fork(nextBlockNumber)
	n.extractorService.Start()

	if n.globalConfig.Mode == MODEL_MINER {
		n.mineNode.Start()
	} else if n.globalConfig.Mode == MODEL_RELAY {
		//n.relayNode.Start()
	} else {
		//n.relayNode.Start()
		n.mineNode.Start()
	}

	return nil
}

func (n *Node) Wait() {
	n.lock.RLock()

	// TODO(fk): states should be judged

	stop := n.stop
	n.lock.RUnlock()

	<-stop
}

func (n *Node) Stop() {
	n.lock.RLock()
	n.mineNode.Stop()
	//
	//n.p2pListener.Stop()
	//n.chainListener.Stop()
	//n.orderbook.Stop()
	//n.miner.Stop()

	//close(n.stop)

	n.lock.RUnlock()
}

func (n *Node) registerCrypto(ks *keystore.KeyStore) {
	c := crypto.NewKSCrypto(true, ks)
	crypto.Initialize(c)
}

func (n *Node) registerMysql() {
	n.rdsService = dao.NewRdsService(n.globalConfig.Mysql)
	n.rdsService.Prepare()
}

func (n *Node) registerAccessor() {
	err := ethaccessor.Initialize(n.globalConfig.Accessor, n.globalConfig.Common, util.WethTokenAddress())
	if nil != err {
		log.Fatalf("err:%s", err.Error())
	}
}

func (n *Node) registerExtractor() {
	n.extractorService = extractor.NewExtractorService(n.globalConfig.Extractor, n.rdsService, &n.accountManager)
}

func (n *Node) registerIPFSSubService() {
	n.ipfsSubService = gateway.NewIPFSSubService(n.globalConfig.Ipfs)
}

func (n *Node) registerOrderManager() {
	n.orderManager = ordermanager.NewOrderManager(&n.globalConfig.OrderManager, n.rdsService, n.userManager, n.marketCapProvider)
}

func (n *Node) registerTrendManager() {
	n.relayNode.trendManager = market.NewTrendManager(n.rdsService)
}

func (n *Node) registerAccountManager() {
	n.accountManager = market.NewAccountManager()
}

func (n *Node) registerTransactionManager() {
	n.relayNode.txManager = txmanager.NewTxManager(n.rdsService, &n.accountManager)
}

func (n *Node) registerTickerCollector() {
	n.relayNode.tickerCollector = *market.NewCollector()
}

func (n *Node) registerWalletService() {
	ethForwarder := gateway.EthForwarder{}
	n.relayNode.walletService = *gateway.NewWalletService(n.relayNode.trendManager, n.orderManager,
		n.accountManager, n.marketCapProvider, &ethForwarder, n.relayNode.tickerCollector, n.rdsService, n.globalConfig.Market.OldVersionWethAddress)
}

func (n *Node) registerJsonRpcService() {
	n.relayNode.jsonRpcService = *gateway.NewJsonrpcService(n.globalConfig.Jsonrpc.Port, &n.relayNode.walletService)
}

func (n *Node) registerWebsocketService() {
	n.relayNode.websocketService = *gateway.NewWebsocketService(n.globalConfig.Websocket.Port, n.relayNode.trendManager, n.accountManager, n.marketCapProvider)
}

func (n *Node) registerSocketIOService() {
	n.relayNode.socketIOService = *gateway.NewSocketIOService(n.globalConfig.Websocket.Port, n.relayNode.walletService)
}

func (n *Node) registerMiner() {
	submitter, err := miner.NewSubmitter(n.globalConfig.Miner, n.rdsService, n.marketCapProvider)
	if nil != err {
		log.Fatalf("failed to init submitter, error:%s", err.Error())
	}
	evaluator := miner.NewEvaluator(n.marketCapProvider, n.globalConfig.Miner.RateRatioCVSThreshold, n.globalConfig.Miner.Subsidy, n.globalConfig.Miner.WalletSplit)
	matcher := timing_matcher.NewTimingMatcher(n.globalConfig.Miner.TimingMatcher, submitter, evaluator, n.orderManager, &n.accountManager)
	submitter.SetMatcher(matcher)
	n.mineNode.miner = miner.NewMiner(submitter, matcher, evaluator, n.marketCapProvider)
}

func (n *Node) registerGateway() {
	gateway.Initialize(&n.globalConfig.GatewayFilters, &n.globalConfig.Gateway, &n.globalConfig.Ipfs, n.orderManager, n.marketCapProvider)
}

func (n *Node) registerUserManager() {
	n.userManager = usermanager.NewUserManager(&n.globalConfig.UserManager, n.rdsService)
}

func (n *Node) registerMarketCap() {
	n.marketCapProvider = marketcap.NewMarketCapProvider(n.globalConfig.MarketCap)
}
