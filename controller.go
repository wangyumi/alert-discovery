package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/golang/glog"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/cache"
	k8s_client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/controller/framework"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/watch"
)

const (
	// Resync period for the kube controller loop.
	resyncPeriod               = 5 * time.Minute
	controllerSyncedPollPeriod = 1 * time.Second
)

type controller struct {
	client        *k8s_client.Client
	shutdown      bool
	stopLock      sync.Mutex
	stopCh        chan struct{}
	store         *store
	podController *framework.Controller
	podStore      cache.Store
	namespace     string
	configmapNs   string
	configmapName string
}

func newController(client *k8s_client.Client, configmapNs, configmapName string) (*controller, error) {
	c := &controller{
		client:        client,
		namespace:     api.NamespaceAll,
		stopCh:        make(chan struct{}),
		store:         newStore(),
		configmapNs:   configmapNs,
		configmapName: configmapName,
	}

	podEventHandler := framework.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pod := obj.(*api.Pod)
			glog.Infof("%s/%s", pod.Namespace, pod.Name)
			changed := c.store.Update(pod)
			if changed {
				c.syncToConfigmap(c.store.GenerateAlertRule(pod))
			}
		},
		UpdateFunc: func(old, cur interface{}) {
			pod := cur.(*api.Pod)
			glog.Infof("%s/%s", pod.Namespace, pod.Name)
			changed := c.store.Update(pod)
			if changed {
				c.syncToConfigmap(c.store.GenerateAlertRule(pod))
			}
		},
		DeleteFunc: func(obj interface{}) {
			pod := obj.(*api.Pod)
			glog.Infof("%s/%s", pod.Namespace, pod.Name)
			c.store.Delete(pod)
			c.syncToConfigmap(c.store.GenerateAlertRule(pod))
		},
	}
	c.podStore, c.podController = framework.NewInformer(
		&cache.ListWatch{
			ListFunc:  podListFunc(c.client, c.namespace),
			WatchFunc: podWatchFunc(c.client, c.namespace),
		},
		&api.Pod{}, resyncPeriod, podEventHandler)
	return c, nil
}

func (c *controller) syncToConfigmap(fn, rules string) {
	cm, err := c.client.ConfigMaps(c.configmapNs).Get(c.configmapName)
	if err != nil {
		glog.Errorf("failed to get configmap: %s/%s, %v", c.configmapNs, c.configmapName, err)
		return
	}
	if rules == "" {
		delete(cm.Data, fn)
	} else if now, ok := cm.Data[fn]; ok {
		if now == rules {
			glog.Infof("no changes of rules, ignore")
			return
		}
		cm.Data[fn] = rules
	} else {
		cm.Data[fn] = rules
	}
	glog.Infof("updating alert rules...")
	if _, err := c.client.ConfigMaps(c.configmapNs).Update(cm); err != nil {
		glog.Errorf("failed to update configmap: %s/%s, %v", c.configmapNs, c.configmapName, err)
		return
	}
}

func podListFunc(c *k8s_client.Client, ns string) func(api.ListOptions) (runtime.Object, error) {
	return func(opts api.ListOptions) (runtime.Object, error) {
		return c.Pods(ns).List(opts)
	}
}

func podWatchFunc(c *k8s_client.Client, ns string) func(options api.ListOptions) (watch.Interface, error) {
	return func(options api.ListOptions) (watch.Interface, error) {
		return c.Pods(ns).Watch(options)
	}
}

func (c *controller) Run() {
	glog.Infof("starting alert discovery...")
	go c.podController.Run(c.stopCh)
	<-c.stopCh
}

func (c *controller) Stop() error {
	c.stopLock.Lock()
	defer c.stopLock.Unlock()

	if !c.shutdown {
		c.shutdown = true
		close(c.stopCh)

		return nil
	}

	return fmt.Errorf("shutdown already in progress")
}

func (c *controller) controllersInSync() bool {
	return c.podController.HasSynced()
}
