// Copyright 2014-2015 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//	http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package api

import (
	"fmt"
	"strconv"
	"sync"
	"time"
)

type TaskStatus int32
type ContainerStatus int32

const (
	TaskStatusNone TaskStatus = iota
	TaskStatusUnknown
	TaskPulled
	TaskCreated
	TaskRunning
	TaskStopped
	TaskDead
)

const (
	ContainerStatusNone ContainerStatus = iota
	ContainerStatusUnknown
	ContainerPulled
	ContainerCreated
	ContainerRunning
	ContainerStopped
	ContainerDead

	ContainerZombie // Impossible status to use as a virtual 'max'
)

type PortBinding struct {
	ContainerPort uint16
	HostPort      uint16
	BindIp        string
}

type TaskOverrides struct{}

type Task struct {
	Arn        string
	Overrides  TaskOverrides `json:"-"`
	Family     string
	Version    string
	Containers []*Container

	DesiredStatus TaskStatus
	KnownStatus   TaskStatus
	KnownTime     time.Time

	SentStatus TaskStatus

	containersByNameLock sync.Mutex
	containersByName     map[string]*Container
}

type ContainerStateChange struct {
	TaskArn       string
	ContainerName string
	Status        ContainerStatus

	Reason       string
	ExitCode     *int
	PortBindings []PortBinding

	TaskStatus TaskStatus // TaskStatusNone if this does not result in a task state change

	Task      *Task
	Container *Container
}

func (t *Task) String() string {
	res := fmt.Sprintf("%s-%s %s, Overrides: %s Status: %s(%s)", t.Family, t.Version, t.Arn, t.Overrides, t.KnownStatus.String(), t.DesiredStatus.String())
	res += " Containers: "
	for _, c := range t.Containers {
		res += c.Name + ","
	}
	return res
}

type ContainerOverrides struct {
	Command *[]string `json:"command"`
}

type Container struct {
	Name        string
	Image       string
	Command     []string
	Cpu         uint
	Memory      uint
	Links       []string
	VolumesFrom []VolumeFrom  `json:"volumesFrom"`
	Ports       []PortBinding `json:"portMappings"`
	Essential   bool
	EntryPoint  *[]string
	Environment map[string]string  `json:"environment"`
	Overrides   ContainerOverrides `json:"overrides"`

	DesiredStatus ContainerStatus `json:"desiredStatus"`
	KnownStatus   ContainerStatus

	AppliedStatus ContainerStatus
	ApplyingError error

	SentStatus ContainerStatus

	KnownExitCode     *int
	KnownPortBindings []PortBinding

	// Not upstream; todo move this out into a wrapper type
	StatusLock sync.Mutex
}

// VolumeFrom is a volume which references another container as its source.
type VolumeFrom struct {
	SourceContainer string `json:"sourceContainer"`
	ReadOnly        bool   `json:"readOnly"`
}

func (c *Container) String() string {
	res := fmt.Sprintf("%s(%s) - Status: %s", c.Name, c.Image, c.KnownStatus.String())
	if c.KnownExitCode != nil {
		res += "; Exited " + strconv.Itoa(*c.KnownExitCode)
	}
	return res
}

type Resource struct {
	Name        string
	Type        string
	DoubleValue float64
	LongValue   int64
}

// This is a mapping between containers-as-docker-knows-them and
// containers-as-we-know-them.
// This is primarily used in DockerState, but lives here such that tasks and
// containers know how to convert themselves into Docker's desired config format
type DockerContainer struct {
	DockerId   string
	DockerName string // needed for linking

	Container *Container
}

func (dc *DockerContainer) String() string {
	if dc == nil {
		return "nil"
	}
	return fmt.Sprintf("Id: %s, Name: %s, Container: %s", dc.DockerId, dc.DockerName, dc.Container.String())
}
