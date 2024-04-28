# velero-plugin-for-terminus

[![](https://github.com/beclab/velero-plugin-for-terminus/actions/workflows/publish_docker_velero-plugin-for-terminus.yaml/badge.svg?branch=main)](https://github.com/beclab/velero-plugin-for-terminus/actions/workflows/publish_docker_velero-plugin-for-terminus.yaml)

Velero is a utility to back up and restore your Kubernetes resource and persistent volumes.

To do backup/restore on Terminus through Velero utility, you need to install and configure velero and velero-plugin for terminus.

## How to use
1. download and install [velero](https://github.com/vmware-tanzu/velero).
2. create backup-location
```bash
velero backup-location create terminus-cloud \
  --provider terminus \
  --namespace os-system \
  --prefix "" \
  --bucket terminus-cloud
```

3. install velero-plugin-for-terminus
```bash
velero install \
  --no-default-backup-location \
  --namespace os-system \
  --image beclab/velero:v1.11.1 \
  --use-volume-snapshots=false \
  --no-secret \
  --plugins beclab/velero-plugin-for-terminus:v1.0.1 \
  --wait
```