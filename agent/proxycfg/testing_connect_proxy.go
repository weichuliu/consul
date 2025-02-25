package proxycfg

import (
	"github.com/mitchellh/go-testing-interface"
	"github.com/stretchr/testify/assert"

	"github.com/hashicorp/consul/agent/cache"
	"github.com/hashicorp/consul/agent/connect"
	"github.com/hashicorp/consul/agent/consul/discoverychain"
	"github.com/hashicorp/consul/agent/structs"
)

// TestConfigSnapshot returns a fully populated snapshot
func TestConfigSnapshot(t testing.T, nsFn func(ns *structs.NodeService), extraUpdates []cache.UpdateEvent) *ConfigSnapshot {
	roots, leaf := TestCerts(t)

	// no entries implies we'll get a default chain
	dbChain := discoverychain.TestCompileConfigEntries(t, "db", "default", "default", "dc1", connect.TestClusterID+".consul", nil)
	assert.True(t, dbChain.IsDefault())

	var (
		upstreams   = structs.TestUpstreams(t)
		dbUpstream  = upstreams[0]
		geoUpstream = upstreams[1]

		dbUID  = NewUpstreamID(&dbUpstream)
		geoUID = NewUpstreamID(&geoUpstream)

		webSN = structs.ServiceIDString("web", nil)
	)

	baseEvents := []cache.UpdateEvent{
		{
			CorrelationID: rootsWatchID,
			Result:        roots,
		},
		{
			CorrelationID: leafWatchID,
			Result:        leaf,
		},
		{
			CorrelationID: intentionsWatchID,
			Result: &structs.IndexedIntentionMatches{
				Matches: []structs.Intentions{
					nil, // no intentions defined
				},
			},
		},
		{
			CorrelationID: svcChecksWatchIDPrefix + webSN,
			Result:        []structs.CheckType{},
		},
		{
			CorrelationID: "upstream:" + geoUID.String(),
			Result: &structs.PreparedQueryExecuteResponse{
				Nodes: TestPreparedQueryNodes(t, "geo-cache"),
			},
		},
		{
			CorrelationID: "discovery-chain:" + dbUID.String(),
			Result: &structs.DiscoveryChainResponse{
				Chain: dbChain,
			},
		},
		{
			CorrelationID: "upstream-target:" + dbChain.ID() + ":" + dbUID.String(),
			Result: &structs.IndexedCheckServiceNodes{
				Nodes: TestUpstreamNodes(t, "db"),
			},
		},
	}

	return testConfigSnapshotFixture(t, &structs.NodeService{
		Kind:    structs.ServiceKindConnectProxy,
		Service: "web-sidecar-proxy",
		Port:    9999,
		Proxy: structs.ConnectProxyConfig{
			DestinationServiceID:   "web",
			DestinationServiceName: "web",
			LocalServiceAddress:    "127.0.0.1",
			LocalServicePort:       8080,
			Config: map[string]interface{}{
				"foo": "bar",
			},
			Upstreams: upstreams,
		},
		Meta:            nil,
		TaggedAddresses: nil,
	}, nsFn, nil, testSpliceEvents(baseEvents, extraUpdates))
}

// TestConfigSnapshotDiscoveryChain returns a fully populated snapshot using a discovery chain
func TestConfigSnapshotDiscoveryChain(
	t testing.T,
	variation string,
	nsFn func(ns *structs.NodeService),
	extraUpdates []cache.UpdateEvent,
	additionalEntries ...structs.ConfigEntry,
) *ConfigSnapshot {
	roots, leaf := TestCerts(t)

	var (
		upstreams   = structs.TestUpstreams(t)
		geoUpstream = upstreams[1]

		geoUID = NewUpstreamID(&geoUpstream)

		webSN = structs.ServiceIDString("web", nil)
	)

	baseEvents := testSpliceEvents([]cache.UpdateEvent{
		{
			CorrelationID: rootsWatchID,
			Result:        roots,
		},
		{
			CorrelationID: leafWatchID,
			Result:        leaf,
		},
		{
			CorrelationID: intentionsWatchID,
			Result: &structs.IndexedIntentionMatches{
				Matches: []structs.Intentions{
					nil, // no intentions defined
				},
			},
		},
		{
			CorrelationID: svcChecksWatchIDPrefix + webSN,
			Result:        []structs.CheckType{},
		},
		{
			CorrelationID: "upstream:" + geoUID.String(),
			Result: &structs.PreparedQueryExecuteResponse{
				Nodes: TestPreparedQueryNodes(t, "geo-cache"),
			},
		},
	}, setupTestVariationConfigEntriesAndSnapshot(
		t, variation, upstreams, additionalEntries...,
	))

	return testConfigSnapshotFixture(t, &structs.NodeService{
		Kind:    structs.ServiceKindConnectProxy,
		Service: "web-sidecar-proxy",
		Port:    9999,
		Proxy: structs.ConnectProxyConfig{
			DestinationServiceID:   "web",
			DestinationServiceName: "web",
			LocalServiceAddress:    "127.0.0.1",
			LocalServicePort:       8080,
			Config: map[string]interface{}{
				"foo": "bar",
			},
			Upstreams: upstreams,
		},
		Meta:            nil,
		TaggedAddresses: nil,
	}, nsFn, nil, testSpliceEvents(baseEvents, extraUpdates))
}

func TestConfigSnapshotExposeConfig(t testing.T, nsFn func(ns *structs.NodeService)) *ConfigSnapshot {
	roots, leaf := TestCerts(t)

	var (
		webSN = structs.ServiceIDString("web", nil)
	)

	baseEvents := []cache.UpdateEvent{
		{
			CorrelationID: rootsWatchID,
			Result:        roots,
		},
		{
			CorrelationID: leafWatchID, Result: leaf,
		},
		{
			CorrelationID: intentionsWatchID,
			Result: &structs.IndexedIntentionMatches{
				Matches: []structs.Intentions{
					nil, // no intentions defined
				},
			},
		},
		{
			CorrelationID: svcChecksWatchIDPrefix + webSN,
			Result:        []structs.CheckType{},
		},
	}

	return testConfigSnapshotFixture(t, &structs.NodeService{
		Kind:    structs.ServiceKindConnectProxy,
		Service: "web-sidecar-proxy",
		Address: "1.2.3.4",
		Port:    8080,
		Proxy: structs.ConnectProxyConfig{
			DestinationServiceID:   "web",
			DestinationServiceName: "web",
			LocalServicePort:       8080,
			Expose: structs.ExposeConfig{
				Checks: false,
				Paths: []structs.ExposePath{
					{
						LocalPathPort: 8080,
						Path:          "/health1",
						ListenerPort:  21500,
					},
					{
						LocalPathPort: 8080,
						Path:          "/health2",
						ListenerPort:  21501,
					},
				},
			},
		},
		Meta:            nil,
		TaggedAddresses: nil,
	}, nsFn, nil, baseEvents)
}

func TestConfigSnapshotGRPCExposeHTTP1(t testing.T) *ConfigSnapshot {
	roots, leaf := TestCerts(t)

	return testConfigSnapshotFixture(t, &structs.NodeService{
		Kind:    structs.ServiceKindConnectProxy,
		Service: "grpc-proxy",
		Address: "1.2.3.4",
		Port:    8080,
		Proxy: structs.ConnectProxyConfig{
			DestinationServiceName: "grpc",
			DestinationServiceID:   "grpc",
			LocalServicePort:       8080,
			Config: map[string]interface{}{
				"protocol": "grpc",
			},
			Expose: structs.ExposeConfig{
				Checks: false,
				Paths: []structs.ExposePath{
					{
						LocalPathPort: 8090,
						Path:          "/healthz",
						ListenerPort:  21500,
						Protocol:      "http",
					},
				},
			},
		},
		Meta:            nil,
		TaggedAddresses: nil,
	}, nil, nil, []cache.UpdateEvent{
		{
			CorrelationID: rootsWatchID,
			Result:        roots,
		},
		{
			CorrelationID: leafWatchID,
			Result:        leaf,
		},
		{
			CorrelationID: intentionsWatchID,
			Result: &structs.IndexedIntentionMatches{
				Matches: []structs.Intentions{
					nil, // no intentions defined
				},
			},
		},
		{
			CorrelationID: svcChecksWatchIDPrefix + structs.ServiceIDString("grpc", nil),
			Result:        []structs.CheckType{},
		},
	})
}
