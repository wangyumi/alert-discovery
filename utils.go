package main

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"text/template"

	// "github.com/golang/glog"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/resource"
)

var (
	warningFactor  float64 = *warningThreshold
	criticalFactor float64 = *criticalThreshold
)

const (
	memRuleTemplate = `
ALERT MemoryUsageHigh
    IF sum(container_memory_usage_bytes{id="[[ .ID ]]"}) > [[ .Target ]]
    FOR 5m
    LABELS { severity = "[[ .Severity ]]", kubernetes_container_name = "[[ .Container ]]", kubernetes_pod_name = "[[ .PodName ]]", kubernetes_namespace = "[[ .Namespace ]]", id = "[[ .ID ]]"}
    ANNOTATIONS {
        summary = "Container [[ .Namespace ]]/[[ .PodName ]]/[[ .Container ]] memory usage high",
        description = "Container [[ .Namespace ]]/[[ .PodName ]]/[[ .Container ]] has high memory usage of {{ $value }}",
    }
`
	cpuRuleTemplate = `
ALERT CPUUsageHigh
    IF sum(rate(container_cpu_usage_seconds_total{id="[[ .ID ]]"}[5m])) > [[ .Target ]]
    FOR 5m
    LABELS { severity = "[[ .Severity ]]", kubernetes_container_name = "[[ .Container ]]", kubernetes_pod_name = "[[ .PodName ]]", kubernetes_namespace = "[[ .Namespace ]]", id = "[[ .ID ]]"}
    ANNOTATIONS {
        summary = "Container [[ .Namespace ]]/[[ .PodName ]]/[[ .Container ]] cpu usage high",
        description = "Container [[ .Namespace ]]/[[ .PodName ]]/[[ .Container ]] has high cpu usage of {{ $value }}",
    }
`
)

type capacity struct {
	CPU    *resource.Quantity // cpu represented as milli value
	Memory *resource.Quantity // memory represented as bytes
	ID     string             // docker id of container
}

type containerStore map[string]*capacity

type podStore map[string]containerStore

type store struct {
	sync.Mutex
	podStore
	memTmpl *template.Template
	cpuTmpl *template.Template
}

func newStore() *store {
	return &store{
		podStore: podStore{},
		cpuTmpl:  template.Must(template.New("cpuRuleTemplate").Delims("[[", "]]").Parse(cpuRuleTemplate)),
		memTmpl:  template.Must(template.New("memRuleTemplate").Delims("[[", "]]").Parse(memRuleTemplate)),
	}
}

func (p *store) Update(pod *api.Pod) bool {
	p.Lock()
	defer p.Unlock()
	podKey := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
	changed := false
	for _, c := range pod.Spec.Containers {
		// Cause the request does not really restricts resource to be used, we use limit to generate alert rules
		cpu := c.Resources.Limits.Cpu()
		memory := c.Resources.Limits.Memory()
		id := fetchID(pod, c.Name)
		if p.updateStoreUnsafe(id, podKey, c.Name, cpu, memory) {
			changed = true
		}
	}
	return changed
}

func fetchID(pod *api.Pod, name string) string {
	for _, status := range pod.Status.ContainerStatuses {
		if status.Name == name {
			splited := strings.Split(status.ContainerID, "//")
			if len(splited) > 0 {
				return fmt.Sprintf("/docker/%s", splited[len(splited)-1])
			}
		}
	}
	return ""
}

func (p *store) CAS(cap *capacity, cpu, memory *resource.Quantity, id string) bool {
	changed := false
	if cap.ID != id {
		cap.ID = id
		changed = true
	}
	if cap.CPU.Cmp(*cpu) != 0 {
		cap.CPU = cpu
		changed = true
	}
	if cap.Memory.Cmp(*memory) != 0 {
		cap.Memory = memory
		changed = true
	}
	return changed
}

func (p *store) updateStoreUnsafe(id, key, name string, cpu *resource.Quantity, memory *resource.Quantity) bool {
	if entry, ok := p.podStore[key]; !ok {
		p.podStore[key] = containerStore{}
		p.podStore[key][name] = &capacity{CPU: cpu, Memory: memory, ID: id}
		return true
	} else if _, ok := entry[name]; !ok {
		entry[name] = &capacity{CPU: cpu, Memory: memory, ID: id}
		return true
	} else {
		return p.CAS(entry[name], cpu, memory, id)
	}
}

func (p *store) Delete(pod *api.Pod) {
	p.Lock()
	defer p.Unlock()
	podKey := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
	p.deleteUnsafe(podKey)
}

func (p *store) deleteUnsafe(key string) {
	if _, ok := p.podStore[key]; ok {
		delete(p.podStore, key)
	}
}

type content struct {
	Namespace string
	PodName   string
	Container string
	ID        string
	Target    string
	Severity  string
}

func (p *store) GenerateAlertRule(pod *api.Pod) (string, string) {
	p.Lock()
	defer p.Unlock()
	writer := bytes.NewBufferString("")
	namespace := pod.Namespace
	name := pod.Name
	fn := fmt.Sprintf("%s.%s.generated.rule", namespace, name)
	podKey := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
	for container, capacity := range p.podStore[podKey] {
		if capacity.ID == "" {
			continue
		}
		if !capacity.CPU.IsZero() {
			c := content{
				Namespace: namespace,
				PodName:   name,
				Container: container,
				ID:        capacity.ID,
				Target:    strconv.FormatFloat(float64(capacity.CPU.MilliValue())*warningFactor/1000, 'f', 3, 32),
				Severity:  "warning",
			}
			p.cpuTmpl.Execute(writer, c)
			c.Target = strconv.FormatFloat(float64(capacity.CPU.MilliValue())*criticalFactor/1000, 'f', 3, 32)
			c.Severity = "critical"
			p.cpuTmpl.Execute(writer, c)
		}
		if !capacity.Memory.IsZero() {
			c := content{
				Namespace: namespace,
				PodName:   name,
				Container: container,
				ID:        capacity.ID,
				Target:    strconv.FormatFloat(float64(capacity.Memory.Value())*warningFactor, 'f', 0, 32),
				Severity:  "warning",
			}
			p.memTmpl.Execute(writer, c)
			c.Target = strconv.FormatFloat(float64(capacity.Memory.Value())*criticalFactor, 'f', 0, 32)
			c.Severity = "critical"
			p.memTmpl.Execute(writer, c)
		}
	}
	return fn, writer.String()
}
