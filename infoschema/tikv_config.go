package infoschema

// TikvConfig is the configuration of TiKV server.
type TikvConfig struct {
	Raftstore RaftstoreConfig `toml:"raftstore"`
}

// RaftstoreConfig is the configuration of TiKV raftstore component.
type RaftstoreConfig struct {
	// true for high reliability, prevent data loss when power failure.
	SyncLog bool `toml:"sync-log"`

	// raft-base-tick-interval is a base tick interval (ms).
	RaftBaseTickInterval     string `toml:"raft-base-tick-interval"`
	RaftHeartbeatTicks       int64  `toml:"raft-heartbeat-ticks"`
	RaftElectionTimeoutTicks int64  `toml:"raft-election-timeout-ticks"`
	// When the entry exceed the max size, reject to propose it.
	RaftEntryMaxSize string `toml:"raft-entry-max-size"`

	// Interval to gc unnecessary raft log (ms).
	RaftLogGCTickInterval string `toml:"raft-log-gc-tick-interval"`
	// A threshold to gc stale raft log, must >= 1.
	RaftLogGCThreshold int64 `toml:"raft-log-gc-threshold"`
	// When entry count exceed this value, gc will be forced trigger.
	RaftLogGCCountLimit int64 `toml:"raft-log-gc-count-limit"`
	// When the approximate size of raft log entries exceed this value
	// gc will be forced trigger.
	RaftLogGCSizeLimit string `toml:"raft-log-gc-size-limit"`
	// When a peer is not responding for this time, leader will not keep entry cache for it.
	RaftEntryCacheLifeTime string `toml:"raft-entry-cache-life-time"`
	// When a peer is newly added, reject transferring leader to the peer for a while.
	RaftRejectTransferLeaderDuration string `toml:"raft-reject-transfer-leader-duration"`

	// Interval (ms) to check region whether need to be split or not.
	SplitRegionCheckTickInterval string `toml:"split-region-check-tick-interval"`
	/// When size change of region exceed the diff since last check, it
	/// will be checked again whether it should be split.
	RegionSplitCheckDiff string `toml:"region-split-check-diff"`
	/// Interval (ms) to check whether start compaction for a region.
	RegionCompactCheckInterval string `toml:"region-compact-check-interval"`
	// delay time before deleting a stale peer
	CleanStalePeerDelay string `toml:"clean-stale-peer-delay"`
	/// Number of regions for each time checking.
	RegionCompactCheckStep int64 `toml:"region-compact-check-step"`
	/// Minimum number of tombstones to trigger manual compaction.
	RegionCompactMinTombstones int64 `toml:"region-compact-min-tombstones"`
	/// Minimum percentage of tombstones to trigger manual compaction.
	/// Should between 1 and 100.
	RegionCompactTombstonesPercent int64  `toml:"region-compact-tombstones-percent"`
	PdHeartbeatTickInterval        string `toml:"pd-heartbeat-tick-interval"`
	PdStoreHeartbeatTickInterval   string `toml:"pd-store-heartbeat-tick-interval"`
	SnapMgrGCTickInterval          string `toml:"snap-mgr-gc-tick-interval"`
	SnapGCTimeout                  string `toml:"snap-gc-timeout"`
	LockCfCompactInterval          string `toml:"lock-cf-compact-interval"`
	LockCfCompactBytesThreshold    string `toml:"lock-cf-compact-bytes-threshold"`

	NotifyCapacity  int64 `toml:"notify-capacity"`
	MessagesPerTick int64 `toml:"messages-per-tick"`

	/// When a peer is not active for max-peer-down-duration
	/// the peer is considered to be down and is reported to PD.
	MaxPeerDownDuration string `toml:"max-peer-down-duration"`

	/// If the leader of a peer is missing for longer than max-leader-missing-duration
	/// the peer would ask pd to confirm whether it is valid in any region.
	/// If the peer is stale and is not valid in any region, it will destroy itself.
	MaxLeaderMissingDuration string `toml:"max-leader-missing-duration"`
	/// Similar to the max-leader-missing-duration, instead it will log warnings and
	/// try to alert monitoring systems, if there is any.
	AbnormalLeaderMissingDuration string `toml:"abnormal-leader-missing-duration"`
	PeerStaleStateCheckInterval   string `toml:"peer-stale-state-check-interval"`

	LeaderTransferMaxLogLag int64 `toml:"leader-transfer-max-log-lag"`

	SnapApplyBatchSize string `toml:"snap-apply-batch-size"`

	// Interval (ms) to check region whether the data is consistent.
	ConsistencyCheckInterval string `toml:"consistency-check-interval"`

	ReportRegionFlowInterval string `toml:"report-region-flow-interval"`

	// The lease provided by a successfully proposed and applied entry.
	RaftStoreMaxLeaderLease string `toml:"raft-store-max-leader-lease"`

	// Right region derive origin region id when split.
	RightDeriveWhenSplit bool `toml:"right-derive-when-split"`

	AllowRemoveLeader bool `toml:"allow-remove-leader"`

	/// Max log gap allowed to propose merge.
	MergeMaxLogGap int64 `toml:"merge-max-log-gap"`
	/// Interval to re-propose merge.
	MergeCheckTickInterval string `toml:"merge-check-tick-interval"`

	UseDeleteRange bool `toml:"use-delete-range"`

	CleanupImportSstInterval string `toml:"cleanup-import-sst-interval"`

	ApplyMaxBatchSize int64 `toml:"apply-max-batch-size"`
	ApplyPoolSize     int64 `toml:"apply-pool-size"`

	StoreMaxBatchSize int64 `toml:"store-max-batch-size"`
	StorePoolSize     int64 `toml:"store-pool-size"`
	HibernateRegions  bool  `toml:"hibernate-regions"`
}
