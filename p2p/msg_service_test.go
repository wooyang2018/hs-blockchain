// Copyright (C) 2021 Aung Maw
// Licensed under the GNU General Public License v3.0

package p2p

import (
	"testing"
	"time"

	"github.com/aungmawjj/juria-blockchain/core"
	p2p_pb "github.com/aungmawjj/juria-blockchain/p2p/pb"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func setupMsgServiceWithLoopBackPeers() (*MsgService, [][]byte, []*Peer) {
	peers := make([]*Peer, 2)
	peers[0] = NewPeer(core.GenerateKey(nil).PublicKey(), nil)
	peers[1] = NewPeer(core.GenerateKey(nil).PublicKey(), nil)

	s1 := peers[0].SubscribeMsg()
	s2 := peers[1].SubscribeMsg()

	raws := make([][]byte, 2)

	go func() {
		for e := range s1.Events() {
			raws[0] = e.([]byte)
		}
	}()

	go func() {
		for e := range s2.Events() {
			raws[1] = e.([]byte)
		}
	}()

	host := new(Host)
	host.peerStore = NewPeerStore()

	svc := NewMsgService(host)

	peers[0].OnConnected(newRWCLoopBack())
	peers[1].OnConnected(newRWCLoopBack())
	host.peerStore.Store(peers[0])
	host.peerStore.Store(peers[1])
	go host.onAddedPeer(peers[0])
	go host.onAddedPeer(peers[1])

	return svc, raws, peers
}

func TestMsgService_BroadcastProposal(t *testing.T) {
	assert := assert.New(t)

	svc, raws, _ := setupMsgServiceWithLoopBackPeers()
	sub := svc.SubscribeProposal(5)
	var recvBlk *core.Block
	var recvCount int
	go func() {
		for e := range sub.Events() {
			recvCount++
			recvBlk = e.(*core.Block)
		}
	}()

	blk := core.NewBlock().SetHeight(10)
	err := svc.BroadcastProposal(blk)

	if !assert.NoError(err) {
		return
	}

	time.Sleep(time.Millisecond)

	assert.NotNil(raws[0])
	assert.Equal(raws[0], raws[1])

	recvMsg := new(p2p_pb.Message)
	proto.Unmarshal(raws[0], recvMsg)
	assert.Equal(p2p_pb.Message_Proposal, recvMsg.Type)

	assert.Equal(2, recvCount)
	if assert.NotNil(recvBlk) {
		assert.Equal(blk.Height(), recvBlk.Height())
	}
}

func TestMsgService_SendVote(t *testing.T) {
	assert := assert.New(t)

	svc, raws, peers := setupMsgServiceWithLoopBackPeers()

	sub := svc.SubscribeVote(5)
	var recvVote *core.Vote
	go func() {
		for e := range sub.Events() {
			recvVote = e.(*core.Vote)
		}
	}()

	validator := core.GenerateKey(nil)
	vote := core.NewBlock().Sign(core.GenerateKey(nil)).Vote(validator)
	err := svc.SendVote(peers[0].PublicKey(), vote)

	if !assert.NoError(err) {
		return
	}

	time.Sleep(time.Millisecond)

	assert.NotNil(raws[0])
	assert.Nil(raws[1])

	recvMsg := new(p2p_pb.Message)
	proto.Unmarshal(raws[0], recvMsg)
	assert.Equal(p2p_pb.Message_Vote, recvMsg.Type)

	if assert.NotNil(recvVote) {
		assert.Equal(vote.BlockHash(), recvVote.BlockHash())
	}
}

func TestMsgService_SendNewView(t *testing.T) {
	assert := assert.New(t)

	svc, raws, peers := setupMsgServiceWithLoopBackPeers()

	sub := svc.SubscribeNewView(5)
	var recvQC *core.QuorumCert
	go func() {
		for e := range sub.Events() {
			recvQC = e.(*core.QuorumCert)
		}
	}()

	vote := core.NewBlock().Sign(core.GenerateKey(nil)).Vote(core.GenerateKey(nil))
	qc := core.NewQuorumCert().Build([]*core.Vote{vote})
	err := svc.SendNewView(peers[0].PublicKey(), qc)

	if !assert.NoError(err) {
		return
	}

	time.Sleep(time.Millisecond)

	assert.NotNil(raws[0])
	assert.Nil(raws[1])

	recvMsg := new(p2p_pb.Message)
	proto.Unmarshal(raws[0], recvMsg)
	assert.Equal(p2p_pb.Message_NewView, recvMsg.Type)

	if assert.NotNil(recvQC) {
		assert.Equal(qc.BlockHash(), recvQC.BlockHash())
	}
}

func TestMsgService_BroadcastTxList(t *testing.T) {
	assert := assert.New(t)

	svc, raws, _ := setupMsgServiceWithLoopBackPeers()
	sub := svc.SubscribeTxList(5)
	var recvTxList core.TxList
	var recvCount int
	go func() {
		for e := range sub.Events() {
			recvCount++
			recvTxList = e.(core.TxList)
		}
	}()

	txList := core.TxList{
		core.NewTransaction().SetNonce(1),
		core.NewTransaction().SetNonce(2),
	}
	err := svc.BroadcastTxList(txList)

	if !assert.NoError(err) {
		return
	}

	time.Sleep(time.Millisecond)

	assert.NotNil(raws[0])
	assert.Equal(raws[0], raws[1])

	recvMsg := new(p2p_pb.Message)
	proto.Unmarshal(raws[0], recvMsg)
	assert.Equal(p2p_pb.Message_TxList, recvMsg.Type)

	assert.Equal(2, recvCount)
	if assert.NotNil(recvTxList) {
		assert.Equal(txList[0].Nonce(), recvTxList[0].Nonce())
		assert.Equal(txList[1].Nonce(), recvTxList[1].Nonce())
	}
}
