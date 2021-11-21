package executor

import (
	"context"
	"fmt"
	"strings"
	"time"

	queue "github.com/luqmanMohammed/k8s-events-runner/queue"
	"github.com/luqmanMohammed/k8s-events-runner/utils"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

type K8sJobExecutor struct {
	k8sClientSet       *kubernetes.Clientset
	namespace          string
	erPodIndentifier   string
	jobQueue           queue.JobQueue
	watchQueue         queue.WatchQueue
	concurrencyTimeout time.Duration `default:"5m"`
	cleanupTimeout     time.Duration `default:"1h"`
}

func New(k8sClientSet *kubernetes.Clientset, namespace, erPodIndentifier string, concurrencyTimeout time.Duration, jobQueue queue.JobQueue) *K8sJobExecutor {
	return &K8sJobExecutor{
		k8sClientSet:       k8sClientSet,
		namespace:          namespace,
		erPodIndentifier:   erPodIndentifier,
		jobQueue:           jobQueue,
		watchQueue:         queue.NewWatchQueue(50),
		concurrencyTimeout: concurrencyTimeout,
	}
}

func (pe K8sJobExecutor) StartExecutors(ctx context.Context) {
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

func (pe K8sJobExecutor) StartWatcher(ctx context.Context) {
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

func (pe K8sJobExecutor) watchPod(ctx context.Context, jb *queue.Job) {
	watchPodChan, err := pe.k8sClientSet.CoreV1().Pods(pe.namespace).Watch(ctx, metav1.ListOptions{
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
		case evt, ok := <-watchPodChan.ResultChan():
			if !ok {
				klog.Info("Pod executor watcher is shutting down")
				return
			}
			fmt.Println(evt.Object)
			if evt.Type == watch.Added {
				pod := evt.Object.(*v1.Pod)
				if pod.Status.Phase == v1.PodFailed {
					klog.Infof("Pod %s failed", jb.PodName)
					jb.RetryCount++
					if jb.RetryCount > jb.RetryLimit {
						klog.Infof("Pod %s failed, retry limit reached, ignoring job ", jb.PodName)
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

func (pe K8sJobExecutor) checkConcurrency(ctx context.Context, jb *queue.Job) (bool, error) {
	jobList, err := pe.k8sClientSet.BatchV1().Jobs(pe.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("erID=%s,erEventType=%s,erResource=%s", pe.erPodIndentifier, jb.EventType, jb.Resource),
		FieldSelector: "status.jobCondition!=Failed,status.jobCondition!=Complete",
	})
	fmt.Println(len(jobList.Items))
	if err != nil {
		return false, err
	}
	if len(jobList.Items) > jb.ConcurrencyLimit {
		return false, nil
	}
	return true, nil
}

func (pe K8sJobExecutor) prepareJob(jb *queue.Job) batchv1.Job {
	podTemplate := v1.PodTemplateSpec(*jb.RunnerConfig)
	if len(podTemplate.Labels) == 0 {
		podTemplate.Labels = make(map[string]string)
	}
	if len(podTemplate.Annotations) == 0 {
		podTemplate.Annotations = make(map[string]string)
	}
	podTemplate.Spec.RestartPolicy = v1.RestartPolicyNever
	for i := range podTemplate.Spec.Containers {
		podTemplate.Spec.Containers[i].ImagePullPolicy = v1.PullIfNotPresent
	}
	retries := int32(jb.RetryLimit)
	cleanupTimeout := int32(pe.cleanupTimeout.Seconds())
	jobLabels := utils.MergeStringStringMaps(podTemplate.Labels, map[string]string{
		"erID":        pe.erPodIndentifier,
		"erEventType": jb.EventType,
		"erResource":  jb.Resource,
	})
	k8sJob := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: strings.ToLower(fmt.Sprintf("%s-%s-%s-", jb.Resource, jb.EventType, jb.Runner)),
			Namespace:    pe.namespace,
			Labels:       jobLabels,
			Annotations:  podTemplate.Annotations,
		},
		Spec: batchv1.JobSpec{
			Template:                podTemplate,
			BackoffLimit:            &retries,
			TTLSecondsAfterFinished: &cleanupTimeout,
		},
	}
	return k8sJob
}

func (pe K8sJobExecutor) executeJob(ctx context.Context, jb *queue.Job) {
	if ok, err := pe.checkConcurrency(ctx, jb); err != nil {
		klog.Errorf("failed to check concurrency: %v", err)
		return
	} else if !ok {
		klog.Infof("concurrency limit reached, skipping job %s:%s and adding back into queue", jb.Resource, jb.EventType)
		fmt.Println(pe.concurrencyTimeout)
		timer := time.NewTimer(pe.concurrencyTimeout)
		select {
		case <-ctx.Done():
			klog.Info("Pod executor is shutting down")
			return
		case <-timer.C:
			klog.Info("Sleep interval done")
			pe.jobQueue <- jb
		}
		return
	}
	pod := pe.prepareJob(jb)
	createdPod, err := pe.k8sClientSet.CoreV1().Pods(pe.namespace).Create(ctx, &pod, metav1.CreateOptions{})
	if err != nil {
		klog.Error(err)
	}
	jb.PodName = createdPod.Name
	pe.watchQueue <- jb
}
