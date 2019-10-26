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
