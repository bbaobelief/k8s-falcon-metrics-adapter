package provider

import "strings"

func PodNameToContainerName(podName string) string {
	p := strings.Split(podName, "-")
	s := p[:len(p)-2]
	return strings.Join(s, "-")
}
