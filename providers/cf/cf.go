package cf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/pkg/errors"
	"github.com/virtual-kubelet/virtual-kubelet/manager"
	"github.com/virtual-kubelet/virtual-kubelet/providers"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CFProvider implements the virtual-kubelet provider interface and stores pods in memory.
type CFProvider struct {
	nodeName           string
	operatingSystem    string
	internalIP         string
	daemonEndpointPort int32
	pods               map[string]*v1.Pod
	providerConfig     providerConfig
	cfClient           *cfclient.Client
	Org                cfclient.Org
	Space              cfclient.Space
}

type DockerAppCreateRequest struct {
	Name        string `json:"name"`
	SpaceGuid   string `json:"space_guid"`
	DockerImage string `json:"docker_image"`
	Diego       bool   `json:"diego"`
}

func (p *CFProvider) CreateDockerApp(req DockerAppCreateRequest) (cfclient.App, error) {
	var appResp cfclient.AppResource
	buf := bytes.NewBuffer(nil)
	err := json.NewEncoder(buf).Encode(req)
	if err != nil {
		return cfclient.App{}, err
	}
	r := p.cfClient.NewRequestWithBody("POST", "/v2/apps", buf)
	resp, err := p.cfClient.DoRequest(r)
	if err != nil {
		return cfclient.App{}, errors.Wrapf(err, "Error creating app %s", req.Name)
	}
	if resp.StatusCode != http.StatusCreated {
		return cfclient.App{}, errors.Wrapf(err, "Error creating app %s, response code: %d", req.Name, resp.StatusCode)
	}
	resBody, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return cfclient.App{}, errors.Wrapf(err, "Error reading app %s http response body", req.Name)
	}
	err = json.Unmarshal(resBody, &appResp)
	if err != nil {
		return cfclient.App{}, errors.Wrapf(err, "Error deserializing app %s response", req.Name)
	}
	return mergeAppResource(appResp), nil

}

func mergeAppResource(app cfclient.AppResource) cfclient.App {
	app.Entity.Guid = app.Meta.Guid
	app.Entity.CreatedAt = app.Meta.CreatedAt
	app.Entity.UpdatedAt = app.Meta.UpdatedAt
	app.Entity.SpaceData.Entity.Guid = app.Entity.SpaceData.Meta.Guid
	app.Entity.SpaceData.Entity.OrgData.Entity.Guid = app.Entity.SpaceData.Entity.OrgData.Meta.Guid
	return app.Entity
}

// NewCFProvider creates a new CFProvider
func NewCFProvider(config string, rm *manager.ResourceManager, nodeName, operatingSystem string, internalIP string, daemonEndpointPort int32) (*CFProvider, error) {

	provider := CFProvider{
		nodeName:           nodeName,
		operatingSystem:    operatingSystem,
		internalIP:         internalIP,
		daemonEndpointPort: daemonEndpointPort,
		pods:               make(map[string]*v1.Pod),
	}

	if config != "" {
		f, err := os.Open(config)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		if err := provider.loadConfig(f); err != nil {
			return nil, err
		}
	}

	cfConfig := &cfclient.Config{
		ApiAddress:        provider.providerConfig.CFAPI,
		Username:          provider.providerConfig.Username,
		Password:          provider.providerConfig.Password,
		SkipSslValidation: true,
	}

	client, err := cfclient.NewClient(cfConfig)
	if err != nil {
		return &CFProvider{}, err
	}

	provider.cfClient = client
	fmt.Printf("%#v\n", &provider)

	err = provider.EnsureOrgAndSpace()
	if err != nil {
		return nil, err
	}

	return &provider, nil
}

func (p *CFProvider) createApp(appRequest DockerAppCreateRequest) error {

	app, err := p.CreateDockerApp(appRequest)

	// app, err := provider.cfClient.CreateApp(appCreateRequest)
	if err != nil {
		return err
	}

	appUpdateResource := cfclient.AppUpdateResource{
		State: "STARTED",
	}

	_, err = p.cfClient.UpdateApp(app.Guid, appUpdateResource)

	if err != nil {
		return err
	}
	return nil

}

// CreatePod accepts a Pod definition and stores it in memory.
func (p *CFProvider) CreatePod(pod *v1.Pod) error {
	log.Printf("receive CreatePod %q\n", pod.Name)

	key, err := buildKey(pod)
	if err != nil {
		return err
	}

	containers := p.getContainers(pod)
	if len(containers) != 1 {
		return errors.New("too many containers")
	}

	appCreateRequest := DockerAppCreateRequest{
		Name:        pod.Name,
		SpaceGuid:   p.Space.Guid,
		DockerImage: containers[0],
		Diego:       true,
	}

	err = p.createApp(appCreateRequest)
	if err != nil {
		return err
	}

	p.pods[key] = pod

	return nil
}

func (p *CFProvider) getContainers(pod *v1.Pod) []string {
	containers := make([]string, 0, len(pod.Spec.Containers))
	for _, container := range pod.Spec.Containers {
		c := container.Image

		// c.EnvironmentVariables = make([]aci.EnvironmentVariable, 0, len(container.Env))
		// for _, e := range container.Env {
		// 	c.EnvironmentVariables = append(c.EnvironmentVariables, aci.EnvironmentVariable{
		// 		Name:  e.Name,
		// 		Value: e.Value,
		// 	})
		// }

		// cpuRequest := 1.00
		// if _, ok := container.Resources.Requests[v1.ResourceCPU]; ok {
		// 	cpuRequest = float64(container.Resources.Requests.Cpu().MilliValue()/10.00) / 100.00
		// 	if cpuRequest < 0.01 {
		// 		cpuRequest = 0.01
		// 	}
		// }

		// memoryRequest := 1.50
		// if _, ok := container.Resources.Requests[v1.ResourceMemory]; ok {
		// 	memoryRequest = float64(container.Resources.Requests.Memory().Value()/100000000.00) / 10.00
		// 	if memoryRequest < 0.10 {
		// 		memoryRequest = 0.10
		// 	}
		// }

		// c.Resources = aci.ResourceRequirements{
		// 	Requests: &aci.ResourceRequests{
		// 		CPU:        cpuRequest,
		// 		MemoryInGB: memoryRequest,
		// 	},
		// }

		containers = append(containers, c)
	}
	return containers
}

func (p *CFProvider) UpdatePod(pod *v1.Pod) error {
	log.Printf("receive UpdatePod %q\n", pod.Name)

	key, err := buildKey(pod)
	if err != nil {
		return err
	}

	p.pods[key] = pod

	return nil
}

// DeletePod deletes the specified pod out of memory.
func (p *CFProvider) DeletePod(pod *v1.Pod) (err error) {
	log.Printf("receive DeletePod %q\n", pod.Name)

	key, err := buildKey(pod)
	if err != nil {
		return err
	}

	delete(p.pods, key)

	return nil
}

// GetPod returns a pod by name that is stored in memory.
func (p *CFProvider) GetPod(namespace, name string) (pod *v1.Pod, err error) {
	log.Printf("receive GetPod %q\n", name)

	key, err := buildKeyFromNames(namespace, name)
	if err != nil {
		return nil, err
	}

	if pod, ok := p.pods[key]; ok {
		return pod, nil
	}

	return nil, nil
}

// GetContainerLogs retrieves the logs of a container by name from the provider.
func (p *CFProvider) GetContainerLogs(namespace, podName, containerName string, tail int) (string, error) {
	log.Printf("receive GetContainerLogs %q\n", podName)
	return "", nil
}

// GetPodStatus returns the status of a pod by name that is "running".
// returns nil if a pod by that name is not found.
func (p *CFProvider) GetPodStatus(namespace, name string) (*v1.PodStatus, error) {
	log.Printf("receive GetPodStatus %q\n", name)

	now := metav1.NewTime(time.Now())

	status := &v1.PodStatus{
		Phase:     v1.PodRunning,
		HostIP:    "1.2.3.4",
		PodIP:     "5.6.7.8",
		StartTime: &now,
		Conditions: []v1.PodCondition{
			{
				Type:   v1.PodInitialized,
				Status: v1.ConditionTrue,
			},
			{
				Type:   v1.PodReady,
				Status: v1.ConditionTrue,
			},
			{
				Type:   v1.PodScheduled,
				Status: v1.ConditionTrue,
			},
		},
	}

	pod, err := p.GetPod(namespace, name)
	if err != nil {
		return status, err
	}

	for _, container := range pod.Spec.Containers {
		status.ContainerStatuses = append(status.ContainerStatuses, v1.ContainerStatus{
			Name:         container.Name,
			Image:        container.Image,
			Ready:        true,
			RestartCount: 0,
			State: v1.ContainerState{
				Running: &v1.ContainerStateRunning{
					StartedAt: now,
				},
			},
		})
	}

	return status, nil
}

// GetPods returns a list of all pods known to be "running".
func (p *CFProvider) GetPods() ([]*v1.Pod, error) {
	log.Printf("receive GetPods\n")

	var pods []*v1.Pod

	for _, pod := range p.pods {
		pods = append(pods, pod)
	}

	return pods, nil
}

// Capacity returns a resource list containing the capacity limits.
func (p *CFProvider) Capacity() v1.ResourceList {
	// TODO: These should be configurable
	return v1.ResourceList{
		"cpu":    resource.MustParse("20"),
		"memory": resource.MustParse("100Gi"),
		"pods":   resource.MustParse("20"),
	}
}

// NodeConditions returns a list of conditions (Ready, OutOfDisk, etc), for updates to the node status
// within Kubernetes.
func (p *CFProvider) NodeConditions() []v1.NodeCondition {
	// TODO: Make this configurable
	return []v1.NodeCondition{
		{
			Type:               "Ready",
			Status:             v1.ConditionTrue,
			LastHeartbeatTime:  metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             "KubeletReady",
			Message:            "kubelet is ready.",
		},
		{
			Type:               "OutOfDisk",
			Status:             v1.ConditionFalse,
			LastHeartbeatTime:  metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             "KubeletHasSufficientDisk",
			Message:            "kubelet has sufficient disk space available",
		},
		{
			Type:               "MemoryPressure",
			Status:             v1.ConditionFalse,
			LastHeartbeatTime:  metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             "KubeletHasSufficientMemory",
			Message:            "kubelet has sufficient memory available",
		},
		{
			Type:               "DiskPressure",
			Status:             v1.ConditionFalse,
			LastHeartbeatTime:  metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             "KubeletHasNoDiskPressure",
			Message:            "kubelet has no disk pressure",
		},
		{
			Type:               "NetworkUnavailable",
			Status:             v1.ConditionFalse,
			LastHeartbeatTime:  metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             "RouteCreated",
			Message:            "RouteController created a route",
		},
	}

}

// NodeAddresses returns a list of addresses for the node status
// within Kubernetes.
func (p *CFProvider) NodeAddresses() []v1.NodeAddress {
	return []v1.NodeAddress{
		{
			Type:    "InternalIP",
			Address: p.internalIP,
		},
	}
}

// NodeDaemonEndpoints returns NodeDaemonEndpoints for the node status
// within Kubernetes.
func (p *CFProvider) NodeDaemonEndpoints() *v1.NodeDaemonEndpoints {
	return &v1.NodeDaemonEndpoints{
		KubeletEndpoint: v1.DaemonEndpoint{
			Port: p.daemonEndpointPort,
		},
	}
}

// OperatingSystem returns the operating system for this provider.
// This is a noop to default to Linux for now.
func (p *CFProvider) OperatingSystem() string {
	return providers.OperatingSystemLinux
}

func buildKeyFromNames(namespace string, name string) (string, error) {
	return fmt.Sprintf("%s-%s", namespace, name), nil
}

// buildKey is a helper for building the "key" for the providers pod store.
func buildKey(pod *v1.Pod) (string, error) {
	if pod.ObjectMeta.Namespace == "" {
		return "", fmt.Errorf("pod namespace not found")
	}

	if pod.ObjectMeta.Name == "" {
		return "", fmt.Errorf("pod name not found")
	}

	return buildKeyFromNames(pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
}

func (p *CFProvider) EnsureOrgAndSpace() error {

	var org cfclient.Org
	var space cfclient.Space

	org, err := p.cfClient.GetOrgByName(p.providerConfig.Org)
	if err != nil {
		orgRequest := cfclient.OrgRequest{Name: p.providerConfig.Org}

		org, err = p.cfClient.CreateOrg(orgRequest)
		if err != nil {
			return err
		}
	}

	space, err = p.cfClient.GetSpaceByName(p.providerConfig.Space, org.Guid)
	if err != nil {
		spaceRequest := cfclient.SpaceRequest{
			Name:             p.providerConfig.Space,
			OrganizationGuid: org.Guid,
		}

		space, err = p.cfClient.CreateSpace(spaceRequest)
		if err != nil {
			return err
		}
	}

	p.Org = org
	p.Space = space

	return nil
}
