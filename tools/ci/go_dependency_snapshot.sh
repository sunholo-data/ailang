#!/bin/bash
# go_dependency_snapshot.sh — build GitHub's dependency-graph snapshot for the root Go module and,
# with --submit, POST it to /repos/{owner}/{repo}/dependency-graph/snapshots.
#
#   tools/ci/go_dependency_snapshot.sh            # print the snapshot JSON (local check, no network
#                                                 # beyond module downloads)
#   tools/ci/go_dependency_snapshot.sh --submit   # CI: POST it; needs GH_TOKEN with contents:write
#
# Replaces actions/go-dependency-submission, which died on every push since it was added with
#   assertion failed: expected no more than one package in cache with namespace+name
# (upstream issue #74, still in v2.0.3). It splits each module path into a purl namespace+name by
# dirname/basename and then asserts that pair is unique; two modules can collide there, so it threw
# before submitting. `continue-on-error: true` hid that, the graph kept serving the last snapshot
# that DID go through (it still listed ollama v0.13.4 after #2206 removed it), and Dependabot's Go
# alerts were computed from that stale list. This script keys everything by full module path, so
# that collision cannot happen.
#
# What is submitted — the same population the action reported:
#   resolved   every module that contributes a package to `go list -deps all` (the build list
#              restricted to modules actually compiled, tests included), replacements applied;
#              modules replaced by a local directory are skipped (no registry package to alert on).
#   relationship  direct unless go.mod marks the requirement `// indirect`.
#   dependencies  edges from `go mod graph`, kept only where both ends are in `resolved` and the
#              parent edge comes from the selected version.
#
# Supersession: the graph keeps only the LATEST snapshot per (job.correlator, detector.name). Both
# are pinned to what the action used — correlator "${job}-${target}" = "dependency-submission-all",
# detector "actions/go-dependency-submission" — so the first submission from this script replaces
# the stale one. They are literals on purpose: renaming the CI job must not fork a new series and
# leave the stale one in force. detector.version/url say this script produced it.
#
# Exit: 0 submitted (or printed), 1 any failure (go, jq, API, or a snapshot the API calls INVALID).
# No silent pass: a broken submission must turn the job red.
set -euo pipefail

mode=print
case "${1:-}" in
  "") ;;
  --submit) mode=submit ;;
  *) echo "usage: $0 [--submit]" >&2; exit 2 ;;
esac

cd "$(git rev-parse --show-toplevel)"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# path <TAB> version <TAB> resolved path <TAB> resolved version <TAB> indirect
go list -deps -f '{{with .Module}}{{if not .Main}}{{.Path}}{{"\t"}}{{.Version}}{{"\t"}}{{if .Replace}}{{.Replace.Path}}{{"\t"}}{{.Replace.Version}}{{else}}{{.Path}}{{"\t"}}{{.Version}}{{end}}{{"\t"}}{{.Indirect}}{{end}}{{end}}' all \
  | sort -u > "$tmp/modules.tsv"
go mod graph > "$tmp/graph.txt"

if [ ! -s "$tmp/modules.tsv" ]; then
  echo "go_dependency_snapshot: go list reported no dependency modules; refusing to submit an empty graph" >&2
  exit 1
fi

sha=${GITHUB_SHA:-$(git rev-parse HEAD)}
ref=${GITHUB_REF:-refs/heads/$(git rev-parse --abbrev-ref HEAD)}
run_id=${GITHUB_RUN_ID:-local}
server=${GITHUB_SERVER_URL:-https://github.com}
repo=${GITHUB_REPOSITORY:-sunholo-data/ailang}

jq -n \
  --rawfile mods "$tmp/modules.tsv" \
  --rawfile graph "$tmp/graph.txt" \
  --arg sha "$sha" --arg ref "$ref" --arg run_id "$run_id" \
  --arg html_url "$server/$repo/actions/runs/$run_id" \
  --arg detector_url "$server/$repo/blob/dev/tools/ci/go_dependency_snapshot.sh" \
  --arg scanned "$(date -u +%Y-%m-%dT%H:%M:%SZ)" '
  def purl($p; $v): "pkg:golang/" + ($p | split("/") | map(@uri) | join("/")) + "@" + ($v | @uri);
  ( $mods | split("\n") | map(select(length > 0) | split("\t")
      | {path: .[0], version: .[1], rpath: .[2], rversion: .[3], indirect: (.[4] == "true")})
    | map(select(.rversion != "")) ) as $m
  | ($m | map({key: .path, value: .}) | from_entries) as $by
  | ( $graph | split("\n") | map(select(length > 0) | split(" ")
        | select(.[0] | contains("@"))
        | (.[0] | split("@")) as [$pp, $pv]
        | (.[1] | split("@")) as [$cp, $cv]
        | select($by[$pp] != null and $by[$pp].version == $pv and $by[$cp] != null and $cp != $pp)
        | {p: $pp, c: $cp})
      | group_by(.p) | map({key: .[0].p, value: (map(.c) | unique)}) | from_entries ) as $edges
  | {
      version: 0,
      sha: $sha,
      ref: $ref,
      job: {correlator: "dependency-submission-all", id: $run_id, html_url: $html_url},
      detector: {name: "actions/go-dependency-submission", version: "ailang-ci-1", url: $detector_url},
      scanned: $scanned,
      manifests: {
        all: {
          name: "all",
          file: {source_location: "go.mod"},
          resolved: ($m | map({key: .path, value: {
            package_url: purl(.rpath; .rversion),
            relationship: (if .indirect then "indirect" else "direct" end),
            dependencies: (($edges[.path] // []) | map($by[.] | purl(.rpath; .rversion)))
          }}) | from_entries)
        }
      }
    }' > "$tmp/snapshot.json"

n=$(jq '.manifests.all.resolved | length' "$tmp/snapshot.json")
direct=$(jq '[.manifests.all.resolved[] | select(.relationship == "direct")] | length' "$tmp/snapshot.json")

if [ "$mode" = print ]; then
  cat "$tmp/snapshot.json"
  echo "go_dependency_snapshot: $n modules ($direct direct)" >&2
  exit 0
fi

echo "go_dependency_snapshot: submitting $n modules ($direct direct) for $ref @ $sha"
attempt=1
while :; do
  if gh api --method POST "repos/$repo/dependency-graph/snapshots" --input "$tmp/snapshot.json" > "$tmp/response.json" 2> "$tmp/err.txt"; then
    break
  fi
  if [ "$attempt" -ge 3 ]; then
    echo "::error title=Dependency snapshot not submitted::POST dependency-graph/snapshots failed after $attempt attempts: $(tr '\n' ' ' < "$tmp/err.txt")"
    exit 1
  fi
  echo "attempt $attempt failed: $(tr '\n' ' ' < "$tmp/err.txt"); retrying in $((attempt * 10))s"
  sleep $((attempt * 10))
  attempt=$((attempt + 1))
done

result=$(jq -r '.result // "unknown"' "$tmp/response.json")
echo "go_dependency_snapshot: API result=$result id=$(jq -r '.id // "?"' "$tmp/response.json") message=$(jq -r '.message // ""' "$tmp/response.json")"
case "$result" in
  SUCCESS|ACCEPTED) ;;
  *) echo "::error title=Dependency snapshot rejected::result=$result: $(jq -c . "$tmp/response.json")"; exit 1 ;;
esac
