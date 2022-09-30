# TerminusDB Benchmark Runner

A simple Go program to run k6 against a specific TerminusDB commit hash.
It is a bit hacky but does its job. Currently it prints the k6 results to
stdout, but the goal is to run other benchmarks as well.

## Requirements

- [k6](https://github.com/grafana/k6)
- git
- Docker (with buildx!)
- [timejson](https://github.com/terminusdb-labs/time-json/)

## Compiling

`go build benchmark.go`

## Running

`./benchmark [git_commit_hash]`

If you are not part of the `docker` user group, run the command as a user who can (root, if you feel unsafe).
