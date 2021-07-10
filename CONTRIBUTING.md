# How to Contribute

Sloth is [Apache 2.0 licensed](LICENSE) and accepts contributions via GitHub
pull requests. This document outlines some of the conventions on development
workflow, commit message formatting, contact points and other resources to make
it easier to get your contribution accepted.

We gratefully welcome improvements to issues and documentation as well as to code.

## Getting Started

- Fork the repository on GitHub
- Read the [README](README.md#getting-started) for getting started.
- If you want to contribute as a developer, continue reading this document for further instructions
- Play with the project, submit bugs, submit pull requests!

## Contribution workflow

This is a rough outline of how to prepare a contribution:

- Fork the repository.
- Create a topic branch from where you want to base your work (usually branched from master).
- Make commits of logical units.
- Make sure your commit messages are clear and self-explanatory.
- Push your changes to a topic branch in your fork of the repository.
- If you changed code, add automated tests to cover your changes.
- Submit a pull request from your fork to the original repository.

## Running the application

### CLI

To run the CLI you can use the example specs. Some examples:

```bash
go run ./cmd/sloth generate  -i ./examples/getting-started.yml

go run ./cmd/sloth/ validate -i ./examples/ -p ./examples/plugins/ -e _gen
```

### Kubernetes

To run Sloth in a controller mode you can run it in multiple ways, depending on the part that you are working on it may be helpful one or the other.

> Apart the options that we will describe next, Kuberentes controller mode has multiple options that can be used to develop, deploy in different ways or apply maintenance, like selecting one single namespace, use a label selector... Check them with `sloth controller --help`

#### Without a cluster

If you are not developing something that needs a real Kubernetes connection, Sloth can run without a Kubernetes cluster with fake memory K8s memory based clients, use `--mode="fake"`

Example:

```bash
go run ./cmd/sloth/ controller --mode=fake --debug
```

#### With a local cluster

If you need a Kubernetes connection or develop using more realistic setup, you can connect to any Kubernetes cluster using local credentials using `--kube-local` flag.

```bash
go run ./cmd/sloth/ controller --kube-local
```

#### Dry run

If you need to set Sloth in dry-run mode (read-only operations), you case use `--mode=dry-run`.

```bash
go run ./cmd/sloth/ controller --mode=dry-run
```

You can use this mode with `--kube-local`.

```bash
go run ./cmd/sloth/ controller --kube-local --mode=dry-run
```

## Automated checks and unit tests

You can check your code satisfies project standards by using:

```bash
make check
```

You can run the unit tests by doing:

```bash
make test
```

## Integration tests

> When running the tests if you don't have any of the required dependencies, the tests will be skipped

### CLI

First you will need to build the binary (you can use `make build`).

Search your binary, for example `./bin/sloth-linux-amd64` and set as the binary to execute the integration tests:

```bash
export SLOTH_INTEGRATION_BINARY=${PWD}/bin/sloth-linux-amd64
```

Now you can run the tests:

```bash
make ci-integration-cli
```

### Kubernetes

For Kubernetes you will need a cluster, the easiest way is to create a cluster using [Kind], lets see an example by creating a cluster and exporting the access configuration.

```bash
kind create cluster --name sloth
kind get kubeconfig --name sloth > /tmp/kind-sloth.kubeconfig
```

Prepare the required CRDs on the cluster:

```bash
kubectl --kubeconfig=/tmp/kind-sloth.kubeconfig apply -f ./pkg/kubernetes/gen/crd/
kubectl --kubeconfig=/tmp/kind-sloth.kubeconfig apply -f ./test/integration/crd
```

Now we are ready, we need to prepare the integration tests settings that point to the binary of sloth we want to use (build with `make build`) and the Kubernetes cluster access config.

```bash
export SLOTH_INTEGRATION_BINARY=${PWD}/bin/sloth-linux-amd64
export SLOTH_INTEGRATION_KUBE_CONFIG=/tmp/kind-sloth.kubeconfig
```

Execute the tests:

```bash
make ci-integration-k8s
```

## Profiling

By default Sloth will set [pprof] on metrics ports (`8081`).

Check this [pprof cheatsheet][pprof-cheatsheet].

[kind]: https://github.com/kubernetes-sigs/kind
[pprof-cheatsheet]: https://gist.github.com/slok/33dad1d0d0bae07977e6d32bcc010188
