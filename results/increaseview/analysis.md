# IncreaseView Attack Analysis and Fix

**Date**: 2026-04-08
**Config**: n=30, relaxed commit rule, `--timeout-multiplier 1.0`, `--duration-samples 100`, 30s duration

## IncreaseView Attack Behavior

The IncreaseView attack (`protocol/rules/byzantine/increaseview.go`) proposes blocks with the view number **inflated by +1000**. When the byzantine replica is leader for view V, it proposes a block at view V+1000 with a valid QC (current highQC).

```go
const ByzViewExtraIncrease hotstuff.View = 1000
proposal = hotstuff.NewProposeMsg(iv.config.ID(), view+ByzViewExtraIncrease, qc, cmd)
```


## Root Cause: Timeout Collector Pollution

The IncreaseView attack causes livelock through the **timeout path**, not the proposal path. The verify-before-advanceView fix (applied for fork attacks) does not prevent this livelock.

### Attack Mechanism

1. Byzantine replica (leader for view V) proposes at view V+1000 -- rejected by `Verify()` (leader check fails)
2. Byzantine replica also times out locally and broadcasts `TimeoutMsg{View: V+1000}` with a valid signature
3. Honest replicas receive this in `OnRemoteTimeout()` -- the signature is valid
4. The timeout is added to the **flat timeout collector**

### The Two Bugs

**Bug 1: No view distance filter on incoming timeouts** (`synchronizer.go`)

`OnRemoteTimeout()` accepted timeout messages for ANY view, including V+1000. These inflated timeouts consumed collector capacity.

**Bug 2: Flat timeout collector with cross-view quorum check** (`timeout_collector.go`)

```go
// BEFORE (buggy): single flat slice, quorum checks total count across ALL views
type timeoutCollector struct {
    timeouts []hotstuff.TimeoutMsg   // all views mixed together
}
func (s *timeoutCollector) add(timeout hotstuff.TimeoutMsg) ([]hotstuff.TimeoutMsg, bool) {
    s.timeouts = append(s.timeouts, timeout)
    if len(s.timeouts) < s.config.QuorumSize() {  // ← counts ALL views
        return nil, false
    }
```

With 3 byzantine timeouts at V+1000 and 24 honest timeouts at V, the total (27) could exceed the quorum threshold (20), triggering on a **mixed-view** set. The resulting TC was invalid or desynchronized replicas.


## The Fixes

### Fix 1: View distance filter in `OnRemoteTimeout` (`synchronizer.go`)

```go
// Reject timeout messages with views too far ahead of the current view.
const alpha hotstuff.View = 10
if timeout.View > currView+alpha {
    return  // drop inflated-view timeouts
}
```

### Fix 2: Per-view timeout collector (`timeout_collector.go`)

```go
// AFTER (fixed): map keyed by view, quorum checks per-view count only
type timeoutCollector struct {
    timeouts map[hotstuff.View][]hotstuff.TimeoutMsg
}
func (s *timeoutCollector) add(timeout hotstuff.TimeoutMsg) ([]hotstuff.TimeoutMsg, bool) {
    view := timeout.View
    viewTimeouts := s.timeouts[view]
    viewTimeouts = append(viewTimeouts, timeout)
    s.timeouts[view] = viewTimeouts
    if len(viewTimeouts) < s.config.QuorumSize() {  // ← counts THIS view only
        return nil, false
    }
```


## Results: Before vs After Fix


| Metric                     | Honest  | Before Fix    | **After Fix** | Silent (ref) | Fork (ref) |
| -------------------------- | ------- | ------------- | ------------- | ------------ | ---------- |
| **Throughput (commits/s)** | **160** | **0**         | **79**        | **81**       | **76**     |
| Total commits              | 153,360 | 90            | 75,930        | 77,880       | 73,110     |
| Consensus latency (ms)     | 20.880  | 31.056        | 38.779        | 37.685       | 34.992     |
| Client latency (ms)        | 35.289  | 501.467       | 72.380        | 70.356       | 61.145     |
| View timeout rate          | 2.30%   | **99.60%**    | **14.83%**    | 15.33%       | 12.44%     |
| Total views                | 156,300 | 37,741        | 86,489        | 88,561       | 93,870     |
| % of honest throughput     | 100%    | 0.1%          | **49.5%**     | 50.8%        | 47.7%      |


### Improvement

| Metric             | Before Fix         | After Fix            |
| ------------------ | ------------------ | -------------------- |
| Throughput         | 0/s (livelock)     | **79/s**             |
| Timeout rate       | 99.60%             | **14.83%**           |
| Client latency     | 501ms (timeout)    | **72ms (stable)**    |
| % of honest        | 0.1%               | **49.5%**            |


### Client Latency Trend (After Fix)

Stable at ~70ms after warmup:

```
interval 1: 134.4ms (42 cmds)
interval 2:  70.0ms (89 cmds)
interval 3:  69.1ms (85 cmds)
interval 4:  70.6ms (86 cmds)
interval 5:  74.0ms (82 cmds)
```


## All Strategies Normalized (10% Byzantine, n=30, After All Fixes)


| Strategy         | Throughput | % of Honest | Client Latency | Timeout Rate |
| ---------------- | ---------- | ----------- | -------------- | ------------ |
| Honest           | 160/s      | 100%        | 35.3ms         | 2.3%         |
| Silent Proposer  | 81/s       | 50.8%       | 70.4ms         | 15.3%        |
| **IncreaseView** | **79/s**   | **49.5%**   | **72.4ms**     | **14.8%**    |
| Fork             | 76/s       | 47.7%       | 61.1ms         | 12.4%        |

After both fixes (verify-before-advanceView for fork, view-distance filter + per-view collector for IncreaseView), **all three byzantine strategies produce similar impact** at 10%: ~48-51% of honest throughput. The protocol correctly neutralizes each attack's unique advantage.


## Full Scaling: IncreaseView 0%-30% (n=30, After Fix)


| Metric                     | Honest  | 10%     | 20%      | 30%       |
| -------------------------- | ------- | ------- | -------- | --------- |
| **Throughput (commits/s)** | **160** | **79**  | **34**   | **11**    |
| Total commits              | 153,360 | 75,930  | 32,280   | 10,680    |
| Consensus latency (ms)     | 20.880  | 38.779  | 88.605   | 286.957   |
| Client latency (ms)        | 35.289  | 72.380  | 171.724  | 468.278   |
| View timeout rate          | 2.30%   | 14.83%  | 26.51%   | 36.76%    |
| Total views                | 156,300 | 86,489  | 41,310   | 15,390    |
| **% of honest throughput** | **100%**| **49.5%**| **21.0%**| **7.0%** |


### IncreaseView Throughput Curve (After Fix)

```
Byzantine %   Throughput    % of Honest
---------     ----------    -----------
  0%            160/s         100.0%
 10%             79/s          49.5%
 20%             34/s          21.0%
 30%             11/s           7.0%
```


## Cross-Strategy Comparison at 20% and 30% (n=30, All Fixes Applied)


### 20% Byzantine

| Metric                 | IncreaseView | Silent Proposer | Fork   |
| ---------------------- | ------------ | --------------- | ------ |
| Throughput (commits/s) | 34           | 37              | **49** |
| Client latency (ms)    | 171.7        | 154.8           | **70.6** |
| Timeout rate           | 26.5%        | 29.5%           | **21.2%** |


### 30% Byzantine

| Metric                 | IncreaseView | **Silent Proposer** | Fork |
| ---------------------- | ------------ | ------------------- | ---- |
| Throughput (commits/s) | 11           | **14**              | 5    |
| Client latency (ms)    | 468.3        | **413.8**           | 396.6 |
| Timeout rate           | **36.8%**    | 46.3%               | 65.2% |


## Key Takeaways

1. **The fix resolves the 10% livelock**: IncreaseView throughput goes from 0/s to 79/s (49.5% of honest), matching other strategies.

2. **Two independent vulnerabilities were fixed**: The view distance filter prevents inflated timeouts from entering the system. The per-view collector prevents cross-view quorum pollution.

3. **After all fixes, all strategies degrade similarly**:
   - At 10%: all retain ~48-51% of honest throughput
   - At 20%: IncView 21%, Silent 23%, Fork 31%
   - At 30%: IncView 7%, Silent 9%, Fork 3%

4. **Fork attack is the least damaging at 20% (after fix)**: Fork retains 31% of honest throughput vs ~21-23% for the other two. This is because the fork fix (verify-before-advanceView) causes fork proposals to be cheaply rejected, making them behave like silent proposers but with faster view recovery.

5. **At 30% (f=f_max), all strategies severely cripple the system**: 5-14/s throughput, 37-65% timeout rates. The fault tolerance boundary remains a practical limit regardless of the attack strategy.

6. **Degradation is consistent and gradual (no more livelock cliffs)**: Before fixes, fork and IncreaseView had sharp cliff edges (functional at 10%, livelock at 20%). After fixes, all strategies show smooth, super-linear degradation matching the silent proposer pattern.
