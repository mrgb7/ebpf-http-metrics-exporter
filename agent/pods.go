package main

import (
	"log"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type podCache struct {
	byIP cache.Indexer
}

type podInfo struct {
	namespace string
	workload  string
}

var unknownPod = podInfo{namespace: "unknown", workload: "unknown"}

func newPodCache() *podCache {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}
	client := kubernetes.NewForConfigOrDie(config)

	factory := informers.NewSharedInformerFactory(client, 0)
	informer := factory.Core().V1().Pods().Informer()
	informer.AddIndexers(cache.Indexers{"ip": podIP})

	stop := make(chan struct{})
	factory.Start(stop)
	factory.WaitForCacheSync(stop)
	log.Println("pod cache synced")

	return &podCache{byIP: informer.GetIndexer()}
}

func podIP(obj any) ([]string, error) {
	pod := obj.(*corev1.Pod)
	if pod.Spec.HostNetwork || pod.Status.PodIP == "" {
		return nil, nil
	}
	return []string{pod.Status.PodIP}, nil
}

func (c *podCache) lookup(ip string) podInfo {
	pods, _ := c.byIP.ByIndex("ip", ip)
	if len(pods) == 0 {
		return unknownPod
	}
	pod := pods[0].(*corev1.Pod)

	workload := pod.Labels["app"]
	if workload == "" {
		workload = pod.Name
	}
	return podInfo{namespace: pod.Namespace, workload: workload}
}
