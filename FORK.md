# xrl integration fork

Baseline: upstream wazero v1.12.0 (`2ab480b5`). Fork PRs target
`xrl/integration`; `main` mirrors upstream. Proposed first release:
`v1.12.0-xrl.0` (not created by this change). Keep this patch independently
reviewable and propose the API, engine guards, and load cleanup upstream;
rebase onto security fixes and drop equivalent patches once upstream supports
the contract. No unrelated dependency or compiler redesign is intended.

## Read-only require-hit cache

Prewarm with `NewCompilationCacheWithDir`, close the runtime and cache, then
open a fresh `NewCompilationCacheWithDirReadOnly` in the consumer. Both the
root and version/OS/architecture directory must exist. No mkdir, entry writes,
or stale-entry deletion occurs through the read-only API. Guest cache misses,
stale entries, and decoding failures return errors before full guest-module
compilation; loaded entries can be reused in memory. `errors.Is` recognizes
`ErrCompilationCacheMiss`, `ErrCompilationCacheStale`,
`ErrCompilationCacheCorrupt`, `ErrCompilationCacheIO`, and
`ErrCompilationCacheUnsupported`. Directory failures are I/O errors (with the
underlying OS error retained). Corrupt means a load/decoding failure, not a
comprehensive integrity or authenticity guarantee.

Host modules remain usable. Shared/host trampolines, listener trampolines, and
bounded per-type cache-hit entry preambles may generate code. This is **not**
a zero-code-generation API. Interpreter guest modules fail with Unsupported,
including auto-selected interpreter fallback; no silent interpretation.
Existing memory-only and read/write constructors retain their behavior.

## Compatibility and security

Cache files are **trusted executable input**. Read-only is a no-mutation/no-
guest-compilation policy, not a parser sandbox, signature check, or protection
against malicious cache producers. Protect the directory and its ancestors
from writes by untrusted users; mount the prepared cache read-only in the
consumer. Do not modify it concurrently. Corrupt-load cleanup releases the
anonymous executable mapping, including stale old-format failures.

Use the same immutable tagged or pseudoversion fork replacement in the
prewarmer and consumer, e.g. a `replace github.com/tetratelabs/wazero =>
github.com/xrl/wazero v1.12.0-xrl.0` after release. Version partitioning uses the
actual replacement version, not the upstream require version. Local unversioned
replacements and tests retain `dev`, which is **not a reproducible deployment
cache identity**. Existing CLI ldflags overrides remain, but must not be used
to relabel unrelated development builds as compatible.

Wasm bytes, compilation configuration (including listeners/termination),
OS/architecture and CPU feature keys must match. CPU features remain part of
the original key: never rename or relabel a cache from another CPU. No shipping
Wasm is changed; the intended consumer's original core.wasm SHA-256 remains
`23d115f4ac7519b48172df3e8615945572dbda7033d51b44c9490fd533ae0f23`.

## Validation and release gates

The fork-only workflow targets `xrl/integration` on `xrl/wazero`, pins action
SHAs and Go **1.26.5**, and runs native Linux AMD64/ARM64 race tests plus 386
interpreter-fallback tests. It uses no private credentials. Local development
uses Go 1.27.1; pinned CI is a separate acceptance gate. Parent owns pushes,
PRs, tags/publication, downstream prewarm/controller validation, artifact
hash checks, and rollout. No production credentials or cluster access is
needed to test this patch.
