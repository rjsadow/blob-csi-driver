/*
Copyright 2017 The Kubernetes Authors.

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

package csicommon

import (
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	fakeDriverName = "fake"
	fakeNodeID     = "fakeNodeID"
)

var (
	vendorVersion = "0.3.0"
)

func TestNewCSIDriver(t *testing.T) {
	name := ""
	str := ""
	nodeID := ""
	assert.Nil(t, NewCSIDriver(name, str, nodeID))
	name = "unit-test"
	assert.Nil(t, NewCSIDriver(name, str, nodeID))
	nodeID = "unit-test"
	driver := CSIDriver{
		Name:    name,
		NodeID:  nodeID,
		Version: str,
	}
	assert.Equal(t, &driver, NewCSIDriver(name, str, nodeID))
}

func NewFakeDriver() *CSIDriver {

	driver := NewCSIDriver(fakeDriverName, vendorVersion, fakeNodeID)

	return driver
}

func TestNewFakeDriver(t *testing.T) {
	// Test New fake driver with invalid arguments.
	d := NewCSIDriver("", vendorVersion, fakeNodeID)
	assert.Nil(t, d)
}

func TestAddControllerServiceCapabilities(t *testing.T) {
	d := NewFakeDriver()
	var cl []csi.ControllerServiceCapability_RPC_Type
	cl = append(cl, csi.ControllerServiceCapability_RPC_UNKNOWN)
	d.AddControllerServiceCapabilities(cl)
}

func TestGetVolumeCapabilityAccessModes(t *testing.T) {
	d := NewFakeDriver()

	// Test no volume access modes.
	// REVISIT: Do we need to support any default access modes.
	c := d.GetVolumeCapabilityAccessModes()
	assert.Zero(t, len(c))

	// Test driver with access modes.
	d.AddVolumeCapabilityAccessModes([]csi.VolumeCapability_AccessMode_Mode{csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER})
	modes := d.GetVolumeCapabilityAccessModes()
	assert.Equal(t, 1, len(modes))
	assert.Equal(t, modes[0].GetMode(), csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER)
}

func TestValidateControllerServiceRequest(t *testing.T) {
	d := NewFakeDriver()

	// Valid requests which require no capabilities
	err := d.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_UNKNOWN)
	assert.NoError(t, err)

	// Test controller service publish/unpublish not supported
	err = d.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME)
	s, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, s.Code(), codes.InvalidArgument)

	// Add controller service publish & unpublish request
	d.AddControllerServiceCapabilities(
		[]csi.ControllerServiceCapability_RPC_Type{
			csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
			csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
			csi.ControllerServiceCapability_RPC_GET_CAPACITY,
			csi.ControllerServiceCapability_RPC_LIST_VOLUMES,
		})

	// Test controller service publish/unpublish is supported
	err = d.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME)
	assert.NoError(t, err)

	// Test controller service create/delete is supported
	err = d.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME)
	assert.NoError(t, err)

	// Test controller service list volumes is supported
	err = d.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_LIST_VOLUMES)
	assert.NoError(t, err)

	// Test controller service get capacity is supported
	err = d.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_GET_CAPACITY)
	assert.NoError(t, err)

}

func TestAddNodeServiceCapabilities(t *testing.T) {
	d := NewFakeDriver()
	nl := []csi.NodeServiceCapability_RPC_Type{csi.NodeServiceCapability_RPC_UNKNOWN, csi.NodeServiceCapability_RPC_EXPAND_VOLUME}
	d.AddNodeServiceCapabilities(nl)
	expectedOutput := []*csi.NodeServiceCapability{NewNodeServiceCapability(nl[0]), NewNodeServiceCapability(nl[1])}
	assert.Equal(t, expectedOutput, d.NSCap, "NS Capabilities must Match")
}
