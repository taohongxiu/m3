	"errors"
	m3clusterkv "github.com/m3db/m3cluster/kv"
	m3clusterkvmem "github.com/m3db/m3cluster/kv/mem"
	"github.com/m3db/m3cluster/shard"
var (
	errNilRetention = errors.New("namespace retention options cannot be empty")
)

	runtimeOpts := m3dbruntime.NewOptions().
		SetPersistRateLimitOptions(ratelimit.NewOptions().
			SetLimitEnabled(true).
			SetLimitMbps(cfg.Filesystem.ThroughputLimitMbps).
			SetLimitCheckEvery(cfg.Filesystem.ThroughputCheckEvery)).
		SetWriteNewSeriesAsync(cfg.WriteNewSeriesAsync).
		SetWriteNewSeriesBackoffDuration(cfg.WriteNewSeriesBackoffDuration)

	if tick := cfg.Tick; tick != nil {
		runtimeOpts = runtimeOpts.
			SetTickSeriesBatchSize(tick.SeriesBatchSize).
			SetTickPerSeriesSleepDuration(tick.PerSeriesSleepDuration).
			SetTickMinimumInterval(tick.MinimumInterval)
	}

	runtimeOptsMgr := m3dbruntime.NewOptionsManager()
	if err := runtimeOptsMgr.Update(runtimeOpts); err != nil {
		logger.Fatalf("could not set initial runtime options: %v", err)
	}
	defer runtimeOptsMgr.Close()

	opts = opts.SetRuntimeOptionsManager(runtimeOptsMgr)

		SetMmapHugePagesThreshold(mmap.HugePages.Threshold).
		SetRuntimeOptionsManager(runtimeOptsMgr)
	var (
		topoInit topology.Initializer
		kv       m3clusterkv.Store
	)
	switch {
	case cfg.ConfigService != nil:
		logger.Info("creating dynamic config service client with m3cluster")
		configSvcClientOpts := cfg.ConfigService.NewOptions().
			SetInstrumentOptions(
				instrument.NewOptions().
					SetLogger(logger).
					SetMetricsScope(scope))
		configSvcClient, err := etcdclient.NewConfigServiceClient(configSvcClientOpts)
		if err != nil {
			logger.Fatalf("could not create m3cluster client: %v", err)
		}
		dynamicOpts := namespace.NewDynamicOptions().
			SetInstrumentOptions(iopts).
			SetConfigServiceClient(configSvcClient).
			SetNamespaceRegistryKey(kvconfig.NamespacesKey)
		nsInit := namespace.NewDynamicInitializer(dynamicOpts)
		opts = opts.SetNamespaceInitializer(nsInit)

		serviceID := services.NewServiceID().
			SetName(cfg.ConfigService.Service).
			SetEnvironment(cfg.ConfigService.Env).
			SetZone(cfg.ConfigService.Zone)

		topoOpts := topology.NewDynamicOptions().
			SetConfigServiceClient(configSvcClient).
			SetServiceID(serviceID).
			SetQueryOptions(services.NewQueryOptions().SetIncludeUnhealthy(true)).
			SetInstrumentOptions(opts.InstrumentOptions()).
			SetHashGen(sharding.NewHashGenWithSeed(cfg.HashingConfiguration.Seed))

		topoInit = topology.NewDynamicInitializer(topoOpts)

		kv, err = configSvcClient.KV()
		if err != nil {
			logger.Fatalf("could not create KV client, %v", err)
		}

	case cfg.StaticConfig != nil && cfg.StaticConfig.TopologyConfig != nil && cfg.StaticConfig.Namespaces != nil:
		logger.Info("creating static config service client with m3cluster")

		shardSet, hostShardSets, err := newStaticShardSet(cfg.StaticConfig.TopologyConfig.Shards, cfg.ListenAddress)
		if err != nil {
			logger.Fatalf("unable to create shard set for static config: %v", err)
		}
		staticOptions := topology.NewStaticOptions().
			SetReplicas(1).
			SetHostShardSets(hostShardSets).
			SetShardSet(shardSet)

		nsList := []namespace.Metadata{}
		for _, ns := range cfg.StaticConfig.Namespaces {
			md, err := newNamespaceMetadata(ns)
			if err != nil {
				logger.Fatalf("unable to create metadata for static config: %v", err)
			}
			nsList = append(nsList, md)
		}
		nsInitStatic := namespace.NewStaticInitializer(nsList)
		topoInit = topology.NewStaticInitializer(staticOptions)
		opts = opts.SetNamespaceInitializer(nsInitStatic)

		kv = m3clusterkvmem.NewStore()

	default:
		logger.Fatal("config service or static configuration required")
	}
	err = setNewSeriesLimitPerShardOnChange(topo, runtimeOptsMgr, initClusterLimit)
	if err != nil {
		logger.Warnf("unable to set cluster new series insert limit: %v", err)
	}
			err = setNewSeriesLimitPerShardOnChange(topo, runtimeOptsMgr, value)
			if err != nil {
				logger.Warnf("unable to set cluster new series insert limit: %v", err)
				continue
			}
) error {
		return nil
	newRuntimeOpts := runtimeOpts.
		SetWriteNewSeriesLimitPerShardPerSecond(perPlacedShardLimit)
	return runtimeOptsMgr.Update(newRuntimeOpts)
			"refillLowWatermark=%f, refillHighWatermark=%f",

func newStaticShardSet(numShards int, listenAddress string) (sharding.ShardSet, []topology.HostShardSet, error) {
	var (
		shardSet      sharding.ShardSet
		hostShardSets []topology.HostShardSet
		shardIDs      []uint32
		err           error
	)

	for i := uint32(0); i < uint32(numShards); i++ {
		shardIDs = append(shardIDs, i)
	}

	shards := sharding.NewShards(shardIDs, shard.Available)
	shardSet, err = sharding.NewShardSet(shards, sharding.DefaultHashFn(1))
	if err != nil {
		return nil, nil, err
	}

	host := topology.NewHost("localhost", listenAddress)
	hostShardSet := topology.NewHostShardSet(host, shardSet)
	hostShardSets = append(hostShardSets, hostShardSet)

	return shardSet, hostShardSets, nil
}

func newNamespaceMetadata(cfg config.StaticNamespaceConfiguration) (namespace.Metadata, error) {
	if cfg.Retention == nil {
		return nil, errNilRetention
	}
	if cfg.Options == nil {
		cfg.Options = &config.StaticNamespaceOptions{
			NeedsBootstrap:      true,
			NeedsFilesetCleanup: true,
			NeedsFlush:          true,
			NeedsRepair:         true,
			WritesToCommitLog:   true,
		}
	}
	md, err := namespace.NewMetadata(
		ts.StringID(cfg.Name),
		namespace.NewOptions().
			SetNeedsBootstrap(cfg.Options.NeedsBootstrap).
			SetNeedsFilesetCleanup(cfg.Options.NeedsFilesetCleanup).
			SetNeedsFlush(cfg.Options.NeedsFlush).
			SetNeedsRepair(cfg.Options.NeedsRepair).
			SetWritesToCommitLog(cfg.Options.WritesToCommitLog).
			SetRetentionOptions(
				retention.NewOptions().
					SetBlockSize(cfg.Retention.BlockSize).
					SetRetentionPeriod(cfg.Retention.RetentionPeriod).
					SetBufferFuture(cfg.Retention.BufferFuture).
					SetBufferPast(cfg.Retention.BufferPast).
					SetBlockDataExpiry(cfg.Retention.BlockDataExpiry).
					SetBlockDataExpiryAfterNotAccessedPeriod(cfg.Retention.BlockDataExpiryAfterNotAccessPeriod)))
	if err != nil {
		return nil, err
	}

	return md, nil
}