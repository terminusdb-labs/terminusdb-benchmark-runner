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

1. Run the `ingest.py` script with the `--no-insert` argument to create the `objs.json` required for the lego benchmark
and place it in the lego Demo folder of your choice. Copy the `schema.json` to this directory too. The script can be found in the `demo_data` folder of the [terminus-cms repo](https://github.com/terminusdb-labs/terminus-cms).
2. Copy config.sample.json to `~/.tdb_benchmark_config.json` and edit the sample values to the right values.

`./benchmark [git_commit_hash] [benchmark_type]` in which benchmark type can be `lego`, `k6`, `js` or `all`.

If you are not part of the `docker` user group, run the command as a user who can (root, if you feel unsafe). Your home dir will be /root, keep this in mind when setting up the config file.
