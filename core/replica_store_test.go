// Copyright (C) 2021 Aung Maw
// Licensed under the GNU General Public License v3.0

package core

import "github.com/stretchr/testify/mock"

type MockReplicaStore struct {
	mock.Mock
}

var _ ReplicaStore = (*MockReplicaStore)(nil)

func (m *MockReplicaStore) ReplicaCount() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockReplicaStore) IsReplica(pubKey *PublicKey) bool {
	args := m.Called(pubKey)
	return args.Bool(0)
}

func (m *MockReplicaStore) GetReplica(idx int) []byte {
	args := m.Called(idx)
	val := args.Get(0)
	if r, ok := val.([]byte); ok {
		return r
	}
	return nil
}

func (m *MockReplicaStore) GetReplicaIndex(pubKey *PublicKey) (int, bool) {
	args := m.Called(pubKey)
	return args.Int(0), args.Bool(1)
}
