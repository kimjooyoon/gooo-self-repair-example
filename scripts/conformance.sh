#!/usr/bin/env bash
set -Eeuo pipefail

if test "$#" -ne 7; then
  echo "usage: conformance.sh BINARY ARTIFACT_DIR SUBJECT_SHA GO_VERSION TEST_JSON TEST_TIME BUILD_TIME" >&2
  exit 64
fi

binary=$1
artifact_dir=$2
subject_sha=$3
go_version=$4
go_test_json=$5
test_time=$6
build_time=$7
repo_root=$(cd "$(dirname "$0")/.." && pwd)
lock="$repo_root/contracts/external-release-lock-v1.json"
work=$(mktemp -d "${RUNNER_TEMP:-/tmp}/gooo-self-repair.XXXXXX")
trap 'rm -rf "$work"' EXIT

before_status=$(git -C "$repo_root" status --porcelain=v1 -z --untracked-files=all | sha256sum | awk '{print $1}')
conformance_start_ns=$(date +%s%N)
mkdir -p "$work/api" "$work/downloads" "$work/evidence"

"$binary" compile \
  --before-source "$repo_root/examples/self-repair/before.gooo" \
  --after-source "$repo_root/examples/self-repair/after.gooo" \
  --contract "$repo_root/contracts/self-repair-lifecycle-denominator-v1.json" \
  --ir "$work/semantic-ir.json" > "$work/compile.stdout"

test -n "${GH_TOKEN:-}" || {
  echo "GH_TOKEN is required for release API observations" >&2
  exit 1
}

download_release_artifact() {
  local id=$1
  local release api_path api_file artifact_file
  local expected_api_digest actual_api_digest
  local tag_ref tag_object_sha tag_object_type target_commit_sha
  local artifact_kind artifact_name artifact_url expected_digest
  local api_asset_digest api_asset_url actual_digest

  release=$(jq -c --arg id "$id" '.releases[] | select(.id == $id)' "$lock")
  api_path=$(jq -r '.release_api_path' <<<"$release")
  api_file="$work/api/$id.json"
  gh api "$api_path" > "$api_file"
  expected_api_digest=$(jq -r '.release_api_identity_digest' <<<"$release")
  actual_api_digest=$(jq -cS '{id,node_id,tag_name,target_commitish,immutable,assets:(.assets|map({id,node_id,name,size,digest,browser_download_url}))}' "$api_file" | sha256sum | awk '{print "sha256:"$1}')
  test "$actual_api_digest" = "$expected_api_digest"
  test "$(jq -r '.tag_name' "$api_file")" = "$(jq -r '.tag' <<<"$release")"
  test "$(jq -r '.immutable' "$api_file")" = true
  test "$(jq -r '.id' "$api_file")" = "$(jq -r '.release_id' <<<"$release")"
  test "$(jq -r '.node_id' "$api_file")" = "$(jq -r '.release_node_id' <<<"$release")"

  tag_ref=$(gh api "repos/$(jq -r '.repository' <<<"$release")/git/ref/tags/$(jq -r '.tag' <<<"$release")")
  tag_object_sha=$(jq -r '.object.sha' <<<"$tag_ref")
  tag_object_type=$(jq -r '.object.type' <<<"$tag_ref")
  test "$tag_object_sha" = "$(jq -r '.tag_object_sha' <<<"$release")"
  if test "$tag_object_type" = tag; then
    target_commit_sha=$(gh api "repos/$(jq -r '.repository' <<<"$release")/git/tags/$tag_object_sha" --jq '.object.sha')
  else
    target_commit_sha=$tag_object_sha
  fi
  test "$target_commit_sha" = "$(jq -r '.target_commit_sha' <<<"$release")"

  artifact_kind=$(jq -r '.artifact.kind' <<<"$release")
  artifact_name=$(jq -r '.artifact.name' <<<"$release")
  artifact_url=$(jq -r '.artifact.url' <<<"$release")
  expected_digest=$(jq -r '.artifact.digest' <<<"$release")
  artifact_file="$work/downloads/$id-$artifact_name"
  if test "$artifact_kind" = release_asset; then
    api_asset_digest=$(jq -r --arg name "$artifact_name" '.assets[] | select(.name == $name) | .digest' "$api_file")
    api_asset_url=$(jq -r --arg name "$artifact_name" '.assets[] | select(.name == $name) | .browser_download_url' "$api_file")
    test "$api_asset_digest" = "$expected_digest"
    test "$api_asset_url" = "$artifact_url"
  else
    test "$artifact_kind" = tag_archive
    test "$(jq '.assets | length' "$api_file")" -eq 0
    test "$artifact_url" = "$(jq -r '.tarball_url' "$api_file")"
  fi
  if test "$artifact_kind" = release_asset; then
    curl -fsSL --retry 3 -H "Authorization: Bearer $GH_TOKEN" -H 'Accept: application/octet-stream' "$artifact_url" -o "$artifact_file"
  else
    curl -fsSL --retry 3 -H "Authorization: Bearer $GH_TOKEN" -H 'Accept: application/vnd.github+json' "$artifact_url" -o "$artifact_file"
  fi
  actual_digest="sha256:$(sha256sum "$artifact_file" | awk '{print $1}')"
  test "$actual_digest" = "$expected_digest"
  jq -cn \
    --arg id "$id" \
    --arg repository "$(jq -r '.repository' <<<"$release")" \
    --arg tag "$(jq -r '.tag' <<<"$release")" \
    --arg release_api_path "$api_path" \
    --arg release_node_id "$(jq -r '.release_node_id' <<<"$release")" \
    --arg tag_object_sha "$tag_object_sha" \
    --arg target_commit_sha "$target_commit_sha" \
    --arg release_api_identity_digest "$actual_api_digest" \
    --arg artifact_kind "$artifact_kind" \
    --arg artifact_name "$artifact_name" \
    --arg artifact_url "$artifact_url" \
    --arg artifact_digest "$expected_digest" \
    --arg artifact_path "release-downloads/$id/$artifact_name" \
    --arg observed_artifact_digest "$actual_digest" \
    --argjson release_id "$(jq '.release_id' <<<"$release")" \
    '{id:$id,repository:$repository,tag:$tag,release_api_path:$release_api_path,release_id:$release_id,release_node_id:$release_node_id,tag_object_sha:$tag_object_sha,target_commit_sha:$target_commit_sha,release_api_identity_digest:$release_api_identity_digest,artifact_kind:$artifact_kind,artifact_name:$artifact_name,artifact_url:$artifact_url,artifact_digest:$artifact_digest,observed_artifact_digest:$observed_artifact_digest,artifact_path:$artifact_path,immutable:true}' \
    >> "$work/evidence/releases.ndjson"
}

: > "$work/evidence/releases.ndjson"
while IFS= read -r id; do
  download_release_artifact "$id"
done < <(jq -r '.releases[].id' "$lock")

extract_archive() {
  local archive=$1 destination=$2
  mkdir -p "$destination"
  tar -xzf "$archive" --strip-components=1 -C "$destination"
}

extract_archive "$work/downloads/improvement-proposer-gooo-improvement-proposer-v0.1.1-evidence.tar.gz" "$work/evidence/proposer"
extract_archive "$work/downloads/improvement-selector-gooo-improvement-selector-v0.1.1-evidence.tar.gz" "$work/evidence/selector"
extract_archive "$work/downloads/proof-kernel-boundary-proof-kernel-boundary-v0.1.0-evidence.tar.gz" "$work/evidence/proof"
extract_archive "$work/downloads/semantic-mutation-lab-gooo-semantic-mutation-lab-v0.1.1-source.tar.gz" "$work/mutation-source"
extract_archive "$work/downloads/test-frontier-gooo-test-frontier-v0.1.1.tar.gz" "$work/frontier-source"
extract_archive "$work/downloads/verification-reuse-gooo-verification-reuse-v0.1.2.tar.gz" "$work/reuse-source"
extract_archive "$work/downloads/semantic-drift-guard-evidence-v0.1.1.tar.gz" "$work/evidence/drift"
extract_archive "$work/downloads/experience-memory-gooo-experience-memory-evidence-v0.1.0.tar.gz" "$work/evidence/memory"

mutation_bin="$work/semantic-mutation-lab"
(cd "$work/mutation-source" && go build -trimpath -o "$mutation_bin" ./cmd/gooo-semantic-mutation-lab)
"$mutation_bin" run \
  --source "$work/mutation-source/examples/semantic-mutation-lab-v1/lab.gooo" \
  --contract "$work/mutation-source/contracts/denominator-v1.json" \
  --out "$work/mutation-run" > "$work/mutation-run.stdout"

frontier_bin="$work/test-frontier"
(cd "$work/frontier-source" && go build -trimpath -o "$frontier_bin" ./cmd/gooo-test-frontier)
"$frontier_bin" conformance \
  --root "$work/frontier-source" \
  --source "$work/frontier-source/examples/test-frontier.gooo" \
  --contract "$work/frontier-source/contracts/test-frontier-denominator-v1.json" \
  --corpus "$work/frontier-source/examples/canonical-corpus.json" \
  --output-dir "$work/frontier-run" > "$work/frontier-run.stdout"

read -r build_seconds build_rss < "$build_time"
read -r test_seconds test_rss < "$test_time"
build_wall_ms=$(awk -v value="$build_seconds" 'BEGIN { printf "%d", (value * 1000) + 0.5 }')
test_wall_ms=$(awk -v value="$test_seconds" 'BEGIN { printf "%d", (value * 1000) + 0.5 }')
peak_rss_kib=$build_rss
if test "$test_rss" -gt "$peak_rss_kib"; then peak_rss_kib=$test_rss; fi
build_result="sha256:$(sha256sum "$build_time" | awk '{print $1}')"
test_result="sha256:$(sha256sum "$go_test_json" | awk '{print $1}')"
jq -S -n --arg operation_id build --arg result_digest "$build_result" --argjson wall_ms "$build_wall_ms" --argjson peak_rss_kib "$build_rss" \
  '{status:"EXECUTED",operation_id:$operation_id,wall_ms:$wall_ms,peak_rss_kib:$peak_rss_kib,clock_domain:"linux/amd64/github.actions.monotonic.v1",result_digest:$result_digest,terminal_result:"PASS"}' > "$work/build-observation.json"
jq -S -n --arg operation_id test --arg result_digest "$test_result" --argjson wall_ms "$test_wall_ms" --argjson peak_rss_kib "$test_rss" \
  '{status:"EXECUTED",operation_id:$operation_id,wall_ms:$wall_ms,peak_rss_kib:$peak_rss_kib,clock_domain:"linux/amd64/github.actions.monotonic.v1",result_digest:$result_digest,terminal_result:"PASS"}' > "$work/test-observation.json"

reuse_bin="$work/verification-reuse"
(cd "$work/reuse-source" && go build -trimpath -o "$reuse_bin" ./cmd/gooo-verification-reuse)
"$reuse_bin" run \
  --source "$work/reuse-source/examples/verification-reuse/main.gooo" \
  --contract "$work/reuse-source/contracts/verification-reuse-denominator-v1.json" \
  --corpus "$work/reuse-source/fixtures/reuse-cases-v1.json" \
  --scenario valid-reuse-plan --tree-root "$work/reuse-source" --out "$work/reuse-run" \
  --build-observation "$work/build-observation.json" --test-observation "$work/test-observation.json" --subject-sha "$subject_sha" > "$work/reuse-run.stdout"

proposer="$work/evidence/proposer/cases/unknown-missing-utility/proposal.json"
selector="$work/evidence/selector/evidence/candidates/unknown-cross-axis/selection.json"
selector_receipt="$work/evidence/selector/evidence/candidates/unknown-cross-axis/receipt.json"
mutation="$work/mutation-run/mutation-report.json"
frontier="$work/frontier-run/unknown-unobserved-execution/plan.json"
reuse="$work/reuse-run/verification-receipt.json"
proof="$work/evidence/proof/evaluation.json"
drift="$work/evidence/drift/conformance/refuted__authority-escalation/comparison-report.json"
drift_positive="$work/evidence/drift/conformance/normal__formatting-comment-order-equivalent/comparison-report.json"
memory="$work/evidence/memory/evaluation.json"

proposer_digest="sha256:$(sha256sum "$proposer" | awk '{print $1}')"
mutation_digest="sha256:$(sha256sum "$mutation" | awk '{print $1}')"
selector_digest="sha256:$(sha256sum "$selector" | awk '{print $1}')"
frontier_digest="sha256:$(sha256sum "$frontier" | awk '{print $1}')"
reuse_digest="sha256:$(sha256sum "$reuse" | awk '{print $1}')"
proof_digest="sha256:$(sha256sum "$proof" | awk '{print $1}')"
drift_digest="sha256:$(sha256sum "$drift_positive" | awk '{print $1}')"
drift_negative_digest="sha256:$(sha256sum "$drift" | awk '{print $1}')"
memory_digest="sha256:$(sha256sum "$memory" | awk '{print $1}')"
before_digest="sha256:$(sha256sum "$repo_root/examples/self-repair/before.gooo" | awk '{print $1}')"

test_total=$(jq -s '[.[] | select(.Test != null and (.Action == "run" or .Action == "skip"))] | length' "$go_test_json")
test_executed=$(jq -s '[.[] | select(.Test != null and (.Action == "pass" or .Action == "fail"))] | length' "$go_test_json")
test_reused=$(jq -s '[.[] | select(.Test != null and .Action == "output" and ((.Output // "") | contains("(cached)")))] | length' "$go_test_json")
test_skipped=$(jq -s '[.[] | select(.Test != null and .Action == "skip")] | length' "$go_test_json")
test_not_observed=$((test_total - test_executed - test_reused - test_skipped))
if test "$test_not_observed" -lt 0; then test_not_observed=0; fi

count_lines() {
  local total=0 file
  while IFS= read -r -d '' file; do
    total=$((total + $(awk 'END { print NR + 0 }' "$file")))
  done
  echo "$total"
}
go_file_list="$work/go-files"
gooo_file_list="$work/gooo-files"
find "$repo_root" -type f -name '*.go' ! -path "$repo_root/.git/*" -print0 | sort -z > "$go_file_list"
find "$repo_root" -type f -name '*.gooo' ! -path "$repo_root/.git/*" -print0 | sort -z > "$gooo_file_list"
go_files=$(find "$repo_root" -type f -name '*.go' ! -path "$repo_root/.git/*" -print | wc -l | tr -d ' ')
gooo_files=$(find "$repo_root" -type f -name '*.gooo' ! -path "$repo_root/.git/*" -print | wc -l | tr -d ' ')
go_lines=$(count_lines < "$go_file_list")
gooo_lines=$(count_lines < "$gooo_file_list")

selector_pair=$(jq -c '.entries[0].pair' "$selector_receipt")
frontier_unknown=$(jq -c '[.activities[] | select(.state == "UNKNOWN" and .unknown != null) | .unknown][0]' "$frontier")
mutation_selected=$(jq -c '.mutants[] | select(.mutant_id == "MUT-02-FIXED-POINT")' "$mutation")
oracle_positive=$(jq -c '.cases[] | select(.case_id == "normal-boundary")' "$proof")
oracle_negative=$(jq -c '.cases[] | select(.case_id == "refuted-verdict-override")' "$proof")
selector_unknown=$(jq -c '.unknown[0]' "$selector")
proposer_unknown=$(jq -c '.unknowns[0]' "$proposer")
memory_metrics=$(jq -c '.metrics' "$memory")
memory_before=$(jq -c '.baseline' "$memory")
memory_after=$(jq -c '.after' "$memory")
frontier_tests=$(jq -c '.execution_counts' "$frontier")
reuse_tests=$(jq -c '{total:(.operations|length),executed:([.operations[]|select(.status=="EXECUTED")]|length),reused:([.operations[]|select(.status=="REUSED")]|length),skipped:([.operations[]|select(.status=="SKIPPED")]|length),not_observed:([.operations[]|select(.status=="NOT_OBSERVED")]|length)}' "$reuse")
conformance_end_ns=$(date +%s%N)
conformance_wall_ms=$(( (conformance_end_ns - conformance_start_ns) / 1000000 ))
if test "$conformance_wall_ms" -lt 1; then conformance_wall_ms=1; fi

jq -S -n \
  --argjson releases "$(jq -s . "$work/evidence/releases.ndjson")" \
  --arg before_digest "$before_digest" --arg proposer_digest "$proposer_digest" --arg mutation_digest "$mutation_digest" --arg selector_digest "$selector_digest" \
  --arg frontier_digest "$frontier_digest" --arg reuse_digest "$reuse_digest" --arg proof_digest "$proof_digest" --arg drift_digest "$drift_digest" --arg drift_negative_digest "$drift_negative_digest" --arg memory_digest "$memory_digest" \
  --argjson state_counts '{"CLOSED":3,"UNKNOWN":3,"REFUTED":3}' --argjson proposer_unknown "$proposer_unknown" --argjson selector_unknown "$selector_unknown" --argjson frontier_unknown "$frontier_unknown" \
  --argjson mutation_selected "$mutation_selected" --argjson oracle_positive "$oracle_positive" --argjson oracle_negative "$oracle_negative" --argjson memory_metrics "$memory_metrics" --argjson memory_before "$memory_before" --argjson memory_after "$memory_after" \
  --argjson selector_pair "$selector_pair" --argjson frontier_tests "$frontier_tests" --argjson reuse_tests "$reuse_tests" \
  --argjson mutation_report "$(cat "$mutation")" --argjson selector "$(cat "$selector")" --argjson frontier "$(cat "$frontier")" --argjson reuse "$(cat "$reuse")" --argjson drift "$(cat "$drift_positive")" --argjson drift_negative "$(cat "$drift")" --argjson memory "$(cat "$memory")" --argjson proposer "$(cat "$proposer")" --argjson selector_receipt "$(cat "$selector_receipt")" \
  --argjson test_total "$test_total" --argjson test_executed "$test_executed" --argjson test_reused "$test_reused" --argjson test_skipped "$test_skipped" --argjson test_not_observed "$test_not_observed" \
  --argjson build_wall_ms "$build_wall_ms" --argjson test_wall_ms "$test_wall_ms" --argjson conformance_wall_ms "$conformance_wall_ms" --argjson peak_rss_kib "$peak_rss_kib" --argjson go_files "$go_files" --argjson go_lines "$go_lines" --argjson gooo_files "$gooo_files" --argjson gooo_lines "$gooo_lines" \
  '{
    schema:"gooo/self-repair/integration-input/v1", denominator_cells:12, state_counts:$state_counts, precedence:["REFUTED","UNKNOWN","CLOSED"], releases:$releases,
    observation:{decision:"UNKNOWN_TOP_LEVEL",observed_state:"CLOSED",expected_state:"UNKNOWN",source_line:3,artifact_digest:$before_digest},
    proposal:{state:$proposer.state,candidate_count:$proposer.candidate_count,candidate_id:$proposer.candidates[0].candidate_id,unknown:$proposer_unknown,artifact_digest:$proposer_digest},
    mutation:{generated:$mutation_report.summary.generated,attempted:$mutation_report.summary.attempted,killed:$mutation_report.summary.killed,unknown:$mutation_report.summary.unknown,refuted:$mutation_report.summary.refuted,selected_mutant_id:$mutation_selected.mutant_id,selected_mutant_state:$mutation_selected.state,artifact_digest:$mutation_digest},
    selection:{state:$selector.state,decision:$selector.decision,selected_id:"alpha",unknown:$selector_unknown,artifact_digest:$selector_digest},
    frontier:{state:$frontier.state,case_id:$frontier.case_id,reason:$frontier.decision_reason,tests:$frontier_tests,unknown:$frontier_unknown,artifact_digest:$frontier_digest},
    reuse:{decision:$reuse.decision,plan_status:$reuse.reuse.plan_status,authorized:$reuse.reuse.authorized,actual_reused:$reuse.reuse.actual_reused,tests:$reuse_tests,artifact_digest:$reuse_digest},
    oracle:{state:$oracle_positive.status,case_id:$oracle_positive.case_id,decision:$oracle_positive.decision,reason:$oracle_positive.kernel.reason,artifact_digest:$proof_digest},
    oracle_negative:{state:$oracle_negative.status,case_id:$oracle_negative.case_id,decision:$oracle_negative.decision,reason:$oracle_negative.kernel.reason,artifact_digest:$proof_digest},
    drift:{decision:$drift.decision,reason:$drift.reason,artifact_digest:$drift_digest}, drift_negative:{decision:$drift_negative.decision,reason:$drift_negative.reason,artifact_digest:$drift_negative_digest},
    experience:{before_state:$memory_before.state,after_state:$memory_after.state,selected_candidate_id:$memory_after.selected_candidate_id,known_recurrences_before:$memory_metrics.known_refuted_recurrences_before,known_recurrences_after:$memory_metrics.known_refuted_recurrences_after,avoided_refuted_candidates:$memory_metrics.avoided_refuted_candidates,replay_comparisons:$memory_metrics.replay_comparisons,replay_mismatches:$memory_metrics.replay_mismatches,artifact_digest:$memory_digest},
    utility:{before:$selector_pair.before,after:$selector_pair.after,source_digest:$selector_receipt.input_digest,decision:"UNKNOWN",decision_reason:"RESOURCE_AXES_CROSS"},
    metrics:{before_utility:$selector_pair.before,after_utility:$selector_pair.after,build_wall_ms:$build_wall_ms,test_wall_ms:$test_wall_ms,conformance_wall_ms:$conformance_wall_ms,peak_rss_kib:$peak_rss_kib,tests:{total:$test_total,executed:$test_executed,reused:$test_reused,skipped:$test_skipped,not_observed:$test_not_observed},go_files:$go_files,go_lines:$go_lines,gooo_files:$gooo_files,gooo_lines:$gooo_lines,repository_writes:0}
  }' > "$work/evidence/integration-input.json"

rm -rf "$artifact_dir"
mkdir -p "$artifact_dir"
"$binary" integrate \
  --before-source "$repo_root/examples/self-repair/before.gooo" --after-source "$repo_root/examples/self-repair/after.gooo" \
  --contract "$repo_root/contracts/self-repair-lifecycle-denominator-v1.json" --ir "$work/semantic-ir.json" \
  --external-inputs "$work/evidence/integration-input.json" --artifact-dir "$artifact_dir" --subject-sha "$subject_sha" --go-version "$go_version"

test "$(find "$artifact_dir" -maxdepth 1 -type f | wc -l | tr -d ' ')" -eq 11
test "$(wc -l < "$artifact_dir/claims.ndjson" | tr -d ' ')" -eq 9
jq -e '
  .denominator.total == 12 and .denominator.exact == true and .denominator.activity_mapping == "1:1" and .denominator.released_gooo_activities == 12 and
  .claim_denominator == {total:9,exact:true} and
  .state_counts == {CLOSED:3,UNKNOWN:3,REFUTED:3} and .precedence == ["REFUTED","UNKNOWN","CLOSED"] and .metrics.repository_writes == 0 and
  .metrics.tests.total == (.metrics.tests.executed + .metrics.tests.reused + .metrics.tests.skipped + .metrics.tests.not_observed) and
  (.metrics.build_wall_ms|type) == "number" and (.metrics.test_wall_ms|type) == "number" and (.metrics.conformance_wall_ms|type) == "number" and
  (.metrics.peak_rss_kib|type) == "number" and (.metrics.go_files|type) == "number" and (.metrics.go_lines|type) == "number" and (.metrics.gooo_files|type) == "number" and (.metrics.gooo_lines|type) == "number" and
  (.external_releases|length) == 8 and .external_utility.state == "UNKNOWN"
' "$artifact_dir/repair-manifest.json" >/dev/null
jq -e -s '
  length == 9 and (map(.state) | sort | join(",")) == "CLOSED,CLOSED,CLOSED,REFUTED,REFUTED,REFUTED,UNKNOWN,UNKNOWN,UNKNOWN" and
  all(.[] | if .state == "UNKNOWN" then .unknown.stage != "" and .unknown.step != "" and .unknown.reason != "" and .unknown.unknown_class != "" and .unknown.next_operation != "" and (.unknown.blocked_by|type) == "array" else true end)
' "$artifact_dir/claims.ndjson" >/dev/null
jq -e '.decision == "CONFORMANCE_CLOSED" and .selected_candidate.state == "CLOSED" and .claims == {CLOSED:3,UNKNOWN:3,REFUTED:3} and .external_utility.state == "UNKNOWN" and (.lifecycle|length) == 10' "$artifact_dir/cycle-receipt.json" >/dev/null
if grep -R -n -E -i 'percentage|percent|aggregate_score|"score"|average' "$artifact_dir"; then
  echo "forbidden aggregate or percentage output" >&2
  exit 1
fi

after_status=$(git -C "$repo_root" status --porcelain=v1 -z --untracked-files=all | sha256sum | awk '{print $1}')
test "$before_status" = "$after_status"
echo "self-repair integration conformance: PASS (denominator=12 closed=3 unknown=3 refuted=3 releases=8 repository_writes=0)"
