#!/usr/bin/env bash
set -Eeuo pipefail

if test "$#" -ne 1; then
  echo "usage: development-authority-guard.sh OUTPUT_JSON" >&2
  exit 64
fi

output=$1
repo=${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is required}
subject_sha=${GITHUB_SHA:?GITHUB_SHA is required}
event_name=${GITHUB_EVENT_NAME:?GITHUB_EVENT_NAME is required}
ref=${GITHUB_REF:?GITHUB_REF is required}
event_path=${GITHUB_EVENT_PATH:?GITHUB_EVENT_PATH is required}
test -n "${GH_TOKEN:-}" || {
  echo "GH_TOKEN is required for development authority observations" >&2
  exit 1
}

v020_release=$(gh api "repos/$repo/releases/tags/v0.2.0")
test "$(jq -r '.tag_name' <<<"$v020_release")" = "v0.2.0"
test "$(jq -r '.immutable' <<<"$v020_release")" = true
test "$(jq '.assets | length' <<<"$v020_release")" -eq 3
test "$(jq '[.assets[] | select((.digest // "") | startswith("sha256:"))] | length' <<<"$v020_release")" -eq 3
v020_release_api_identity_digest=$(jq -cS '{id,node_id,tag_name,target_commitish,immutable,assets:(.assets|map({id,node_id,name,size,digest,browser_download_url}))}' <<<"$v020_release" | sha256sum | awk '{print "sha256:"$1}')

current_guard_state=UNKNOWN
current_pr_associated_path=0
current_pr_number=0
current_merge_commit_sha=""
if test "$event_name" = push && test "$ref" = refs/heads/main; then
  associated_pulls=$(gh api "repos/$repo/commits/$subject_sha/pulls")
  merged_pull=$(jq -c --arg sha "$subject_sha" '[.[] | select(.base.ref == "main" and .merged_at != null and .merge_commit_sha == $sha)]' <<<"$associated_pulls")
  if test "$(jq 'length' <<<"$merged_pull")" -ne 1; then
    jq -S -n \
      --arg schema "gooo/self-repair/development-authority-receipt/v1" \
      --arg offending_commit "5dca56d" \
      --arg expected_pr_association "merged pull request to main for the current commit" \
      --arg current_commit "$subject_sha" \
      --arg v020_digest "$v020_release_api_identity_digest" \
      '{schema:$schema,direct_main_push:1,offending_commit:$offending_commit,expected_pr_association:$expected_pr_association,state:"REFUTED",stage:"DEVELOPMENT_PROCESS_AUTHORITY",step:"CHECK_MAIN_COMMIT_PR_ASSOCIATION",reason:"DIRECT_MAIN_PUSHED_BEFORE_PR_ASSOCIATION",unknown_class:"PROCESS_VIOLATION",next_operation:"REQUIRE_MERGED_PR_ASSOCIATION_AND_RECHECK",blocked_by:[$offending_commit],historical_violation_count:1,current_guard_state:"REFUTED",current_commit:$current_commit,current_pr_associated_path:0,current_pr_number:0,current_merge_commit_sha:"",repository_direct_writes_after_guard:0,semantic_artifacts_state:"UNKNOWN",v020_release_tag:"v0.2.0",v020_release_immutable:true,v020_release_api_identity_digest:$v020_digest,v020_release_asset_count:3}' > "$output"
    echo "main commit $subject_sha is not associated with exactly one merged PR" >&2
    exit 1
  fi
  current_guard_state=CLOSED
  current_pr_associated_path=1
  current_pr_number=$(jq -r '.[0].number' <<<"$merged_pull")
  current_merge_commit_sha=$(jq -r '.[0].merge_commit_sha' <<<"$merged_pull")
elif test "$event_name" = pull_request; then
  current_pr_number=$(jq -r '.number // .pull_request.number // 0' "$event_path")
  current_merge_commit_sha=$(jq -r '.pull_request.merge_commit_sha // ""' "$event_path")
else
  echo "development authority guard requires pull_request or push to main" >&2
  exit 1
fi

jq -S -n \
  --arg schema "gooo/self-repair/development-authority-receipt/v1" \
  --arg offending_commit "5dca56d" \
  --arg expected_pr_association "merged pull request to main for the current commit" \
  --arg current_guard_state "$current_guard_state" \
  --arg current_commit "$subject_sha" \
  --arg current_merge_commit_sha "$current_merge_commit_sha" \
  --arg v020_digest "$v020_release_api_identity_digest" \
  --argjson current_pr_number "$current_pr_number" \
  --argjson current_pr_associated_path "$current_pr_associated_path" \
  '{schema:$schema,direct_main_push:1,offending_commit:$offending_commit,expected_pr_association:$expected_pr_association,state:"REFUTED",stage:"DEVELOPMENT_PROCESS_AUTHORITY",step:"CHECK_MAIN_COMMIT_PR_ASSOCIATION",reason:"DIRECT_MAIN_PUSHED_BEFORE_PR_ASSOCIATION",unknown_class:"PROCESS_VIOLATION",next_operation:"REQUIRE_MERGED_PR_ASSOCIATION_AND_RECHECK",blocked_by:[$offending_commit],historical_violation_count:1,current_guard_state:$current_guard_state,current_commit:$current_commit,current_pr_associated_path:$current_pr_associated_path,current_pr_number:$current_pr_number,current_merge_commit_sha:$current_merge_commit_sha,repository_direct_writes_after_guard:0,semantic_artifacts_state:"CLOSED",v020_release_tag:"v0.2.0",v020_release_immutable:true,v020_release_api_identity_digest:$v020_digest,v020_release_asset_count:3}' > "$output"

echo "development authority guard: historical violation=REFUTED current=$current_guard_state pr_path=$current_pr_associated_path"
