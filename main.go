/*
Copyright 2017 Heptio Inc.

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
	"fmt"
	"net/http"
	"strings"

	plugin "github.com/hashicorp/go-plugin"
	"github.com/heptio/ark/pkg/cloudprovider"
	arkplugin "github.com/heptio/ark/pkg/plugin"
	"github.com/heptio/ark/pkg/util/collections"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	rookRestAPIURL = "rookRestAPIURL"
)

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: arkplugin.Handshake,
		Plugins: map[string]plugin.Plugin{
			string(arkplugin.PluginKindBlockStore): arkplugin.NewBlockStorePlugin(NewBlockStore()),
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}

type rook struct {
	apiUrl string
}

func NewBlockStore() cloudprovider.BlockStore {
	return &rook{}
}

func (b *rook) Init(config map[string]string) error {
	api := config[rookRestAPIURL]
	if api == "" {
		return errors.Errorf("missing %s in rook configuration", rookRestAPIURL)
	}

	b.apiUrl = api
	logrus.Infof("Api URL passed:", b.apiUrl)

	return nil
}

func (b *rook) CreateVolumeFromSnapshot(snapshotID, volumeType, volumeAZ string, iops *int64) (volumeID string, err error) {

	pool, image := b.getPoolImageFromVolumeID(snapshotID)

	httpURL := fmt.Sprintf("%s/block/%s/%s/%s", b.apiUrl, pool, snapshotID, image)

	// Make an http request to rookapi
	client := &http.Client{}
	req, err := http.NewRequest("POST", httpURL, nil)
	resp, err := client.Do(req)

	// Process response
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return fmt.Sprintf("%s||%s", pool, image), nil
}

func (b *rook) GetVolumeInfo(volumeID, volumeAZ string) (string, *int64, error) {
	return "rook", nil, nil
}

func (b *rook) IsVolumeReady(poolName, volumeID string) (ready bool, err error) {
	return true, nil
}

func (b *rook) ListSnapshots(tagFilters map[string]string) ([]string, error) {
	return []string{}, nil
}

func (b *rook) CreateSnapshot(volumeID, volumeAZ string, tags map[string]string) (snapshotID string, err error) {

	pool, image := b.getPoolImageFromVolumeID(volumeID)
	snapID := fmt.Sprintf("%s||%s", volumeID, uuid.NewV4())

	httpURL := fmt.Sprintf("%s/snapshot/%s/%s/%s", b.apiUrl, pool, snapID, image)

	// Make an http request to rookapi
	client := &http.Client{}
	req, err := http.NewRequest("POST", httpURL, nil)
	resp, err := client.Do(req)

	// Process response
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return snapID, nil
}

func (b *rook) DeleteSnapshot(snapshotID string) error {

	pool, _ := b.getPoolImageFromVolumeID(snapshotID)

	httpURL := fmt.Sprintf("%s/snapshot/%s/%s", b.apiUrl, pool, snapshotID)

	// Make an http request to rookapi
	client := &http.Client{}
	req, err := http.NewRequest("DELETE", httpURL, nil)
	resp, err := client.Do(req)

	// Process response
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (b *rook) GetVolumeID(pv runtime.Unstructured) (string, error) {
	if !collections.Exists(pv.UnstructuredContent(), "spec.flexVolume.options.pool") || !collections.Exists(pv.UnstructuredContent(), "spec.flexVolume.options.image") {
		return "", nil
	}

	pool, image, err := b.getPoolImageFromUnstructured(pv)

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s||%s", pool, image), nil
}

func (b *rook) SetVolumeID(pv runtime.Unstructured, volumeID string) (runtime.Unstructured, error) {

	rk, err := collections.GetMap(pv.UnstructuredContent(), "spec.flexVolume.options")
	if err != nil {
		return nil, err
	}

	pool, image := b.getPoolImageFromVolumeID(volumeID)

	rk["pool"] = pool
	rk["image"] = image

	return pv, nil
}

func (b *rook) getPoolImageFromUnstructured(pv runtime.Unstructured) (string, string, error) {
	pool, err := collections.GetString(pv.UnstructuredContent(), "spec.flexVolume.options.pool")
	if err != nil {
		return "", "", err
	}

	image, err := collections.GetString(pv.UnstructuredContent(), "spec.flexVolume.options.image")
	if err != nil {
		return "", "", err
	}

	return pool, image, nil
}

// Returns pool / imageId
func (b *rook) getPoolImageFromVolumeID(volumeID string) (string, string) {
	splits := strings.Split(volumeID, "||")
	return splits[0], splits[1]
}
