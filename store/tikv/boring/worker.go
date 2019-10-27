package boring

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/juju/errors"
	"github.com/pingcap/log"
	pd "github.com/pingcap/pd/client"
	"github.com/pingcap/tidb/privilege"
	"github.com/pingcap/tidb/session"
	"github.com/pingcap/tidb/store/tikv"
	"github.com/pingcap/tidb/util/logutil"
	"github.com/plainboring/config_client/pkg/cfgclient"
	pdcfg "github.com/plainboring/config_client/pkg/pd"
	kvcfg "github.com/plainboring/config_client/pkg/tikv"
	"go.uber.org/zap"
)

var defaultWorker *configWorker
var pdAddress string

// ConfigWorker ..
type configWorker struct {
	pdClient  pd.Client
	cfgClient cfgclient.ConfigClient
	session   session.Session
}

// NewConfigWorker returns a new config worker
func NewConfigWorker(storage tikv.Storage, pdClient pd.Client) (tikv.ConfigHandler, error) {
	cfgClient, err := cfgclient.NewConfigClient(context.Background(), pdAddress)
	if err != nil {
		return nil, errors.Trace(err)
	}

	log.Info("[config worker] start")

	defaultWorker = &configWorker{
		pdClient:  pdClient,
		cfgClient: cfgClient,
		session:   createSession(storage),
	}
	return defaultWorker, nil
}

// SetPDAddr sets the pd address
func SetPDAddr(addr string) {
	pdAddress = addr
}

// GetDefaultWorker returns the default worker
func GetDefaultWorker() tikv.ConfigHandler {
	return defaultWorker
}

func createSession(store tikv.Storage) session.Session {
	for {
		se, err := session.CreateSession(store)
		if err != nil {
			logutil.BgLogger().Warn("[config worker] create session", zap.Error(err))
			continue
		}
		// Disable privilege check for gc worker session.
		privilege.BindPrivilegeManager(se, nil)
		se.GetSessionVars().InRestrictedSQL = true
		return se
	}
}

// Start starts a background config worker.
func (w *configWorker) Start() {
	log.Info("start config worker")
	c := context.Background()
	ctx, cancel := context.WithTimeout(c, time.Second*5)
	defer cancel()

	stores, err := w.pdClient.GetAllStores(ctx)
	if err != nil {
		log.Error("get store failed", zap.Error(err))
	}
	for _, s := range stores {
		storeID := s.GetId()
		err = w.initTiKVConfig(storeID)
		if err != nil {
			log.Error("init tikv failed", zap.Error(err),
				zap.Uint64("storeID", storeID))
		}
	}
	err = w.initPDConfig()
	if err != nil {
		log.Error("init pd failed", zap.Error(err))
	}
}

// UpdateTiKV updates tikv config
func (w *configWorker) UpdateTiKV(
	storeID uint64, subs []string, name, value string) error {
	log.Info("config client",
		zap.Uint64("storeID", storeID), zap.Strings("sub", subs),
		zap.String("name", name), zap.String("value", value),
		zap.String("comp", "tikv"),
	)
	err := w.cfgClient.Update("tikv", subs, name, value, storeID)
	return errors.Trace(err)
}

// UpdatePD updates pd config
func (w *configWorker) UpdatePD(
	subs []string, name, value string) error {
	log.Info("config client",
		zap.Strings("sub", subs),
		zap.String("name", name), zap.String("value", value),
		zap.String("comp", "pd"),
	)
	err := w.cfgClient.Update("pd", subs, name, value, 0)
	return errors.Trace(err)
}

func (w *configWorker) initPDConfig() error {
	cfg, err := w.cfgClient.Get("pd", 0)
	if err != nil {
		return errors.Trace(err)
	}
	cfgReader := strings.NewReader(cfg)
	pdCfg := new(pdcfg.Config)
	_, err = toml.DecodeReader(cfgReader, pdCfg)
	if err != nil {
		log.Error("decode pd failed", zap.Error(err))
		return errors.Trace(err)
	}

	serverCfg := pdCfg.PDServerCfg
	err = w.updatePDConfig([]string{"pd-server"}, "use-region-storage", fmt.Sprintf("%t", serverCfg.UseRegionStorage))
	if err != nil {
		return errors.Trace(err)
	}

	replicaCfg := pdCfg.Replication
	err = w.updatePDConfig([]string{"replication"}, "max-replicas", fmt.Sprintf("%d", replicaCfg.MaxReplicas))
	if err != nil {
		return errors.Trace(err)
	}
	err = w.updatePDConfig([]string{"replication"}, "location-labels", fmt.Sprintf("%s", replicaCfg.LocationLabels))
	if err != nil {
		return errors.Trace(err)
	}
	err = w.updatePDConfig([]string{"replication"}, "strictly-match-label", fmt.Sprintf("%t", replicaCfg.StrictlyMatchLabel))
	if err != nil {
		return errors.Trace(err)
	}

	scheduleCfg := pdCfg.Schedule
	err = w.updatePDConfig([]string{"schedule"}, "max-snapshot-count", fmt.Sprintf("%d", scheduleCfg.MaxSnapshotCount))
	if err != nil {
		return errors.Trace(err)
	}
	err = w.updatePDConfig([]string{"schedule"}, "max-pending-peer-count", fmt.Sprintf("%d", scheduleCfg.MaxPendingPeerCount))
	if err != nil {
		return errors.Trace(err)
	}
	err = w.updatePDConfig([]string{"schedule"}, "max-merge-region-size", fmt.Sprintf("%d", scheduleCfg.MaxMergeRegionSize))
	if err != nil {
		return errors.Trace(err)
	}
	err = w.updatePDConfig([]string{"schedule"}, "max-merge-region-keys", fmt.Sprintf("%d", scheduleCfg.MaxMergeRegionKeys))
	if err != nil {
		return errors.Trace(err)
	}
	// err = w.updatePDConfig([]string{"schedule"}, "split-merge-interval", fmt.Sprintf("%s", scheduleCfg.SplitMergeInterval))
	// if err != nil {
	// 	return errors.Trace(err)
	// }
	err = w.updatePDConfig([]string{"schedule"}, "enable-one-way-merge", fmt.Sprintf("%t", scheduleCfg.EnableOneWayMerge))
	if err != nil {
		return errors.Trace(err)
	}
	// err = w.updatePDConfig([]string{"schedule"}, "patrol-region-interval", fmt.Sprintf("%s", scheduleCfg.PatrolRegionInterval))
	// if err != nil {
	// 	return errors.Trace(err)
	// }
	// err = w.updatePDConfig([]string{"schedule"}, "max-store-down-time", fmt.Sprintf("%s", scheduleCfg.MaxStoreDownTime))
	// if err != nil {
	// 	return errors.Trace(err)
	// }
	err = w.updatePDConfig([]string{"schedule"}, "leader-schedule-limit", fmt.Sprintf("%d", scheduleCfg.LeaderScheduleLimit))
	if err != nil {
		return errors.Trace(err)
	}
	err = w.updatePDConfig([]string{"schedule"}, "leader-schedule-strategy", fmt.Sprintf("%s", scheduleCfg.LeaderScheduleStrategy))
	if err != nil {
		return errors.Trace(err)
	}
	err = w.updatePDConfig([]string{"schedule"}, "region-schedule-limit", fmt.Sprintf("%d", scheduleCfg.RegionScheduleLimit))
	if err != nil {
		return errors.Trace(err)
	}
	err = w.updatePDConfig([]string{"schedule"}, "replica-schedule-limit", fmt.Sprintf("%d", scheduleCfg.ReplicaScheduleLimit))
	if err != nil {
		return errors.Trace(err)
	}
	err = w.updatePDConfig([]string{"schedule"}, "merge-schedule-limit", fmt.Sprintf("%d", scheduleCfg.MergeScheduleLimit))
	if err != nil {
		return errors.Trace(err)
	}
	err = w.updatePDConfig([]string{"schedule"}, "hot-region-schedule-limit", fmt.Sprintf("%d", scheduleCfg.HotRegionScheduleLimit))
	if err != nil {
		return errors.Trace(err)
	}
	err = w.updatePDConfig([]string{"schedule"}, "hot-region-cache-hits-threshold", fmt.Sprintf("%d", scheduleCfg.HotRegionCacheHitsThreshold))
	if err != nil {
		return errors.Trace(err)
	}
	err = w.updatePDConfig([]string{"schedule"}, "store-balance-rate", fmt.Sprintf("%f", scheduleCfg.StoreBalanceRate))
	if err != nil {
		return errors.Trace(err)
	}
	err = w.updatePDConfig([]string{"schedule"}, "tolerant-size-ratio", fmt.Sprintf("%f", scheduleCfg.TolerantSizeRatio))
	if err != nil {
		return errors.Trace(err)
	}
	err = w.updatePDConfig([]string{"schedule"}, "low-space-ratio", fmt.Sprintf("%f", scheduleCfg.LowSpaceRatio))
	if err != nil {
		return errors.Trace(err)
	}
	err = w.updatePDConfig([]string{"schedule"}, "high-space-ratio", fmt.Sprintf("%f", scheduleCfg.HighSpaceRatio))
	if err != nil {
		return errors.Trace(err)
	}
	err = w.updatePDConfig([]string{"schedule"}, "scheduler-max-waiting-operator", fmt.Sprintf("%d", scheduleCfg.SchedulerMaxWaitingOperator))
	if err != nil {
		return errors.Trace(err)
	}
	err = w.updatePDConfig([]string{"schedule"}, "disable-raft-learner", fmt.Sprintf("%t", scheduleCfg.DisableLearner))
	if err != nil {
		return errors.Trace(err)
	}
	err = w.updatePDConfig([]string{"schedule"}, "disable-remove-down-replica", fmt.Sprintf("%t", scheduleCfg.DisableRemoveDownReplica))
	if err != nil {
		return errors.Trace(err)
	}
	err = w.updatePDConfig([]string{"schedule"}, "disable-replace-offline-replica", fmt.Sprintf("%t", scheduleCfg.DisableReplaceOfflineReplica))
	if err != nil {
		return errors.Trace(err)
	}
	err = w.updatePDConfig([]string{"schedule"}, "disable-make-up-replica", fmt.Sprintf("%t", scheduleCfg.DisableMakeUpReplica))
	if err != nil {
		return errors.Trace(err)
	}
	err = w.updatePDConfig([]string{"schedule"}, "disable-remove-extra-replica", fmt.Sprintf("%t", scheduleCfg.DisableRemoveExtraReplica))
	if err != nil {
		return errors.Trace(err)
	}
	err = w.updatePDConfig([]string{"schedule"}, "disable-location-replacement", fmt.Sprintf("%t", scheduleCfg.DisableLocationReplacement))
	if err != nil {
		return errors.Trace(err)
	}
	err = w.updatePDConfig([]string{"schedule"}, "disable-namespace-relocation", fmt.Sprintf("%t", scheduleCfg.DisableNamespaceRelocation))
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (w *configWorker) updatePDConfig(
	comp []string, name, value string) error {

	comps := strings.Join(comp, ",")
	stmt := fmt.Sprintf(`
INSERT HIGH_PRIORITY INTO mysql.pd VALUES ('%[1]s', '%[2]s', '%[3]s')
ON DUPLICATE KEY UPDATE variable_value = '%[3]s'`,
		comps, name, value)

	if w.session == nil {
		return errors.New("[saveValueToSysTable session is nil]")
	}

	_, err := w.session.Execute(context.Background(), stmt)
	logutil.BgLogger().Debug("[config worker] save config",
		zap.String("name", name),
		zap.String("value", value),
		zap.String("comp", "PD"),
		zap.Error(err))

	return errors.Trace(err)
}

func (w *configWorker) initTiKVConfig(storeID uint64) error {
	tikvcfg, err := w.cfgClient.Get("tikv", storeID)
	if err != nil {
		return errors.Trace(err)
	}
	cfgReader := strings.NewReader(tikvcfg)
	tikvCfg := new(kvcfg.Config)
	_, err = toml.DecodeReader(cfgReader, tikvCfg)
	if err != nil {
		return err
	}

	rocksdbCfg := tikvCfg.Rocksdb
	w.updateTiKVConfig(storeID, []string{"rocksdb"}, "wal-recovery-mode", fmt.Sprintf("%d", rocksdbCfg.WalRecoveryMode))
	w.updateTiKVConfig(storeID, []string{"rocksdb"}, "wal-dir", fmt.Sprintf("%s", rocksdbCfg.WalDir))
	w.updateTiKVConfig(storeID, []string{"rocksdb"}, "wal-ttl-seconds", fmt.Sprintf("%d", rocksdbCfg.WalTTLSeconds))
	w.updateTiKVConfig(storeID, []string{"rocksdb"}, "wal-size-limit", fmt.Sprintf("%s", rocksdbCfg.WalSizeLimit))
	w.updateTiKVConfig(storeID, []string{"rocksdb"}, "max-total-wal-size", fmt.Sprintf("%s", rocksdbCfg.MaxTotalWalSize))
	w.updateTiKVConfig(storeID, []string{"rocksdb"}, "max-background-jobs", fmt.Sprintf("%d", rocksdbCfg.MaxBackgroundJobs))
	w.updateTiKVConfig(storeID, []string{"rocksdb"}, "max-manifest-file-size", fmt.Sprintf("%s", rocksdbCfg.MaxManifestFileSize))
	w.updateTiKVConfig(storeID, []string{"rocksdb"}, "create-if-missing", fmt.Sprintf("%t", rocksdbCfg.CreateIfMissing))
	w.updateTiKVConfig(storeID, []string{"rocksdb"}, "max-open-files", fmt.Sprintf("%d", rocksdbCfg.MaxOpenFiles))
	w.updateTiKVConfig(storeID, []string{"rocksdb"}, "enable-statistics", fmt.Sprintf("%t", rocksdbCfg.EnableStatistics))
	w.updateTiKVConfig(storeID, []string{"rocksdb"}, "stats-dump-period", fmt.Sprintf("%s", rocksdbCfg.StatsDumpPeriod))
	w.updateTiKVConfig(storeID, []string{"rocksdb"}, "compaction-readahead-size", fmt.Sprintf("%s", rocksdbCfg.CompactionReadaheadSize))
	w.updateTiKVConfig(storeID, []string{"rocksdb"}, "info-log-max-size", fmt.Sprintf("%s", rocksdbCfg.InfoLogMaxSize))
	w.updateTiKVConfig(storeID, []string{"rocksdb"}, "info-log-roll-time", fmt.Sprintf("%s", rocksdbCfg.InfoLogRollTime))
	w.updateTiKVConfig(storeID, []string{"rocksdb"}, "info-log-keep-log-file-num", fmt.Sprintf("%d", rocksdbCfg.InfoLogKeepLogFileNum))
	w.updateTiKVConfig(storeID, []string{"rocksdb"}, "info-log-dir", fmt.Sprintf("%s", rocksdbCfg.InfoLogDir))
	w.updateTiKVConfig(storeID, []string{"rocksdb"}, "rate-bytes-per-sec", fmt.Sprintf("%s", rocksdbCfg.RateBytesPerSec))
	w.updateTiKVConfig(storeID, []string{"rocksdb"}, "rate-limiter-mode", fmt.Sprintf("%d", rocksdbCfg.RateLimiterMode))
	w.updateTiKVConfig(storeID, []string{"rocksdb"}, "auto-tuned", fmt.Sprintf("%t", rocksdbCfg.AutoTuned))
	w.updateTiKVConfig(storeID, []string{"rocksdb"}, "bytes-per-sync", fmt.Sprintf("%s", rocksdbCfg.BytesPerSync))
	w.updateTiKVConfig(storeID, []string{"rocksdb"}, "wal-bytes-per-sync", fmt.Sprintf("%s", rocksdbCfg.WalBytesPerSync))
	w.updateTiKVConfig(storeID, []string{"rocksdb"}, "max-sub-compactions", fmt.Sprintf("%d", rocksdbCfg.MaxSubCompactions))
	w.updateTiKVConfig(storeID, []string{"rocksdb"}, "writable-file-max-buffer-size", fmt.Sprintf("%s", rocksdbCfg.WritableFileMaxBufferSize))
	w.updateTiKVConfig(storeID, []string{"rocksdb"}, "use-direct-io-for-flush-and-compaction", fmt.Sprintf("%t", rocksdbCfg.UseDirectIoForFlushAndCompaction))
	w.updateTiKVConfig(storeID, []string{"rocksdb"}, "enable-pipelined-write", fmt.Sprintf("%t", rocksdbCfg.EnablePipelinedWrite))

	cfCfgs := []kvcfg.CfConfig{rocksdbCfg.Defaultcf, rocksdbCfg.Writecf, rocksdbCfg.Lockcf}
	cfNames := []string{"defaultcf", "writecf", "lockcf"}
	for i, cfg := range cfCfgs {
		cf := cfNames[i]
		w.updateTiKVConfig(storeID, []string{"rocksdb", cf}, "block-size", fmt.Sprintf("%s", cfg.BlockSize))
		w.updateTiKVConfig(storeID, []string{"rocksdb", cf}, "block-cache-size", fmt.Sprintf("%s", cfg.BlockCacheSize))
		w.updateTiKVConfig(storeID, []string{"rocksdb", cf}, "disable-block-cache", fmt.Sprintf("%t", cfg.DisableBlockCache))
		w.updateTiKVConfig(storeID, []string{"rocksdb", cf}, "cache-index-and-filter-blocks", fmt.Sprintf("%t", cfg.CacheIndexAndFilterBlocks))
		w.updateTiKVConfig(storeID, []string{"rocksdb", cf}, "pin-l0-filter-and-index-blocks", fmt.Sprintf("%t", cfg.PinL0FilterAndIndexBlocks))
		w.updateTiKVConfig(storeID, []string{"rocksdb", cf}, "use-bloom-filter", fmt.Sprintf("%t", cfg.UseBloomFilter))
		w.updateTiKVConfig(storeID, []string{"rocksdb", cf}, "optimize-filters-for-hits", fmt.Sprintf("%t", cfg.OptimizeFiltersForHits))
		w.updateTiKVConfig(storeID, []string{"rocksdb", cf}, "whole-key-filtering", fmt.Sprintf("%t", cfg.WholeKeyFiltering))
		w.updateTiKVConfig(storeID, []string{"rocksdb", cf}, "bloom-filter-bits-per-key", fmt.Sprintf("%d", cfg.BloomFilterBitsPerKey))
		w.updateTiKVConfig(storeID, []string{"rocksdb", cf}, "block-based-bloom-filter", fmt.Sprintf("%t", cfg.BlockBasedBloomFilter))
		w.updateTiKVConfig(storeID, []string{"rocksdb", cf}, "read-amp-bytes-per-bit", fmt.Sprintf("%d", cfg.ReadAmpBytesPerBit))
		w.updateTiKVConfig(storeID, []string{"rocksdb", cf}, "compression-per-level", fmt.Sprintf("%s", cfg.CompressionPerLevel))
		w.updateTiKVConfig(storeID, []string{"rocksdb", cf}, "write-buffer-size", fmt.Sprintf("%s", cfg.WriteBufferSize))
		w.updateTiKVConfig(storeID, []string{"rocksdb", cf}, "max-write-buffer-number", fmt.Sprintf("%d", cfg.MaxWriteBufferNumber))
		w.updateTiKVConfig(storeID, []string{"rocksdb", cf}, "min-write-buffer-number-to-merge", fmt.Sprintf("%d", cfg.MinWriteBufferNumberToMerge))
		w.updateTiKVConfig(storeID, []string{"rocksdb", cf}, "max-bytes-for-level-base", fmt.Sprintf("%s", cfg.MaxBytesForLevelBase))
		w.updateTiKVConfig(storeID, []string{"rocksdb", cf}, "target-file-size-base", fmt.Sprintf("%s", cfg.TargetFileSizeBase))
		w.updateTiKVConfig(storeID, []string{"rocksdb", cf}, "level0-file-num-compaction-trigger", fmt.Sprintf("%d", cfg.Level0FileNumCompactionTrigger))
		w.updateTiKVConfig(storeID, []string{"rocksdb", cf}, "level0-slowdown-writes-trigger", fmt.Sprintf("%d", cfg.Level0SlowdownWritesTrigger))
		w.updateTiKVConfig(storeID, []string{"rocksdb", cf}, "level0-stop-writes-trigger", fmt.Sprintf("%d", cfg.Level0StopWritesTrigger))
		w.updateTiKVConfig(storeID, []string{"rocksdb", cf}, "max-compaction-bytes", fmt.Sprintf("%s", cfg.MaxCompactionBytes))
		w.updateTiKVConfig(storeID, []string{"rocksdb", cf}, "compaction-pri", fmt.Sprintf("%d", cfg.CompactionPri))
		w.updateTiKVConfig(storeID, []string{"rocksdb", cf}, "dynamic-level-bytes", fmt.Sprintf("%t", cfg.DynamicLevelBytes))
		w.updateTiKVConfig(storeID, []string{"rocksdb", cf}, "num-levels", fmt.Sprintf("%d", cfg.NumLevels))
		w.updateTiKVConfig(storeID, []string{"rocksdb", cf}, "max-bytes-for-level-multiplier", fmt.Sprintf("%d", cfg.MaxBytesForLevelMultiplier))
		w.updateTiKVConfig(storeID, []string{"rocksdb", cf}, "compaction-style", fmt.Sprintf("%d", cfg.CompactionStyle))
		w.updateTiKVConfig(storeID, []string{"rocksdb", cf}, "disable-auto-compactions", fmt.Sprintf("%t", cfg.DisableAutoCompactions))
		w.updateTiKVConfig(storeID, []string{"rocksdb", cf}, "soft-pending-compaction-bytes-limit", fmt.Sprintf("%s", cfg.SoftPendingCompactionBytesLimit))
		w.updateTiKVConfig(storeID, []string{"rocksdb", cf}, "hard-pending-compaction-bytes-limit", fmt.Sprintf("%s", cfg.HardPendingCompactionBytesLimit))
		w.updateTiKVConfig(storeID, []string{"rocksdb", cf}, "force-consistency-checks", fmt.Sprintf("%t", cfg.ForceConsistencyChecks))
		w.updateTiKVConfig(storeID, []string{"rocksdb", cf}, "prop-size-index-distance", fmt.Sprintf("%d", cfg.PropSizeIndexDistance))
		w.updateTiKVConfig(storeID, []string{"rocksdb", cf}, "prop-keys-index-distance", fmt.Sprintf("%d", cfg.PropKeysIndexDistance))
		w.updateTiKVConfig(storeID, []string{"rocksdb", cf}, "enable-doubly-skiplist", fmt.Sprintf("%t", cfg.EnableDoublySkiplist))
	}

	titanCfg := rocksdbCfg.Titan
	w.updateTiKVConfig(storeID, []string{"rocksdb", "titan"}, "enabled", fmt.Sprintf("%t", titanCfg.Enabled))
	w.updateTiKVConfig(storeID, []string{"rocksdb", "titan"}, "dirname", fmt.Sprintf("%s", titanCfg.Dirname))
	w.updateTiKVConfig(storeID, []string{"rocksdb", "titan"}, "disable-gc", fmt.Sprintf("%t", titanCfg.DisableGc))
	w.updateTiKVConfig(storeID, []string{"rocksdb", "titan"}, "max-background-gc", fmt.Sprintf("%d", titanCfg.MaxBackgroundGc))
	w.updateTiKVConfig(storeID, []string{"rocksdb", "titan"}, "purge-obsolete-files-period", fmt.Sprintf("%s", titanCfg.PurgeObsoleteFilesPeriod))

	stoargeCfg := tikvCfg.Storage
	err = w.updateTiKVConfig(storeID, []string{"storage"}, "data-dir", fmt.Sprintf("%s", stoargeCfg.DataDir))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"storage"}, "max-key-size", fmt.Sprintf("%d", stoargeCfg.MaxKeySize))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"storage"}, "scheduler-notify-capacity", fmt.Sprintf("%d", stoargeCfg.SchedulerNotifyCapacity))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"storage"}, "scheduler-concurrency", fmt.Sprintf("%d", stoargeCfg.SchedulerConcurrency))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"storage"}, "scheduler-worker-pool-size", fmt.Sprintf("%d", stoargeCfg.SchedulerWorkerPoolSize))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"storage"}, "scheduler-pending-write-threshold", fmt.Sprintf("%s", stoargeCfg.SchedulerPendingWriteThreshold))
	if err != nil {
		return err
	}

	blockCfg := tikvCfg.Storage.BlockCache
	err = w.updateTiKVConfig(storeID, []string{"storage", "block-cache"}, "shared", fmt.Sprintf("%t", blockCfg.Shared))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"storage", "block-cache"}, "capacity", fmt.Sprintf("%s", blockCfg.Capacity))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"storage", "block-cache"}, "num-shard-bits", fmt.Sprintf("%d", blockCfg.NumShardBits))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"storage", "block-cache"}, "strict-capacity-limit", fmt.Sprintf("%t", blockCfg.StrictCapacityLimit))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"storage", "block-cache"}, "high-pri-pool-ratio", fmt.Sprintf("%f", blockCfg.HighPriPoolRatio))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"storage", "block-cache"}, "memory-allocator", fmt.Sprintf("%s", blockCfg.MemoryAllocator))
	if err != nil {
		return err
	}

	serverCfg := tikvCfg.Server
	err = w.updateTiKVConfig(storeID, []string{"server"}, "grpc-compression-type", fmt.Sprintf("%s", serverCfg.GrpcCompressionType))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"server"}, "grpc-concurrency", fmt.Sprintf("%d", serverCfg.GrpcConcurrency))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"server"}, "grpc-concurrent-stream", fmt.Sprintf("%d", serverCfg.GrpcConcurrentStream))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"server"}, "grpc-raft-conn-num", fmt.Sprintf("%d", serverCfg.GrpcRaftConnNum))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"server"}, "grpc-stream-initial-window-size", fmt.Sprintf("%s", serverCfg.GrpcStreamInitialWindowSize))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"server"}, "grpc-keepalive-time", fmt.Sprintf("%s", serverCfg.GrpcKeepaliveTime))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"server"}, "grpc-keepalive-timeout", fmt.Sprintf("%s", serverCfg.GrpcKeepaliveTimeout))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"server"}, "concurrent-send-snap-limit", fmt.Sprintf("%d", serverCfg.ConcurrentSendSnapLimit))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"server"}, "concurrent-recv-snap-limit", fmt.Sprintf("%d", serverCfg.ConcurrentRecvSnapLimit))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"server"}, "end-point-recursion-limit", fmt.Sprintf("%d", serverCfg.EndPointRecursionLimit))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"server"}, "end-point-stream-channel-size", fmt.Sprintf("%d", serverCfg.EndPointStreamChannelSize))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"server"}, "end-point-batch-row-limit", fmt.Sprintf("%d", serverCfg.EndPointBatchRowLimit))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"server"}, "end-point-stream-batch-row-limit", fmt.Sprintf("%d", serverCfg.EndPointStreamBatchRowLimit))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"server"}, "end-point-enable-batch-if-possible", fmt.Sprintf("%t", serverCfg.EndPointEnableBatchIfPossible))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"server"}, "end-point-request-max-handle-duration", fmt.Sprintf("%s", serverCfg.EndPointRequestMaxHandleDuration))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"server"}, "snap-max-write-bytes-per-sec", fmt.Sprintf("%s", serverCfg.SnapMaxWriteBytesPerSec))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"server"}, "snap-max-total-size", fmt.Sprintf("%s", serverCfg.SnapMaxTotalSize))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"server"}, "stats-concurrency", fmt.Sprintf("%d", serverCfg.StatsConcurrency))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"server"}, "heavy-load-threshold", fmt.Sprintf("%d", serverCfg.HeavyLoadThreshold))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"server"}, "heavy-load-wait-duration", fmt.Sprintf("%s", serverCfg.HeavyLoadWaitDuration))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"server"}, "labels", fmt.Sprintf("%s", serverCfg.Labels))
	if err != nil {
		return err
	}

	raftstoreCfg := tikvCfg.Raftstore
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "sync-log", fmt.Sprintf("%t", raftstoreCfg.SyncLog))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "raft-base-tick-interval", raftstoreCfg.RaftBaseTickInterval)
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "raft-heartbeat-ticks", fmt.Sprintf("%d", raftstoreCfg.RaftHeartbeatTicks))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "raft-election-timeout-ticks", fmt.Sprintf("%d", raftstoreCfg.RaftElectionTimeoutTicks))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "raft-entry-max-size", raftstoreCfg.RaftEntryMaxSize)
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "raft-log-gc-tick-interval", raftstoreCfg.RaftLogGCTickInterval)
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "raft-log-gc-threshold", fmt.Sprintf("%d", raftstoreCfg.RaftLogGCThreshold))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "raft-log-gc-count-limit", fmt.Sprintf("%d", raftstoreCfg.RaftLogGCCountLimit))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "raft-log-gc-size-limit", raftstoreCfg.RaftLogGCSizeLimit)
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "raft-entry-cache-life-time", raftstoreCfg.RaftEntryCacheLifeTime)
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "raft-reject-transfer-leader-duration", raftstoreCfg.RaftRejectTransferLeaderDuration)
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "split-region-check-tick-interval", raftstoreCfg.SplitRegionCheckTickInterval)
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "region-split-check-diff", raftstoreCfg.RegionSplitCheckDiff)
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "region-compact-check-interval", raftstoreCfg.RegionCompactCheckInterval)
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "clean-stale-peer-delay", raftstoreCfg.CleanStalePeerDelay)
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "region-compact-check-step", fmt.Sprintf("%d", raftstoreCfg.RegionCompactCheckStep))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "region-compact-min-tombstones", fmt.Sprintf("%d", raftstoreCfg.RegionCompactMinTombstones))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "region-compact-tombstones-percent", fmt.Sprintf("%d", raftstoreCfg.RegionCompactTombstonesPercent))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "pd-heartbeat-tick-interval", raftstoreCfg.PdHeartbeatTickInterval)
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "pd-store-heartbeat-tick-interval", raftstoreCfg.PdStoreHeartbeatTickInterval)
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "snap-mgr-gc-tick-interval", raftstoreCfg.SnapMgrGCTickInterval)
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "snap-gc-timeout", raftstoreCfg.SnapGCTimeout)
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "lock-cf-compact-interval", raftstoreCfg.LockCfCompactInterval)
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "lock-cf-compact-bytes-threshold", raftstoreCfg.LockCfCompactBytesThreshold)
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "notify-capacity", fmt.Sprintf("%d", raftstoreCfg.NotifyCapacity))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "messages-per-tick", fmt.Sprintf("%d", raftstoreCfg.MessagesPerTick))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "max-peer-down-duration", raftstoreCfg.MaxPeerDownDuration)
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "max-leader-missing-duration", raftstoreCfg.MaxLeaderMissingDuration)
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "abnormal-leader-missing-duration", raftstoreCfg.AbnormalLeaderMissingDuration)
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "peer-stale-state-check-interval", raftstoreCfg.PeerStaleStateCheckInterval)
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "leader-transfer-max-log-lag", fmt.Sprintf("%d", raftstoreCfg.LeaderTransferMaxLogLag))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "snap-apply-batch-size", raftstoreCfg.SnapApplyBatchSize)
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "consistency-check-interval", raftstoreCfg.ConsistencyCheckInterval)
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "report-region-flow-interval", raftstoreCfg.ReportRegionFlowInterval)
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "raft-store-max-leader-lease", raftstoreCfg.RaftStoreMaxLeaderLease)
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "right-derive-when-split", fmt.Sprintf("%t", raftstoreCfg.RightDeriveWhenSplit))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "allow-remove-leader", fmt.Sprintf("%t", raftstoreCfg.AllowRemoveLeader))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "merge-max-log-gap", fmt.Sprintf("%d", raftstoreCfg.MergeMaxLogGap))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "merge-check-tick-interval", raftstoreCfg.MergeCheckTickInterval)
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "use-delete-range", fmt.Sprintf("%t", raftstoreCfg.UseDeleteRange))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "cleanup-import-sst-interval", raftstoreCfg.CleanupImportSstInterval)
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "apply-max-batch-size", fmt.Sprintf("%d", raftstoreCfg.ApplyMaxBatchSize))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "apply-pool-size", fmt.Sprintf("%d", raftstoreCfg.ApplyPoolSize))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "store-max-batch-size", fmt.Sprintf("%d", raftstoreCfg.StoreMaxBatchSize))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "store-pool-size", fmt.Sprintf("%d", raftstoreCfg.StorePoolSize))
	if err != nil {
		return err
	}
	err = w.updateTiKVConfig(storeID, []string{"raftstore"}, "hibernate-regions", fmt.Sprintf("%t", raftstoreCfg.HibernateRegions))
	if err != nil {
		return err
	}

	return nil
}

func (w *configWorker) updateTiKVConfig(
	storeID uint64, comp []string, name, value string) error {

	comps := strings.Join(comp, ",")
	stmt := fmt.Sprintf(`
INSERT HIGH_PRIORITY INTO mysql.tikv VALUES ('%[1]d', '%[2]s', '%[3]s', '%[4]s')
ON DUPLICATE KEY UPDATE variable_value = '%[4]s'`,
		storeID, comps, name, value)

	if w.session == nil {
		return errors.New("[saveValueToSysTable session is nil]")
	}

	_, err := w.session.Execute(context.Background(), stmt)
	logutil.BgLogger().Debug("[config worker] save config",
		zap.String("name", name),
		zap.String("value", value),
		zap.Error(err))

	return errors.Trace(err)
}
