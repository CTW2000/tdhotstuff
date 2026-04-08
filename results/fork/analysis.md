# Fork Attack Analysis (n=30, 0%-30% Byzantine)

**Date**: 2026-04-08
**Config**: n=30, relaxed commit rule, `--timeout-multiplier 1.0`, `--duration-samples 100`, 30s duration

## Fork Attack Behavior

The fork attack (`protocol/rules/byzantine/fork.go`) proposes blocks based on the **grandparent's QC** instead of the current highQC. This builds on an older block, attempting to create a competing chain branch. Unlike the silent proposer which produces no block, the fork attacker actively produces blocks -- but on the wrong chain.

## Byzantine Configurations


| Experiment | Fork Attackers | Replica IDs                       | f used / f max |
| ---------- | -------------- | --------------------------------- | -------------- |
| Honest     | 0              | -                                 | 0 / 9          |
| 10%        | 3              | [2, 12, 22]                       | 3 / 9          |
| 20%        | 6              | [2, 7, 12, 17, 22, 27]            | 6 / 9          |
| 30%        | 9              | [2, 5, 8, 12, 15, 18, 22, 25, 28] | 9 / 9 (limit)  |


## Results: Fork Attack Scaling (n=30)


| Metric                     | Honest   | 10% (3 byz) | 20% (6 byz) | 30% (9 byz) |
| -------------------------- | -------- | ----------- | ----------- | ----------- |
| **Throughput (commits/s)** | **160**  | **76**      | **0**       | **0**       |
| Total commits/replica      | 5,112    | 2,437       | 1           | 0           |
| Consensus latency (ms)     | 20.880   | 34.992      | 24.229      | N/A         |
| Client latency (ms)        | 35.289   | 61.145      | 502.677     | 503.577     |
| View timeout rate          | 2.30%    | 12.44%      | 99.27%      | 99.01%      |
| Total views                | 156,300  | 93,870      | 23,850      | 18,249      |
| **% of honest throughput** | **100%** | **47.7%**   | **0%**      | **0%**      |


### Fork Throughput Curve

```
Byzantine %   Throughput    % of Honest
---------     ----------    -----------
  0%            160/s         100.0%
 10%             76/s          47.7%
 20%              0/s           0.0%  (livelock)
 30%              0/s           0.0%  (livelock)
```

## Cross-Strategy Comparison: Fork vs Silent Proposer (n=30)


| Metric                     | Fork 0% | Fork 10% | Fork 20% | Fork 30% | Silent 0% | Silent 10% | Silent 20% | Silent 30% |
| -------------------------- | ------- | -------- | -------- | -------- | --------- | ---------- | ---------- | ---------- |
| **Throughput (commits/s)** | **160** | **76**   | **0**    | **0**    | **159**   | **81**     | **37**     | **14**     |
| % of honest throughput     | 100%    | 47.7%    | 0%       | 0%       | 100%      | 50.9%      | 23.1%      | 8.5%       |
| Client latency (ms)        | 35.3    | 61.1     | 502.7    | 503.6    | 35.4      | 70.4       | 154.8      | 413.8      |
| Timeout rate               | 2.30%   | 12.44%   | 99.27%   | 99.01%   | 2.28%     | 15.33%     | 29.51%     | 46.33%     |


### Throughput Comparison Chart

```
Byzantine %    Fork          Silent Proposer
---------      ----          ---------------
  0%           160/s (100%)  159/s (100%)
 10%            76/s (47.7%)  81/s (50.9%)
 20%             0/s  (0.0%)  37/s (23.1%)     <-- Fork livelocks, Silent survives
 30%             0/s  (0.0%)  14/s  (8.5%)     <-- Fork livelocks, Silent survives
```

## Analysis

### 10% Byzantine: Fork and Silent Proposer Are Comparable

At 10%, both strategies produce similar impact:

- Fork retains 47.7% of honest throughput; Silent retains 50.9%
- Fork has slightly lower client latency (61ms vs 70ms) because fork proposals trigger faster view advancement
- Timeout rates are similar (~12-15%)

### 20% Byzantine: Fork Causes Complete Livelock, Silent Survives

This is the critical divergence:

- **Fork 20%**: 99.27% timeout rate, 0 throughput -- complete livelock
- **Silent 20%**: 29.51% timeout rate, 37/s throughput -- degraded but functional

**Why fork is worse at 20%+**: The fork attacker proposes blocks on stale QCs (grandparent), which honest replicas receive and process. These invalid proposals:

1. Waste processing time (signature verification, blockchain lookups)
2. May cause replicas to temporarily disagree on the highest QC, desynchronizing their view advancement
3. With 6 out of 30 leaders producing fork proposals, the rate of confusing messages is high enough to prevent honest replicas from forming quorums in time
4. Unlike silent proposer views (which cleanly timeout after 500ms), fork views partially process before failing, leaving replicas in inconsistent states

The silent proposer's "clean failure" (no proposal at all) is paradoxically easier for the protocol to handle than the fork attacker's "noisy failure" (invalid proposals that waste resources).

### 30% Byzantine: Both Strategies Severely Impact the System

- Fork: 0/s (complete livelock, 99% timeout rate)
- Silent: 14/s (near-livelock, 46% timeout rate, 162 client timeouts)

At the fault tolerance boundary (f=9, n=30), both strategies cripple the system, but the silent proposer still allows minimal progress.

## Key Takeaways

1. **Fork attack is MORE destructive than silent proposer at higher percentages**: At 20%+, fork causes complete livelock while silent proposer retains partial throughput. The fork attacker's invalid proposals actively disrupt view synchronization.
2. **Fork attack is slightly LESS destructive at 10%**: At low percentages, fork proposals trigger faster view transitions than waiting for full timeouts, resulting in comparable or slightly better throughput than the silent proposer case.
3. **The "cliff edge" for fork attack is between 10% and 20%**: Throughput drops from 47.7% to 0% -- a binary transition from functional to livelock. The silent proposer degrades more gradually (50.9% -> 23.1% -> 8.5%).
4. **Fork attack is the more dangerous strategy for an adversary**: If an adversary controls 20%+ of replicas, the fork attack is strictly more effective than the silent proposer at denying service. The silent proposer is only more effective if the adversary controls exactly the f_max boundary (30%).
5. **Safety is maintained in all cases**: Despite the fork attacker's attempts to create conflicting chains, no safety violations were detected. The VoteRule and bLock mechanism prevent honest replicas from voting for blocks on the forked chain.
6. **Practical implication**: Systems expecting fork attacks need a lower byzantine threshold for liveness than those only expecting silent proposers. With fork attackers, the practical liveness limit is ~10% of nodes, vs ~20% for silent proposers.

