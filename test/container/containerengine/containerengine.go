/*
© Copyright IBM Corporation 2017, 2023

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
package containerengine

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type ContainerInterface interface {
	ContainerCreate(config *ContainerConfig, hostConfig *ContainerHostConfig, networkingConfig *ContainerNetworkSettings, containerName string) (string, error)
	ContainerStop(container string, timeout *time.Duration) error
	ContainerKill(container string, signal string) error
	ContainerRemove(container string, options ContainerRemoveOptions) error
	ContainerStart(container string, options ContainerStartOptions) error
	ContainerWait(ctx context.Context, container string, condition string) (<-chan int64, <-chan error)
	GetContainerLogs(ctx context.Context, container string, options ContainerLogsOptions) (string, error)
	CopyFromContainer(container, srcPath string) ([]byte, error)

	GetContainerPort(ID string, hostPort int) (string, error)
	GetContainerIPAddress(ID string) (string, error)
	ContainerInspectWithFormat(format string, ID string) (string, error)
	ExecContainer(ID string, user string, cmd []string) (int, string)
	GetMQVersion(image string) (string, error)
	ContainerInspect(containerID string) (ContainerDetails, error)

	NetworkCreate(name string, options NetworkCreateOptions) (string, error)
	NetworkRemove(network string) error

	VolumeCreate(options VolumeCreateOptions) (string, error)
	VolumeRemove(volumeID string, force bool) error

	ImageBuild(context io.Reader, tag string, dockerfilename string) (string, error)
	ImageRemove(image string, options ImageRemoveOptions) (bool, error)
	ImageInspectWithFormat(format string, ID string) (string, error)
}

type ContainerClient struct {
	ContainerTool string
	Version       string
}

// objects
var objVolume = "volume"
var objImage = "image"
var objPort = "port"
var objNetwork = "network"

// verbs
var listContainers = "ps"
var listImages = "images"
var create = "create"
var startContainer = "start"
var waitContainer = "wait"
var execContainer = "exec"
var getLogs = "logs"
var stopContainer = "stop"
var remove = "rm"
var inspect = "inspect"
var copyFile = "cp"
var build = "build"
var killContainer = "kill"

// args
var argEntrypoint = "--entrypoint"
var argUser = "--user"
var argExpose = "--expose"
var argVolume = "--volume"
var argPublish = "--publish"
var argPrivileged = "--privileged"
var argAddCapability = "--cap-add"
var argDropCapability = "--cap-drop"
var argName = "--name"
var argCondition = "--condition"
var argEnvironmentVariable = "--env"
var argTail = "--tail"
var argForce = "--force"
var argVolumes = "--volumes"
var argHostname = "--hostname"
var argDriver = "--driver"
var argFile = "--file"
var argQuiet = "--quiet"
var argTag = "--tag"
var argFormat = "--format"
var argNetwork = "--network"
var argSecurityOptions = "--security-opt"
var argSignal = "--signal"
var argReadOnlyRootfs = "--read-only"

// generic
var toolVersion = "version"
var ContainerStateNotRunning = "not-running"
var ContainerStateStopped = "stopped"

type ContainerConfig struct {
	Image        string
	Hostname     string
	User         string
	Entrypoint   []string
	Env          []string
	ExposedPorts []string
}

type ContainerDetails struct {
	ID         string
	Name       string
	Image      string
	Path       string
	Args       []string
	Config     ContainerConfig
	HostConfig ContainerHostConfig
}

type ContainerDetailsLogging struct {
	ID      string
	Name    string
	Image   string
	Path    string
	Args    []string
	CapAdd  []string
	CapDrop []string
	User    string
	Env     []string
}

type ContainerHostConfig struct {
	Binds          []string      // Bindings onto a volume
	PortBindings   []PortBinding //Bindings from a container port to a port on the host
	Privileged     bool          // Give extended privileges to container
	CapAdd         []string      // Linux capabilities to add to the container
	CapDrop        []string      // Linux capabilities to drop from the container
	SecurityOpt    []string
	ReadOnlyRootfs bool // Readonly root file system
}

type ContainerNetworkSettings struct {
	Networks []string // A list of networks to connect the container to
}

type ContainerRemoveOptions struct {
	Force         bool
	RemoveVolumes bool
}

type ContainerStartOptions struct {
}

type NetworkCreateOptions struct {
}

type ContainerLogsOptions struct {
}

type ImageRemoveOptions struct {
	Force bool
}

type VolumeCreateOptions struct {
	Name   string
	Driver string
}

// Binding from a container port to a port on the host
type PortBinding struct {
	HostIP        string
	HostPort      string //Port to map to on the host
	ContainerPort string //Exposed port on the container
}

// NewContainerClient returns a new container client
// Defaults to using podman
func NewContainerClient() ContainerClient {
	tool, set := os.LookupEnv("COMMAND")
	if !set {
		tool = "podman"
	}
	return ContainerClient{
		ContainerTool: tool,
		Version:       GetContainerToolVersion(tool),
	}
}

// GetContainerToolVersion returns the version of the container tool being used
func GetContainerToolVersion(containerTool string) string {
	if containerTool == "docker" {
		args := []string{"version", "--format", "'{{.Client.Version}}'"}
		v, err := exec.Command("docker", args...).Output()
		if err != nil {
			return "0.0.0"
		}
		return string(v)
	} else if containerTool == "podman" {
		//Default to checking the version of podman
		args := []string{"version", "--format", "'{{.Version}}'"}
		v, err := exec.Command("podman", args...).Output()
		if err != nil {
			return "0.0.0"
		}
		return string(v)
	}
	return "0.0.0"
}

// GetMQVersion returns the MQ version of a given container image
func (cli ContainerClient) GetMQVersion(image string) (string, error) {
	v, err := cli.ImageInspectWithFormat("{{.Config.Labels.version}}", image)
	if err != nil {
		return "", err
	}
	return v, nil
}

// ImageInspectWithFormat inspects an image with a given formatting string
func (cli ContainerClient) ImageInspectWithFormat(format string, ID string) (string, error) {
	args := []string{
		objImage,
		inspect,
		ID,
	}
	if format != "" {
		args = append(args, []string{argFormat, format}...)
	}
	output, err := exec.Command(cli.ContainerTool, args...).Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// ContainerInspectWithFormat inspects a container with a given formatting string
func (cli ContainerClient) ContainerInspectWithFormat(format string, ID string) (string, error) {
	args := []string{
		inspect,
		ID,
	}
	if format != "" {
		args = append(args, []string{argFormat, format}...)
	}
	output, err := exec.Command(cli.ContainerTool, args...).Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// GetContainerPort gets the ports on a container
func (cli ContainerClient) GetContainerPort(ID string, hostPort int) (string, error) {
	args := []string{
		objPort,
		ID,
		strconv.Itoa(hostPort),
	}
	output, err := exec.Command(cli.ContainerTool, args...).Output()
	if err != nil {
		return "", err
	}
	o := SanitizeString(string(output))
	return strings.Split((o), ":")[1], nil
}

// GetContainerIPAddress gets the IP address of a container
func (cli ContainerClient) GetContainerIPAddress(ID string) (string, error) {
	v, err := cli.ContainerInspectWithFormat("{{.NetworkSettings.IPAddress}}", ID)
	if err != nil {
		return "", err
	}
	return v, nil
}

// CopyFromContainer copies a file from a container and returns its contents
func (cli ContainerClient) CopyFromContainer(container, srcPath string) ([]byte, error) {
	tmpDir, err := os.MkdirTemp("", "tmp")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)
	args := []string{
		copyFile,
		container + ":" + srcPath,
		tmpDir + "/.",
	}
	_, err = exec.Command(cli.ContainerTool, args...).CombinedOutput()
	if err != nil {
		return nil, err
	}
	//Get file name
	fname := filepath.Base(srcPath)
	data, err := os.ReadFile(filepath.Join(tmpDir, fname))
	if err != nil {
		return nil, err
	}

	//Remove the file
	err = os.Remove(filepath.Join(tmpDir, fname))
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (cli ContainerClient) ContainerInspect(containerID string) (ContainerDetails, error) {
	args := []string{
		inspect,
		containerID,
	}
	output, err := exec.Command(cli.ContainerTool, args...).Output()
	if err != nil {
		return ContainerDetails{}, err
	}

	var container ContainerDetails
	err = json.Unmarshal(output, &container)
	if err != nil {
		return ContainerDetails{}, err
	}
	return container, err
}

func (cli ContainerClient) ContainerStop(container string, timeout *time.Duration) error {
	args := []string{
		stopContainer,
		container,
	}
	_, err := exec.Command(cli.ContainerTool, args...).Output()
	return err
}

func (cli ContainerClient) ContainerKill(container string, signal string) error {
	args := []string{
		killContainer,
		container,
	}
	if signal != "" {
		args = append(args, []string{argSignal, signal}...)
	}
	_, err := exec.Command(cli.ContainerTool, args...).Output()
	return err
}

func (cli ContainerClient) ContainerRemove(container string, options ContainerRemoveOptions) error {
	args := []string{
		remove,
		container,
	}
	if options.Force {
		args = append(args, argForce)
	}
	if options.RemoveVolumes {
		args = append(args, argVolumes)
	}
	_, err := exec.Command(cli.ContainerTool, args...).Output()
	if err != nil {
		//Silently error as the exit code 125 is present on sucessful deletion
		if strings.Contains(err.Error(), "125") {
			return nil
		}
		return err
	}
	return nil
}

func (cli ContainerClient) ExecContainer(ID string, user string, cmd []string) (int, string) {
	args := []string{
		execContainer,
	}
	if user != "" {
		args = append(args, []string{argUser, user}...)
	}
	args = append(args, ID)
	args = append(args, cmd...)
	ctx := context.Background()
	output, err := exec.CommandContext(ctx, cli.ContainerTool, args...).CombinedOutput()
	if err != nil {
		if err.(*exec.ExitError) != nil {
			return err.(*exec.ExitError).ExitCode(), string(output)
		} else {
			return 9897, string(output)
		}
	}
	return 0, string(output)
}

func (cli ContainerClient) ContainerStart(container string, options ContainerStartOptions) error {
	args := []string{
		startContainer,
		container,
	}
	_, err := exec.Command(cli.ContainerTool, args...).Output()
	return err
}

// ContainerWait starts waiting for a container. It returns an int64 channel for receiving an exit code and an error channel for receiving errors.
// The channels returned from this function should be used to receive the results from the wait command.
func (cli ContainerClient) ContainerWait(ctx context.Context, container string, condition string) (<-chan int64, <-chan error) {
	args := []string{
		waitContainer,
		container,
	}
	if cli.ContainerTool == "podman" {
		if condition == ContainerStateNotRunning {
			condition = ContainerStateStopped
		}
		args = append(args, []string{argCondition, string(condition)}...)
	}

	resultC := make(chan int64)
	errC := make(chan error, 1)

	output, err := exec.CommandContext(ctx, cli.ContainerTool, args...).Output()
	if err != nil {
		errC <- err
		return resultC, errC
	}

	go func() {
		out := strings.TrimSuffix(string(output), "\n")
		exitCode, err := strconv.Atoi(out)
		if err != nil {
			errC <- err
			return
		}
		resultC <- int64(exitCode)
	}()

	return resultC, errC
}

func (cli ContainerClient) GetContainerLogs(ctx context.Context, container string, options ContainerLogsOptions) (string, error) {
	args := []string{
		getLogs,
		container,
	}
	output, err := exec.Command(cli.ContainerTool, args...).CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func (cli ContainerClient) NetworkCreate(name string, options NetworkCreateOptions) (string, error) {
	args := []string{
		objNetwork,
		create,
	}
	netID, err := exec.Command(cli.ContainerTool, args...).CombinedOutput()
	if err != nil {
		return "", err
	}
	networkID := SanitizeString(string(netID))

	return networkID, nil
}

func (cli ContainerClient) NetworkRemove(network string) error {
	args := []string{
		objNetwork,
		remove,
	}
	_, err := exec.Command(cli.ContainerTool, args...).CombinedOutput()
	return err
}

func (cli ContainerClient) VolumeCreate(options VolumeCreateOptions) (string, error) {
	args := []string{
		objVolume,
		create,
		options.Name,
	}
	if options.Driver != "" {
		args = append(args, []string{argDriver, options.Driver}...)
	}
	output, err := exec.Command(cli.ContainerTool, args...).Output()
	if err != nil {
		return "", err
	}
	name := SanitizeString(string(output))
	return name, nil
}

func (cli ContainerClient) VolumeRemove(volumeID string, force bool) error {
	args := []string{
		objVolume,
		remove,
		volumeID,
	}
	if force {
		args = append(args, argForce)
	}
	_, err := exec.Command(cli.ContainerTool, args...).Output()
	return err
}

func (cli ContainerClient) ImageBuild(context io.Reader, tag string, dockerfilename string) (string, error) {
	args := []string{
		objImage,
		build,
	}
	//dockerfilename includes the path to the dockerfile
	//When using podman use the full path including the name of the Dockerfile
	if cli.ContainerTool == "podman" {
		args = append(args, []string{argFile, dockerfilename}...)
	}
	if tag != "" {
		args = append(args, []string{argTag, tag}...)
	}
	args = append(args, argQuiet)
	//When using docker remove the name 'DockerFile' from the string
	if cli.ContainerTool == "docker" {
		dfn := strings.ReplaceAll(dockerfilename, "Dockerfile", "")
		args = append(args, dfn)
	}
	output, err := exec.Command(cli.ContainerTool, args...).Output()
	if err != nil {
		return "", err
	}
	sha := SanitizeString(string(output))
	return sha, nil
}

func (cli ContainerClient) ImageRemove(image string, options ImageRemoveOptions) (bool, error) {
	args := []string{
		objImage,
		remove,
		image,
	}
	if options.Force {
		args = append(args, argForce)
	}
	_, err := exec.Command(cli.ContainerTool, args...).Output()
	if err != nil {
		return false, err
	}
	return true, nil
}

func (cli ContainerClient) ContainerCreate(config *ContainerConfig, hostConfig *ContainerHostConfig, networkingConfig *ContainerNetworkSettings, containerName string) (string, error) {
	args := []string{
		create,
		argName,
		containerName,
	}
	args = getHostConfigArgs(args, hostConfig)
	args = getNetworkConfigArgs(args, networkingConfig)
	args = getContainerConfigArgs(args, config, cli.ContainerTool)
	output, err := exec.Command(cli.ContainerTool, args...).Output()
	lines := strings.Split(strings.ReplaceAll(string(output), "\r\n", "\n"), "\n")
	if err != nil {
		return lines[0], err
	}
	return lines[0], nil
}

// getContainerConfigArgs converts a ContainerConfig into a set of cli arguments
func getContainerConfigArgs(args []string, config *ContainerConfig, toolName string) []string {
	argList := []string{}
	if config.Entrypoint != nil && toolName == "podman" {
		entrypoint := "[\""
		for i, commandPart := range config.Entrypoint {
			if i != len(config.Entrypoint)-1 {
				entrypoint += commandPart + "\",\""
			} else {
				//terminate list
				entrypoint += commandPart + "\"]"
			}
		}
		args = append(args, []string{argEntrypoint, entrypoint}...)
	}
	if config.Entrypoint != nil && toolName == "docker" {
		ep1 := ""
		for i, commandPart := range config.Entrypoint {
			if i == 0 {
				ep1 = commandPart
			} else {
				argList = append(argList, commandPart)
			}
		}
		args = append(args, []string{argEntrypoint, ep1}...)
	}
	if config.User != "" {
		args = append(args, []string{argUser, config.User}...)
	}
	if config.ExposedPorts != nil {
		for _, port := range config.ExposedPorts {
			args = append(args, []string{argExpose, port}...)
		}
	}
	if config.Hostname != "" {
		args = append(args, []string{argHostname, config.Hostname}...)
	}
	for _, env := range config.Env {
		args = append(args, []string{argEnvironmentVariable, env}...)
	}
	if config.Image != "" {
		args = append(args, config.Image)
	}
	if config.Entrypoint != nil && toolName == "docker" {
		args = append(args, argList...)
	}
	return args
}

// getHostConfigArgs converts a ContainerHostConfig into a set of cli arguments
func getHostConfigArgs(args []string, hostConfig *ContainerHostConfig) []string {
	if hostConfig.Binds != nil {
		for _, volume := range hostConfig.Binds {
			args = append(args, []string{argVolume, volume}...)
		}
	}
	if hostConfig.PortBindings != nil {
		for _, binding := range hostConfig.PortBindings {
			pub := binding.HostIP + ":" + binding.HostPort + ":" + binding.ContainerPort
			args = append(args, []string{argPublish, pub}...)
		}
	}
	if hostConfig.Privileged {
		args = append(args, []string{argPrivileged}...)
	}
	if hostConfig.CapAdd != nil {
		for _, capability := range hostConfig.CapAdd {
			args = append(args, []string{argAddCapability, string(capability)}...)
		}
	}
	if hostConfig.CapDrop != nil {
		for _, capability := range hostConfig.CapDrop {
			args = append(args, []string{argDropCapability, string(capability)}...)
		}
	}
	if hostConfig.SecurityOpt != nil {
		for _, securityOption := range hostConfig.SecurityOpt {
			args = append(args, []string{argSecurityOptions, string(securityOption)}...)
		}
	}
	// Add --read-only flag to enable Read Only Root File system on the container
	if hostConfig.ReadOnlyRootfs {
		args = append(args, []string{argReadOnlyRootfs}...)
	}

	return args
}

// getNetworkConfigArgs converts a set of ContainerNetworkSettings into a set of cli arguments
func getNetworkConfigArgs(args []string, networkingConfig *ContainerNetworkSettings) []string {
	if networkingConfig.Networks != nil {
		for _, netID := range networkingConfig.Networks {
			args = append(args, []string{argNetwork, netID}...)
		}
	}
	return args
}

func SanitizeString(s string) string {
	s = strings.Replace(s, " ", "", -1)
	s = strings.Replace(s, "\t", "", -1)
	s = strings.Replace(s, "\n", "", -1)
	return s
}
