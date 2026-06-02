# Quality checks (Phase 5)

## Redundancy test

HIG and Material both cover gestures, but from different platform vantage
points. Not redundant — kept both.

## Coverage test

Foundations and patterns each have at least one source. Craft has none in v1.
Flag in synthesis as out-of-scope.

## Disagreement test

HIG and Material disagree on back-navigation and overflow patterns. Both
included; synthesis frames the divergence.

## Framing-gap test

Initial scope was iOS-centric. Material was added late to surface the
cross-platform divergence. Still narrow on craft topics (motion, typography);
revisit before library contribution.
