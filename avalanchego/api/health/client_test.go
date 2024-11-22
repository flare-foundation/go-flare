// Copyright (C) 2019-2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package health

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanchego/utils/rpc"
)

type mockClient struct {
	reply  APIHealthReply
	err    error
	onCall func()
}

func (mc *mockClient) SendRequest(ctx context.Context, method string, params interface{}, replyIntf interface{}, options ...rpc.Option) error {
	reply := replyIntf.(*APIHealthReply)
	*reply = mc.reply
	mc.onCall()
	return mc.err
}

func TestNewClient(t *testing.T) {
	require := require.New(t)

	c := NewClient("")
	require.NotNil(c)
}

func TestClient(t *testing.T) {
	require := require.New(t)

	mc := &mockClient{
		reply: APIHealthReply{
			Healthy: true,
		},
		err:    nil,
		onCall: func() {},
	}
	c := client{
		requester: mc,
	}

	{
		readiness, err := c.Readiness(context.Background())
		require.NoError(err)
		require.True(readiness.Healthy)
	}

	{
		health, err := c.Health(context.Background())
		require.NoError(err)
		require.True(health.Healthy)
	}

	{
		liveness, err := c.Liveness(context.Background())
		require.NoError(err)
		require.True(liveness.Healthy)
	}

	{
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		healthy, err := c.AwaitHealthy(ctx, time.Second)
		cancel()
		require.NoError(err)
		require.True(healthy)
	}

	mc.reply.Healthy = false

	{
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Microsecond)
		healthy, err := c.AwaitHealthy(ctx, time.Microsecond)
		cancel()
		require.Error(err)
		require.False(healthy)
	}

	mc.onCall = func() {
		mc.reply.Healthy = true
	}

	{
		healthy, err := c.AwaitHealthy(context.Background(), time.Microsecond)
		require.NoError(err)
		require.True(healthy)
	}
}
