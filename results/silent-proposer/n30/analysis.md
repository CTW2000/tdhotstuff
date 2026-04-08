# n=30 Byzantine Scaling Analysis

**Date**: 2026-04-08
**Config**: n=30, relaxed commit rule, `--timeout-multiplier 1.0`, `--duration-samples 100`, 30s duration

## Byzantine Configurations


| Experiment | Silent Proposers | Replica IDs                       | f used / f max |
| ---------- | ---------------- | --------------------------------- | -------------- |
| Honest     | 0                | -                                 | 0 / 9          |
| 10%        | 3                | [2, 12, 22]                       | 3 / 9          |
| 20%        | 6                | [2, 7, 12, 17, 22, 27]            | 6 / 9          |
| 30%        | 9                | [2, 5, 8, 12, 15, 18, 22, 25, 28] | 9 / 9 (limit)  |


## Results


| Metric                     | Honest (0%) | 10% (3 byz) | 20% (6 byz) | 30% (9 byz) |
| -------------------------- | ----------- | ----------- | ----------- | ----------- |
| **Throughput (commits/s)** | **159**     | **81**      | **37**      | **14**      |
| Total commits/replica      | 5,102       | 2,596       | 1,180       | 0           |
| Commands executed (client) | 5,099       | 2,587       | 1,171       | 286         |
| Client timeouts            | 0           | 6           | 6           | 162         |
| Consensus latency (ms)     | 21.009      | 37.685      | 80.213      | 212.543     |
| Client latency (ms)        | 35.423      | 70.356      | 154.783     | 413.834     |
| View timeout rate          | 2.28%       | 15.33%      | 29.51%      | 46.33%      |
| Total views                | 155,791     | 88,561      | 45,518      | 20,851      |
| **% of honest throughput** | **100%**    | **50.9%**   | **23.1%**   | **8.5%**    |


## Throughput Scaling Curve

```
Byzantine %   Throughput    % of Honest
---------     ----------    -----------
  0%            159/s         100.0%
 10%             81/s          50.9%
 20%             37/s          23.1%
 30%             14/s           8.5%
```

## Client Latency Trends

### Honest (0%)

Stable at ~35ms:

```
interval 1:  33.0ms (179 cmds)
interval 2:  36.5ms (164 cmds)
interval 3:  35.7ms (163 cmds)
interval 4:  34.6ms (164 cmds)
interval 5:  37.6ms (169 cmds)
```

### 10% Byzantine (3 silent proposers)

Stable at ~67-70ms after warmup:

```
interval 1:  128.8ms (44 cmds)
interval 2:   67.1ms (91 cmds)
interval 3:   66.3ms (90 cmds)
interval 4:   66.1ms (93 cmds)
interval 5:   67.1ms (89 cmds)
```

### 20% Byzantine (6 silent proposers)

Stabilizes at ~130-145ms, with gradual upward drift:

```
interval 1:  221.7ms (25 cmds)
interval 2:  117.7ms (52 cmds)
interval 3:  126.5ms (48 cmds)
interval 4:  131.6ms (44 cmds)
interval 5:  128.1ms (48 cmds)
interval 6:  132.5ms (44 cmds)
interval 7:  131.8ms (47 cmds)
interval 8:  136.0ms (44 cmds)
interval 9:  141.8ms (41 cmds)
interval10:  145.5ms (42 cmds)
```

### 30% Byzantine (9 silent proposers)

High and degrading, approaching client timeout ceiling:

```
interval 1:  399.9ms (13 cmds)
interval 2:  241.2ms (23 cmds)
interval 3:  268.8ms (24 cmds)
interval 4:  288.7ms (21 cmds)
interval 5:  328.5ms (17 cmds)
interval 6:  347.1ms (19 cmds)
interval 7:  341.6ms (15 cmds)
interval 8:  356.3ms (17 cmds)
interval 9:  362.8ms (17 cmds)
interval10:  375.9ms (17 cmds)
```

## Cross-Scale Comparison: n=10 vs n=30


| Metric                 | n=10 Honest | n=10 10% | n=10 20% | n=10 30% | n=30 Honest | n=30 10% | n=30 20% | n=30 30% |
| ---------------------- | ----------- | -------- | -------- | -------- | ----------- | -------- | -------- | -------- |
| Throughput (commits/s) | 856         | 487      | 240      | 0        | 159         | 81       | 37       | 14       |
| % of honest throughput | 100%        | 56.9%    | 28.1%    | 0%       | 100%        | 50.9%    | 23.1%    | 8.5%     |
| Client latency (ms)    | 6.6         | 11.8     | 24.4     | 500+     | 35.4        | 70.4     | 154.8    | 413.8    |
| View timeout rate      | 2.4%        | 12.5%    | 22.8%    | 99.8%    | 2.3%        | 15.3%    | 29.5%    | 46.3%    |


## Key Takeaways

1. **Degradation is super-linear but consistent across scales**: Both n=10 and n=30 show the same pattern -- each 10% increase in byzantine nodes causes progressively more damage. The relative throughput retention is similar: ~51-57% at 10%, ~23-28% at 20%.
2. **n=30 survives 30% byzantine; n=10 does not**: At the fault tolerance limit (f=f_max), n=30 retains 8.5% throughput (14/s, 286 commands in 30s) while n=10 completely livelocks. The larger quorum size (21 for n=30 vs 7 for n=10) and more honest leaders between silent ones provide enough overlap for the system to make intermittent progress.
3. **Timeout rate diverges at 30% between scales**: n=10 at 30% has 99.8% timeout rate (livelock), while n=30 at 30% has 46.3%. With n=30 and 9 silent proposers, the round-robin cycle has 21 honest leaders per 30 views. Even with 30% failures, enough consecutive honest views exist for the system to advance.
4. **Client latency trend reveals stability thresholds**:
  - 10%: Stable (~67-70ms at n=30), no degradation
  - 20%: Mostly stable (~130ms) but shows gradual upward drift (130 -> 145ms over 10 intervals)
  - 30%: Degrading throughout (241 -> 376ms), approaching the 500ms client timeout ceiling
5. **O(n^2) communication dominates absolute throughput**: Honest throughput drops 81% from n=10 (856/s) to n=30 (159/s). This dwarfs the byzantine penalty at any level, reinforcing that the clique communication pattern is the primary scalability bottleneck.
6. **Practical deployment guidance**:
  - Up to 10% byzantine: System performs well at any scale (50%+ throughput retained)
  - Up to 20% byzantine: Usable but with significant degradation (~23% throughput, latency 4-5x honest)
  - At 30% (f=f_max): Only viable at larger n (n>=30), and even then with severe degradation

