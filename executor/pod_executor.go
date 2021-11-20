package executor

import (
	"context"
	"fmt"
	"strconv"
	"time"

	queue "github.com/luqmanMohammed/k8s-events-runner/queue"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

type PodExecutor struct {
	k8sClientSet       *kubernetes.Clientset
	namespace          string
	erPodIndentifier   string
	jobQueue           queue.JobQueue
	concurrencyTimeout time.Duration
}

func New(k8sClientSet *kubernetes.Clientset, namespace, erPodIndentifier string, jobQueue queue.JobQueue) *PodExecutor {
	return &PodExecutor{
		k8sClientSet:     k8sClientSet,
		namespace:        namespace,
		erPodIndentifier: erPodIndentifier,
		jobQueue:         jobQueue,
	}
}

func (pe PodExecutor) StartExecutors() {
	for job := range pe.jobQueue {
		go pe.executeJob(job)
	}
}

func (pe PodExecutor) checkConcurrency(jb *queue.Job) (bool, error) {
	podList, err := pe.k8sClientSet.CoreV1().Pods(pe.namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("er=%s,erEventType=%s,erResource=%s", pe.erPodIndentifier, jb.EventType, jb.Resource),
	})
	if err != nil {
		return false, err
	}
	if len(podList.Items) >= jb.ConcurrencyLimit {
		return false, nil
	}
	return true, nil
}

func (pe PodExecutor) preparePod(jb *queue.Job) v1.Pod {
	pod := v1.Pod(*jb.RunnerConfig)
	if len(pod.Labels) == 0 {
		pod.Labels = make(map[string]string)
	}
	if len(pod.Annotations) == 0 {
		pod.Annotations = make(map[string]string)
	}
	pod.Labels["erID"] = pe.erPodIndentifier
	pod.Labels["erEventType"] = jb.EventType
	pod.Labels["erResource"] = jb.Resource
	pod.Spec.RestartPolicy = v1.RestartPolicyNever
	pod.Annotations["erRetries"] = strconv.Itoa(jb.RetryCount)
	for i := range pod.Spec.Containers {
		pod.Spec.Containers[i].ImagePullPolicy = v1.PullIfNotPresent
	}
	return pod
}

func (pe PodExecutor) executeJob(jb *queue.Job) {
	if ok, err := pe.checkConcurrency(jb); err != nil {
		klog.Errorf("failed to check concurrency: %v", err)
		return
	} else if !ok {
		klog.Infof("concurrency limit reached, skipping job %s:%s and adding back into queue", jb.Resource, jb.EventType, jb.RunnerConfig.Name)
		time.Sleep(pe.concurrencyTimeout)
		pe.jobQueue <- jb
		return
	}
	pod := pe.preparePod(jb)
	_, err := pe.k8sClientSet.CoreV1().Pods(pe.namespace).Create(context.Background(), &pod, metav1.CreateOptions{})
	if err != nil {
		klog.Error(err)
	}
}

// func (pe PodExecutor) StartWatcher() error {
// 	watchInt, err := pe.K8sClientSet.CoreV1().Pods(pe.Namespace).Watch(context.Background(), metav1.ListOptions{
// 		LabelSelector: "erID=" + pe.ErPodIndentifier,
// 		FieldSelector: "status.phase!=Running,status.phase!=Pending",
// 	})
// 	if err != nil {
// 		fmt.Println(err)
// 		return err
// 	}
// 	for podEvent := range watchInt.ResultChan() {
// 		if podEvent.Type == watch.Added {
// 			fmt.Println(podEvent.Type)
// 			pod := podEvent.Object.(*v1.Pod)
// 			print(pod.Status.Phase)
// 			if pod.Status.Phase == v1.PodSucceeded {
// 				fmt.Println(pod.Name, " Should be deleted")
// 			} else if pod.Status.Phase == v1.PodFailed {
// 				fmt.Println(pod.Name, " Should be retried")
// 			}
// 		}
// 	}

// 	return nil
// }
