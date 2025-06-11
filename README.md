# regmaid 

`regmaid` is a CLI tool to enforce tag retention policies on (private) container registries.

It works entirely by talking to the [Docker Registry HTTP API V2](https://docker-docs.uclv.cu/registry/spec/api/), so any registry supporting this API spec is supported.

## The Problem
Oftentimes, private, self-hosted Docker registries are used in development environments, where images are pushed at a high frequency and become obsolete quickly.

A lot of garbage accumulates that you want to get rid of.

Normally once an artifact is pushed, you would not want to modify or delete it, but in those scenarios you often have just a single point where images are being consumed, so it's fine and sometimes even necessary (costs) to get rid of old content.

Vanilla self-hosted registries (https://distribution.github.io/distribution/) have internal garbage-collection mechanisms, but those are for unreferenced blobs. There are no native retention policies for tags.

## Getting Started

### Installation

You can install regmaid with Go:
```sh
go install github.com/moritzrinow/regmaid@latest
```

You can also run regmaid with Docker:
```sh
docker run -it -v /path/to/regmaid.yaml:/etc/regmaid/regmaid.yaml ghcr.io/moritzrinow/regmaid:latest
```

Running as automated jobs with auto-confirm:
```sh
docker run -it -v /path/to/regmaid.yaml:/etc/regmaid/regmaid.yaml ghcr.io/moritzrinow/regmaid:latest --yes
```

### Configuration
Regmaid uses a YAML configuration file where registries and policies are defined. See [regmaid.yaml](regmaid.yaml) for reference.

Example:
```yaml
registries:
  - name: dev
    host: internal.registry.com
    username: user
    password: password

policies:
  - name: example-app-dev
    registry: dev
    repository: example-app # Policies may target a single repository, or multiple via wildcard expression
    match: *-dev # Match tags ending with '-dev'
    retention: 30d # Delete tags older than 30 days
    keep: 5 # Always keep at least newest 5 tags
```

### Cleaning Registry

Run command `regmaid`, which will output all tags eligible for deletion according to the configured policies.

Confirm with `yes` to delete the tags/manifests (or provide argument `--yes` to auto-confirm).

### Running CronJob on Kubernetes

Since Regmaid is available as a Docker image, it can also be run periodically as a [CronJob](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/) on Kubernetes.

See [kubernetes.yaml](examples/kubernetes.yaml) for an example.

## Limitations

### Large Repositories

Regmaid needs to make two requests against the registry for every tag in the repository. One for reading the manifest and one for reading the config blob. For large repositories this would mean a ton of requests. Configure rate-limiting accordingly for the registry (`maxConcurrentRequests`, `maxRequestsPerSecond`).
