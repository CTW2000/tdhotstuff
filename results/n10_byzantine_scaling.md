# n=10 Byzantine Scaling Analysis

**Date**: 2026-04-08
**Config**: n=10, relaxed commit rule, `--timeout-multiplier 1.0`, `--duration-samples 100`, 30s duration

## Byzantine Configurations

| Experiment | Silent Proposers | Replica IDs     | f used / f max |
| ---------- | ---------------- | --------------- | -------------- |
| Honest     | 0                | -               | 0 / 3          |
| 10%        | 1                | [2]             | 1 / 3          |
| 20%        | 2                | [2, 7]          | 2 / 3          |
| 30%        | 3                | [2, 4, 6]       | 3 / 3 (limit)  |

## Results


| Metric                     | Honest    | 10% (1 byz)  | 20% (2 byz)  | 30% (3 byz)  |
| -------------------------- | --------- | ------------ | ------------ | ------------ |
| **Throughput (commits/s)** | **856**   | **487**      | **240**      | **0**        |
| Total commits/replica      | 27,380    | 15,573       | 7,684        | 3            |
| Commands executed (client) | 27,377    | 15,564       | 7,675        | 0            |
| Client timeouts            | 0         | 6            | 6            | 5            |
| Consensus latency (ms)     | 3.925     | 6.285        | 12.210       | 22.195       |
| Client latency (ms)        | 6.570     | 11.754       | 24.406       | 500.422      |
| View timeout rate          | 2.42%     | 12.49%       | 22.79%       | 99.78%       |
| Total views                | 279,722   | 177,371      | 99,021       | 23,291       |
| **% of honest throughput** | **100%**  | **56.9%**    | **28.1%**    | **0%**       |


## Throughput Scaling Curve

```
Byzantine %   Throughput    % of Honest
---------     ----------    -----------
  0%            856/s         100.0%
 10%            487/s          56.9%
 20%            240/s          28.1%
 30%              0/s           0.0%  (livelock at f=f_max)
```
