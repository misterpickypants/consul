package cachetype

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/consul/agent/cache"
	"github.com/hashicorp/consul/agent/structs"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestInternalServiceDump(t *testing.T) {
	rpc := TestRPC(t)
	typ := &InternalServiceDump{RPC: rpc}

	// Expect the proper RPC call. This also sets the expected value
	// since that is return-by-pointer in the arguments.
	var resp *structs.IndexedCheckServiceNodes
	rpc.On("RPC", "Internal.ServiceDump", mock.Anything, mock.Anything).Return(nil).
		Run(func(args mock.Arguments) {
			req := args.Get(1).(*structs.ServiceDumpRequest)
			require.Equal(t, uint64(24), req.QueryOptions.MinQueryIndex)
			require.Equal(t, 1*time.Second, req.QueryOptions.MaxQueryTime)
			require.True(t, req.AllowStale)

			reply := args.Get(2).(*structs.IndexedCheckServiceNodes)
			reply.Nodes = []structs.CheckServiceNode{
				{Service: &structs.NodeService{Kind: req.ServiceKind, Service: "foo"}},
			}
			reply.QueryMeta.Index = 48
			resp = reply
		})

	// Fetch
	resultA, err := typ.Fetch(cache.FetchOptions{
		MinIndex: 24,
		Timeout:  1 * time.Second,
	}, &structs.ServiceDumpRequest{
		Datacenter:     "dc1",
		ServiceKind:    structs.ServiceKindMeshGateway,
		UseServiceKind: true,
	})
	require.NoError(t, err)
	require.Equal(t, cache.FetchResult{
		Value: resp,
		Index: 48,
	}, resultA)

	rpc.AssertExpectations(t)
}

func TestInternalServiceDump_badReqType(t *testing.T) {
	rpc := TestRPC(t)
	typ := &CatalogServices{RPC: rpc}

	// Fetch
	_, err := typ.Fetch(cache.FetchOptions{}, cache.TestRequest(
		t, cache.RequestInfo{Key: "foo", MinIndex: 64}))
	require.Error(t, err)
	require.Contains(t, err.Error(), "wrong type")
	rpc.AssertExpectations(t)
}

func TestInternalServiceDump_IntegrationWithCache_NotModifiedResponse(t *testing.T) {
	rpc := &MockRPC{}
	typ := &InternalServiceDump{RPC: rpc}

	nodes := []structs.CheckServiceNode{
		{Service: &structs.NodeService{Service: "foo"}},
	}
	rpc.On("RPC", "Internal.ServiceDump", mock.Anything, mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			req := args.Get(1).(*structs.ServiceDumpRequest)
			require.True(t, req.AllowStale)
			require.True(t, req.AllowNotModifiedResponse)

			reply := args.Get(2).(*structs.IndexedCheckServiceNodes)
			reply.QueryMeta.Index = 44
			reply.NotModified = true
		})

	c := cache.New(cache.Options{})
	c.RegisterType(InternalServiceDumpName, typ)
	last := cache.FetchResult{
		Value: &structs.IndexedCheckServiceNodes{
			Nodes:     nodes,
			QueryMeta: structs.QueryMeta{Index: 42},
		},
		Index: 42,
	}
	req := &structs.ServiceDumpRequest{
		Datacenter: "dc1",
		QueryOptions: structs.QueryOptions{
			Token:         "token",
			MinQueryIndex: 44,
			MaxQueryTime:  time.Second,
		},
	}

	err := c.Prepopulate(InternalServiceDumpName, last, "dc1", "token", req.CacheInfo().Key)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	actual, _, err := c.Get(ctx, InternalServiceDumpName, req)
	require.NoError(t, err)

	expected := &structs.IndexedCheckServiceNodes{
		Nodes:     nodes,
		QueryMeta: structs.QueryMeta{Index: 42},
	}
	require.Equal(t, expected, actual)

	rpc.AssertExpectations(t)
}
