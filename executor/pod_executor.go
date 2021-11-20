package executor

import (
	"context"
	"fmt"
	"strconv"
	"time"

	queue "github.com/luqmanMohammed/k8s-events-runner/queue"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

var (
	WatcherRoutinesCount int
)

type PodExecutor struct {
	k8sClientSet       *kubernetes.Clientset
	namespace          string
	erPodIndentifier   string
	jobQueue           queue.JobQueue
	watchQueue         queue.WatchQueue
	concurrencyTimeout time.Duration
}

func New(k8sClientSet *kubernetes.Clientset, namespace, erPodIndentifier string, jobQueue queue.JobQueue) *PodExecutor {
	return &PodExecutor{
		k8sClientSet:     k8sClientSet,
		namespace:        namespace,
		erPodIndentifier: erPodIndentifier,
		jobQueue:         jobQueue,
		watchQueue:       queue.NewWatchQueue(50),
	}
}

func (pe PodExecutor) StartExecutors(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			klog.Info("Pod executor is shutting down")
			return
		case jb := <-pe.jobQueue:
			go pe.executeJob(ctx, jb)
		}
	}
}

func (pe PodExecutor) StartWatcher(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			klog.Info("Pod executor watcher is shutting down")
			return
		case w := <-pe.watchQueue:
			go pe.watchPod(ctx, w)
		}
	}
}

func (pe PodExecutor) watchPod(ctx context.Context, jb *queue.Job) {
	watchPod, err := pe.k8sClientSet.CoreV1().Pods(pe.namespace).Watch(ctx, metav1.ListOptions{
		LabelSelector: "run=" + jb.PodName,
		FieldSelector: "status.phase!=Running,status.phase!=Pending",
	})
	if err != nil {
		klog.Errorf("failed to watch pod: %v", err)
	}
	for {
		select {
		case <-ctx.Done():
			klog.Info("Pod executor watcher is shutting down")
			return
		case evt, ok := <-watchPod.ResultChan():
			if !ok {
				klog.Info("Pod executor watcher is shutting down")
				return
			}
			if evt.Type == watch.Added {
				pod := evt.Object.(*v1.Pod)
				if pod.Status.Phase == v1.PodFailed {
					klog.Infof("Pod %s failed", jb.PodName)
					jb.RetryCount++
					if jb.RetryCount > jb.RetryLimit {
						klog.Infof("Pod %s failed, retry limit reached, ignoring job %s:%s", jb.PodName)
						return
					}
					pe.jobQueue <- jb
					return
				} else if pod.Status.Phase == v1.PodSucceeded {
					klog.Infof("Pod %s succeeded", jb.PodName)
					return
				}
				return
			}
		}
	}
}

func (pe PodExecutor) checkConcurrency(ctx context.Context, jb *queue.Job) (bool, error) {
	podList, err := pe.k8sClientSet.CoreV1().Pods(pe.namespace).List(ctx, metav1.ListOptions{
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

func (pe PodExecutor) executeJob(ctx context.Context, jb *queue.Job) {
	if ok, err := pe.checkConcurrency(ctx, jb); err != nil {
		klog.Errorf("failed to check concurrency: %v", err)
		return
	} else if !ok {
		klog.Infof("concurrency limit reached, skipping job %s:%s and adding back into queue", jb.Resource, jb.EventType, jb.RunnerConfig.Name)
		time.Sleep(pe.concurrencyTimeout)
		pe.jobQueue <- jb
		return
	}
	pod := pe.preparePod(jb)
	_, err := pe.k8sClientSet.CoreV1().Pods(pe.namespace).Create(ctx, &pod, metav1.CreateOptions{})
	if err != nil {
		klog.Error(err)
	}
	pe.watchQueue <- jb
}
