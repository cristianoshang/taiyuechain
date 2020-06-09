// Copyright 2016 The go-ethereum Authors
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

// Package les implements the Light Taiyuechain Subprotocol.
package les

import (
	"fmt"
	"sync"
	"time"

	"github.com/taiyuechain/taiyuechain/accounts"
	"github.com/taiyuechain/taiyuechain/common"
	"github.com/taiyuechain/taiyuechain/consensus"
	"github.com/taiyuechain/taiyuechain/core"
	"github.com/taiyuechain/taiyuechain/core/bloombits"
	"github.com/taiyuechain/taiyuechain/core/rawdb"
	"github.com/taiyuechain/taiyuechain/core/types"
	"github.com/taiyuechain/taiyuechain/event"
	"github.com/taiyuechain/taiyuechain/internal/taiapi"
	"github.com/taiyuechain/taiyuechain/light"
	"github.com/taiyuechain/taiyuechain/log"
	"github.com/taiyuechain/taiyuechain/node"
	"github.com/taiyuechain/taiyuechain/p2p"
	"github.com/taiyuechain/taiyuechain/p2p/discv5"
	"github.com/taiyuechain/taiyuechain/params"
	"github.com/taiyuechain/taiyuechain/rpc"
	"github.com/taiyuechain/taiyuechain/yue"
	"github.com/taiyuechain/taiyuechain/yue/downloader"
	"github.com/taiyuechain/taiyuechain/yue/filters"
	"github.com/taiyuechain/taiyuechain/yue/gasprice"
	"github.com/taiyuechain/taiyuechain/yuedb"
)

type LightEtrue struct {
	config *yue.Config

	odr         *LesOdr
	relay       *LesTxRelay
	chainConfig *params.ChainConfig
	// Channel for shutting down the service
	shutdownChan chan bool
	// Handlers
	peers           *peerSet
	txPool          *light.TxPool
	blockchain      *light.LightChain
	protocolManager *ProtocolManager
	serverPool      *serverPool
	reqDist         *requestDistributor
	retriever       *retrieveManager
	// DB interfaces
	chainDb yuedb.Database // Block chain database

	bloomRequests                              chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer, chtIndexer, bloomTrieIndexer *core.ChainIndexer

	ApiBackend *LesApiBackend

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	networkId     uint64
	netRPCService *taiapi.PublicNetAPI

	wg sync.WaitGroup
}

func New(ctx *node.ServiceContext, config *yue.Config) (*LightEtrue, error) {
	chainDb, err := yue.CreateDB(ctx, config, "lightchaindata")
	if err != nil {
		return nil, err
	}
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, isCompat := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !isCompat {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	peers := newPeerSet()
	quitSync := make(chan struct{})

	leth := &LightEtrue{
		config:           config,
		chainConfig:      chainConfig,
		chainDb:          chainDb,
		eventMux:         ctx.EventMux,
		peers:            peers,
		reqDist:          newRequestDistributor(peers, quitSync),
		accountManager:   ctx.AccountManager,
		engine:           yue.CreateConsensusEngine(ctx, &config.MinervaHash, chainConfig, chainDb),
		shutdownChan:     make(chan bool),
		networkId:        config.NetworkId,
		bloomRequests:    make(chan chan *bloombits.Retrieval),
		bloomIndexer:     yue.NewBloomIndexer(chainDb, light.BloomTrieFrequency),
		chtIndexer:       light.NewChtIndexer(chainDb, true),
		bloomTrieIndexer: light.NewBloomTrieIndexer(chainDb, true),
	}

	var trustedNodes []string
	if leth.config.ULC != nil {
		trustedNodes = leth.config.ULC.TrustedServers
	}
	leth.relay = NewLesTxRelay(peers, leth.reqDist)
	leth.serverPool = newServerPool(chainDb, quitSync, &leth.wg, trustedNodes)
	leth.retriever = newRetrieveManager(peers, leth.reqDist, leth.serverPool)
	leth.odr = NewLesOdr(chainDb, leth.chtIndexer, leth.bloomTrieIndexer, leth.bloomIndexer, leth.retriever)
	if leth.blockchain, err = light.NewLightChain(leth.odr, leth.chainConfig, leth.engine); err != nil {
		return nil, err
	}
	leth.bloomIndexer.Start(leth.blockchain)
	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		leth.blockchain.SetHead(compat.RewindTo)
		rawdb.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}

	leth.txPool = light.NewTxPool(leth.chainConfig, leth.blockchain, leth.relay)
	if leth.protocolManager, err = NewProtocolManager(leth.chainConfig, true, ClientProtocolVersions, config.NetworkId, leth.eventMux, leth.engine, leth.peers, nil, nil, chainDb, leth.odr, leth.relay, leth.serverPool, quitSync, &leth.wg); err != nil {
		return nil, err
	}
	leth.ApiBackend = &LesApiBackend{leth, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	leth.ApiBackend.gpo = gasprice.NewOracle(leth.ApiBackend, gpoParams)
	return leth, nil
}

func lesTopic(genesisHash common.Hash, protocolVersion uint) discv5.Topic {
	var name string
	switch protocolVersion {
	case lpv1:
		name = "LES"
	case lpv2:
		name = "LES2"
	default:
		panic(nil)
	}
	return discv5.Topic(name + "@" + common.Bytes2Hex(genesisHash.Bytes()[0:8]))
}

type LightDummyAPI struct{}

// Etherbase is the address that mining rewards will be send to
func (s *LightDummyAPI) Etherbase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("not supported")
}

// Coinbase is the address that mining rewards will be send to (alias for Etherbase)
func (s *LightDummyAPI) Coinbase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("not supported")
}

// APIs returns the collection of RPC services the ethereum package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *LightEtrue) APIs() []rpc.API {
	apis := taiapi.GetAPIs(s.ApiBackend)
	namespaces := []string{"etrue", "yue", "eth"}
	for _, name := range namespaces {
		apis = append(apis, []rpc.API{
			{
				Namespace: name,
				Version:   "1.0",
				Service:   &LightDummyAPI{},
				Public:    true,
			}, {
				Namespace: name,
				Version:   "1.0",
				Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
				Public:    true,
			}, {
				Namespace: name,
				Version:   "1.0",
				Service:   filters.NewPublicFilterAPI(s.ApiBackend, true),
				Public:    true,
			},
		}...)
	}
	apis = append(apis, []rpc.API{
		{
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
	return apis
}

func (s *LightEtrue) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *LightEtrue) BlockChain() *light.LightChain      { return s.blockchain }
func (s *LightEtrue) TxPool() *light.TxPool              { return s.txPool }
func (s *LightEtrue) Engine() consensus.Engine           { return s.engine }
func (s *LightEtrue) LesVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *LightEtrue) Downloader() *downloader.Downloader { return s.protocolManager.downloader }
func (s *LightEtrue) EventMux() *event.TypeMux           { return s.eventMux }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *LightEtrue) Protocols() []p2p.Protocol {
	return s.protocolManager.SubProtocols
}

// Start implements node.Service, starting all internal goroutines needed by the
// Taiyuechain protocol implementation.
func (s *LightEtrue) Start(srvr *p2p.Server) error {
	s.startBloomHandlers()
	log.Warn("Light client mode is an experimental feature")
	s.netRPCService = taiapi.NewPublicNetAPI(srvr, s.networkId)
	// clients are searching for the first advertised protocol in the list
	protocolVersion := AdvertiseProtocolVersions[0]
	s.serverPool.start(srvr, lesTopic(s.blockchain.Genesis().Hash(), protocolVersion))
	s.protocolManager.Start(s.config.LightPeers)
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// Taiyuechain protocol.
func (s *LightEtrue) Stop() error {
	s.odr.Stop()
	if s.bloomIndexer != nil {
		s.bloomIndexer.Close()
	}
	if s.chtIndexer != nil {
		s.chtIndexer.Close()
	}
	if s.bloomTrieIndexer != nil {
		s.bloomTrieIndexer.Close()
	}
	s.blockchain.Stop()
	s.protocolManager.Stop()
	s.txPool.Stop()

	s.eventMux.Stop()

	time.Sleep(time.Millisecond * 200)
	s.chainDb.Close()
	close(s.shutdownChan)

	return nil
}
