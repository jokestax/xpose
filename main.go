package main

import (
	"fmt"
	"time"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	kubeconfig := "/Users/mrrishi/.config/k3d/kubeconfig-dev.yaml"
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		fmt.Print(err)
		return
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Print(err)
		return
	}
	ch := make(chan struct{})

	informers := informers.NewSharedInformerFactory(clientset, 10*time.Minute)
	c := newController(clientset, informers.Apps().V1().Deployments())
	informers.Start(ch)
	c.run(ch)
	fmt.Print(informers)

}
