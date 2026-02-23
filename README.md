# directorysync-s3-service

A Go service that synchronizes one or more local directories to Amazon S3 on a schedule, with optional service-mode execution, live status reporting, and per-profile filtering rules.

## Overview

`directorysync-s3-service` runs one or more configured sync profiles. Each profile:

- scans a local source directory
- filters files by extension/exclusions
- checks whether files already exist in S3
- uploads missing files (or all files when overwrite is enabled)
- exposes runtime status through an HTTP API

The service supports:

- interactive/foreground execution
- system service execution (`install`, `start`, `stop`, `uninstall`)
- scheduled jobs with cron expressions
- configurable concurrent upload workers
- optional throttling support for copy operations
- simulation mode (dry-run copy behavior)

## Repository Status

This repository is currently in a package layout migration (`data/`, `service/`, `main/` moved to `internal/*` and `cmd/`).

Important note:
- the current source tree has import/build path mismatches (for example code imports `bitbucket.org/kpsgo/directorysync-s3-service/data` while files live under `internal/types`)
- `go test ./...` currently fails until import paths are aligned

The behavior documented below reflects the implemented runtime logic in the codebase.

## Architecture

High-level components:

- `cmd/main.go`: entrypoint, CLI flags, interactive vs service mode, service control commands
- `internal/conf/conf.go`: loads `config.yaml`, resolves execution folder, stores global config
- `internal/service/service.go`: manages synchronizer instances and service lifecycle
- `internal/service/synchronizer.go`: file discovery, S3 comparison, upload execution, scheduling, `.environ` loading
- `api/statusapi.go`: REST status endpoints (`/status`, `/status/{profile}`)
- `internal/types/*.go`: runtime status model shared by synchronizers/API

Execution flow:

1. Load `config.yaml`.
2. Build synchronizer instances from `syncprofiles`.
3. For each run:
   - load `.environ` variables (if present)
   - list keys in target S3 bucket
   - walk local source tree and create `search.dat` for candidate files
   - upload files using N worker goroutines (`maxconcurrentcopies`)
   - write failed file paths to `errors.dat`
   - update in-memory service status
4. Expose status over HTTP API.

## Requirements

- Go `1.26` (per `go.mod`)
- AWS credentials available through:
  - environment variables, or
  - shared profile/credentials chain, or
  - static credentials in connection config
- Network access to S3-compatible endpoint

Optional:

- Docker (for `make build-linux` / `make build-win`)

## Configuration

The service expects `config.yaml` in the current working directory (interactive mode) or executable directory (service mode).

### Top-level structure

```yaml
syncprofiles:
  profile-name:
    name: profile-name
    description: sample profile
    s3config:
      connect:
        region: us-east-1
        endpoint: http://localhost:4566
        s3usepathstyle: true
        profile: default
      bucket: my-target-bucket
      s3storageclass: standard_ia
    filecopyoptions:
      throttlebucketsizekb: 2048
      throttle: false
      maxconcurrentcopies: 4
    source: /path/to/source
    ext: [txt, csv]
    exclusions: [tmp, bak]
    scheduledef: "* * * * *"
    runatstartup: true
    recursive: true

serviceconfig:
  Name: dirsynctos3
  DisplayName: Directory Synchronization to AWS S3
  Description: Directory Synchronization to AWS S3

apiconfig:
  listenerip: 0.0.0.0
  port: 8080
  protocol: tcp
  issecure: false

executionflags:
  overwrite: false
```

### Profile fields

- `name`: profile identifier (used for status lookup)
- `description`: human-readable description
- `source`: local source directory
- `recursive`: include subdirectories
- `runatstartup`: run immediately on service startup
- `scheduledef`: cron expression for recurring runs
- `ext`: allowed file extensions (empty = all)
- `exclusions`: excluded extensions
- `s3config`: S3 target and connection details
- `filecopyoptions.maxconcurrentcopies`: number of parallel upload workers
- `filecopyoptions.throttle`: enable throttled copy mode
- `filecopyoptions.throttlebucketsizekb`: throttle bucket size in KB

### `.environ` support

Before each run, the synchronizer attempts to load a `.environ` file from the execution folder and set values in the process environment.

Example:

```env
AWS_ACCESS_KEY_ID=...
AWS_SECRET_ACCESS_KEY=...
AWS_DEFAULT_REGION=us-east-1
```

Variables from this file are unset during cleanup after the run.

## Running

### Foreground (interactive)

Run all configured profiles:

```bash
go run ./cmd
```

Run one profile:

```bash
go run ./cmd --profile musicbak-local
```

List remote S3 keys for one profile:

```bash
go run ./cmd --profile musicbak-local --remote
```

Simulate copy operations:

```bash
go run ./cmd --simulate
```

Enable overwrite from CLI:

```bash
go run ./cmd --overwrite
```

### Service commands

The executable accepts service control commands:

- `install`
- `uninstall`
- `start`
- `stop`

Examples:

```bash
./dirsynctos3 install
./dirsynctos3 start
./dirsynctos3 stop
./dirsynctos3 uninstall
```

## REST Status API

Configured by `apiconfig` and started with the synchronizer service.

Endpoints:

- `GET /status`: service-wide status map (all profiles)
- `GET /status/{profile}`: status for one profile

Example:

```bash
curl http://localhost:8080/status
curl http://localhost:8080/status/musicbak-local
```

Returned status includes fields such as:

- profile configuration
- start/end timestamps
- next scheduled run time
- processing duration
- active transfers by worker
- synchronized file count
- error count
- running state

## Build

### Make targets

Current `Makefile` targets:

- `make build-linux`
- `make build-win`
- `make clean`
- dependency helpers (`tidy`, `deps-upgrade`, etc.)

These Docker-based targets are currently tied to legacy source paths and may require adjustment after the package migration.

### Direct Go build

```bash
go build -o dirsynctos3 ./cmd
```

## Runtime Artifacts

Created in execution folder:

- `search.dat`: discovered local files queued for upload
- `errors.dat`: file paths that failed to upload
- `syncservice.log`: service-mode log output

## Behavior Details

- S3 remote existence is determined by listing bucket keys and comparing normalized lower-case paths.
- If `executionflags.overwrite` is `false`, files already present in S3 are skipped.
- If simulation mode is enabled, uploads are not executed but run flow and logging still occur.
- `scheduledef` is evaluated to calculate `NextRunTime` and trigger periodic runs.

## Development Notes

- Module path: `bitbucket.org/kpsgo/directorysync-s3-service`
- Go workspace file (`go.work`) references local `../lib/go-supportlib`
- Vendor directory is present

Given the active migration state, align package imports before relying on CI/test status.

## Contributing

1. Create/update profile config under `config.yaml`.
2. Validate behavior in interactive mode with `--profile` and `--simulate`.
3. Verify API output from `/status`.
4. Keep changes compatible with both foreground and service-mode execution.

## License

No standalone `LICENSE` file is currently present in this repository. Source headers indicate proprietary/confidential ownership.
