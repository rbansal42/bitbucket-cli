# bb pipeline

Manage Bitbucket Pipelines.

## Synopsis

```
bb pipeline <subcommand> [flags]
```

## Description

Work with Bitbucket Pipelines for continuous integration and deployment. List pipeline runs, view details, trigger new builds, monitor logs, and manage pipeline execution.

## Subcommands

- [bb pipeline list](#bb-pipeline-list) - List pipeline runs
- [bb pipeline view](#bb-pipeline-view) - View pipeline details
- [bb pipeline run](#bb-pipeline-run) - Trigger a pipeline run
- [bb pipeline logs](#bb-pipeline-logs) - View pipeline logs
- [bb pipeline steps](#bb-pipeline-steps) - List pipeline steps
- [bb pipeline stop](#bb-pipeline-stop) - Stop a running pipeline

---

# bb pipeline list

List pipeline runs for a repository.

## Synopsis

```
bb pipeline list [flags]
```

## Description

Display a list of pipeline runs for the current or specified repository. By default, shows the most recent pipeline runs with their status, branch, and trigger information.

Results are sorted by creation time, with the most recent pipelines first.

## Flags

| Flag | Description |
|------|-------------|
| `-R, --repo <owner/repo>` | Select a repository (default: current repository) |
| `-b, --branch <name>` | Filter by branch name |
| `-s, --status <status>` | Filter by status (PENDING, IN_PROGRESS, SUCCESSFUL, FAILED, STOPPED) |
| `-L, --limit <number>` | Maximum number of results to return (default: 30) |
| `--json` | Output in JSON format |
| `-h, --help` | Show help for command |

## Examples

List recent pipelines:

```
$ bb pipeline list
ID      STATUS       BRANCH    TRIGGER        DURATION  CREATED
1234    SUCCESSFUL   main      push           2m 34s    2026-02-05 09:15:00
1233    FAILED       feature   push           1m 12s    2026-02-05 08:45:00
1232    SUCCESSFUL   main      pull_request   3m 01s    2026-02-04 16:30:00
```

Filter by branch:

```
$ bb pipeline list --branch main
```

Filter by status:

```
$ bb pipeline list --status FAILED
```

List pipelines for a specific repository:

```
$ bb pipeline list -R myworkspace/myrepo
```

## See also

- [bb pipeline view](#bb-pipeline-view) - View pipeline details
- [bb pipeline run](#bb-pipeline-run) - Trigger a pipeline run

---

# bb pipeline view

View details of a specific pipeline run.

## Synopsis

```
bb pipeline view <pipeline-id> [flags]
```

## Description

Display detailed information about a specific pipeline run, including its status, duration, trigger information, and step summary.

If no pipeline ID is provided, the most recent pipeline run for the current branch is shown.

## Flags

| Flag | Description |
|------|-------------|
| `-R, --repo <owner/repo>` | Select a repository (default: current repository) |
| `-w, --web` | Open the pipeline in a browser |
| `--json` | Output in JSON format |
| `-h, --help` | Show help for command |

## Examples

View a specific pipeline:

```
$ bb pipeline view 1234
Pipeline #1234
Status:     SUCCESSFUL
Branch:     main
Commit:     abc1234 - Fix authentication bug
Trigger:    push by johndoe
Started:    2026-02-05 09:15:00 UTC
Duration:   2m 34s

Steps:
  ✓ Build         45s
  ✓ Test          1m 20s
  ✓ Deploy        29s
```

View the most recent pipeline for current branch:

```
$ bb pipeline view
```

Open pipeline in browser:

```
$ bb pipeline view 1234 --web
Opening https://bitbucket.org/myworkspace/myrepo/pipelines/results/1234
```

## See also

- [bb pipeline list](#bb-pipeline-list) - List pipeline runs
- [bb pipeline logs](#bb-pipeline-logs) - View pipeline logs
- [bb pipeline steps](#bb-pipeline-steps) - List pipeline steps

---

# bb pipeline run

Trigger a new pipeline run.

## Synopsis

```
bb pipeline run [flags]
```

## Description

Trigger a new pipeline run for the current or specified branch. By default, runs the default pipeline defined in `bitbucket-pipelines.yml`.

You can specify a custom pipeline or target using the `--pipeline` and `--target` flags.

## Flags

| Flag | Description |
|------|-------------|
| `-R, --repo <owner/repo>` | Select a repository (default: current repository) |
| `-b, --branch <name>` | Branch to run pipeline on (default: current branch) |
| `-p, --pipeline <name>` | Custom pipeline name to run |
| `-t, --target <type>` | Pipeline target type (branch, tag, bookmark, custom) |
| `--commit <sha>` | Specific commit SHA to run pipeline on |
| `-v, --variable <key=value>` | Pipeline variable (can be specified multiple times) |
| `--wait` | Wait for pipeline to complete |
| `--json` | Output in JSON format |
| `-h, --help` | Show help for command |

## Examples

Trigger a pipeline on the current branch:

```
$ bb pipeline run
Triggered pipeline #1235 on branch main
https://bitbucket.org/myworkspace/myrepo/pipelines/results/1235
```

Run pipeline on a specific branch:

```
$ bb pipeline run --branch feature-branch
```

Run a custom pipeline:

```
$ bb pipeline run --pipeline deploy-staging
```

Run with custom variables:

```
$ bb pipeline run -v ENV=staging -v DEBUG=true
```

Trigger and wait for completion:

```
$ bb pipeline run --wait
Triggered pipeline #1235 on branch main
Waiting for pipeline to complete...
Pipeline #1235 completed with status: SUCCESSFUL
```

## See also

- [bb pipeline list](#bb-pipeline-list) - List pipeline runs
- [bb pipeline view](#bb-pipeline-view) - View pipeline details
- [bb pipeline stop](#bb-pipeline-stop) - Stop a running pipeline

---

# bb pipeline logs

View logs from a pipeline run.

## Synopsis

```
bb pipeline logs <pipeline-id> [flags]
```

## Description

Display logs from a specific pipeline run. By default, shows logs from all steps combined. Use the `--step` flag to view logs from a specific step.

## Flags

| Flag | Description |
|------|-------------|
| `-R, --repo <owner/repo>` | Select a repository (default: current repository) |
| `-s, --step <name>` | Show logs for a specific step only |
| `-f, --follow` | Follow log output (for in-progress pipelines) |
| `--failed` | Show only failed step logs |
| `-h, --help` | Show help for command |

## Examples

View all logs from a pipeline:

```
$ bb pipeline logs 1234
==> Step: Build
+ npm install
added 523 packages in 12.5s
+ npm run build
Build completed successfully.

==> Step: Test
+ npm test
All 42 tests passed.
```

View logs for a specific step:

```
$ bb pipeline logs 1234 --step Test
+ npm test
Running test suite...
All 42 tests passed.
```

Follow logs for a running pipeline:

```
$ bb pipeline logs 1235 --follow
==> Step: Build (in progress)
+ npm install
Installing dependencies...
```

View only failed step logs:

```
$ bb pipeline logs 1233 --failed
==> Step: Test (FAILED)
+ npm test
FAIL src/auth.test.js
  ✕ should validate token (15ms)
    Expected: true
    Received: false
```

## See also

- [bb pipeline view](#bb-pipeline-view) - View pipeline details
- [bb pipeline steps](#bb-pipeline-steps) - List pipeline steps

---

# bb pipeline steps

List steps in a pipeline run.

## Synopsis

```
bb pipeline steps <pipeline-id> [flags]
```

## Description

Display a list of all steps in a specific pipeline run with their status, duration, and execution order.

## Flags

| Flag | Description |
|------|-------------|
| `-R, --repo <owner/repo>` | Select a repository (default: current repository) |
| `--json` | Output in JSON format |
| `-h, --help` | Show help for command |

## Examples

List steps in a pipeline:

```
$ bb pipeline steps 1234
#   NAME      STATUS       DURATION  IMAGE
1   Build     SUCCESSFUL   45s       atlassian/default-image:4
2   Test      SUCCESSFUL   1m 20s    atlassian/default-image:4
3   Deploy    SUCCESSFUL   29s       atlassian/default-image:4
```

List steps with JSON output:

```
$ bb pipeline steps 1234 --json
[
  {
    "name": "Build",
    "status": "SUCCESSFUL",
    "duration_in_seconds": 45,
    "image": "atlassian/default-image:4"
  },
  ...
]
```

## See also

- [bb pipeline view](#bb-pipeline-view) - View pipeline details
- [bb pipeline logs](#bb-pipeline-logs) - View pipeline logs

---

# bb pipeline stop

Stop a running pipeline.

## Synopsis

```
bb pipeline stop <pipeline-id> [flags]
```

## Description

Stop a pipeline that is currently in progress. This will terminate all running steps and mark the pipeline as STOPPED.

This action cannot be undone. You can trigger a new pipeline run using `bb pipeline run`.

## Flags

| Flag | Description |
|------|-------------|
| `-R, --repo <owner/repo>` | Select a repository (default: current repository) |
| `-y, --yes` | Skip confirmation prompt |
| `-h, --help` | Show help for command |

## Examples

Stop a running pipeline:

```
$ bb pipeline stop 1235
? Are you sure you want to stop pipeline #1235? Yes
Pipeline #1235 has been stopped
```

Stop without confirmation:

```
$ bb pipeline stop 1235 --yes
Pipeline #1235 has been stopped
```

## See also

- [bb pipeline list](#bb-pipeline-list) - List pipeline runs
- [bb pipeline run](#bb-pipeline-run) - Trigger a pipeline run
