/*
Copyright 2017, 2019 the Velero contributors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"math/rand"
	"strconv"

	"bytetrade.io/web3os/velero-plugin-for-terminus/utils/collections"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"

	veleroplugin "github.com/vmware-tanzu/velero/pkg/plugin/framework"
)

const (
	ackClusterNameKey = "ACK_CLUSTER_NAME"
)

type Volume struct {
	volType, az string
	iops        int64
}

// Snapshot keeps track of snapshots created by this plugin
type Snapshot struct {
	volID, az string
	tags      map[string]string
}

// VolumeSnapshotter struct
type VolumeSnapshotter struct {
	config    map[string]string
	log       logrus.FieldLogger
	volumes   map[string]Volume
	snapshots map[string]Snapshot
}

// NewVolumeSnapshotter init a VolumeSnapshotter
func newVolumeSnapshotter(logger logrus.FieldLogger) *VolumeSnapshotter {
	return &VolumeSnapshotter{log: logger}
}

// Init init ecs client with os env
func (b *VolumeSnapshotter) Init(config map[string]string) error {
	if err := veleroplugin.ValidateVolumeSnapshotterConfigKeys(config); err != nil {
		return err
	}
	b.config = config

	if b.volumes == nil {
		b.volumes = make(map[string]Volume)
	}
	if b.snapshots == nil {
		b.snapshots = make(map[string]Snapshot)
	}

	return nil
}

// CreateVolumeFromSnapshot restore a volume from a snapshot
func (b *VolumeSnapshotter) CreateVolumeFromSnapshot(snapshotID, volumeType, volumeAZ string, iops *int64) (string, error) {
	b.log.Infof("<plugin> CreateVolumeFromSnapshot called snapshotID: %s, volumeType: %s, volumeAZ: %s, iops: %d", snapshotID, volumeType, volumeAZ, *iops)
	var volumeID string
	for {
		volumeID := snapshotID + ".vol." + strconv.FormatUint(rand.Uint64(), 10)
		if _, ok := b.volumes[volumeID]; ok {
			// Duplicate ? Retry
			continue
		}
		break
	}

	b.volumes[volumeID] = Volume{
		volType: volumeType,
		az:      volumeAZ,
		iops:    *iops,
	}
	return volumeID, nil
}

// GetVolumeInfo get a volume's details
func (b *VolumeSnapshotter) GetVolumeInfo(volumeID, volumeAZ string) (string, *int64, error) {
	b.log.Infof("<plugin> GetVolumeInfo called volumeID: %s, volumeAZ: %s", volumeID, volumeAZ)
	if val, ok := b.volumes[volumeID]; ok {
		iops := val.iops
		return val.volType, &iops, nil
	}
	return "", nil, errors.New("Volume " + volumeID + " not found")
}

// CreateSnapshot create a snapshot
func (b *VolumeSnapshotter) CreateSnapshot(volumeID, volumeAZ string, tags map[string]string) (string, error) {
	b.log.Infof("<plugin> CreateSnapshot called volumeID: %s, volumeAZ: %s, tags: %v", volumeID, volumeAZ, tags)
	var snapshotID string
	for {
		snapshotID = volumeID + ".snap." + strconv.FormatUint(rand.Uint64(), 10)
		b.log.Infof("<plugin> CreateSnapshot trying to create snapshot snapshotID: %s", snapshotID)
		if _, ok := b.snapshots[snapshotID]; ok {
			// Duplicate ? Retry
			continue
		}
		break
	}

	// Remember the "original" volume, only required for the first
	// time.
	if _, exists := b.volumes[volumeID]; !exists {
		b.volumes[volumeID] = Volume{
			volType: "orignalVolumeType",
			az:      volumeAZ,
			iops:    100,
		}
	}

	// Remember the snapshot
	b.snapshots[snapshotID] = Snapshot{volID: volumeID,
		az:   volumeAZ,
		tags: tags}

	b.log.Infof("<plugin> CreateSnapshot returning snapshotID: %s", snapshotID)
	return snapshotID, nil
}

// DeleteSnapshot delete a snapshot
func (b *VolumeSnapshotter) DeleteSnapshot(snapshotID string) error {
	b.log.Infof("<plugin> DeleteSnapshot called snapshotID: %s", snapshotID)
	delete(b.snapshots, snapshotID)
	return nil
}

// GetVolumeID get a volume's id
func (b *VolumeSnapshotter) GetVolumeID(pv runtime.Unstructured) (string, error) {
	b.log.Infof("<plugin> GetVolumeID called pv: %v", pv)
	if !collections.Exists(pv.UnstructuredContent(), "spec.hostPath.path") {
		return "", errors.New("Example plugin failed to get volume ID. ")
	}

	// Seed the volume info so that GetVolumeInfo doesn't fail later.
	volumeID, _ := collections.GetString(pv.UnstructuredContent(), "spec.hostPath.path")
	if _, exists := b.volumes[volumeID]; !exists {
		b.log.Info("L134")
		b.volumes[volumeID] = Volume{
			volType: "orignalVolumeType",
			iops:    100,
		}
	}

	return collections.GetString(pv.UnstructuredContent(), "spec.hostPath.path")
}

// SetVolumeID set the volume's id
func (b *VolumeSnapshotter) SetVolumeID(pv runtime.Unstructured, volumeID string) (runtime.Unstructured, error) {
	b.log.Infof("<plugin> SetVolumeID called pv: %v, volumeID: %s", pv, volumeID)
	metadataMap, err := collections.GetMap(pv.UnstructuredContent(), "spec.hostPath.path")
	if err != nil {
		return nil, err
	}

	metadataMap["volumeID"] = volumeID
	return pv, nil
}
