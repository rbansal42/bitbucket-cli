# Scripting and Automation Guide

This guide covers using `bb` in scripts, CI/CD pipelines, and automation workflows.

## Table of Contents

- [Using bb in Shell Scripts](#using-bb-in-shell-scripts)
- [JSON Output Mode](#json-output-mode)
- [Raw API Access](#raw-api-access)
- [Common Automation Patterns](#common-automation-patterns)
- [Exit Codes and Error Handling](#exit-codes-and-error-handling)
- [Working with jq](#working-with-jq)
- [Non-Interactive Mode](#non-interactive-mode)
- [Example Scripts](#example-scripts)
- [CI/CD Integration](#cicd-integration)

---

## Using bb in Shell Scripts

The `bb` CLI is designed to work well in scripts and automation. Here are the key principles:

### Basic Script Structure

```bash
#!/bin/bash
set -euo pipefail

# Authenticate using environment variable
export BB_TOKEN="your-access token"

# Run bb commands
bb pr list --json | jq '.[] | .title'
```

### Authentication in Scripts

For scripted use, set authentication via environment variables:

```bash
# Option 1: BB_TOKEN (preferred)
export BB_TOKEN="your-bitbucket-access token"

# Option 2: BITBUCKET_TOKEN (alternative)
export BITBUCKET_TOKEN="your-bitbucket-access token"
```

Create an access token in Bitbucket:
1. Go to Personal Settings > access tokens
2. Create a new access token with required permissions
3. Store it securely (environment variable, secrets manager, etc.)

### Specifying Repository Context

Always specify the repository explicitly in scripts to avoid dependency on git context:

```bash
# Use -R or --repo flag
bb pr list -R myworkspace/myrepo --json

# Or use the global flag
bb --repo myworkspace/myrepo pr list --json
```

---

## JSON Output Mode

Most `bb` commands support `--json` flag for machine-readable output:

```bash
# List PRs as JSON
bb pr list --json

# List issues as JSON
bb issue list --json

# List pipelines as JSON
bb pipeline list --json
```

### JSON Output Examples

**Pull Requests:**
```bash
bb pr list --json
```
```json
[
  {
    "id": 42,
    "title": "Add new feature",
    "state": "OPEN",
    "source": {
      "branch": {
        "name": "feature/new-thing"
      }
    },
    "destination": {
      "branch": {
        "name": "main"
      }
    },
    "author": {
      "display_name": "John Doe"
    }
  }
]
```

**Issues:**
```bash
bb issue list --json
```
```json
[
  {
    "id": 1,
    "title": "Bug in login flow",
    "state": "open",
    "kind": "bug",
    "priority": "critical",
    "assignee": "johndoe"
  }
]
```

**Pipelines:**
```bash
bb pipeline list --json
```
```json
[
  {
    "build_number": 123,
    "state": "COMPLETED",
    "result": "SUCCESSFUL",
    "branch": "main",
    "commit": "abc1234",
    "duration": 180
  }
]
```

---

## Raw API Access

Use `bb api` for direct access to any Bitbucket API endpoint:

### Basic Usage

```bash
# GET request (default)
bb api /user

# GET with full endpoint
bb api /repositories/myworkspace/myrepo

# Specify HTTP method
bb api /repositories/myworkspace/myrepo/issues --method POST \
  --json title="New issue" \
  --json priority="major"
```

### Request Options

```bash
# Add custom headers
bb api /user --header "Accept: application/json"

# POST with JSON body
bb api /repositories/workspace/repo/issues \
  --method POST \
  --json title="Bug report" \
  --json content.raw="Description here" \
  --json priority="major"

# Read request body from file
bb api /repositories/workspace/repo/src/main/config.json \
  --method PUT \
  --input config.json

# Read from stdin
echo '{"title": "Test"}' | bb api /repositories/workspace/repo/issues \
  --method POST \
  --input -
```

### Pagination

```bash
# Automatically fetch all pages
bb api /repositories/myworkspace --paginate

# This returns all results combined into a single JSON array
```

### Response Options

```bash
# Include response headers
bb api /user --include

# Silent mode (no output, useful for checking status codes)
bb api /user --silent
```

### API Examples

```bash
# Get repository details
bb api /repositories/myworkspace/myrepo

# List workspace members
bb api /workspaces/myworkspace/members

# Get pipeline configuration
bb api /repositories/myworkspace/myrepo/pipelines_config

# Create a branch restriction
bb api /repositories/myworkspace/myrepo/branch-restrictions \
  --method POST \
  --json kind="push" \
  --json pattern="main"

# Add a webhook
bb api /repositories/myworkspace/myrepo/hooks \
  --method POST \
  --json description="CI webhook" \
  --json url="https://ci.example.com/webhook" \
  --json active=true \
  --json events='["repo:push", "pullrequest:created"]'
```

---

## Common Automation Patterns

### Auto-Creating PRs in CI/CD

Create a PR automatically after pushing a feature branch:

```bash
#!/bin/bash
set -euo pipefail

BRANCH=$(git rev-parse --abbrev-ref HEAD)
BASE_BRANCH="${BASE_BRANCH:-main}"
WORKSPACE="myworkspace"
REPO="myrepo"

# Skip if on main/master
if [[ "$BRANCH" == "main" || "$BRANCH" == "master" ]]; then
  echo "Skipping PR creation on main branch"
  exit 0
fi

# Check if PR already exists
EXISTING_PR=$(bb pr list -R "$WORKSPACE/$REPO" --json | \
  jq -r --arg branch "$BRANCH" '.[] | select(.source.branch.name == $branch) | .id')

if [[ -n "$EXISTING_PR" ]]; then
  echo "PR #$EXISTING_PR already exists for branch $BRANCH"
  exit 0
fi

# Create PR with auto-filled content from commits
bb pr create -R "$WORKSPACE/$REPO" \
  --fill \
  --base "$BASE_BRANCH" \
  --head "$BRANCH"
```

### Batch Operations on Issues

Close all issues with a specific label:

```bash
#!/bin/bash
set -euo pipefail

WORKSPACE="myworkspace"
REPO="myrepo"

# Get all open issues of a certain kind
ISSUE_IDS=$(bb issue list -R "$WORKSPACE/$REPO" --state open --kind bug --json | \
  jq -r '.[].id')

for id in $ISSUE_IDS; do
  echo "Closing issue #$id"
  bb issue close -R "$WORKSPACE/$REPO" "$id"
done
```

Bulk update issue priority:

```bash
#!/bin/bash
set -euo pipefail

WORKSPACE="myworkspace"
REPO="myrepo"

# Get all trivial priority issues
bb issue list -R "$WORKSPACE/$REPO" --priority trivial --json | \
  jq -r '.[].id' | while read -r id; do
    echo "Updating issue #$id to minor priority"
    bb issue edit -R "$WORKSPACE/$REPO" "$id" --priority minor
done
```

### Pipeline Triggering from Scripts

Trigger and monitor pipelines:

```bash
#!/bin/bash
set -euo pipefail

WORKSPACE="myworkspace"
REPO="myrepo"
BRANCH="${1:-main}"

# Trigger pipeline
echo "Triggering pipeline on branch $BRANCH..."
bb pipeline run -R "$WORKSPACE/$REPO" --branch "$BRANCH"

# Wait for pipeline to complete (with timeout)
TIMEOUT=1800  # 30 minutes
ELAPSED=0
INTERVAL=30

while [[ $ELAPSED -lt $TIMEOUT ]]; do
  # Get latest pipeline status
  STATUS=$(bb pipeline list -R "$WORKSPACE/$REPO" --branch "$BRANCH" --limit 1 --json | \
    jq -r '.[0].state')
  RESULT=$(bb pipeline list -R "$WORKSPACE/$REPO" --branch "$BRANCH" --limit 1 --json | \
    jq -r '.[0].result // empty')
  
  echo "Pipeline status: $STATUS ${RESULT:+($RESULT)}"
  
  if [[ "$STATUS" == "COMPLETED" ]]; then
    if [[ "$RESULT" == "SUCCESSFUL" ]]; then
      echo "Pipeline completed successfully!"
      exit 0
    else
      echo "Pipeline failed with result: $RESULT"
      exit 1
    fi
  fi
  
  sleep $INTERVAL
  ELAPSED=$((ELAPSED + INTERVAL))
done

echo "Pipeline timed out after ${TIMEOUT}s"
exit 1
```

Run custom pipeline with variables:

```bash
#!/bin/bash
# Trigger a deployment pipeline

WORKSPACE="myworkspace"
REPO="myrepo"
ENVIRONMENT="${1:-staging}"

bb pipeline run -R "$WORKSPACE/$REPO" \
  --branch main \
  --custom "deploy-$ENVIRONMENT"
```

---

## Exit Codes and Error Handling

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error (command failed) |

### Handling Errors in Scripts

```bash
#!/bin/bash
set -euo pipefail

# Method 1: Check exit code explicitly
if ! bb pr create --title "Test" --body "Test body"; then
  echo "Failed to create PR"
  exit 1
fi

# Method 2: Capture output and check
OUTPUT=$(bb pr list --json 2>&1) || {
  echo "Failed to list PRs: $OUTPUT"
  exit 1
}

# Method 3: Use trap for cleanup
cleanup() {
  echo "Script failed, cleaning up..."
}
trap cleanup ERR

bb pr create --title "Test"
```

### Checking for Empty Results

```bash
#!/bin/bash

# Check if any PRs exist
PR_COUNT=$(bb pr list --json | jq 'length')

if [[ "$PR_COUNT" -eq 0 ]]; then
  echo "No open PRs found"
  exit 0
fi

echo "Found $PR_COUNT open PRs"
```

### Validating API Responses

```bash
#!/bin/bash

# Check if API call succeeded and returned expected data
RESPONSE=$(bb api /user 2>&1)
EXIT_CODE=$?

if [[ $EXIT_CODE -ne 0 ]]; then
  echo "API call failed: $RESPONSE"
  exit 1
fi

USERNAME=$(echo "$RESPONSE" | jq -r '.username // empty')
if [[ -z "$USERNAME" ]]; then
  echo "Unexpected API response format"
  exit 1
fi

echo "Logged in as: $USERNAME"
```

---

## Working with jq

### Common jq Patterns

**Extract specific fields:**
```bash
# Get PR titles
bb pr list --json | jq -r '.[].title'

# Get PR IDs and titles
bb pr list --json | jq -r '.[] | "\(.id): \(.title)"'
```

**Filter results:**
```bash
# PRs by specific author
bb pr list --json | jq '[.[] | select(.author.display_name == "John Doe")]'

# Issues with high priority
bb issue list --json | jq '[.[] | select(.priority == "critical" or .priority == "blocker")]'

# Failed pipelines
bb pipeline list --json | jq '[.[] | select(.result == "FAILED")]'
```

**Transform output:**
```bash
# Create CSV from PRs
bb pr list --json | jq -r '.[] | [.id, .title, .state] | @csv'

# Create markdown list
bb pr list --json | jq -r '.[] | "- [\(.title)](\(.url))"'
```

**Aggregate data:**
```bash
# Count PRs by state
bb pr list --json | jq 'group_by(.state) | map({state: .[0].state, count: length})'

# Sum pipeline durations
bb pipeline list --json | jq '[.[].duration] | add'
```

**Complex transformations:**
```bash
# Get PR summary with reviewers
bb pr list --json | jq '
  .[] | {
    id,
    title,
    branch: .source.branch.name,
    author: .author.display_name,
    created: .created_on
  }
'
```

---

## Non-Interactive Mode

When running in non-interactive environments (CI/CD, cron jobs), ensure all inputs are provided via flags:

### PR Creation

```bash
# Fully non-interactive PR creation
bb pr create \
  --title "Automated PR: Update dependencies" \
  --body "This PR was created automatically by CI." \
  --base main \
  --head feature/deps-update \
  --repo myworkspace/myrepo
```

### Issue Creation

```bash
# Fully non-interactive issue creation
bb issue create \
  --title "Automated: Performance regression detected" \
  --body "Performance dropped by 15% in latest build." \
  --kind bug \
  --priority critical \
  --repo myworkspace/myrepo
```

### Detecting Non-Interactive Mode

The `bb` CLI automatically detects when stdin is not a TTY and will error if required input is missing:

```bash
# This will fail in non-interactive mode without --title
echo "" | bb pr create
# Error: --title flag is required when not running interactively
```

### Environment Variables

Disable color output in scripts:
```bash
export NO_COLOR=1
# or
export BB_NO_COLOR=1
```

---

## Example Scripts

### Daily PR Summary Report

```bash
#!/bin/bash
# Generate a daily summary of open PRs

set -euo pipefail

WORKSPACE="myworkspace"
REPO="myrepo"
OUTPUT_FILE="pr-summary-$(date +%Y%m%d).md"

cat > "$OUTPUT_FILE" << EOF
# Pull Request Summary - $(date +%Y-%m-%d)

## Open PRs

EOF

bb pr list -R "$WORKSPACE/$REPO" --state OPEN --json | jq -r '
  .[] | "### PR #\(.id): \(.title)\n- **Author:** \(.author.display_name)\n- **Branch:** \(.source.branch.name) -> \(.destination.branch.name)\n- **Created:** \(.created_on)\n"
' >> "$OUTPUT_FILE"

echo "Summary written to $OUTPUT_FILE"
```

### Stale PR Reminder

```bash
#!/bin/bash
# Find PRs older than 7 days without recent activity

set -euo pipefail

WORKSPACE="myworkspace"
REPO="myrepo"
DAYS_OLD=7
CUTOFF_DATE=$(date -d "$DAYS_OLD days ago" +%Y-%m-%dT%H:%M:%S 2>/dev/null || \
              date -v-${DAYS_OLD}d +%Y-%m-%dT%H:%M:%S)

echo "Finding PRs older than $DAYS_OLD days..."

bb pr list -R "$WORKSPACE/$REPO" --state OPEN --json | jq -r --arg cutoff "$CUTOFF_DATE" '
  .[] | select(.updated_on < $cutoff) | 
  "PR #\(.id): \(.title) (last updated: \(.updated_on))"
'
```

### Release Automation

```bash
#!/bin/bash
# Create a release PR merging develop into main

set -euo pipefail

WORKSPACE="myworkspace"
REPO="myrepo"
VERSION="${1:?Usage: $0 <version>}"
RELEASE_BRANCH="release/$VERSION"

echo "Creating release $VERSION..."

# Create release branch
git checkout develop
git pull origin develop
git checkout -b "$RELEASE_BRANCH"
git push -u origin "$RELEASE_BRANCH"

# Create PR
bb pr create -R "$WORKSPACE/$REPO" \
  --title "Release $VERSION" \
  --body "## Release $VERSION

### Changes
$(git log --oneline main..HEAD | sed 's/^/- /')

### Checklist
- [ ] Version bumped
- [ ] Changelog updated
- [ ] Tests passing" \
  --base main \
  --head "$RELEASE_BRANCH"

echo "Release PR created!"
```

### Sync Fork with Upstream

```bash
#!/bin/bash
# Sync a forked repository with upstream

set -euo pipefail

UPSTREAM_WORKSPACE="upstream-workspace"
UPSTREAM_REPO="upstream-repo"
FORK_WORKSPACE="my-workspace"
FORK_REPO="my-fork"

# Get latest from upstream using API
LATEST_SHA=$(bb api /repositories/$UPSTREAM_WORKSPACE/$UPSTREAM_REPO/refs/branches/main | \
  jq -r '.target.hash')

echo "Latest upstream commit: $LATEST_SHA"

# Sync fork (via git)
git fetch upstream
git checkout main
git merge upstream/main
git push origin main

echo "Fork synced!"
```

### Monitor Pipeline and Notify

```bash
#!/bin/bash
# Monitor pipeline and send notification on completion

set -euo pipefail

WORKSPACE="myworkspace"
REPO="myrepo"
SLACK_WEBHOOK="${SLACK_WEBHOOK_URL:-}"
BUILD_NUM="${1:?Usage: $0 <build-number>}"

wait_for_pipeline() {
  local build=$1
  local timeout=3600
  local elapsed=0
  
  while [[ $elapsed -lt $timeout ]]; do
    local status=$(bb pipeline list -R "$WORKSPACE/$REPO" --json | \
      jq -r --argjson num "$build" '.[] | select(.build_number == $num) | .state')
    local result=$(bb pipeline list -R "$WORKSPACE/$REPO" --json | \
      jq -r --argjson num "$build" '.[] | select(.build_number == $num) | .result // empty')
    
    if [[ "$status" == "COMPLETED" ]]; then
      echo "$result"
      return 0
    fi
    
    sleep 30
    elapsed=$((elapsed + 30))
  done
  
  echo "TIMEOUT"
  return 1
}

RESULT=$(wait_for_pipeline "$BUILD_NUM")

# Send Slack notification if webhook configured
if [[ -n "$SLACK_WEBHOOK" ]]; then
  if [[ "$RESULT" == "SUCCESSFUL" ]]; then
    COLOR="good"
    TEXT="Pipeline #$BUILD_NUM completed successfully!"
  else
    COLOR="danger"
    TEXT="Pipeline #$BUILD_NUM failed: $RESULT"
  fi
  
  curl -s -X POST "$SLACK_WEBHOOK" \
    -H 'Content-type: application/json' \
    -d "{\"attachments\":[{\"color\":\"$COLOR\",\"text\":\"$TEXT\"}]}"
fi

[[ "$RESULT" == "SUCCESSFUL" ]]
```

---

## CI/CD Integration

### GitHub Actions

```yaml
name: Bitbucket Sync

on:
  push:
    branches: [main]

jobs:
  sync:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Install bb CLI
        run: |
          curl -sL https://github.com/rbansal42/bitbucket-cli/releases/latest/download/bb_linux_amd64.tar.gz | tar xz
          sudo mv bb /usr/local/bin/
      
      - name: Create PR in Bitbucket
        env:
          BB_TOKEN: ${{ secrets.BITBUCKET_access token }}
        run: |
          bb pr create \
            --repo myworkspace/myrepo \
            --title "Sync from GitHub: ${{ github.sha }}" \
            --body "Automated sync from GitHub repository" \
            --base main \
            --head sync-${{ github.sha }}
```

### GitLab CI

```yaml
stages:
  - deploy

variables:
  BB_TOKEN: $BITBUCKET_access token

deploy:
  stage: deploy
  image: alpine:latest
  before_script:
    - apk add --no-cache curl jq
    - curl -sL https://github.com/rbansal42/bitbucket-cli/releases/latest/download/bb_linux_amd64.tar.gz | tar xz
    - mv bb /usr/local/bin/
  script:
    - |
      bb pipeline run \
        --repo myworkspace/myrepo \
        --branch main \
        --custom deploy-production
  only:
    - main
```

### Jenkins Pipeline

```groovy
pipeline {
    agent any
    
    environment {
        BB_TOKEN = credentials('bitbucket-access token')
    }
    
    stages {
        stage('Setup') {
            steps {
                sh '''
                    curl -sL https://github.com/rbansal42/bitbucket-cli/releases/latest/download/bb_linux_amd64.tar.gz | tar xz
                    chmod +x bb
                '''
            }
        }
        
        stage('Create PR') {
            steps {
                sh '''
                    ./bb pr create \
                        --repo myworkspace/myrepo \
                        --title "Jenkins Build #${BUILD_NUMBER}" \
                        --body "Automated PR from Jenkins" \
                        --fill
                '''
            }
        }
        
        stage('Trigger Pipeline') {
            steps {
                sh '''
                    ./bb pipeline run \
                        --repo myworkspace/myrepo \
                        --branch ${BRANCH_NAME}
                '''
            }
        }
    }
}
```

### CircleCI

```yaml
version: 2.1

jobs:
  bitbucket-sync:
    docker:
      - image: cimg/base:stable
    steps:
      - checkout
      - run:
          name: Install bb CLI
          command: |
            curl -sL https://github.com/rbansal42/bitbucket-cli/releases/latest/download/bb_linux_amd64.tar.gz | tar xz
            sudo mv bb /usr/local/bin/
      - run:
          name: Sync to Bitbucket
          command: |
            bb pr create \
              --repo myworkspace/myrepo \
              --title "CircleCI Sync: ${CIRCLE_SHA1:0:7}" \
              --body "Automated sync from CircleCI build #${CIRCLE_BUILD_NUM}"

workflows:
  sync:
    jobs:
      - bitbucket-sync:
          context: bitbucket-credentials
```

### Azure DevOps Pipeline

```yaml
trigger:
  - main

pool:
  vmImage: 'ubuntu-latest'

variables:
  - group: bitbucket-credentials

steps:
  - script: |
      curl -sL https://github.com/rbansal42/bitbucket-cli/releases/latest/download/bb_linux_amd64.tar.gz | tar xz
      sudo mv bb /usr/local/bin/
    displayName: 'Install bb CLI'

  - script: |
      bb pipeline run \
        --repo $(BITBUCKET_WORKSPACE)/$(BITBUCKET_REPO) \
        --branch main
    displayName: 'Trigger Bitbucket Pipeline'
    env:
      BB_TOKEN: $(BITBUCKET_access token)
```

### Bitbucket Pipelines (Self-Triggering)

```yaml
# bitbucket-pipelines.yml
pipelines:
  custom:
    create-release-pr:
      - step:
          name: Create Release PR
          script:
            - curl -sL https://github.com/rbansal42/bitbucket-cli/releases/latest/download/bb_linux_amd64.tar.gz | tar xz
            - chmod +x bb
            - |
              ./bb pr create \
                --repo $BITBUCKET_WORKSPACE/$BITBUCKET_REPO_SLUG \
                --title "Release $(date +%Y.%m.%d)" \
                --body "Automated release PR" \
                --base main \
                --head develop
          services:
            - docker
```

---

## Best Practices

1. **Always use `--repo` in scripts** - Don't rely on git context detection
2. **Use `--json` for parsing** - Human-readable output may change between versions
3. **Handle errors gracefully** - Check exit codes and validate JSON responses
4. **Use environment variables for secrets** - Never hardcode tokens
5. **Set timeouts** - Especially for pipeline monitoring scripts
6. **Use `set -euo pipefail`** - Fail fast on errors in bash scripts
7. **Log actions** - Include echo statements for debugging
8. **Test locally first** - Verify scripts before running in CI/CD
