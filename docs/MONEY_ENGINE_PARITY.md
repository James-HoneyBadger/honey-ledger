# Shared Money engine compatibility

The shared Go decoder and its tests are mirrored between HB SunRise and Honey Ledger.
The checked-in manifest fingerprints every Go file in the shared decoder tree,
normalizing only line endings and the two module import prefixes. Five Ledger-specific
adapter files are explicitly excluded. New shared files are detected automatically.

## Coordinated changes

1. Make and test the decoder change in both working trees.
2. Run the decoder regression suite in each: `go test -race ./pkg/converters/msmoney/...`.
3. Run `python3 scripts/check-money-engine.py --update` in each checkout.
4. Run `python3 scripts/check-money-engine.py --peer /path/to/other/checkout`.
5. Review both manifests and source changes, then merge the paired PRs.

CI checks the local manifest on every normal CI run. A separate daily or manually
triggered workflow checks both public main branches for divergence. It may flag
a temporary mismatch while a coordinated pair of PRs is being merged; do not
silence a persistent mismatch by regenerating only one manifest.

The fingerprint is a source parity check, not a proof of financial correctness.
Synthetic fixtures, genuine Money files, and balance reconciliation are still
required. Never commit private financial fixtures or use them in public CI.
Set MSISAM_SAMPLE_FILE locally for real-file tests. Unsupported historical formats,
investment data and protected files remain subject to documented limitations.
