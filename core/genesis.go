// Copyright 2014 The go-ethereum Authors
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

package core

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/taiyuechain/taiyuechain/common"
	"github.com/taiyuechain/taiyuechain/common/hexutil"
	"github.com/taiyuechain/taiyuechain/common/math"
	"github.com/taiyuechain/taiyuechain/consensus"
	"github.com/taiyuechain/taiyuechain/core/rawdb"
	"github.com/taiyuechain/taiyuechain/core/state"
	"github.com/taiyuechain/taiyuechain/core/types"
	"github.com/taiyuechain/taiyuechain/crypto"
	"github.com/taiyuechain/taiyuechain/log"
	"github.com/taiyuechain/taiyuechain/params"
	"github.com/taiyuechain/taiyuechain/rlp"
	"github.com/taiyuechain/taiyuechain/yuedb"
)

//go:generate gencodec -type Genesis -field-override genesisSpecMarshaling -out gen_genesis.go
//go:generate gencodec -type GenesisAccount -field-override genesisAccountMarshaling -out gen_genesis_account.go

var errGenesisNoConfig = errors.New("genesis has no chain configuration")

const (
	SYMMETRICCRYPTOSM4    = 1
	SYMMETRICCRYPTOAES    = 2
	ASYMMETRICCRYPTOECDSA = 3
	ASYMMETRICCRYPTOSM2   = 4
	HASHCRYPTOSM3         = 5
	HASHCRYPTOHAS3        = 6
)

// Genesis specifies the header fields, state of a genesis block. It also defines hard
// fork switch-over blocks through the chain configuration.
type Genesis struct {
	Config       *params.ChainConfig      `json:"config"`
	Timestamp    uint64                   `json:"timestamp"`
	ExtraData    []byte                   `json:"extraData"`
	GasLimit     uint64                   `json:"gasLimit"   gencodec:"required"`
	UseGas       uint8                    `json:"useGas" 		gencodec:"required"`
	IsCoin   uint8                    	  `json:"isCoin" 		gencodec:"required"`
	KindOfCrypto uint8                    `json:"kindOfCrypto" 		gencodec:"required"`
	PermisionWlSendTx   uint8             `json:"permisionWlSendTx" 		gencodec:"required"`
	PermisionWlCreateTx  uint8            `json:"permisionWlCreateTx" 		gencodec:"required"`
	Coinbase     common.Address           `json:"coinbase"`
	Alloc        types.GenesisAlloc       `json:"alloc"`
	Committee    []*types.CommitteeMember `json:"committee"      gencodec:"required"`
	CertList     [][]byte                 `json:"CertList"      gencodec:"required"`

	// These fields are used for consensus tests. Please don't use them
	// in actual genesis blocks.
	Number     uint64      `json:"number"`
	GasUsed    uint64      `json:"gasUsed"`
	ParentHash common.Hash `json:"parentHash"`
}

// GenesisAccount is an account in the state of the genesis block.
type GenesisAccount struct {
	Code       []byte                      `json:"code,omitempty"`
	Storage    map[common.Hash]common.Hash `json:"storage,omitempty"`
	Balance    *big.Int                    `json:"balance" gencodec:"required"`
	Nonce      uint64                      `json:"nonce,omitempty"`
	PrivateKey []byte                      `json:"secretKey,omitempty"` // for tests
}

// field type overrides for gencodec
type genesisSpecMarshaling struct {
	Timestamp math.HexOrDecimal64
	ExtraData hexutil.Bytes
	GasLimit  math.HexOrDecimal64
	GasUsed   math.HexOrDecimal64
	Number    math.HexOrDecimal64
	CertList  []hexutil.Bytes
	Alloc     map[common.UnprefixedAddress]types.GenesisAccount
}

type genesisAccountMarshaling struct {
	Code       hexutil.Bytes
	Balance    *math.HexOrDecimal256
	Nonce      math.HexOrDecimal64
	Storage    map[storageJSON]storageJSON
	PrivateKey hexutil.Bytes
}

type specialString string
type Foo2 struct{ M map[string]string }

type foo2Marshaling struct{ S map[string]specialString }

// storageJSON represents a 256 bit byte array, but allows less than 256 bits when
// unmarshaling from hex.
type storageJSON common.Hash

func (h *storageJSON) UnmarshalText(text []byte) error {
	text = bytes.TrimPrefix(text, []byte("0x"))
	if len(text) > 64 {
		return fmt.Errorf("too many hex characters in storage key/value %q", text)
	}
	offset := len(h) - len(text)/2 // pad on the left
	if _, err := hex.Decode(h[offset:], text); err != nil {
		fmt.Println(err)
		return fmt.Errorf("invalid hex storage key/value %q", text)
	}
	return nil
}

func (h storageJSON) MarshalText() ([]byte, error) {
	return hexutil.Bytes(h[:]).MarshalText()
}

// GenesisMismatchError is raised when trying to overwrite an existing
// genesis block with an incompatible one.
type GenesisMismatchError struct {
	Stored, New common.Hash
}

func (e *GenesisMismatchError) Error() string {
	return fmt.Sprintf("database already contains an incompatible genesis block (have %x, new %x)", e.Stored[:8], e.New[:8])
}

// SetupGenesisBlock writes or updates the genesis block in db.
// The block that will be used is:
//
//                          genesis == nil       genesis != nil
//                       +------------------------------------------
//     db has no genesis |  main-net default  |  genesis
//     db has genesis    |  from DB           |  genesis (if compatible)
//
// The stored chain configuration will be updated if it is compatible (i.e. does not
// specify a fork block below the local head block). In case of a conflict, the
// error is a *params.ConfigCompatError and the new, unwritten config is returned.
//
// The returned chain configuration is never nil.
func SetupGenesisBlock(db yuedb.Database, genesis *Genesis) (*params.ChainConfig, common.Hash, error) {
	if genesis != nil && genesis.Config == nil {
		return params.AllMinervaProtocolChanges, common.Hash{}, errGenesisNoConfig
	}

	fastConfig, fastHash, fastErr := setupGenesisBlock(db, genesis)
	genesisBlock := rawdb.ReadBlock(db, fastHash, 0)
	if genesisBlock != nil {
		data := genesisBlock.Header().Extra
		params.ParseExtraDataFromGenesis(data)
		GasUsed, IsCoin, KindOfCrypto := data[0], data[1], data[2]
		if err := baseCheck(GasUsed,IsCoin,KindOfCrypto); err != nil {
			return nil,common.Hash{},err
		}
	}
	return fastConfig, fastHash, fastErr

}

// setupGenesisBlock writes or updates the fast genesis block in db.
// The block that will be used is:
//
//                          genesis == nil       genesis != nil
//                       +------------------------------------------
//     db has no genesis |  main-net default  |  genesis
//     db has genesis    |  from DB           |  genesis (if compatible)
//
// The stored chain configuration will be updated if it is compatible (i.e. does not
// specify a fork block below the local head block). In case of a conflict, the
// error is a *params.ConfigCompatError and the new, unwritten config is returned.
//
// The returned chain configuration is never nil.
func setupGenesisBlock(db yuedb.Database, genesis *Genesis) (*params.ChainConfig, common.Hash, error) {
	if genesis != nil && genesis.Config == nil {
		return params.AllMinervaProtocolChanges, common.Hash{}, errGenesisNoConfig
	}

	// Just commit the new block if there is no stored genesis block.
	stored := rawdb.ReadCanonicalHash(db, 0)
	if (stored == common.Hash{}) {
		if genesis == nil {
			log.Info("Writing default main-net genesis block")
			genesis = DefaultGenesisBlock()
		} else {
			log.Info("Writing custom genesis block")
		}
		block, err := genesis.Commit(db)
		return genesis.Config, block.Hash(), err
	}

	// Check whether the genesis block is already written.
	if genesis != nil {
		hash := genesis.ToBlock(nil).Hash()
		if hash != stored {
			return genesis.Config, hash, &GenesisMismatchError{stored, hash}
		}
	}

	// Get the existing chain configuration.
	newcfg := genesis.configOrDefault(stored)
	storedcfg := rawdb.ReadChainConfig(db, stored)
	if storedcfg == nil {
		log.Warn("Found genesis block without chain config")
		rawdb.WriteChainConfig(db, stored, newcfg)
		return newcfg, stored, nil
	}
	// Special case: don't change the existing config of a non-mainnet chain if no new
	// config is supplied. These chains would get AllProtocolChanges (and a compat error)
	// if we just continued here.
	if genesis == nil && stored != params.MainnetGenesisHash {
		return storedcfg, stored, nil
	}

	// Check config compatibility and write the config. Compatibility errors
	// are returned to the caller unless we're already at block zero.
	height := rawdb.ReadHeaderNumber(db, rawdb.ReadHeadHeaderHash(db))
	if height == nil {
		return newcfg, stored, fmt.Errorf("missing block number for head header hash")
	}
	compatErr := storedcfg.CheckCompatible(newcfg, *height)
	if compatErr != nil && *height != 0 && compatErr.RewindTo != 0 {
		return newcfg, stored, compatErr
	}
	rawdb.WriteChainConfig(db, stored, newcfg)
	return newcfg, stored, nil
}

// Commit writes the block and state of a genesis specification to the database.
// The block is committed as the canonical head block.
func (g *Genesis) Commit(db yuedb.Database) (*types.Block, error) {
	block := g.ToBlock(db)
	if block.Number().Sign() != 0 {
		return nil, fmt.Errorf("can't commit genesis block with number > 0")
	}
	rawdb.WriteBlock(db, block)
	rawdb.WriteReceipts(db, block.Hash(), block.NumberU64(), nil)
	rawdb.WriteCanonicalHash(db, block.Hash(), block.NumberU64())
	rawdb.WriteHeadBlockHash(db, block.Hash())
	rawdb.WriteHeadHeaderHash(db, block.Hash())
	rawdb.WriteStateGcBR(db, block.NumberU64())

	config := g.Config
	if config == nil {
		config = params.AllMinervaProtocolChanges
	}
	rawdb.WriteChainConfig(db, block.Hash(), config)
	return block, nil
}
func (g *Genesis) makeExtraData() []byte {
	h := []byte{g.UseGas, g.IsCoin, g.KindOfCrypto,g.PermisionWlSendTx,g.PermisionWlCreateTx}
	g.ExtraData = h
	return g.ExtraData
}

// ToBlock creates the genesis block and writes state of a genesis specification
// to the given database (or discards it if nil).
func (g *Genesis) ToBlock(db yuedb.Database) *types.Block {

	if db == nil {
		db = yuedb.NewMemDatabase()
	}
	g.ExtraData = g.makeExtraData()
	params.ParseExtraDataFromGenesis(g.ExtraData)
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(db))
	for addr, account := range g.Alloc {
		if g.IsCoin > 0 {
			statedb.AddBalance(addr, account.Balance)
		}
		statedb.SetCode(addr, account.Code)
		statedb.SetNonce(addr, account.Nonce)
		for key, value := range account.Storage {
			statedb.SetState(addr, key, value)
		}
	}
	pubk := [][]byte{}
	coinAddr := []common.Address{}
	for _,v:= range  g.Committee{
		pubk = append(pubk,v.Publickey)
		coinAddr = append(coinAddr,v.Coinbase)
	}
	consensus.OnceInitCAState(statedb, new(big.Int).SetUint64(g.Number), g.CertList,pubk,coinAddr)
	root := statedb.IntermediateRoot(false)

	head := &types.Header{
		Number:     new(big.Int).SetUint64(g.Number),
		Time:       new(big.Int).SetUint64(g.Timestamp),
		ParentHash: g.ParentHash,
		Extra:      g.ExtraData,
		GasLimit:   g.GasLimit,
		GasUsed:    g.GasUsed,
		Root:       root,
	}
	if g.GasLimit == 0 {
		head.GasLimit = params.GenesisGasLimit
	}
	statedb.Commit(false)
	statedb.Database().TrieDB().Commit(root, true)

	// All genesis committee members are included in switchinfo of block #0
	committee := &types.SwitchInfos{CID: common.Big0, Members: g.Committee, BackMembers: make([]*types.CommitteeMember, 0), Vals: make([]*types.SwitchEnter, 0)}
	for _, member := range committee.Members {
		//caolaing modify
		//pubkey, _ := crypto.UnmarshalPubkey(member.Publickey)
		// cc := hex.EncodeToString(member.Publickey)
		// fmt.Sprintln("cccccc" + cc)
		pubkey, e := crypto.UnmarshalPubkey(member.Publickey)
		if e != nil {
			fmt.Println(e)
		}
		member.Flag = types.StateUsedFlag
		member.MType = types.TypeFixed
		//caolaing modify
		member.CommitteeBase = crypto.PubkeyToAddress(*pubkey)
		///member.CommitteeBase = taipublic.PubkeyToAddress(*pubkey)
	}
	return types.NewBlock(head, nil, nil, nil, committee.Members)
}

// MustCommit writes the genesis block and state to db, panicking on error.
// The block is committed as the canonical head block.
func (g *Genesis) MustCommit(db yuedb.Database) *types.Block {
	if err := baseCheck(g.UseGas,g.IsCoin,g.KindOfCrypto); err != nil {
		panic(err)
	}
	block, err := g.Commit(db)
	if err != nil {
		panic(err)
	}
	return block
}

// DefaultGenesisBlock returns the Taiyuechain main net snail block.
func DefaultGenesisBlock() *Genesis {
	seedkey1 := hexutil.MustDecode("0x04bdf9699d20b4ebabe76e76260480e5492c87aaeda51b138bd22c6d66b69549313dc3eb8c96dc9a1cbbf3b347322c51c05afdd609622277444e0f07e6bd35d8bd")
	seedkey2 := hexutil.MustDecode("0x045e1711b6cd8550a5e5466f7f0868b5507929cb69c2f0fca84f8f94816eb40a808ea8a77c3d83c9d16341acb037fbea2f7d9d4af46326defa39b408f40f28fb98")
	seedkey3 := hexutil.MustDecode("0x041b931d350257e881f27bce2563d98c99b13ca4f525a0662f5e7d53f085edff0dca8ceaae550c9f4ceecf217f72806a48a48fb024916392ae41d7c45168e89b94")
	seedkey4 := hexutil.MustDecode("0x049923777d866fd80485be57a126d638cc7dda78a5d6958aff784ca7ed9d9c7be494125bf75fd0328490ae51020274427b9fbb07f59e4c9b5104ac6924721a4438")

	strcert1 := "3082028e3082023aa0030201020201ff300a06082a811ccf55018375303031133011060355040a0c0acea32041636d6520436f3119301706035504031310746573742e6578616d706c652e636f6d301e170d3230303630313034313133385a170d3233303830323133353831385a303031133011060355040a0c0acea32041636d6520436f3119301706035504031310746573742e6578616d706c652e636f6d3059301306072a8648ce3d020106082a811ccf5501822d03420004bdf9699d20b4ebabe76e76260480e5492c87aaeda51b138bd22c6d66b69549313dc3eb8c96dc9a1cbbf3b347322c51c05afdd609622277444e0f07e6bd35d8bda38201433082013f300e0603551d0f0101ff04040302020430260603551d25041f301d06082b0601050507030206082b0601050507030106022a030603810b01300f0603551d130101ff040530030101ff300d0603551d0e0406040401020304305f06082b0601050507010104533051302306082b060105050730018617687474703a2f2f6f6373702e6578616d706c652e636f6d302a06082b06010505073002861e687474703a2f2f6372742e6578616d706c652e636f6d2f6361312e637274301a0603551d1104133011820f666f6f2e6578616d706c652e636f6d300f0603551d2004083006300406022a0330570603551d1f0450304e3025a023a021861f687474703a2f2f63726c312e6578616d706c652e636f6d2f6361312e63726c3025a023a021861f687474703a2f2f63726c322e6578616d706c652e636f6d2f6361312e63726c300a06082a811ccf550183750342008851ca997c3b35b6de11fa5e43d04dfb76cd4177c4517e60f72db9373fec1a3731c46b70b562240a1cbd98e22dec6e1fd857e6b88fee893897c39e61e9bb502c01"
	strcert2 := "3082028e3082023aa0030201020201ff300a06082a811ccf55018375303031133011060355040a0c0acea32041636d6520436f3119301706035504031310746573742e6578616d706c652e636f6d301e170d3230303630313034313134335a170d3233303830323133353832335a303031133011060355040a0c0acea32041636d6520436f3119301706035504031310746573742e6578616d706c652e636f6d3059301306072a8648ce3d020106082a811ccf5501822d034200045e1711b6cd8550a5e5466f7f0868b5507929cb69c2f0fca84f8f94816eb40a808ea8a77c3d83c9d16341acb037fbea2f7d9d4af46326defa39b408f40f28fb98a38201433082013f300e0603551d0f0101ff04040302020430260603551d25041f301d06082b0601050507030206082b0601050507030106022a030603810b01300f0603551d130101ff040530030101ff300d0603551d0e0406040401020304305f06082b0601050507010104533051302306082b060105050730018617687474703a2f2f6f6373702e6578616d706c652e636f6d302a06082b06010505073002861e687474703a2f2f6372742e6578616d706c652e636f6d2f6361312e637274301a0603551d1104133011820f666f6f2e6578616d706c652e636f6d300f0603551d2004083006300406022a0330570603551d1f0450304e3025a023a021861f687474703a2f2f63726c312e6578616d706c652e636f6d2f6361312e63726c3025a023a021861f687474703a2f2f63726c322e6578616d706c652e636f6d2f6361312e63726c300a06082a811ccf55018375034200626bafd3a7e8a296305bba2e097c340c48345fafd02eb305721c08c808cf9456728a64a8f2b5af08caa36c24b1f994c918ece5aa90efb859e81dc6dd4111591f03"
	strcert3 := "3082028e3082023aa0030201020201ff300a06082a811ccf55018375303031133011060355040a0c0acea32041636d6520436f3119301706035504031310746573742e6578616d706c652e636f6d301e170d3230303630313034313134345a170d3233303830323133353832345a303031133011060355040a0c0acea32041636d6520436f3119301706035504031310746573742e6578616d706c652e636f6d3059301306072a8648ce3d020106082a811ccf5501822d034200041b931d350257e881f27bce2563d98c99b13ca4f525a0662f5e7d53f085edff0dca8ceaae550c9f4ceecf217f72806a48a48fb024916392ae41d7c45168e89b94a38201433082013f300e0603551d0f0101ff04040302020430260603551d25041f301d06082b0601050507030206082b0601050507030106022a030603810b01300f0603551d130101ff040530030101ff300d0603551d0e0406040401020304305f06082b0601050507010104533051302306082b060105050730018617687474703a2f2f6f6373702e6578616d706c652e636f6d302a06082b06010505073002861e687474703a2f2f6372742e6578616d706c652e636f6d2f6361312e637274301a0603551d1104133011820f666f6f2e6578616d706c652e636f6d300f0603551d2004083006300406022a0330570603551d1f0450304e3025a023a021861f687474703a2f2f63726c312e6578616d706c652e636f6d2f6361312e63726c3025a023a021861f687474703a2f2f63726c322e6578616d706c652e636f6d2f6361312e63726c300a06082a811ccf550183750342000c17b347c9fe8d7721bcbebdbe111ebe9c41841a9dc87d2043efc45269c130878cb7dfb84804373070e1a8a19cd2a6d0b06b4db580e50c6030ea16232c814b5c01"
	strcert4 := "3082028e3082023aa0030201020201ff300a06082a811ccf55018375303031133011060355040a0c0acea32041636d6520436f3119301706035504031310746573742e6578616d706c652e636f6d301e170d3230303630313034313134345a170d3233303830323133353832345a303031133011060355040a0c0acea32041636d6520436f3119301706035504031310746573742e6578616d706c652e636f6d3059301306072a8648ce3d020106082a811ccf5501822d034200049923777d866fd80485be57a126d638cc7dda78a5d6958aff784ca7ed9d9c7be494125bf75fd0328490ae51020274427b9fbb07f59e4c9b5104ac6924721a4438a38201433082013f300e0603551d0f0101ff04040302020430260603551d25041f301d06082b0601050507030206082b0601050507030106022a030603810b01300f0603551d130101ff040530030101ff300d0603551d0e0406040401020304305f06082b0601050507010104533051302306082b060105050730018617687474703a2f2f6f6373702e6578616d706c652e636f6d302a06082b06010505073002861e687474703a2f2f6372742e6578616d706c652e636f6d2f6361312e637274301a0603551d1104133011820f666f6f2e6578616d706c652e636f6d300f0603551d2004083006300406022a0330570603551d1f0450304e3025a023a021861f687474703a2f2f63726c312e6578616d706c652e636f6d2f6361312e63726c3025a023a021861f687474703a2f2f63726c322e6578616d706c652e636f6d2f6361312e63726c300a06082a811ccf550183750342005eaa2519c0ef76a2243d08caa7b0306a11238d64b080d0b5d300e56aad312856fbef8394703b51e8faa7996cfa69923927253f6fe5e2ff7587f661f967ca7d3b01"
	cert1, _ := hex.DecodeString(strcert1)
	cert2, _ := hex.DecodeString(strcert2)
	cert3, _ := hex.DecodeString(strcert3)
	cert4, _ := hex.DecodeString(strcert4)
	var certList = [][]byte{cert1, cert2, cert3, cert4}
	coinbase := common.HexToAddress("0x9331cf34D0e3E43bce7de1bFd30a59d3EEc106B6")
	amount1, _ := new(big.Int).SetString("24000000000000000000000000", 10)

	return &Genesis{
		Config:       params.MainnetChainConfig,
		GasLimit:     16777216,
		UseGas:       1,
		IsCoin:   1,
		KindOfCrypto: 2,
		PermisionWlSendTx:		1,
		PermisionWlCreateTx:		1,
		Timestamp:    1537891200,
		Coinbase:   common.HexToAddress("0x0000000000000000000000000000000000000000"),
		ParentHash: common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
		Alloc: map[common.Address]types.GenesisAccount{
			common.HexToAddress("0x9dA04184dB45870Ee6A5F8A415F93015886cC768"): {Balance: amount1},
			common.HexToAddress("0x5A778953403352839Faf865C82309B63965f15F2"): {Balance: amount1},
			common.HexToAddress("0x1b3d007C0D5318D241F26374F379E882cDCbc371"): {Balance: amount1},
			common.HexToAddress("0xFE9cFAc0EDf17FB746069f1d12885217fF30234C"): {Balance: amount1},
		},
		Committee: []*types.CommitteeMember{
			{Coinbase: coinbase, Publickey: seedkey1},
			{Coinbase: coinbase, Publickey: seedkey2},
			{Coinbase: coinbase, Publickey: seedkey3},
			{Coinbase: coinbase, Publickey: seedkey4},
		},
		CertList: certList,
	}
}

func (g *Genesis) configOrDefault(ghash common.Hash) *params.ChainConfig {
	switch {
	case g != nil:
		return g.Config
	case ghash == params.MainnetGenesisHash:
		return params.MainnetChainConfig
	case ghash == params.MainnetSnailGenesisHash:
		return params.MainnetChainConfig
	case ghash == params.TestnetGenesisHash:
		return params.TestnetChainConfig
	case ghash == params.TestnetSnailGenesisHash:
		return params.TestnetChainConfig
	default:
		return params.AllMinervaProtocolChanges
	}
}

func decodePrealloc(data string) types.GenesisAlloc {
	var p []struct{ Addr, Balance *big.Int }
	if err := rlp.NewStream(strings.NewReader(data), 0).Decode(&p); err != nil {
		panic(err)
	}
	ga := make(types.GenesisAlloc, len(p))
	for _, account := range p {
		ga[common.BigToAddress(account.Addr)] = types.GenesisAccount{Balance: account.Balance}
	}
	return ga
}

// GenesisBlockForTesting creates and writes a block in which addr has the given wei balance.
func GenesisBlockForTesting(db yuedb.Database, addr common.Address, balance *big.Int) *types.Block {
	g := Genesis{Alloc: types.GenesisAlloc{addr: {Balance: balance}}, Config: params.AllMinervaProtocolChanges}
	return g.MustCommit(db)
}

// DefaultDevGenesisBlock returns the Rinkeby network genesis block.
func DefaultDevGenesisBlock() *Genesis {
	i, _ := new(big.Int).SetString("90000000000000000000000", 10)

	return &Genesis{
		Config:       params.DevnetChainConfig,
		GasLimit:     88080384,
		UseGas:       1,
		IsCoin:   1,
		KindOfCrypto: 2,
		PermisionWlSendTx:		1,
		PermisionWlCreateTx:		1,
		//Alloc:      decodePrealloc(mainnetAllocData),
		Alloc: map[common.Address]types.GenesisAccount{
			common.HexToAddress("0x3f9061bf173d8f096c94db95c40f3658b4c7eaad"): {Balance: i},
			common.HexToAddress("0x2cdac3658f85b5da3b70223cc3ad3b2dfe7c1930"): {Balance: i},
			common.HexToAddress("0x41acde8dd7611338c2a30e90149e682566716e9d"): {Balance: i},
			common.HexToAddress("0x0ffd116a3bf97a7112ff8779cc770b13ea3c66a5"): {Balance: i},
		},
		Committee: []*types.CommitteeMember{},
	}
}

func DefaultSingleNodeGenesisBlock() *Genesis {
	i, _ := new(big.Int).SetString("90000000000000000000000", 10)
	key1 := hexutil.MustDecode(
		"0x04bdf9699d20b4ebabe76e76260480e5492c87aaeda51b138bd22c6d66b69549313dc3eb8c96dc9a1cbbf3b347322c51c05afdd609622277444e0f07e6bd35d8bd")

	strcert1 := "3082028e3082023aa0030201020201ff300a06082a811ccf55018375303031133011060355040a0c0acea32041636d6520436f3119301706035504031310746573742e6578616d706c652e636f6d301e170d3230303630313034313133385a170d3233303830323133353831385a303031133011060355040a0c0acea32041636d6520436f3119301706035504031310746573742e6578616d706c652e636f6d3059301306072a8648ce3d020106082a811ccf5501822d03420004bdf9699d20b4ebabe76e76260480e5492c87aaeda51b138bd22c6d66b69549313dc3eb8c96dc9a1cbbf3b347322c51c05afdd609622277444e0f07e6bd35d8bda38201433082013f300e0603551d0f0101ff04040302020430260603551d25041f301d06082b0601050507030206082b0601050507030106022a030603810b01300f0603551d130101ff040530030101ff300d0603551d0e0406040401020304305f06082b0601050507010104533051302306082b060105050730018617687474703a2f2f6f6373702e6578616d706c652e636f6d302a06082b06010505073002861e687474703a2f2f6372742e6578616d706c652e636f6d2f6361312e637274301a0603551d1104133011820f666f6f2e6578616d706c652e636f6d300f0603551d2004083006300406022a0330570603551d1f0450304e3025a023a021861f687474703a2f2f63726c312e6578616d706c652e636f6d2f6361312e63726c3025a023a021861f687474703a2f2f63726c322e6578616d706c652e636f6d2f6361312e63726c300a06082a811ccf550183750342008851ca997c3b35b6de11fa5e43d04dfb76cd4177c4517e60f72db9373fec1a3731c46b70b562240a1cbd98e22dec6e1fd857e6b88fee893897c39e61e9bb502c01"
	cert1, _ := hex.DecodeString(strcert1)

	var certList = [][]byte{cert1}
	return &Genesis{
		Config:       params.SingleNodeChainConfig,
		GasLimit:     22020096,
		UseGas:       1,
		IsCoin:   1,
		KindOfCrypto: 2,
		PermisionWlSendTx:		1,
		PermisionWlCreateTx:		1,
		//Alloc:      decodePrealloc(mainnetAllocData),
		Alloc: map[common.Address]types.GenesisAccount{
			common.HexToAddress("0x9dA04184dB45870Ee6A5F8A415F93015886cC768"): {Balance: i},
		},
		Committee: []*types.CommitteeMember{
			{Coinbase: common.HexToAddress("0x76ea2f3a002431fede1141b660dbb75c26ba6d97"), Publickey: key1},
		},
		CertList: certList,
	}
}

// DefaultTestnetGenesisBlock returns the Ropsten network genesis block.
func DefaultTestnetGenesisBlock() *Genesis {
	seedkey1 := hexutil.MustDecode("0x04bdf9699d20b4ebabe76e76260480e5492c87aaeda51b138bd22c6d66b69549313dc3eb8c96dc9a1cbbf3b347322c51c05afdd609622277444e0f07e6bd35d8bd")
	seedkey2 := hexutil.MustDecode("0x045e1711b6cd8550a5e5466f7f0868b5507929cb69c2f0fca84f8f94816eb40a808ea8a77c3d83c9d16341acb037fbea2f7d9d4af46326defa39b408f40f28fb98")
	seedkey3 := hexutil.MustDecode("0x041b931d350257e881f27bce2563d98c99b13ca4f525a0662f5e7d53f085edff0dca8ceaae550c9f4ceecf217f72806a48a48fb024916392ae41d7c45168e89b94")
	seedkey4 := hexutil.MustDecode("0x049923777d866fd80485be57a126d638cc7dda78a5d6958aff784ca7ed9d9c7be494125bf75fd0328490ae51020274427b9fbb07f59e4c9b5104ac6924721a4438")

	strcert1 := "3082028e3082023aa0030201020201ff300a06082a811ccf55018375303031133011060355040a0c0acea32041636d6520436f3119301706035504031310746573742e6578616d706c652e636f6d301e170d3230303630313034313133385a170d3233303830323133353831385a303031133011060355040a0c0acea32041636d6520436f3119301706035504031310746573742e6578616d706c652e636f6d3059301306072a8648ce3d020106082a811ccf5501822d03420004bdf9699d20b4ebabe76e76260480e5492c87aaeda51b138bd22c6d66b69549313dc3eb8c96dc9a1cbbf3b347322c51c05afdd609622277444e0f07e6bd35d8bda38201433082013f300e0603551d0f0101ff04040302020430260603551d25041f301d06082b0601050507030206082b0601050507030106022a030603810b01300f0603551d130101ff040530030101ff300d0603551d0e0406040401020304305f06082b0601050507010104533051302306082b060105050730018617687474703a2f2f6f6373702e6578616d706c652e636f6d302a06082b06010505073002861e687474703a2f2f6372742e6578616d706c652e636f6d2f6361312e637274301a0603551d1104133011820f666f6f2e6578616d706c652e636f6d300f0603551d2004083006300406022a0330570603551d1f0450304e3025a023a021861f687474703a2f2f63726c312e6578616d706c652e636f6d2f6361312e63726c3025a023a021861f687474703a2f2f63726c322e6578616d706c652e636f6d2f6361312e63726c300a06082a811ccf550183750342008851ca997c3b35b6de11fa5e43d04dfb76cd4177c4517e60f72db9373fec1a3731c46b70b562240a1cbd98e22dec6e1fd857e6b88fee893897c39e61e9bb502c01"
	strcert2 := "3082028e3082023aa0030201020201ff300a06082a811ccf55018375303031133011060355040a0c0acea32041636d6520436f3119301706035504031310746573742e6578616d706c652e636f6d301e170d3230303630313034313134335a170d3233303830323133353832335a303031133011060355040a0c0acea32041636d6520436f3119301706035504031310746573742e6578616d706c652e636f6d3059301306072a8648ce3d020106082a811ccf5501822d034200045e1711b6cd8550a5e5466f7f0868b5507929cb69c2f0fca84f8f94816eb40a808ea8a77c3d83c9d16341acb037fbea2f7d9d4af46326defa39b408f40f28fb98a38201433082013f300e0603551d0f0101ff04040302020430260603551d25041f301d06082b0601050507030206082b0601050507030106022a030603810b01300f0603551d130101ff040530030101ff300d0603551d0e0406040401020304305f06082b0601050507010104533051302306082b060105050730018617687474703a2f2f6f6373702e6578616d706c652e636f6d302a06082b06010505073002861e687474703a2f2f6372742e6578616d706c652e636f6d2f6361312e637274301a0603551d1104133011820f666f6f2e6578616d706c652e636f6d300f0603551d2004083006300406022a0330570603551d1f0450304e3025a023a021861f687474703a2f2f63726c312e6578616d706c652e636f6d2f6361312e63726c3025a023a021861f687474703a2f2f63726c322e6578616d706c652e636f6d2f6361312e63726c300a06082a811ccf55018375034200626bafd3a7e8a296305bba2e097c340c48345fafd02eb305721c08c808cf9456728a64a8f2b5af08caa36c24b1f994c918ece5aa90efb859e81dc6dd4111591f03"
	strcert3 := "3082028e3082023aa0030201020201ff300a06082a811ccf55018375303031133011060355040a0c0acea32041636d6520436f3119301706035504031310746573742e6578616d706c652e636f6d301e170d3230303630313034313134345a170d3233303830323133353832345a303031133011060355040a0c0acea32041636d6520436f3119301706035504031310746573742e6578616d706c652e636f6d3059301306072a8648ce3d020106082a811ccf5501822d034200041b931d350257e881f27bce2563d98c99b13ca4f525a0662f5e7d53f085edff0dca8ceaae550c9f4ceecf217f72806a48a48fb024916392ae41d7c45168e89b94a38201433082013f300e0603551d0f0101ff04040302020430260603551d25041f301d06082b0601050507030206082b0601050507030106022a030603810b01300f0603551d130101ff040530030101ff300d0603551d0e0406040401020304305f06082b0601050507010104533051302306082b060105050730018617687474703a2f2f6f6373702e6578616d706c652e636f6d302a06082b06010505073002861e687474703a2f2f6372742e6578616d706c652e636f6d2f6361312e637274301a0603551d1104133011820f666f6f2e6578616d706c652e636f6d300f0603551d2004083006300406022a0330570603551d1f0450304e3025a023a021861f687474703a2f2f63726c312e6578616d706c652e636f6d2f6361312e63726c3025a023a021861f687474703a2f2f63726c322e6578616d706c652e636f6d2f6361312e63726c300a06082a811ccf550183750342000c17b347c9fe8d7721bcbebdbe111ebe9c41841a9dc87d2043efc45269c130878cb7dfb84804373070e1a8a19cd2a6d0b06b4db580e50c6030ea16232c814b5c01"
	strcert4 := "3082028e3082023aa0030201020201ff300a06082a811ccf55018375303031133011060355040a0c0acea32041636d6520436f3119301706035504031310746573742e6578616d706c652e636f6d301e170d3230303630313034313134345a170d3233303830323133353832345a303031133011060355040a0c0acea32041636d6520436f3119301706035504031310746573742e6578616d706c652e636f6d3059301306072a8648ce3d020106082a811ccf5501822d034200049923777d866fd80485be57a126d638cc7dda78a5d6958aff784ca7ed9d9c7be494125bf75fd0328490ae51020274427b9fbb07f59e4c9b5104ac6924721a4438a38201433082013f300e0603551d0f0101ff04040302020430260603551d25041f301d06082b0601050507030206082b0601050507030106022a030603810b01300f0603551d130101ff040530030101ff300d0603551d0e0406040401020304305f06082b0601050507010104533051302306082b060105050730018617687474703a2f2f6f6373702e6578616d706c652e636f6d302a06082b06010505073002861e687474703a2f2f6372742e6578616d706c652e636f6d2f6361312e637274301a0603551d1104133011820f666f6f2e6578616d706c652e636f6d300f0603551d2004083006300406022a0330570603551d1f0450304e3025a023a021861f687474703a2f2f63726c312e6578616d706c652e636f6d2f6361312e63726c3025a023a021861f687474703a2f2f63726c322e6578616d706c652e636f6d2f6361312e63726c300a06082a811ccf550183750342005eaa2519c0ef76a2243d08caa7b0306a11238d64b080d0b5d300e56aad312856fbef8394703b51e8faa7996cfa69923927253f6fe5e2ff7587f661f967ca7d3b01"
	cert1, _ := hex.DecodeString(strcert1)
	cert2, _ := hex.DecodeString(strcert2)
	cert3, _ := hex.DecodeString(strcert3)
	cert4, _ := hex.DecodeString(strcert4)
	var certList = [][]byte{cert1, cert2, cert3, cert4}
	coinbase := common.HexToAddress("0x9331cf34D0e3E43bce7de1bFd30a59d3EEc106B6")
	amount1, _ := new(big.Int).SetString("24000000000000000000000000", 10)
	return &Genesis{
		Config:       params.TestnetChainConfig,
		GasLimit:     20971520,
		UseGas:       1,
		IsCoin:   1,
		KindOfCrypto: 2,
		PermisionWlSendTx:		1,
		PermisionWlCreateTx:		1,
		Timestamp:    1537891200,
		Coinbase:     common.HexToAddress("0x0000000000000000000000000000000000000000"),
		ParentHash:   common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
		Alloc: map[common.Address]types.GenesisAccount{
			common.HexToAddress("0x9dA04184dB45870Ee6A5F8A415F93015886cC768"): {Balance: amount1},
			common.HexToAddress("0x5A778953403352839Faf865C82309B63965f15F2"): {Balance: amount1},
			common.HexToAddress("0x1b3d007C0D5318D241F26374F379E882cDCbc371"): {Balance: amount1},
			common.HexToAddress("0xFE9cFAc0EDf17FB746069f1d12885217fF30234C"): {Balance: amount1},
		},
		Committee: []*types.CommitteeMember{
			{Coinbase: coinbase, Publickey: seedkey1},
			{Coinbase: coinbase, Publickey: seedkey2},
			{Coinbase: coinbase, Publickey: seedkey3},
			{Coinbase: coinbase, Publickey: seedkey4},
		},
		CertList: certList,
	}
}
func baseCheck(useGas,isCoin,kindCrypto byte) error {
	if int(kindCrypto) < crypto.CRYPTO_P256_SH3_AES || int(kindCrypto) > crypto.CRYPTO_S256_SH3_AES {
		return errors.New("wrong param on kindCrypto")
	}
	if isCoin == 0 && useGas != 0 {
		return errors.New("has gas used on no any rewards")
	}
	return nil
}