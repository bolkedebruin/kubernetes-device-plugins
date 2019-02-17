package fuse

import (
	"os"
	"strconv"

	"context"
	"github.com/golang/glog"
	"github.com/kubevirt/device-plugin-manager/pkg/dpm"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
)

const (
	FUSEPath  = "/dev/fuse"
	FUSEName  = "fuse"
	Namespace = "devices.kubevirt.io"
)

type message struct{}

type FusePlugin struct {
	counter int
	devs    []*pluginapi.Device
	update  chan message
}

// object responsible for discovering initial pool of devices and their allocation.
type FuseLister struct{}

func (l FuseLister) GetResourceNamespace() string {
	return Namespace
}

// Discovery discovers the FUSE device within the system.
func (l FuseLister) Discover(pluginListCh chan dpm.PluginNameList) {
	var plugins = make(dpm.PluginNameList, 0)

	if _, err := os.Stat(FUSEPath); err == nil {
		glog.V(3).Infof("Discovered %s", FUSEPath)
		plugins = append(plugins, FUSEName)
	}
	pluginListCh <- plugins
}

func (FuseLister) NewPlugin(deviceID string) dpm.PluginInterface {
	glog.V(3).Infof("Creating device plugin %s", deviceID)

	return &FusePlugin{
		counter: 0,
		devs:    make([]*pluginapi.Device, 0),
		update:  make(chan message),
	}
}

func (p *FusePlugin) ListAndWatch(e *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
	// initialize with one available device
	p.devs = append(p.devs, &pluginapi.Device{
		ID:	FUSEName + strconv.Itoa(p.counter),
		Health: pluginapi.Healthy,
	})

	glog.V(3).Infof("Returning %d available devices", len(p.devs))

	s.Send(&pluginapi.ListAndWatchResponse{Devices: p.devs})

	for {
		select {
		case <-p.update:
			s.Send(&pluginapi.ListAndWatchResponse{Devices: p.devs})
		}
	}
}

func (p *FusePlugin) Allocate(ctx context.Context, request *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	var response pluginapi.AllocateResponse
	var car pluginapi.ContainerAllocateResponse
	var dev *pluginapi.DeviceSpec

	p.devs = append(p.devs, &pluginapi.Device{
		ID:     FUSEPath + strconv.Itoa(p.counter),
		Health: pluginapi.Healthy,
	})

	glog.V(3).Infof("Allocated virtual fuse device %d", p.counter)

	p.counter += 1
	p.update <- message{}

	car = pluginapi.ContainerAllocateResponse{}

	dev = new(pluginapi.DeviceSpec)
	dev.HostPath = FUSEPath
	dev.ContainerPath = FUSEPath
	dev.Permissions = "rw"
	car.Devices = append(car.Devices, dev)

	response.ContainerResponses = append(response.ContainerResponses, &car)

	return &response, nil
}

// GetDevicePluginOptions returns options to be communicated with Device
// Manager
func (p *FusePlugin) GetDevicePluginOptions(context.Context, *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return nil, nil
}

// PreStartContainer is called, if indicated by Device Plugin during registration phase,
// before each container start. Device plugin can run device specific operations
// such as resetting the device before making devices available to the container
func (p *FusePlugin) PreStartContainer(context.Context, *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return nil, nil
}
