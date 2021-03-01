package ps

import (
	"errors"
	"github.com/diegostock12/kubeml/ml/pkg/api"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"time"
)

func (ps *ParameterServer) isPodReady(podName string) wait.ConditionFunc {
	return func() (done bool, err error) {

		pod, err := ps.kubeClient.CoreV1().Pods(KubeMlNamespace).Get(podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		switch pod.Status.Phase {
		case corev1.PodRunning:
			return true, nil
		case corev1.PodFailed, corev1.PodSucceeded:
			return false, errors.New("pod failed or was succeeded")
		}

		return false, nil
	}
}

func (ps *ParameterServer) waitForPodRunning(pod *corev1.Pod, timeout time.Duration) error {
	return wait.PollImmediate(time.Second, timeout, ps.isPodReady(pod.Name))
}

// createJobPod creates a pod for a new train job with a specific ID
func (ps *ParameterServer) createJobPod(task api.TrainTask) (*corev1.Pod, error) {

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "job-" + task.Job.JobId,
			Namespace: KubeMlNamespace,
			Labels: map[string]string{
				"svc": "job",
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:            "job",
					Image:           KubeMlContainer,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         []string{"/kubeml"},
					Args: []string{
						"--jobPort",
						"9090",
						"--jobId",
						task.Job.JobId,
					},
					// TODO for now limit parallelism to two in minikube
					Env: []corev1.EnvVar{
						{
							Name: "LIMIT_PARALLELISM",
							Value: "true",
						},
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          "http",
							ContainerPort: 9090,
							Protocol:      "TCP",
						},
					},
					ReadinessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							Exec: nil,
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/health",
								Port:   intstr.IntOrString{Type: intstr.Int, IntVal: 9090, StrVal: "9090"},
								Scheme: "HTTP",
							},
						},
						InitialDelaySeconds: 1,
						TimeoutSeconds:      1,
						PeriodSeconds:       1,
						SuccessThreshold:    1,
						FailureThreshold:    30,
					},
				},
			},
		},
	}

	podRef, err := ps.kubeClient.CoreV1().Pods(KubeMlNamespace).Create(pod)
	if err != nil {
		return nil, err
	}

	ps.logger.Debug("data from pod",
		zap.Any("name", podRef.Name),
		zap.Any("ip", podRef.Status.PodIP),
		zap.Any("phase", podRef.Status.Phase))

	err = ps.waitForPodRunning(podRef, 20*time.Second)
	if err != nil {
		return nil, err
	}

	ps.logger.Debug("Created pod")


	// get the reference of the pod with the IP for creation of the client
	pod, err = ps.kubeClient.CoreV1().Pods(KubeMlNamespace).Get(pod.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return pod, nil
}