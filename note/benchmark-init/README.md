 ● Done. note/benchmark-init/ now contains:

   • main.go — the benchmark (cold vs file-cached compile, interpreter, instantiate-only, full WASI VM setup)
   • go.mod / go.sum — with a replace directive pointing at this repo, so it benchmarks your fork directly

   Verified with go run . from the new directory — same numbers as before (cold 9.6ms, cached 0.7ms, interpreter 1.4ms). Also updated note/wazevo-init-time.md to reference the new location instead of
   /tmp/wazerobench.

   Note: the file cache directory is still hardcoded to /tmp/wazerobench/filecache in main.go — delete it between runs if you want to re-measure a truly cold first compile.
