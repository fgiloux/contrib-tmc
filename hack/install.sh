#!/usr/bin/env bash

# Copyright 2023 The TMC Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o pipefail

kubectl ws root
kubectl apply -f ./config/rbac
kubectl apply -f ./config/tmc/workspacetype-tmc.yaml
kubectl apply -f ./config/tmc/clusterworkspace-tmc.yaml
kubectl apply -f ./config/rootcompute/clusterworkspace-compute.yaml
id_hash=$(kubectl get apiexport shards.core.kcp.io -o=jsonpath='{.status.identityHash}')

sleep 1
kubectl ws root:tmc
kubectl apply -f ./config/tmc/resources
kubectl apply -f ./config/tmc/resources/export/apiexport-identity-secret.yaml
cat ./config/tmc/resources/export/apiexport-tmc.yaml | sed "s/{{core_id_hash}}/$id_hash/g" | kubectl apply -f -
cat ./config/tmc/resources/export/apibinding-tmc.yaml | sed "s/{{core_id_hash}}/$id_hash/g" | kubectl apply -f -

kubectl ws root:compute
kubectl apply -f ./config/rootcompute/kube-1.24

kubectl ws root:tmc


