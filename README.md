# TMC - Transparent Multi-Cluster

Experimental multi-cluster workload management platform for Kubernetes, built on top of [kcp](https://github.com/kcp-dev/kcp)

## What is TMC?

TMC is a multi-cluster management platform that allows you to manage multiple Kubernetes clusters from a single control plane. TMC is to be layered on top of KCP and extends its functionality to multi-cluster workload management.

## Quick start

```
# Build CLI binaries

  make build

# Copy binaries to PATH

  # Either copy the binaries to a directory in your PATH
  cp bin/{kubectl-tmc,kubectl-workloads} /usr/local/bin/kubectl-tmc

  # Or add the bin directory to your PATH
  export PATH=$PATH:$(pwd)/bin

# Deploy TMC on top of kcp 

  TBD

# Create TMC workspace

  kubectl tmc workspace create tmc-ws --type tmc --enter

# Create SyncTarget for remote cluster

  kubectl tmc workload sync cluster-1 --syncer-image ghcr.io/kcp-dev/contrib-tmc/syncer:latest --output-file cluster-1.yaml

# Login into child cluster

  KUBECONFIG=<pcluster-config> kubectl apply -f "cluster-1.yaml"

# Bind compute resources

  kubectl tmc bind compute root:tmc-ws

# Create some workload in a kcp workspace

  kubectl create deployment kuard --image gcr.io/kuar-demo/kuard-amd64:blue
```

## Known issues

- [ ] TMC currently does not support sharding
- [ ] Log tunneling needs to be brought back

## Background

Historically TMC was part of kcp. It has now been spinned off as a separate project, whose capabilities can be layered on top of kcp. This leaves kcp more slim for use cases not needing TMC.

https://github.com/kcp-dev/kcp/issues/2954

