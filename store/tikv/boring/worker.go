package boring

import (
	"context"
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/juju/errors"
	"github.com/pingcap/log"
	pd "github.com/pingcap/pd/client"
	"github.com/pingcap/tidb/privilege"
	"github.com/pingcap/tidb/session"
	"github.com/pingcap/tidb/store/tikv"
	"github.com/pingcap/tidb/util/logutil"
	"github.com/plainboring/config_client/pkg/cfgclient"
	kvcfg "github.com/plainboring/config_client/pkg/tikv"
	"go.uber.org/zap"
)

var defaultWorker *configWorker

// ConfigWorker ..
type configWorker struct {
	pdClient  pd.Client
	cfgClient cfgclient.ConfigClient
	session   session.Session
}

// NewConfigWorker returns a new config worker
func NewConfigWorker(storage tikv.Storage, pdClient pd.Client) (tikv.ConfigHandler, error) {
	cfgClient, err := cfgclient.NewMockClient()
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
	w.initTiKVConfig(1)
}

// UpdateTiKV updates tikv config
func (w *configWorker) UpdateTiKV(
	storeID uint64, subs []string, name, value string) error {
	log.Info("config client",
		zap.Uint64("storeID", storeID), zap.Strings("sub", subs),
		zap.String("name", name), zap.String("value", value),
	)
	err := w.cfgClient.Update("tikv", subs, name, value, storeID)
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
