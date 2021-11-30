package executor

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	queue "github.com/luqmanMohammed/k8s-events-runner/queue"
	"github.com/luqmanMohammed/k8s-events-runner/utils"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

type K8sJobExecutor struct {
	k8sClientSet       *kubernetes.Clientset
	namespace          string
	erPodIndentifier   string
	jobQueue           queue.JobQueue
	concurrencyTimeout time.Duration `default:"5m"`
	manageCleanup      bool          `default:"false"`
	cleanupTimeout     time.Duration `default:"1h"`
	completions        int32         `default:"1"`
	executorCount      int           `default:"5"`
}

func New(k8sClientSet *kubernetes.Clientset, namespace, erPodIndentifier string, concurrencyTimeout, cleanupTimeout time.Duration, jobQueue queue.JobQueue) *K8sJobExecutor {
	k8sMajorVersion, k8sMinorVersion, err := utils.GetKubeVersion(k8sClientSet)
	if err != nil {
		klog.Fatal(err)
	}

	return &K8sJobExecutor{
		k8sClientSet:       k8sClientSet,
		namespace:          namespace,
		erPodIndentifier:   erPodIndentifier,
		jobQueue:           jobQueue,
		concurrencyTimeout: concurrencyTimeout,
		cleanupTimeout:     cleanupTimeout,
		completions:        1,
		executorCount:      5,
		manageCleanup:      k8sMajorVersion >= 1 && k8sMinorVersion >= 21,
	}
}

func (pe K8sJobExecutor) runJobCleaner(ctx context.Context) {

}

func (pe K8sJobExecutor) StartExecutors(ctx context.Context) {
	wg := sync.WaitGroup{}
	wg.Add(pe.executorCount)
	for i := 0; i < pe.executorCount; i++ {
		go func(ctx context.Context) {
			for {
				select {
				case <-ctx.Done():
					klog.Info("Pod executor is shutting down")
					wg.Done()
					return
				case jb := <-pe.jobQueue:
					klog.Infof("executing job %s:%s", jb.Resource, jb.EventType)
					pe.executeJob(ctx, jb)
				}
			}
		}(ctx)
	}
	wg.Wait()
}

func (pe K8sJobExecutor) checkConcurrency(ctx context.Context, jb *queue.Job) (bool, error) {
	if jb.ConcurrencyLimit == -1 {
		return true, nil
	}
	jobList, err := pe.k8sClientSet.BatchV1().Jobs(pe.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("erID=%s,erEventType=%s,erResource=%s", pe.erPodIndentifier, jb.EventType, jb.Resource),
		FieldSelector: "status.successful!=1",
	})
	if err != nil {
		return false, err
	}
	concCount := 0
	for _, job := range jobList.Items {
		if len(job.Status.Conditions) == 0 {
			concCount++
		}
	}
	if concCount > jb.ConcurrencyLimit {
		return false, nil
	}
	return true, nil
}

func (pe K8sJobExecutor) prepareJob(jb *queue.Job) batchv1.Job {
	podTemplate := v1.PodTemplateSpec(*jb.RunnerTemplate)
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

	jobAnnotations := podTemplate.Annotations
	if pe.manageCleanup {
		jobAnnotations["erCleanTime"] = strconv.Itoa(int(time.Now().Add(pe.cleanupTimeout).Unix()))
	}
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
			Annotations:  jobAnnotations,
		},
		Spec: batchv1.JobSpec{
			Template:                podTemplate,
			BackoffLimit:            &retries,
			TTLSecondsAfterFinished: &cleanupTimeout,
			Completions:             &pe.completions,
		},
	}
	return k8sJob
}

func (pe *K8sJobExecutor) executeJob(ctx context.Context, jb *queue.Job) {
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
	k8sJob := pe.prepareJob(jb)
	_, err := pe.k8sClientSet.BatchV1().Jobs(pe.namespace).Create(ctx, &k8sJob, metav1.CreateOptions{})
	if err != nil {
		klog.Error(err)
	}
}
