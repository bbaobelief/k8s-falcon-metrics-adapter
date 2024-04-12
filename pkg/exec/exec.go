package exec

import (
	"bytes"
	"io"
	corev1 "k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/klog/v2"
	"net/url"
	"strings"
)

type ExecOptions struct {
	Namespace     string
	PodName       string
	Stdin         io.Reader
	CaptureStdout bool
	CaptureStderr bool

	ContainerName string
	//Command       []string
}

func NewExecWithOptions(nameSpace, podName, containerName string) *ExecOptions {
	return &ExecOptions{
		Namespace:     nameSpace,
		PodName:       podName,
		ContainerName: containerName,
		Stdin:         nil,
		CaptureStdout: true,
		CaptureStderr: true,
	}
}

func (e *ExecOptions) ExecCommandInPod(clientSet clientset.Interface, config *rest.Config) (string, string, error) {
	klog.V(4).Info("ExecWithOptions %+v", e)

	req := clientSet.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(e.PodName).
		Namespace(e.Namespace).
		SubResource("exec").
		Param("container", e.ContainerName).
		VersionedParams(&corev1.PodExecOptions{
			Container: e.ContainerName,
			Command:   []string{"bash", "-c", "hostname"},
			Stdin:     e.Stdin != nil,
			Stdout:    e.CaptureStdout,
			Stderr:    e.CaptureStderr,
			TTY:       false,
		}, scheme.ParameterCodec)

	var stdout, stderr bytes.Buffer
	err := execute("POST", req.URL(), config, e.Stdin, &stdout, &stderr, false)

	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
}

func execute(method string, url *url.URL, config *rest.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool) error {
	exec, err := remotecommand.NewSPDYExecutor(config, method, url)
	if err != nil {
		return err
	}
	return exec.Stream(remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
		Tty:    tty,
	})
}
