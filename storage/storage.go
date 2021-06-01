// Copyright (C) 2021 Aung Maw
// Licensed under the GNU General Public License v3.0

package storage

import (
	"crypto"
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/aungmawjj/juria-blockchain/core"
	"github.com/aungmawjj/juria-blockchain/logger"
	"github.com/aungmawjj/juria-blockchain/merkle"
	"github.com/dgraph-io/badger/v3"
	_ "golang.org/x/crypto/sha3"
)

type CommitData struct {
	Block        *core.Block
	QC           *core.QuorumCert // QC for commited block
	Transactions []*core.Transaction
	BlockCommit  *core.BlockCommit
	TxCommits    []*core.TxCommit
	merkleUpdate *merkle.UpdateResult
}

type Config struct {
	MerkleBranchFactor uint8
}

var DefaultConfig = Config{
	MerkleBranchFactor: 8,
}

type Storage struct {
	db          *badger.DB
	chainStore  *chainStore
	stateStore  *stateStore
	merkleStore *merkleStore
	merkleTree  *merkle.Tree

	// for writeStateTree and VerifyState
	mtxWriteState sync.RWMutex
}

func New(db *badger.DB, config Config) *Storage {
	strg := new(Storage)
	strg.db = db
	getter := &badgerGetter{db}
	strg.chainStore = &chainStore{getter}
	strg.stateStore = &stateStore{getter, crypto.SHA3_256}
	strg.merkleStore = &merkleStore{getter}
	strg.merkleTree = merkle.NewTree(strg.merkleStore, crypto.SHA3_256, config.MerkleBranchFactor)
	return strg
}

func (strg *Storage) Commit(data *CommitData) error {
	return strg.commit(data)
}

func (strg *Storage) GetBlock(hash []byte) (*core.Block, error) {
	return strg.chainStore.getBlock(hash)
}

func (strg *Storage) GetLastBlock() (*core.Block, error) {
	return strg.chainStore.getLastBlock()
}

func (strg *Storage) GetLastQC() (*core.QuorumCert, error) {
	return strg.chainStore.getLastQC()
}

func (strg *Storage) GetBlockHeight() uint64 {
	height, _ := strg.chainStore.getBlockHeight()
	return height
}

func (strg *Storage) GetBlockByHeight(height uint64) (*core.Block, error) {
	return strg.chainStore.getBlockByHeight(height)
}

func (strg *Storage) GetBlockCommit(hash []byte) (*core.BlockCommit, error) {
	return strg.chainStore.getBlockCommit(hash)
}

func (strg *Storage) GetTx(hash []byte) (*core.Transaction, error) {
	return strg.chainStore.getTx(hash)
}

func (strg *Storage) HasTx(hash []byte) bool {
	return strg.chainStore.hasTx(hash)
}

func (strg *Storage) GetTxCommit(hash []byte) (*core.TxCommit, error) {
	return strg.chainStore.getTxCommit(hash)
}

func (strg *Storage) GetState(key []byte) []byte {
	return strg.stateStore.getState(key)
}

func (strg *Storage) VerifyState(key []byte) ([]byte, error) {
	strg.mtxWriteState.RLock()
	defer strg.mtxWriteState.RUnlock()

	value := strg.stateStore.getState(key)
	merkleIdx, err := strg.stateStore.getMerkleIndex(key)
	if err != nil {
		return nil, errors.New("state not found: " + err.Error())
	}
	node := &merkle.Node{
		Data:     strg.stateStore.sumStateValue(value),
		Position: merkle.NewPosition(0, big.NewInt(0).SetBytes(merkleIdx)),
	}
	if !strg.merkleTree.Verify([]*merkle.Node{node}) {
		return nil, errors.New("merkle verification failed")
	}
	return value, nil
}

func (strg *Storage) GetMerkleRoot() []byte {
	root := strg.merkleTree.Root()
	if root == nil {
		return nil
	}
	return root.Data
}

func (strg *Storage) commit(data *CommitData) error {
	if len(data.BlockCommit.StateChanges()) > 0 {
		start := time.Now()
		strg.computeMerkleUpdate(data)
		elapsed := time.Since(start)
		data.BlockCommit.SetElapsedMerkle(elapsed.Seconds())
		logger.I().Debugw("compute merkle update",
			"leaf nodes", len(data.merkleUpdate.Leaves), "elapsed", elapsed)
	}

	start := time.Now()
	if err := strg.writeCommitData(data); err != nil {
		return err
	}
	elapsed := time.Since(start)
	logger.I().Debugw("write commit data", "elapsed", elapsed)
	return nil
}

func (strg *Storage) writeCommitData(data *CommitData) error {
	if err := strg.writeChainData(data); err != nil {
		return err
	}
	if err := strg.writeBlockCommit(data); err != nil {
		return err
	}
	if err := strg.writeStateMerkleTree(data); err != nil {
		return err
	}
	return strg.setCommitedBlockHeight(data.Block.Height())
}

func (strg *Storage) computeMerkleUpdate(data *CommitData) {
	strg.stateStore.loadPrevValues(data.BlockCommit.StateChanges())
	strg.stateStore.loadPrevTreeIndexes(data.BlockCommit.StateChanges())
	prevLeafCount := strg.merkleStore.getLeafCount()
	leafCount := strg.stateStore.setNewTreeIndexes(data.BlockCommit.StateChanges(), prevLeafCount)
	nodes := strg.stateStore.computeUpdatedTreeNodes(data.BlockCommit.StateChanges())
	data.merkleUpdate = strg.merkleTree.Update(nodes, leafCount)

	data.BlockCommit.
		SetLeafCount(data.merkleUpdate.LeafCount.Bytes()).
		SetMerkleRoot(data.merkleUpdate.Root.Data)
}

func (strg *Storage) writeChainData(data *CommitData) error {
	updFns := make([]updateFunc, 0)
	updFns = append(updFns, strg.chainStore.setBlock(data.Block)...)
	updFns = append(updFns, strg.chainStore.setLastQC(data.QC))
	updFns = append(updFns, strg.chainStore.setTxs(data.Transactions)...)
	updFns = append(updFns, strg.chainStore.setTxCommits(data.TxCommits)...)
	return updateBadgerDB(strg.db, updFns)
}

func (strg *Storage) writeBlockCommit(data *CommitData) error {
	updFn := strg.chainStore.setBlockCommit(data.BlockCommit)
	return updateBadgerDB(strg.db, []updateFunc{updFn})
}

// commit state values and merkle tree in one transaction
func (strg *Storage) writeStateMerkleTree(data *CommitData) error {
	if len(data.BlockCommit.StateChanges()) == 0 {
		return nil
	}
	strg.mtxWriteState.Lock()
	defer strg.mtxWriteState.Unlock()

	updFns := strg.stateStore.commitStateChanges(data.BlockCommit.StateChanges())
	updFns = append(updFns, strg.merkleStore.commitUpdate(data.merkleUpdate)...)
	return updateBadgerDB(strg.db, updFns)
}

func (strg *Storage) setCommitedBlockHeight(height uint64) error {
	updFn := strg.chainStore.setBlockHeight(height)
	return updateBadgerDB(strg.db, []updateFunc{updFn})
}
