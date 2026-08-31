#!/usr/bin/env bash
set -euo pipefail

binary=$1
artifact_dir=$2
subject_sha=$3
go_version=$4
repo_root=$(cd "$(dirname "$0")/.." && pwd)
temp_root=${RUNNER_TEMP:-$(mktemp -d)}
ir_path="$temp_root/self-repair.semantic-ir.json"
peak_path="$temp_root/self-repair.peak-rss-kib"

count_lines() {
  local total=0
  local path
  while IFS= read -r path; do
    total=$((total + $(wc -l < "$path")))
  done
  echo "$total"
}

file_list=$(mktemp)
find "$repo_root" -type f -not -path "$repo_root/.git/*" -not -path "$repo_root/.git" -not -path "$repo_root/README.md" -print > "$file_list"
files=$(wc -l < "$file_list" | tr -d ' ')
directories=$(find "$repo_root" -type d -not -path "$repo_root/.git/*" -not -path "$repo_root/.git" -print | wc -l | tr -d ' ')
physical_lines=$(count_lines < "$file_list")
go_file_list=$(mktemp)
find "$repo_root" -type f -name '*.go' -not -path "$repo_root/.git/*" -print | sort > "$go_file_list"
go_files=$(wc -l < "$go_file_list" | tr -d ' ')
go_lines=$(count_lines < "$go_file_list")
gooo_file_list=$(mktemp)
find "$repo_root" -type f -name '*.gooo' -not -path "$repo_root/.git/*" -print | sort > "$gooo_file_list"
gooo_files=$(wc -l < "$gooo_file_list" | tr -d ' ')
gooo_lines=$(count_lines < "$gooo_file_list")

mkdir -p "$artifact_dir"
"$binary" compile \
  --before-source "$repo_root/examples/self-repair/before.gooo" \
  --after-source "$repo_root/examples/self-repair/after.gooo" \
  --contract "$repo_root/contracts/self-repair-lifecycle-denominator-v1.json" \
  --ir "$ir_path"

evaluate_start_ns=$(date +%s%N)
/usr/bin/time -f '%M' -o "$peak_path" "$binary" evaluate \
  --before-source "$repo_root/examples/self-repair/before.gooo" \
  --after-source "$repo_root/examples/self-repair/after.gooo" \
  --contract "$repo_root/contracts/self-repair-lifecycle-denominator-v1.json" \
  --ir "$ir_path" \
  --cases "$repo_root/fixtures/scenarios.json" \
  --parent-fixture "$repo_root/fixtures/proof-kernel/parent-evaluator.json" \
  --proof-fixture "$repo_root/fixtures/proof-kernel/proof-kernel.json" \
  --generated-go "$repo_root/generated/evaluator.go" \
  --evaluator "$repo_root/scripts/conformance.sh" \
  --artifact-dir "$artifact_dir" \
  --subject-sha "$subject_sha" \
  --go-version "$go_version" \
  --peak-rss-kib 0 \
  --wall-ms 0 \
  --directories "$directories" \
  --files "$files" \
  --physical-lines "$physical_lines" \
  --go-files "$go_files" \
  --go-lines "$go_lines" \
  --gooo-files "$gooo_files" \
  --gooo-lines "$gooo_lines" \
  --repository-writes 0 \
  --local-test-executions 0 \
  --cross-project-required-gates 0
evaluate_end_ns=$(date +%s%N)
wall_ms=$(((evaluate_end_ns - evaluate_start_ns) / 1000000))
peak_rss_kib=$(tr -d '[:space:]' < "$peak_path")

manifest_tmp=$(mktemp)
jq --argjson wall "$wall_ms" --argjson peak "$peak_rss_kib" \
  '.runtime.wall_ms = $wall | .runtime.peak_rss_kib = $peak' \
  "$artifact_dir/repair-manifest.json" > "$manifest_tmp"
mv "$manifest_tmp" "$artifact_dir/repair-manifest.json"

test "$(find "$artifact_dir" -maxdepth 1 -type f | wc -l | tr -d ' ')" -eq 8
test "$(wc -l < "$artifact_dir/claims.ndjson" | tr -d ' ')" -eq 12
test "$(wc -l < "$artifact_dir/counterexamples.ndjson" | tr -d ' ')" -eq 3
jq -e '
  .contracts.denominator_cells == 12 and
  .contracts.released_gooo_activities == 12 and
  .contracts.executable_cases == 12 and
  .contracts.activity_mapping == "1:1" and
  .case_states.CLOSED == 9 and
  .case_states.REFUTED == 3 and
  .case_states.UNKNOWN == 0 and
  .case_kinds.BEFORE_NORMAL == 2 and
  .case_kinds.BEFORE_COUNTEREXAMPLE == 3 and
  .case_kinds.AFTER_NORMAL == 3 and
  .case_kinds.AFTER_DEFENSE == 3 and
  .case_kinds.REPLAY == 1 and
  .authority.repository_writes == 0 and
  .authority.local_test_executions == 0 and
  .authority.cross_project_required_gates == 0 and
  .inventory.root_readme_excluded == true and
  (.runtime.peak_rss_kib | type) == "number" and
  (.runtime.wall_ms | type) == "number"
' "$artifact_dir/repair-manifest.json" >/dev/null
jq -e -s 'length == 12 and all(.[] | select(.kind == "COUNTEREXAMPLE"); .expected_refutation_receipt == true and .final_state == "REFUTED" and .before_outcome == "CLOSED" and (.lifecycle | map(.state) | join(">") == "BEFORE_REFUTED>CANDIDATE>INDEPENDENT_EVALUATION>AUTHORIZED_ADOPTION>AFTER_CLOSED"))' "$artifact_dir/claims.ndjson" >/dev/null
jq -e -s 'length == 3 and all(.[]; .kind == "DEFENSE" and .after_outcome == "FAIL_CLOSED")' <(jq -c 'select(.kind == "DEFENSE")' "$artifact_dir/claims.ndjson") >/dev/null
jq -e -s 'length == 3 and (.[0].previous_record_digest == "") and all(.[]; .record_type == "BEFORE_COUNTEREXAMPLE")' "$artifact_dir/counterexamples.ndjson" >/dev/null
if grep -R -n -E -i 'percentage|percent|aggregate_score|"score"' "$artifact_dir"; then
  echo "forbidden aggregate or percentage output" >&2
  exit 1
fi
if jq -e '.provenance.before_source_digest == .provenance.after_source_digest or .provenance.semantic_ir_digest == .provenance.generated_go_digest or .provenance.parent_evaluator_fixture_digest == .provenance.proof_kernel_fixture_digest' "$artifact_dir/repair-manifest.json" >/dev/null; then
  echo "digest chain is not independently bound" >&2
  exit 1
fi

if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
  cat "$artifact_dir/self-repair-report.md" >> "$GITHUB_STEP_SUMMARY"
fi

echo "conformance passed: artifacts=8 cases=12 closed=9 refuted=3 unknown=0"
