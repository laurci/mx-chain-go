package resolverscontainer

import (
	"github.com/multiversx/mx-chain-core-go/data/typeConverters"
	"github.com/multiversx/mx-chain-core-go/marshal"
	"github.com/multiversx/mx-chain-go/common"
	"github.com/multiversx/mx-chain-go/config"
	"github.com/multiversx/mx-chain-go/dataRetriever"
	"github.com/multiversx/mx-chain-go/p2p"
	"github.com/multiversx/mx-chain-go/sharding"
)

// FactoryArgs will hold the arguments for ResolversContainerFactory for both shard and meta
type FactoryArgs struct {
	ResolverConfig              config.ResolverConfig
	NumConcurrentResolvingJobs  int32
	ShardCoordinator            sharding.Coordinator
	Messenger                   dataRetriever.TopicMessageHandler
	Store                       dataRetriever.StorageService
	Marshalizer                 marshal.Marshalizer
	DataPools                   dataRetriever.PoolsHolder
	Uint64ByteSliceConverter    typeConverters.Uint64ByteSliceConverter
	DataPacker                  dataRetriever.DataPacker
	TriesContainer              common.TriesHolder
	InputAntifloodHandler       dataRetriever.P2PAntifloodHandler
	OutputAntifloodHandler      dataRetriever.P2PAntifloodHandler
	CurrentNetworkEpochProvider dataRetriever.CurrentNetworkEpochProviderHandler
	PreferredPeersHolder        p2p.PreferredPeersHolderHandler
	PeersRatingHandler          dataRetriever.PeersRatingHandler
	SizeCheckDelta              uint32
	IsFullHistoryNode           bool
	PayloadValidator            dataRetriever.PeerAuthenticationPayloadValidator
}
