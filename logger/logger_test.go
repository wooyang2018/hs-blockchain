// Copyright (C) 2021 Aung Maw
// Licensed under the GNU General Public License v3.0

package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInit(t *testing.T) {
	assert := assert.New(t)
	assert.Panics(func() { Instance() }, "should panic")
	Set(New())
	assert.NotNil(Instance(), "instance should not be nil")
}