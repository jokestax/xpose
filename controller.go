package main

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	appinformers "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	applisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

/* custom controller datatype */
type controller struct {
	clientset      kubernetes.Interface
	depLister      applisters.DeploymentLister
	depCacheSynced cache.InformerSynced
	queue          workqueue.RateLimitingInterface
}

/* defines clientset,Lister,cache stored in informer,queue for performing custom functions*/
func newController(clientset kubernetes.Interface, depInformer appinformers.DeploymentInformer) *controller {
	c := &controller{
		clientset:      clientset,
		depLister:      depInformer.Lister(),
		depCacheSynced: depInformer.Informer().HasSynced,
		queue:          workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "xpose"),
	}

	depInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.handleAdd,
			DeleteFunc: c.handleDel,
		},
	)
	return c
}

/* function to perform serive creation for deployments created */
func (c *controller) run(ch <-chan struct{}) {
	fmt.Print("starting controller")
	if !cache.WaitForCacheSync(ch, c.depCacheSynced) {
		fmt.Printf("waiting for cache to be synced")
	}
	/* waits until channel is closed */
	go wait.Until(c.worker, 1*time.Second, ch)

	<-ch
}

func (c *controller) worker() {
	for c.processItem() {

	}
}

/* process items stored in queue added when new deployment was created */
func (c *controller) processItem() bool {

	item, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	defer c.queue.Forget(item)

	key, err := cache.MetaNamespaceKeyFunc(item)
	if err != nil {
		fmt.Print(err)
		return false
	}

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		fmt.Print(err)
		return false
	}

	err = c.SyncDeployment(ns, name)
	if err != nil {
		fmt.Print(err)
		return false
	}

	return true

}

/* create the service for particular deployment*/
func (c *controller) SyncDeployment(ns, name string) error {
	dep, err := c.depLister.Deployments(ns).Get(name)

	if err != nil {
		fmt.Print(err)
		return err
	}
	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dep.Name,
			Namespace: ns,
		},
		Spec: corev1.ServiceSpec{
			Selector: depLabels(*dep),
			Ports: []corev1.ServicePort{
				corev1.ServicePort{
					Name: "http",
					Port: 80,
				},
			},
		},
	}
	_, err = c.clientset.CoreV1().Services(ns).Create(context.Background(), &svc, metav1.CreateOptions{})

	if err != nil {
		return err
	}
	return nil
}

/* labels of pods for service */
func depLabels(dep appsv1.Deployment) map[string]string {
	return dep.Spec.Template.Labels
}

/* handler functions for informer*/
func (c *controller) handleAdd(obj interface{}) {
	fmt.Println("add was called")
	c.queue.Add(obj)
}

/* handler functions for informer*/
func (c *controller) handleDel(obj interface{}) {
	fmt.Println("Delete was called")
	c.queue.Add(obj)
}
