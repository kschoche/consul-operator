package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	v1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	consulv1alpha1 "github.com/ashwin-venkatesh/consul-operator/api/v1alpha1"
)

// OperatorReconciler reconciles a Operator object
type OperatorReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=consul.hashicorp.com,resources=operators,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=consul.hashicorp.com,resources=operators/status,verbs=get;update;patch

func (r *OperatorReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	operator := &consulv1alpha1.Operator{}
	ctx := context.Background()
	log := r.Log.WithValues("operator", req.NamespacedName)

	err := r.Get(ctx, req.NamespacedName, operator)
	if err != nil {
		log.Error(err, "failed getting operator")
		return ctrl.Result{}, err
	}

	// Server Configuration
	if operator.Spec.Global.Enabled || operator.Spec.Server.Enabled {
		// Server Service Account
		serverServiceAccount := &corev1.ServiceAccount{}
		err = r.Get(ctx, types.NamespacedName{
			Namespace: operator.Namespace,
			Name:      fmt.Sprintf("%s-server", operator.Name),
		}, serverServiceAccount)

		if err != nil && errors.IsNotFound(err) {
			serverServiceAccount = operator.ServerServiceAccount()
			if err := r.Create(ctx, serverServiceAccount); err != nil {
				log.Error(err, "failed creating server serviceaccount")
				return ctrl.Result{}, err
			}
		} else if err != nil {
			log.Error(err, "failed getting server serviceaccount")
			return ctrl.Result{}, err
		} else {
			//return ctrl.Result{}, nil
		}

		// Server Role
		serverRole := &rbacv1.Role{}
		err := r.Get(ctx, types.NamespacedName{
			Namespace: operator.Namespace,
			Name:      fmt.Sprintf("%s-server", operator.Name),
		}, serverRole)

		if err != nil && errors.IsNotFound(err) {
			serverRole = operator.ServerRole()
			if err := r.Create(ctx, serverRole); err != nil {
				log.Error(err, "failed creating server role")
				return ctrl.Result{}, err
			}
		} else if err != nil {
			log.Error(err, "failed getting server role")
			return ctrl.Result{}, err
		} else {
			//return ctrl.Result{}, nil
		}

		// Server RoleBinding
		serverRoleBinding := &rbacv1.RoleBinding{}
		err = r.Get(ctx, types.NamespacedName{
			Namespace: operator.Namespace,
			Name:      fmt.Sprintf("%s-server", operator.Name),
		}, serverRoleBinding)

		if err != nil && errors.IsNotFound(err) {
			serverRoleBinding = operator.ServerRoleBinding()
			if err := r.Create(ctx, serverRoleBinding); err != nil {
				log.Error(err, "failed creating server rolebinding")
				return ctrl.Result{}, err
			}
		} else if err != nil {
			log.Error(err, "failed getting server rolebinding")
			return ctrl.Result{}, err
		} else {
			//return ctrl.Result{}, nil
		}

		// Server ConfigMap
		serverConfigMap := &corev1.ConfigMap{}
		err = r.Get(ctx, types.NamespacedName{
			Namespace: operator.Namespace,
			Name:      fmt.Sprintf("%s-server-config", operator.Name),
		}, serverConfigMap)

		if err != nil && errors.IsNotFound(err) {
			serverConfigMap = operator.ServerConfigMap()
			if err := r.Create(ctx, serverConfigMap); err != nil {
				log.Error(err, "failed creating server configmap")
				return ctrl.Result{}, err
			}
		} else if err != nil {
			log.Error(err, "failed getting server configmap")
			return ctrl.Result{}, err
		} else {
			//return ctrl.Result{}, nil
		}

		// Server Service
		serverService := &corev1.Service{}
		err = r.Get(ctx, types.NamespacedName{
			Namespace: operator.Namespace,
			Name:      fmt.Sprintf("%s-server", operator.Name),
		}, serverService)

		if err != nil && errors.IsNotFound(err) {
			serverService = operator.ServerService()
			if err := r.Create(ctx, serverService); err != nil {
				log.Error(err, "failed creating server service")
				return ctrl.Result{}, err
			}
		} else if err != nil {
			log.Error(err, "failed getting server service")
			return ctrl.Result{}, err
		} else {
			//return ctrl.Result{}, nil
		}

		// Server StatefulSet
		serverStatefulSet := &appsv1.StatefulSet{}
		err = r.Get(ctx, types.NamespacedName{
			Namespace: operator.Namespace,
			Name:      fmt.Sprintf("%s-server", operator.Name),
		}, serverStatefulSet)

		if err != nil && errors.IsNotFound(err) {
			serverStatefulSet = operator.ServerStatefulSet()
			if err := r.Create(ctx, serverStatefulSet); err != nil {
				log.Error(err, "failed creating server statefulset")
				return ctrl.Result{}, err
			}
		} else if err != nil {
			log.Error(err, "failed getting server statefulset")
			return ctrl.Result{}, err
		} else {
			expectedStatefulSet := operator.ServerStatefulSet()
			if !cmp.Equal(expectedStatefulSet.Spec.Template.Spec.Containers[0].Command, serverStatefulSet.Spec.Template.Spec.Containers[0].Command) {
				if err := r.Update(ctx, expectedStatefulSet); err != nil {
					log.Error(err, "failed updating server statefulset")
					return ctrl.Result{}, err
				}
			}
		}
	}
	if operator.Spec.UI.Enabled || operator.Spec.Global.Enabled {
		// UI Service
		uiService := &corev1.Service{}
		err = r.Get(ctx, types.NamespacedName{
			Namespace: operator.Namespace,
			Name:      fmt.Sprintf("%s-ui", operator.Name),
		}, uiService)

		if err != nil && errors.IsNotFound(err) {
			uiService = operator.UIService()
			if err := r.Create(ctx, uiService); err != nil {
				log.Error(err, "failed creating ui service")
				return ctrl.Result{}, err
			}
		} else if err != nil {
			log.Error(err, "failed getting ui service")
			return ctrl.Result{}, err
		} else {
			//return ctrl.Result{}, nil
		}
	}

	// Connect Inject Webhook
	if operator.Spec.Connect.Enabled || operator.Spec.Global.Enabled {
		// Connect Service Account
		connectServiceAccount := &corev1.ServiceAccount{}
		err = r.Get(ctx, types.NamespacedName{
			Namespace: operator.Namespace,
			Name:      fmt.Sprintf("%s-connect-injector-webhook-svc-account", operator.Name),
		}, connectServiceAccount)
		if err != nil && errors.IsNotFound(err) {
			connectServiceAccount = operator.ConnectServiceAccount()
			if err := r.Create(ctx, connectServiceAccount); err != nil {
				log.Error(err, "failed creating connect serviceaccount")
				return ctrl.Result{}, err
			}
		} else if err != nil {
			log.Error(err, "failed getting connect serviceaccount")
			return ctrl.Result{}, err
		} else {
			//return ctrl.Result{}, nil
		}

		// Connect ClusterRole
		connectClusterRole := &rbacv1.ClusterRole{}
		err := r.Get(ctx, types.NamespacedName{
			//Namespace: operator.Namespace,
			Name: fmt.Sprintf("%s-connect-injector-webhook", operator.Name),
		}, connectClusterRole)

		if err != nil && errors.IsNotFound(err) {
			connectClusterRole = operator.ConnectClusterRole()
			if err := r.Create(ctx, connectClusterRole); err != nil {
				log.Error(err, "failed creating connect cluster role")
				return ctrl.Result{}, err
			}
		} else if err != nil {
			log.Error(err, "failed getting connect cluster role")
			return ctrl.Result{}, err
		} else {
			//return ctrl.Result{}, nil
		}

		// Connect ClusterRolebinding
		clientClusterRoleBinding := &rbacv1.ClusterRoleBinding{}
		err = r.Get(ctx, types.NamespacedName{
			//Namespace: operator.Namespace,
			Name: fmt.Sprintf("%s-connect-injector-webhook-admin-role-binding", operator.Name),
		}, clientClusterRoleBinding)

		if err != nil && errors.IsNotFound(err) {
			clientClusterRoleBinding = operator.ClientClusterRoleBinding()
			if err := r.Create(ctx, clientClusterRoleBinding); err != nil {
				log.Error(err, "failed creating client clusterrolebinding")
				return ctrl.Result{}, err
			}
		} else if err != nil {
			log.Error(err, "failed getting client clusterrolebinding")
			return ctrl.Result{}, err
		} else {
			//return ctrl.Result{}, nil
		}

		// Connect Service
		connectService := &corev1.Service{}
		err = r.Get(ctx, types.NamespacedName{
			Namespace: operator.Namespace,
			Name:      fmt.Sprintf("%s-connect-injector-svc", operator.Name),
		}, connectService)

		if err != nil && errors.IsNotFound(err) {
			connectService = operator.ConnectService()
			if err := r.Create(ctx, connectService); err != nil {
				log.Error(err, "failed creating connect service")
				return ctrl.Result{}, err
			}
		} else if err != nil {
			log.Error(err, "failed getting connect service")
			return ctrl.Result{}, err
		} else {
			//return ctrl.Result{}, nil
		}

		// Connect Webhook
		connectWebhook := &v1.MutatingWebhookConfiguration{}
		err = r.Get(ctx, types.NamespacedName{
			//Namespace: operator.Namespace,
			Name: fmt.Sprintf("%s-connect-injector-cfg", operator.Name),
		}, connectWebhook)
		if err != nil && errors.IsNotFound(err) {
			connectWebhook = operator.ConnectWebhook()
			if err := r.Create(ctx, connectWebhook); err != nil {
				log.Error(err, "failed creating connect webhook")
				return ctrl.Result{}, err
			}
		} else if err != nil {
			log.Error(err, "failed getting connect webhook")
			return ctrl.Result{}, err
		} else {
			//return ctrl.Result{}, nil
		}

		// Connect Deployment
		connectDeployment := &appsv1.Deployment{}
		err = r.Get(ctx, types.NamespacedName{
			Namespace: operator.Namespace,
			Name:      fmt.Sprintf("%s-connect-injector-webhook-deployment", operator.Name),
		}, connectDeployment)

		if err != nil && errors.IsNotFound(err) {
			connectDeployment = operator.ConnectDeployment()
			if err := r.Create(ctx, connectDeployment); err != nil {
				log.Error(err, "failed creating connect deployment")
				return ctrl.Result{}, err
			}
		} else if err != nil {
			log.Error(err, "failed getting connect deployment")
			return ctrl.Result{}, err
		} else {
			expectedDeployment := operator.ConnectDeployment()
			if !cmp.Equal(expectedDeployment.Spec.Template.Spec.Containers[0].Command, connectDeployment.Spec.Template.Spec.Containers[0].Command) {
				if err := r.Update(ctx, expectedDeployment); err != nil {
					log.Error(err, "failed updating connect deployment")
					return ctrl.Result{}, err
				}
			}
		}

	}

	// Client Configuration
	if operator.Spec.Client.Enabled || operator.Spec.Global.Enabled {
		// Client Service Account
		clientServiceAccount := &corev1.ServiceAccount{}
		err = r.Get(ctx, types.NamespacedName{
			Namespace: operator.Namespace,
			Name:      fmt.Sprintf("%s-client", operator.Name),
		}, clientServiceAccount)

		if err != nil && errors.IsNotFound(err) {
			clientServiceAccount = operator.ClientServiceAccount()
			if err := r.Create(ctx, clientServiceAccount); err != nil {
				log.Error(err, "failed creating client serviceaccount")
				return ctrl.Result{}, err
			}
		} else if err != nil {
			log.Error(err, "failed getting client serviceaccount")
			return ctrl.Result{}, err
		} else {
			//return ctrl.Result{}, nil
		}

		// Client Role
		clientRole := &rbacv1.Role{}
		err := r.Get(ctx, types.NamespacedName{
			Namespace: operator.Namespace,
			Name:      fmt.Sprintf("%s-client", operator.Name),
		}, clientRole)

		if err != nil && errors.IsNotFound(err) {
			clientRole = operator.ClientRole()
			if err := r.Create(ctx, clientRole); err != nil {
				log.Error(err, "failed creating client role")
				return ctrl.Result{}, err
			}
		} else if err != nil {
			log.Error(err, "failed getting client role")
			return ctrl.Result{}, err
		} else {
			//return ctrl.Result{}, nil
		}

		// Client Role Binding
		clientRoleBinding := &rbacv1.RoleBinding{}
		err = r.Get(ctx, types.NamespacedName{
			Namespace: operator.Namespace,
			Name:      fmt.Sprintf("%s-client", operator.Name),
		}, clientRoleBinding)

		if err != nil && errors.IsNotFound(err) {
			clientRoleBinding = operator.ClientRoleBinding()
			if err := r.Create(ctx, clientRoleBinding); err != nil {
				log.Error(err, "failed creating client rolebinding")
				return ctrl.Result{}, err
			}
		} else if err != nil {
			log.Error(err, "failed getting client rolebinding")
			return ctrl.Result{}, err
		} else {
			//return ctrl.Result{}, nil
		}

		// Client ConfigMap
		clientConfigMap := &corev1.ConfigMap{}
		err = r.Get(ctx, types.NamespacedName{
			Namespace: operator.Namespace,
			Name:      fmt.Sprintf("%s-client-config", operator.Name),
		}, clientConfigMap)

		if err != nil && errors.IsNotFound(err) {
			clientConfigMap = operator.ClientConfigMap()
			if err := r.Create(ctx, clientConfigMap); err != nil {
				log.Error(err, "failed creating client configmap")
				return ctrl.Result{}, err
			}
		} else if err != nil {
			log.Error(err, "failed getting client configmap")
			return ctrl.Result{}, err
		} else {
			//return ctrl.Result{}, nil
		}

		// Client Daemonset
		clientDaemonset := &appsv1.DaemonSet{}
		err = r.Get(ctx, types.NamespacedName{
			Namespace: operator.Namespace,
			Name:      fmt.Sprintf("%s", operator.Name),
		}, clientDaemonset)
		if err != nil && errors.IsNotFound(err) {
			clientDaemonset = operator.ClientDaemonSet()
			if err := r.Create(ctx, clientDaemonset); err != nil {
				log.Error(err, "failed creating client daemonset")
				return ctrl.Result{}, err
			}
		} else if err != nil {
			log.Error(err, "failed getting client daemonset")
			return ctrl.Result{}, err
		} else {
			expectedDaemonset := operator.ClientDaemonSet()
			if !cmp.Equal(expectedDaemonset.Spec.Template.Spec.Containers[0].Command, clientDaemonset.Spec.Template.Spec.Containers[0].Command) {
				if err := r.Update(ctx, expectedDaemonset); err != nil {
					log.Error(err, "failed updating client daemonset")
					return ctrl.Result{}, err
				}
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *OperatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&consulv1alpha1.Operator{}).
		Complete(r)
}
