package kubetest

import (
	"context"
	"testing"
	"flag"
	"path/filepath"
	"log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	api "k8s.io/api/core/v1"
)

func TestPodCreate(t *testing.T) {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	pod := &api.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pod",
		},
		Spec: api.PodSpec{
			Containers: []api.Container{
				{
					Name:  "nginx",
					Image: "nginx",
				},
			},
		},
	}
	log.Printf("[INFO] Creating new pod: %#v", pod)
	out, err := clientset.CoreV1().Pods("docker").Create(context.Background(), pod, metav1.CreateOptions{})

	if err != nil {
		panic(err)
	}
	log.Printf("[INFO] Submitted new pod: %#v", out)

}
