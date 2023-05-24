package v1alpha1

import (
	"fmt"
	"strings"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// OperatorSpec defines the desired state of Operator
type OperatorSpec struct {
	Global  GlobalConfig  `json:"global,omitempty"`
	Server  ServerConfig  `json:"server,omitempty"`
	Client  ClientConfig  `json:"client,omitempty"`
	Connect ConnectConfig `json:"connect,omitempty"`
	UI      UIConfig      `json:"ui,omitempty"`
}

type GlobalConfig struct {
	Datacenter       string   `json:"datacenter,omitempty"`
	Domain           string   `json:"domain,omitempty"`
	Enabled          bool     `json:"enabled,omitempty"`
	ConsulImage      string   `json:"consulImage,omitempty"`
	ConsulK8sImage   string   `json:"consulK8sImage,omitempty"`
	EnvoyImage       string   `json:"imageEnvoy,omitempty"`
	ImagePullSecrets []string `json:"imagePullSecrets,omitempty"`
}

type ServerConfig struct {
	Enabled  bool  `json:"enabled,omitempty"`
	Replicas int32 `json:"replicas,omitempty"`
}

type ClientConfig struct {
	Enabled bool `json:"enabled,omitempty"`
}

type UIConfig struct {
	Enabled bool   `json:"enabled,omitempty"`
	Type    string `json:"type,omitempty"`
}

type ConnectConfig struct {
	Enabled      bool `json:"enabled,omitempty"`
	HealthChecks bool `json:"health,omitempty"`
}

// OperatorStatus defines the observed state of Operator
type OperatorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Operator is the Schema for the operators API
type Operator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OperatorSpec   `json:"spec,omitempty"`
	Status OperatorStatus `json:"status,omitempty"`
}

func (in *Operator) ServerStatefulSet() *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-server", in.Name),
			Namespace: in.Namespace,
			Labels: map[string]string{
				"app":       in.Name,
				"release":   in.Name,
				"component": "server",
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(in, schema.GroupVersionKind{
					Group:   "consul.hashicorp.com",
					Version: "v1alpha1",
					Kind:    "Operator",
				}),
			},
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName:         fmt.Sprintf("%s-server", in.Name),
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Replicas:            &in.Spec.Server.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":       in.Name,
					"release":   in.Name,
					"component": "server",
					"hasDNS":    "true",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":       in.Name,
						"release":   in.Name,
						"component": "server",
						"hasDNS":    "true",
					},
					Annotations: map[string]string{
						"consul.hashicorp.com/connect-inject": "false",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: fmt.Sprintf("%s-server", in.Name),
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: fmt.Sprintf("%s-server-config", in.Name),
									},
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "consul",
							Image: in.Spec.Global.ConsulImage,
							Env: []corev1.EnvVar{
								{
									Name: "POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name: "NAMESPACE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
							},
							Command: []string{"/bin/sh", "-ec", fmt.Sprintf(
								`exec /bin/consul agent \
-advertise="${POD_IP}" \
-bind=0.0.0.0 \
-bootstrap-expect=%d \
-client=0.0.0.0 \
-config-dir=/consul/config \
-datacenter=%s \
-data-dir=/consul/data \
-domain=%s \
-hcl="connect { enabled = true }" \
-ui \
%s \
-server`,
								in.Spec.Server.Replicas, in.Spec.Global.Datacenter, in.Spec.Global.Domain, in.retryJoinString(),
							)},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/consul/config",
								},
								{
									Name:      fmt.Sprintf("data-%s", in.Namespace),
									MountPath: "/consul/data",
								},
							},
							Lifecycle: &corev1.Lifecycle{
								PreStop: &corev1.Handler{
									Exec: &corev1.ExecAction{
										Command: []string{"/bin/sh", "-c", "consul leave"},
									},
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 8500,
								},
								{
									Name:          "serflan",
									ContainerPort: 8301,
								},
								{
									Name:          "serfwan",
									ContainerPort: 8302,
								},
								{
									Name:          "server",
									ContainerPort: 8300,
								},
								{
									Name:          "dns-tcp",
									ContainerPort: 8600,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          "dns-udp",
									ContainerPort: 8600,
									Protocol:      corev1.ProtocolUDP,
								},
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									Exec: &corev1.ExecAction{
										Command: []string{"/bin/sh", "-ec", `curl http://127.0.0.1:8500/v1/status/leader 2>/dev/null | grep -E '".+"'`},
									},
								},
								InitialDelaySeconds: 5,
								TimeoutSeconds:      5,
								PeriodSeconds:       3,
								SuccessThreshold:    1,
								FailureThreshold:    2,
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: fmt.Sprintf("data-%s", in.Namespace),
						OwnerReferences: []metav1.OwnerReference{
							*metav1.NewControllerRef(in, schema.GroupVersionKind{
								Group:   "consul.hashicorp.com",
								Version: "v1alpha1",
								Kind:    "Operator",
							}),
						},
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("10Gi"),
							},
						},
					},
				},
			},
		},
	}
}

func (in *Operator) ServerRole() *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-server", in.Name),
			Namespace: in.Namespace,
			Labels: map[string]string{
				"app":       in.Name,
				"release":   in.Name,
				"component": "server",
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(in, schema.GroupVersionKind{
					Group:   "consul.hashicorp.com",
					Version: "v1alpha1",
					Kind:    "Operator",
				}),
			},
		},
		Rules: []rbacv1.PolicyRule{},
	}
}

func (in *Operator) ServerRoleBinding() *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-server", in.Name),
			Namespace: in.Namespace,
			Labels: map[string]string{
				"app":       in.Name,
				"release":   in.Name,
				"component": "server",
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(in, schema.GroupVersionKind{
					Group:   "consul.hashicorp.com",
					Version: "v1alpha1",
					Kind:    "Operator",
				}),
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: rbacv1.ServiceAccountKind,
				Name: fmt.Sprintf("%s-server", in.Name),
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     fmt.Sprintf("%s-server", in.Name),
		},
	}
}

func (in *Operator) ServerServiceAccount() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-server", in.Name),
			Namespace: in.Namespace,
			Labels: map[string]string{
				"app":       in.Name,
				"release":   in.Name,
				"component": "server",
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(in, schema.GroupVersionKind{
					Group:   "consul.hashicorp.com",
					Version: "v1alpha1",
					Kind:    "Operator",
				}),
			},
		},
		ImagePullSecrets: in.imagePullSecretsList(),
	}
}

func (in *Operator) imagePullSecretsList() []corev1.LocalObjectReference {
	var secrets []corev1.LocalObjectReference
	for _, s := range in.Spec.Global.ImagePullSecrets {
		secrets = append(secrets, corev1.LocalObjectReference{Name: s})
	}
	return secrets
}

func (in *Operator) ServerConfigMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-server-config", in.Name),
			Namespace: in.Namespace,
			Labels: map[string]string{
				"app":       in.Name,
				"release":   in.Name,
				"component": "server",
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(in, schema.GroupVersionKind{
					Group:   "consul.hashicorp.com",
					Version: "v1alpha1",
					Kind:    "Operator",
				}),
			},
		},
		Data: map[string]string{
			"extra-from-values.json": "{}",
		},
	}
}

func (in *Operator) ServerService() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-server", in.Name),
			Namespace: in.Namespace,
			Labels: map[string]string{
				"app":       in.Name,
				"release":   in.Name,
				"component": "server",
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(in, schema.GroupVersionKind{
					Group:   "consul.hashicorp.com",
					Version: "v1alpha1",
					Kind:    "Operator",
				}),
			},
			Annotations: map[string]string{
				"service.alpha.kubernetes.io/tolerate-unready-endpoints": "true",
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:     "serflan-tcp",
					Protocol: corev1.ProtocolTCP,
					Port:     8301,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 8301,
					},
				},
				{
					Name:     "serflan-udp",
					Protocol: corev1.ProtocolUDP,
					Port:     8301,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 8301,
					},
				},
				{
					Name:     "serfwan-tcp",
					Protocol: corev1.ProtocolTCP,
					Port:     8302,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 8302,
					},
				},
				{
					Name:     "serfwan-udp",
					Protocol: corev1.ProtocolUDP,
					Port:     8302,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 8302,
					},
				},
				{
					Name:     "dns-tcp",
					Protocol: corev1.ProtocolTCP,
					Port:     8600,
					TargetPort: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "dns-tcp",
					},
				},
				{
					Name:     "dns-udp",
					Protocol: corev1.ProtocolUDP,
					Port:     8600,
					TargetPort: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "dns-udp",
					},
				},
				{
					Name: "server",
					Port: 8300,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 8300,
					},
				},
				{
					Name: "http",
					Port: 8500,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 8500,
					},
				},
			},
			Selector: map[string]string{
				"app":       in.Name,
				"release":   in.Name,
				"component": "server",
			},
			ClusterIP:                corev1.ClusterIPNone,
			PublishNotReadyAddresses: true,
		},
	}
}

func (in *Operator) UIService() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-ui", in.Name),
			Namespace: in.Namespace,
			Labels: map[string]string{
				"app":       in.Name,
				"release":   in.Name,
				"component": "ui",
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(in, schema.GroupVersionKind{
					Group:   "consul.hashicorp.com",
					Version: "v1alpha1",
					Kind:    "Operator",
				}),
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app":       in.Name,
				"release":   in.Name,
				"component": "server",
			},
			Ports: []corev1.ServicePort{
				{
					Name: "http",
					Port: 80,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 8500,
					},
				},
			},
			Type: in.uiServiceType(),
		},
	}
}

func (in *Operator) ClientServiceAccount() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-client", in.Name),
			Namespace: in.Namespace,
			Labels: map[string]string{
				"app":     in.Name,
				"release": in.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(in, schema.GroupVersionKind{
					Group:   "consul.hashicorp.com",
					Version: "v1alpha1",
					Kind:    "Operator",
				}),
			},
		},
		ImagePullSecrets: in.imagePullSecretsList(),
	}
}

func (in *Operator) ClientRole() *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-client", in.Name),
			Namespace: in.Namespace,
			Labels: map[string]string{
				"app":     in.Name,
				"release": in.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(in, schema.GroupVersionKind{
					Group:   "consul.hashicorp.com",
					Version: "v1alpha1",
					Kind:    "Operator",
				}),
			},
		},
		Rules: []rbacv1.PolicyRule{},
	}
}

func (in *Operator) ClientRoleBinding() *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-client", in.Name),
			Namespace: in.Namespace,
			Labels: map[string]string{
				"app":     in.Name,
				"release": in.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(in, schema.GroupVersionKind{
					Group:   "consul.hashicorp.com",
					Version: "v1alpha1",
					Kind:    "Operator",
				}),
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     fmt.Sprintf("%s-client", in.Name),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: rbacv1.ServiceAccountKind,
				Name: fmt.Sprintf("%s-client", in.Name),
			},
		},
	}
}

func (in *Operator) ClientConfigMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-client-config", in.Name),
			Namespace: in.Namespace,
			Labels: map[string]string{
				"app":     in.Name,
				"release": in.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(in, schema.GroupVersionKind{
					Group:   "consul.hashicorp.com",
					Version: "v1alpha1",
					Kind:    "Operator",
				}),
			},
		},
		Data: map[string]string{
			"extra-from-values.json": "{}",
		},
	}

}

func (in *Operator) retryJoinString() string {
	var retryJoinString []string
	for i := int32(0); i < in.Spec.Server.Replicas; i++ {
		retryJoinString = append(retryJoinString, fmt.Sprintf("-retry-join=%s-server-%d.%s-server.%s.svc ", in.Name, i, in.Name, in.Namespace))
	}

	return strings.Join(retryJoinString, ``)
}

func (in *Operator) ClientDaemonSet() *appsv1.DaemonSet {
	tenSeconds := int64(10)
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      in.Name,
			Namespace: in.Namespace,
			Labels: map[string]string{
				"app":     in.Name,
				"release": in.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(in, schema.GroupVersionKind{
					Group:   "consul.hashicorp.com",
					Version: "v1alpha1",
					Kind:    "Operator",
				}),
			},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":       in.Name,
					"release":   in.Name,
					"component": "client",
					"hasDNS":    "true",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":       in.Name,
						"release":   in.Name,
						"component": "client",
						"hasDNS":    "true",
					},
					Annotations: map[string]string{
						"consul.hashicorp.com/connect-inject": "false",
					},
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: &tenSeconds,
					ServiceAccountName:            fmt.Sprintf("%s-client", in.Name),
					Volumes: []corev1.Volume{
						{
							Name: "data",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: fmt.Sprintf("%s-client-config", in.Name),
									},
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "consul",
							Image: in.Spec.Global.ConsulImage,
							Env: []corev1.EnvVar{
								{
									Name: "ADVERTISE_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name: "NAMESPACE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
								{
									Name: "NODE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name:  "HOST_IP",
									Value: "",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.hostIP",
										},
									},
								},
							},
							Command: []string{"/bin/sh", "-ec", fmt.Sprintf(`exec /bin/consul agent \
-node="${NODE}" \
-advertise="${ADVERTISE_IP}" \
-bind=0.0.0.0 \
-client=0.0.0.0 \
-node-meta=pod-name:${HOSTNAME} \
-hcl='leave_on_terminate = true' \
-hcl='ports { grpc = 8502 }' \
-config-dir=/consul/config \
-datacenter=%s \
-data-dir=/consul/data \
%s \
-domain=%s`, in.Spec.Global.Datacenter, in.retryJoinString(), in.Spec.Global.Domain)},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "data",
									MountPath: "/consul/data",
								},
								{
									Name:      "config",
									MountPath: "/consul/config",
								},
							},
							Ports: []corev1.ContainerPort{
								{ // TODO: to support TLS we need to remove this port and add 8501
									Name:          "http",
									HostPort:      8500,
									ContainerPort: 8500,
								},
								{
									Name:          "grpc",
									HostPort:      8502,
									ContainerPort: 8502,
								},
								{
									Name:          "serflan-tcp",
									ContainerPort: 8301,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          "serflan-udp",
									ContainerPort: 8301,
									Protocol:      corev1.ProtocolUDP,
								},
								{
									Name:          "serfwan",
									ContainerPort: 8302,
								},
								{
									Name:          "server",
									ContainerPort: 8300,
								},
								{
									Name:          "dns-tcp",
									ContainerPort: 8600,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          "dns-udp",
									ContainerPort: 8600,
									Protocol:      corev1.ProtocolUDP,
								},
							},
							ReadinessProbe: &corev1.Probe{
								// TODO: this can be stubbed w/ server
								Handler: corev1.Handler{
									Exec: &corev1.ExecAction{
										Command: []string{"/bin/sh", "-ec", `curl http://127.0.0.1:8500/v1/status/leader 2>/dev/null | grep -E '".+"'`},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (in *Operator) ConnectServiceAccount() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-connect-injector-webhook-svc-account", in.Name),
			Namespace: in.Namespace,
			Labels: map[string]string{
				"app":     in.Name,
				"release": in.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(in, schema.GroupVersionKind{
					Group:   "consul.hashicorp.com",
					Version: "v1alpha1",
					Kind:    "Operator",
				}),
			},
		},
		ImagePullSecrets: in.imagePullSecretsList(),
	}
}

func (in *Operator) ConnectClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-connect-injector-webhook", in.Name),
			Namespace: in.Namespace,
			Labels: map[string]string{
				"app":     in.Name,
				"release": in.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(in, schema.GroupVersionKind{
					Group:   "consul.hashicorp.com",
					Version: "v1alpha1",
					Kind:    "Operator",
				}),
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				// TODO find the const for all these
				APIGroups: []string{"admissionregistration.k8s.io"},
				Resources: []string{"mutatingwebhookconfigurations"},
				Verbs: []string{
					"get",
					"list",
					"watch",
					"patch",
				},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs: []string{
					"get",
					"list",
					"watch",
				},
			},
		},
	}
}

func (in *Operator) ClientClusterRoleBinding() *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-connect-injector-webhook-admin-role-binding", in.Name),
			Labels: map[string]string{
				"app":     in.Name,
				"release": in.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(in, schema.GroupVersionKind{
					Group:   "consul.hashicorp.com",
					Version: "v1alpha1",
					Kind:    "Operator",
				}),
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     fmt.Sprintf("%s-connect-injector-webhook", in.Name),
		},
		Subjects: []rbacv1.Subject{
			{
				Namespace: in.Namespace,
				Kind:      rbacv1.ServiceAccountKind,
				Name:      fmt.Sprintf("%s-connect-injector-webhook-svc-account", in.Name),
			},
		},
	}

}

func (in *Operator) ConnectService() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-connect-injector-svc", in.Name),
			Namespace: in.Namespace,
			Labels: map[string]string{
				"app":     in.Name,
				"release": in.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(in, schema.GroupVersionKind{
					Group:   "consul.hashicorp.com",
					Version: "v1alpha1",
					Kind:    "Operator",
				}),
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port: 443,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 8080,
					},
				},
			},
			Selector: map[string]string{
				"app":       in.Name,
				"release":   in.Name,
				"component": "connect-injector",
			},
		},
	}
}

func (in *Operator) ConnectDeployment() *appsv1.Deployment {
	oneReplica := int32(1)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-connect-injector-webhook-deployment", in.Name),
			Namespace: in.Namespace,
			Labels: map[string]string{
				"app":     in.Name,
				"release": in.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(in, schema.GroupVersionKind{
					Group:   "consul.hashicorp.com",
					Version: "v1alpha1",
					Kind:    "Operator",
				}),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &oneReplica,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":       in.Name,
					"release":   in.Name,
					"component": "connect-injector",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":       in.Name,
						"release":   in.Name,
						"component": "connect-injector",
					},
					Annotations: map[string]string{
						"consul.hashicorp.com/connect-inject": "false",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: fmt.Sprintf("%s-connect-injector-webhook-svc-account", in.Name),
					Containers: []corev1.Container{
						{
							Name:  "sidecar-injector",
							Image: in.Spec.Global.ConsulK8sImage,
							Env: []corev1.EnvVar{
								{
									Name: "NAMESPACE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
								{
									Name:  "CONSUL_HTTP_ADDR",
									Value: "http://$(HOST_IP):8500",
								},
								{
									Name: "HOST_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.hostIP",
										},
									},
								},
							},
							Command: []string{"/bin/sh", "-ec", fmt.Sprintf(
								`exec consul-k8s inject-connect \
-default-inject=false \
-consul-image="%s" \
-envoy-image="%s" \
-consul-k8s-image="%s" \
-listen=:8080 \
-log-level=info \
-enable-health-checks-controller=%t \
-health-checks-reconcile-period=1m \
-enable-central-config=true \
-allow-k8s-namespace="*" \
-consul-destination-namespace=default \
-tls-auto=%s-connect-injector-cfg \
-tls-auto-hosts=%s-connect-injector-svc,%s-connect-injector-svc.%s,%s-connect-injector-svc.%s.svc`,
								in.Spec.Global.ConsulImage, in.Spec.Global.EnvoyImage, in.Spec.Global.ConsulK8sImage, in.Spec.Connect.HealthChecks,
								in.Name, in.Name, in.Name, in.Namespace, in.Name, in.Namespace,
							)},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/health/ready",
										Port: intstr.IntOrString{
											Type:   intstr.Int,
											IntVal: 8080,
										},
										Scheme: corev1.URISchemeHTTPS,
									},
								},
								FailureThreshold:    2,
								InitialDelaySeconds: 1,
								PeriodSeconds:       2,
								SuccessThreshold:    1,
								TimeoutSeconds:      5,
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/health/ready",
										Port: intstr.IntOrString{
											Type:   intstr.Int,
											IntVal: 8080,
										},
										Scheme: corev1.URISchemeHTTPS,
									},
								},
								FailureThreshold:    2,
								InitialDelaySeconds: 1,
								PeriodSeconds:       2,
								SuccessThreshold:    1,
								TimeoutSeconds:      5,
							},
							Ports:           nil,
							VolumeDevices:   nil,
							StartupProbe:    nil,
							Lifecycle:       nil,
							ImagePullPolicy: "",
							SecurityContext: nil,
						},
					},
				},
			},
		},
	}
}

func (in *Operator) ConnectWebhook() *admissionv1.MutatingWebhookConfiguration {
	failurePolicy := admissionv1.Ignore
	sideEffects := admissionv1.SideEffectClassNone
	webhookPath := "/mutate"
	return &admissionv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-connect-injector-cfg", in.Name),
			Namespace: in.Namespace,
			Labels: map[string]string{
				"app":     in.Name,
				"release": in.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(in, schema.GroupVersionKind{
					Group:   "consul.hashicorp.com",
					Version: "v1alpha1",
					Kind:    "Operator",
				}),
			},
		},
		Webhooks: []admissionv1.MutatingWebhook{
			{
				Name: fmt.Sprintf("%s-connect-injector.consul.hashicorp.com", in.Name),
				ClientConfig: admissionv1.WebhookClientConfig{
					Service: &admissionv1.ServiceReference{
						Namespace: in.Namespace,
						Name:      fmt.Sprintf("%s-connect-injector-svc", in.Name),
						Path:      &webhookPath,
					},
					CABundle: []byte(""),
				},
				Rules: []admissionv1.RuleWithOperations{
					{
						Operations: []admissionv1.OperationType{admissionv1.Create},
						Rule: admissionv1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"pods"},
						},
					},
				},
				FailurePolicy:           &failurePolicy,
				SideEffects:             &sideEffects,
				AdmissionReviewVersions: []string{"v1beta1", "v1"},
			},
		},
	}
}

func (in *Operator) uiServiceType() corev1.ServiceType {
	switch in.Spec.UI.Type {
	case "LoadBalancer":
		return corev1.ServiceTypeLoadBalancer
	case "ClusterIP":
		return corev1.ServiceTypeClusterIP
	case "NodePort":
		return corev1.ServiceTypeNodePort
	default:
		return corev1.ServiceTypeExternalName
	}
}

// +kubebuilder:object:root=true

// OperatorList contains a list of Operator
type OperatorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Operator `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Operator{}, &OperatorList{})
}
