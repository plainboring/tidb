package infoschema

var tikvConfigFixture string = `
log-level = "info"
log-file = "tikv.log"
log-rotation-timespan = "1d"
panic-when-unexpected-key-or-data = false
[readpool.storage]
high-concurrency = 4
normal-concurrency = 4
low-concurrency = 4
max-tasks-per-worker-high = 2000
max-tasks-per-worker-normal = 2000
max-tasks-per-worker-low = 2000
stack-size = "10MiB"

[readpool.coprocessor]
high-concurrency = 32
normal-concurrency = 32
low-concurrency = 32
max-tasks-per-worker-high = 2000
max-tasks-per-worker-normal = 2000
max-tasks-per-worker-low = 2000
stack-size = "10MiB"

[server]
addr = "127.0.0.1:20161"
advertise-addr = ""
status-addr = "127.0.0.1:20180"
status-thread-pool-size = 1
grpc-compression-type = "none"
grpc-concurrency = 4
grpc-concurrent-stream = 1024
grpc-raft-conn-num = 1
grpc-stream-initial-window-size = "2MiB"
grpc-keepalive-time = "10s"
grpc-keepalive-timeout = "3s"
concurrent-send-snap-limit = 32
concurrent-recv-snap-limit = 32
end-point-recursion-limit = 1000
end-point-stream-channel-size = 8
end-point-batch-row-limit = 64
end-point-stream-batch-row-limit = 128
end-point-enable-batch-if-possible = true
end-point-request-max-handle-duration = "1m"
snap-max-write-bytes-per-sec = "100MiB"
snap-max-total-size = "0KiB"
stats-concurrency = 1
heavy-load-threshold = 300
heavy-load-wait-duration = "1ms"

[server.labels]

[storage]
data-dir = "./"
gc-ratio-threshold = 1.1
max-key-size = 4096
scheduler-notify-capacity = 10240
scheduler-concurrency = 2048000
scheduler-worker-pool-size = 8
scheduler-pending-write-threshold = "100MiB"

[storage.block-cache]
shared = true
num-shard-bits = 6
strict-capacity-limit = false
high-pri-pool-ratio = 0.8
memory-allocator = "nodump"

[pd]
endpoints = ["http://127.0.0.1:2371"]
retry-interval = "300ms"
retry-max-count = 9223372036854775807
retry-log-every = 10

[metric]
interval = "15s"
address = ""
job = "tikv"

[raftstore]
sync-log = true
prevote = true
raftdb-path = ""
capacity = "0KiB"
raft-base-tick-interval = "1s"
raft-heartbeat-ticks = 2
raft-election-timeout-ticks = 10
raft-min-election-timeout-ticks = 0
raft-max-election-timeout-ticks = 0
raft-max-size-per-msg = "1MiB"
raft-max-inflight-msgs = 256
raft-entry-max-size = "8MiB"
raft-log-gc-tick-interval = "10s"
raft-log-gc-threshold = 50
raft-log-gc-count-limit = 73728
raft-log-gc-size-limit = "72MiB"
raft-entry-cache-life-time = "30s"
raft-reject-transfer-leader-duration = "3s"
split-region-check-tick-interval = "10s"
region-split-check-diff = "6MiB"
region-compact-check-interval = "5m"
clean-stale-peer-delay = "10m"
region-compact-check-step = 100
region-compact-min-tombstones = 10000
region-compact-tombstones-percent = 30
pd-heartbeat-tick-interval = "1m"
pd-store-heartbeat-tick-interval = "10s"
snap-mgr-gc-tick-interval = "1m"
snap-gc-timeout = "4h"
lock-cf-compact-interval = "10m"
lock-cf-compact-bytes-threshold = "256MiB"
notify-capacity = 40960
messages-per-tick = 4096
max-peer-down-duration = "5m"
max-leader-missing-duration = "2h"
abnormal-leader-missing-duration = "10m"
peer-stale-state-check-interval = "5m"
leader-transfer-max-log-lag = 10
snap-apply-batch-size = "10MiB"
consistency-check-interval = "0s"
report-region-flow-interval = "1m"
raft-store-max-leader-lease = "9s"
right-derive-when-split = true
allow-remove-leader = false
merge-max-log-gap = 10
merge-check-tick-interval = "10s"
use-delete-range = false
cleanup-import-sst-interval = "10m"
local-read-batch-size = 1024
apply-max-batch-size = 1024
apply-pool-size = 2
store-max-batch-size = 1024
store-pool-size = 2
future-poll-size = 1
hibernate-regions = false

[coprocessor]
split-region-on-table = true
batch-split-limit = 10
region-max-size = "144MiB"
region-split-size = "96MiB"
region-max-keys = 1440000
region-split-keys = 960000

[rocksdb]
wal-recovery-mode = 2
wal-dir = ""
wal-ttl-seconds = 0
wal-size-limit = "0KiB"
max-total-wal-size = "4GiB"
max-background-jobs = 6
max-manifest-file-size = "128MiB"
create-if-missing = true
max-open-files = 40960
enable-statistics = true
stats-dump-period = "10m"
compaction-readahead-size = "0KiB"
info-log-max-size = "1GiB"
info-log-roll-time = "0s"
info-log-keep-log-file-num = 10
info-log-dir = ""
rate-bytes-per-sec = "0KiB"
rate-limiter-mode = 2
auto-tuned = false
bytes-per-sync = "1MiB"
wal-bytes-per-sync = "512KiB"
max-sub-compactions = 1
writable-file-max-buffer-size = "1MiB"
use-direct-io-for-flush-and-compaction = false
enable-pipelined-write = true

[rocksdb.defaultcf]
block-size = "64KiB"
block-cache-size = "32163MiB"
disable-block-cache = false
cache-index-and-filter-blocks = true
pin-l0-filter-and-index-blocks = true
use-bloom-filter = true
optimize-filters-for-hits = true
whole-key-filtering = true
bloom-filter-bits-per-key = 10
block-based-bloom-filter = false
read-amp-bytes-per-bit = 0
compression-per-level = ["no", "no", "lz4", "lz4", "lz4", "zstd", "zstd"]
write-buffer-size = "128MiB"
max-write-buffer-number = 5
min-write-buffer-number-to-merge = 1
max-bytes-for-level-base = "512MiB"
target-file-size-base = "8MiB"
level0-file-num-compaction-trigger = 4
level0-slowdown-writes-trigger = 20
level0-stop-writes-trigger = 36
max-compaction-bytes = "2GiB"
compaction-pri = 3
dynamic-level-bytes = true
num-levels = 7
max-bytes-for-level-multiplier = 10
compaction-style = 0
disable-auto-compactions = false
soft-pending-compaction-bytes-limit = "64GiB"
hard-pending-compaction-bytes-limit = "256GiB"
force-consistency-checks = true
prop-size-index-distance = 4194304
prop-keys-index-distance = 40960
enable-doubly-skiplist = true

[rocksdb.defaultcf.titan]
min-blob-size = "1KiB"
blob-file-compression = "lz4"
blob-cache-size = "0KiB"
min-gc-batch-size = "16MiB"
max-gc-batch-size = "64MiB"
discardable-ratio = 0.5
sample-ratio = 0.1
merge-small-file-threshold = "8MiB"
blob-run-mode = "normal"

[rocksdb.writecf]
block-size = "64KiB"
block-cache-size = "19298MiB"
disable-block-cache = false
cache-index-and-filter-blocks = true
pin-l0-filter-and-index-blocks = true
use-bloom-filter = true
optimize-filters-for-hits = false
whole-key-filtering = false
bloom-filter-bits-per-key = 10
block-based-bloom-filter = false
read-amp-bytes-per-bit = 0
compression-per-level = ["no", "no", "lz4", "lz4", "lz4", "zstd", "zstd"]
write-buffer-size = "128MiB"
max-write-buffer-number = 5
min-write-buffer-number-to-merge = 1
max-bytes-for-level-base = "512MiB"
target-file-size-base = "8MiB"
level0-file-num-compaction-trigger = 4
level0-slowdown-writes-trigger = 20
level0-stop-writes-trigger = 36
max-compaction-bytes = "2GiB"
compaction-pri = 3
dynamic-level-bytes = true
num-levels = 7
max-bytes-for-level-multiplier = 10
compaction-style = 0
disable-auto-compactions = false
soft-pending-compaction-bytes-limit = "64GiB"
hard-pending-compaction-bytes-limit = "256GiB"
force-consistency-checks = true
prop-size-index-distance = 4194304
prop-keys-index-distance = 40960
enable-doubly-skiplist = true

[rocksdb.writecf.titan]
min-blob-size = "1KiB"
blob-file-compression = "lz4"
blob-cache-size = "0KiB"
min-gc-batch-size = "16MiB"
max-gc-batch-size = "64MiB"
discardable-ratio = 0.5
sample-ratio = 0.1
merge-small-file-threshold = "8MiB"
blob-run-mode = "read-only"

[rocksdb.lockcf]
block-size = "16KiB"
block-cache-size = "1GiB"
disable-block-cache = false
cache-index-and-filter-blocks = true
pin-l0-filter-and-index-blocks = true
use-bloom-filter = true
optimize-filters-for-hits = false
whole-key-filtering = true
bloom-filter-bits-per-key = 10
block-based-bloom-filter = false
read-amp-bytes-per-bit = 0
compression-per-level = ["no", "no", "no", "no", "no", "no", "no"]
write-buffer-size = "128MiB"
max-write-buffer-number = 5
min-write-buffer-number-to-merge = 1
max-bytes-for-level-base = "128MiB"
target-file-size-base = "8MiB"
level0-file-num-compaction-trigger = 1
level0-slowdown-writes-trigger = 20
level0-stop-writes-trigger = 36
max-compaction-bytes = "2GiB"
compaction-pri = 0
dynamic-level-bytes = true
num-levels = 7
max-bytes-for-level-multiplier = 10
compaction-style = 0
disable-auto-compactions = false
soft-pending-compaction-bytes-limit = "64GiB"
hard-pending-compaction-bytes-limit = "256GiB"
force-consistency-checks = true
prop-size-index-distance = 4194304
prop-keys-index-distance = 40960
enable-doubly-skiplist = true

[rocksdb.lockcf.titan]
min-blob-size = "1KiB"
blob-file-compression = "lz4"
blob-cache-size = "0KiB"
min-gc-batch-size = "16MiB"
max-gc-batch-size = "64MiB"
discardable-ratio = 0.5
sample-ratio = 0.1
merge-small-file-threshold = "8MiB"
blob-run-mode = "read-only"

[rocksdb.raftcf]
block-size = "16KiB"
block-cache-size = "128MiB"
disable-block-cache = false
cache-index-and-filter-blocks = true
pin-l0-filter-and-index-blocks = true
use-bloom-filter = true
optimize-filters-for-hits = true
whole-key-filtering = true
bloom-filter-bits-per-key = 10
block-based-bloom-filter = false
read-amp-bytes-per-bit = 0
compression-per-level = ["no", "no", "no", "no", "no", "no", "no"]
write-buffer-size = "128MiB"
max-write-buffer-number = 5
min-write-buffer-number-to-merge = 1
max-bytes-for-level-base = "128MiB"
target-file-size-base = "8MiB"
level0-file-num-compaction-trigger = 1
level0-slowdown-writes-trigger = 20
level0-stop-writes-trigger = 36
max-compaction-bytes = "2GiB"
compaction-pri = 0
dynamic-level-bytes = true
num-levels = 7
max-bytes-for-level-multiplier = 10
compaction-style = 0
disable-auto-compactions = false
soft-pending-compaction-bytes-limit = "64GiB"
hard-pending-compaction-bytes-limit = "256GiB"
force-consistency-checks = true
prop-size-index-distance = 4194304
prop-keys-index-distance = 40960
enable-doubly-skiplist = true

[rocksdb.raftcf.titan]
min-blob-size = "1KiB"
blob-file-compression = "lz4"
blob-cache-size = "0KiB"
min-gc-batch-size = "16MiB"
max-gc-batch-size = "64MiB"
discardable-ratio = 0.5
sample-ratio = 0.1
merge-small-file-threshold = "8MiB"
blob-run-mode = "read-only"

[rocksdb.titan]
enabled = false
dirname = ""
disable-gc = false
max-background-gc = 1
purge-obsolete-files-period = "10s"

[raftdb]
wal-recovery-mode = 2
wal-dir = ""
wal-ttl-seconds = 0
wal-size-limit = "0KiB"
max-total-wal-size = "4GiB"
max-background-jobs = 4
max-manifest-file-size = "20MiB"
create-if-missing = true
max-open-files = 40960
enable-statistics = true
stats-dump-period = "10m"
compaction-readahead-size = "0KiB"
info-log-max-size = "1GiB"
info-log-roll-time = "0s"
info-log-keep-log-file-num = 10
info-log-dir = ""
max-sub-compactions = 2
writable-file-max-buffer-size = "1MiB"
use-direct-io-for-flush-and-compaction = false
enable-pipelined-write = true
allow-concurrent-memtable-write = false
bytes-per-sync = "1MiB"
wal-bytes-per-sync = "512KiB"

[raftdb.defaultcf]
block-size = "64KiB"
block-cache-size = "2GiB"
disable-block-cache = false
cache-index-and-filter-blocks = true
pin-l0-filter-and-index-blocks = true
use-bloom-filter = false
optimize-filters-for-hits = true
whole-key-filtering = true
bloom-filter-bits-per-key = 10
block-based-bloom-filter = false
read-amp-bytes-per-bit = 0
compression-per-level = ["no", "no", "lz4", "lz4", "lz4", "zstd", "zstd"]
write-buffer-size = "128MiB"
max-write-buffer-number = 5
min-write-buffer-number-to-merge = 1
max-bytes-for-level-base = "512MiB"
target-file-size-base = "8MiB"
level0-file-num-compaction-trigger = 4
level0-slowdown-writes-trigger = 20
level0-stop-writes-trigger = 36
max-compaction-bytes = "2GiB"
compaction-pri = 0
dynamic-level-bytes = true
num-levels = 7
max-bytes-for-level-multiplier = 10
compaction-style = 0
disable-auto-compactions = false
soft-pending-compaction-bytes-limit = "64GiB"
hard-pending-compaction-bytes-limit = "256GiB"
force-consistency-checks = true
prop-size-index-distance = 4194304
prop-keys-index-distance = 40960
enable-doubly-skiplist = true

[raftdb.defaultcf.titan]
min-blob-size = "1KiB"
blob-file-compression = "lz4"
blob-cache-size = "0KiB"
min-gc-batch-size = "16MiB"
max-gc-batch-size = "64MiB"
discardable-ratio = 0.5
sample-ratio = 0.1
merge-small-file-threshold = "8MiB"
blob-run-mode = "normal"

[security]
ca-path = ""
cert-path = ""
key-path = ""
cipher-file = ""

[import]
num-threads = 8
stream-channel-window = 128

[pessimistic-txn]
enabled = true
wait-for-lock-timeout = 3000
wake-up-delay-duration = 1
`
