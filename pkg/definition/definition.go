package definition

import (
	"bytes"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	certificatesv1 "k8s.io/api/certificates/v1"
	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	flowcontrolv1 "k8s.io/api/flowcontrol/v1"
	networkingv1 "k8s.io/api/networking/v1"
	nodev1 "k8s.io/api/node/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	rbacv1beta1 "k8s.io/api/rbac/v1beta1"
	resourcev1beta1 "k8s.io/api/resource/v1beta1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/util/certificate/csr"
	"k8s.io/utils/ptr"
)

const (
	loadBalancerWidth = 16
	// labelNodeRolePrefix is a label prefix for node roles
	// It's copied over to here until it's merged in core: https://github.com/kubernetes/kubernetes/pull/39112
	labelNodeRolePrefix = "node-role.kubernetes.io/"
	// nodeLabelRole specifies the role of a node
	nodeLabelRole = "kubernetes.io/role"
)

var mapping = map[string]runtime.Object{
	"Pod":                            &corev1.PodList{},
	"PodDisruptionBudget":            &policyv1.PodDisruptionBudgetList{},
	"ReplicaSet":                     &appsv1.ReplicaSetList{},
	"DaemonSet":                      &appsv1.DaemonSetList{},
	"Deployment":                     &appsv1.DeploymentList{},
	"StatefulSet":                    &appsv1.StatefulSetList{},
	"Job":                            &batchv1.JobList{},
	"CronJob":                        &batchv1beta1.CronJobList{},
	"Ingress":                        &networkingv1.IngressList{},
	"Service":                        &corev1.ServiceList{},
	"Endpoints":                      &corev1.EndpointsList{},
	"Node":                           &corev1.NodeList{},
	"Namespace":                      &corev1.NamespaceList{},
	"Secret":                         &corev1.SecretList{},
	"ConfigMap":                      &corev1.ConfigMapList{},
	"PersistentVolume":               &corev1.PersistentVolumeList{},
	"PersistentVolumeClaim":          &corev1.PersistentVolumeClaimList{},
	"StorageClass":                   &storagev1.StorageClassList{},
	"PriorityClass":                  &schedulingv1.PriorityClassList{},
	"ServiceAccount":                 &corev1.ServiceAccountList{},
	"Role":                           &rbacv1.RoleList{},
	"ClusterRole":                    &rbacv1.ClusterRoleList{},
	"RoleBinding":                    &rbacv1.RoleBindingList{},
	"ClusterRoleBinding":             &rbacv1.ClusterRoleBindingList{},
	"HorizontalPodAutoscaler":        &autoscalingv2.HorizontalPodAutoscalerList{},
	"PriorityLevelConfiguration":     &flowcontrolv1.PriorityLevelConfigurationList{},
	"CSR":                            &certificatesv1.CertificateSigningRequestList{},
	"Lease":                          &coordinationv1.LeaseList{},
	"EndpointSlice":                  &discoveryv1.EndpointSliceList{},
	"LimitRange":                     &corev1.LimitRangeList{},
	"ResourceQuota":                  &corev1.ResourceQuotaList{},
	"Event":                          &corev1.EventList{},
	"MutatingWebhookConfiguration":   &admissionregistrationv1.MutatingWebhookConfigurationList{},
	"ValidatingWebhookConfiguration": &admissionregistrationv1.ValidatingWebhookConfigurationList{},
	"ResourceClaim":                  &resourcev1beta1.ResourceClaimList{},
	"ResourceSlice":                  &resourcev1beta1.ResourceSliceList{},
}

func IsSupportedKind(kind string) (runtime.Object, bool) {
	if _, ok := mapping[kind]; !ok {
		return nil, false
	}
	return mapping[kind], true
}

// AddHandlers adds print handlers for default Kubernetes types dealing with internal versions.
func AddHandlers(h *HumanReadableGenerator) {
	podColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Ready", Type: "string", Description: "The aggregate readiness state of this pod for accepting traffic."},
		{Name: "Status", Type: "string", Description: "The aggregate status of the containers in this pod."},
		{Name: "Restarts", Type: "string", Description: "The number of times the containers in this pod have been restarted and when the last container in this pod has restarted."},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
		{Name: "IP", Type: "string", Priority: 1, Description: corev1.PodStatus{}.SwaggerDoc()["podIP"]},
		{Name: "Node", Type: "string", Priority: 1, Description: corev1.PodSpec{}.SwaggerDoc()["nodeName"]},
		{Name: "Nominated Node", Type: "string", Priority: 1, Description: corev1.PodStatus{}.SwaggerDoc()["nominatedNodeName"]},
		{Name: "Readiness Gates", Type: "string", Priority: 1, Description: corev1.PodSpec{}.SwaggerDoc()["readinessGates"]},
	}
	_ = h.TableHandler(podColumnDefinitions, printPodList)

	podDisruptionBudgetColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Min Available", Type: "string", Description: "The minimum number of pods that must be available."},
		{Name: "Max Unavailable", Type: "string", Description: "The maximum number of pods that may be unavailable."},
		{Name: "Allowed Disruptions", Type: "integer", Description: "Calculated number of pods that may be disrupted at this time."},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(podDisruptionBudgetColumnDefinitions, printPodDisruptionBudgetList)

	replicaSetColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Desired", Type: "integer", Description: appsv1.ReplicaSetSpec{}.SwaggerDoc()["replicas"]},
		{Name: "Current", Type: "integer", Description: appsv1.ReplicaSetStatus{}.SwaggerDoc()["replicas"]},
		{Name: "Ready", Type: "integer", Description: appsv1.ReplicaSetStatus{}.SwaggerDoc()["readyReplicas"]},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
		{Name: "Containers", Type: "string", Priority: 1, Description: "Names of each container in the template."},
		{Name: "Images", Type: "string", Priority: 1, Description: "Images referenced by each container in the template."},
		{Name: "Selector", Type: "string", Priority: 1, Description: appsv1.ReplicaSetSpec{}.SwaggerDoc()["selector"]},
	}
	_ = h.TableHandler(replicaSetColumnDefinitions, printReplicaSetList)

	daemonSetColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Desired", Type: "integer", Description: appsv1.DaemonSetStatus{}.SwaggerDoc()["desiredNumberScheduled"]},
		{Name: "Current", Type: "integer", Description: appsv1.DaemonSetStatus{}.SwaggerDoc()["currentNumberScheduled"]},
		{Name: "Ready", Type: "integer", Description: appsv1.DaemonSetStatus{}.SwaggerDoc()["numberReady"]},
		{Name: "Up-to-date", Type: "integer", Description: appsv1.DaemonSetStatus{}.SwaggerDoc()["updatedNumberScheduled"]},
		{Name: "Available", Type: "integer", Description: appsv1.DaemonSetStatus{}.SwaggerDoc()["numberAvailable"]},
		{Name: "Node Selector", Type: "string", Description: corev1.PodSpec{}.SwaggerDoc()["nodeSelector"]},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
		{Name: "Containers", Type: "string", Priority: 1, Description: "Names of each container in the template."},
		{Name: "Images", Type: "string", Priority: 1, Description: "Images referenced by each container in the template."},
		{Name: "Selector", Type: "string", Priority: 1, Description: appsv1.DaemonSetSpec{}.SwaggerDoc()["selector"]},
	}
	_ = h.TableHandler(daemonSetColumnDefinitions, printDaemonSetList)

	jobColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Status", Type: "string", Description: "Status of the job."},
		{Name: "Completions", Type: "string", Description: batchv1.JobStatus{}.SwaggerDoc()["succeeded"]},
		{Name: "Duration", Type: "string", Description: "Time required to complete the job."},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
		{Name: "Containers", Type: "string", Priority: 1, Description: "Names of each container in the template."},
		{Name: "Images", Type: "string", Priority: 1, Description: "Images referenced by each container in the template."},
		{Name: "Selector", Type: "string", Priority: 1, Description: batchv1.JobSpec{}.SwaggerDoc()["selector"]},
	}
	_ = h.TableHandler(jobColumnDefinitions, printJobList)

	cronJobColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Schedule", Type: "string", Description: batchv1beta1.CronJobSpec{}.SwaggerDoc()["schedule"]},
		{Name: "Timezone", Type: "string", Description: batchv1beta1.CronJobSpec{}.SwaggerDoc()["timeZone"]},
		{Name: "Suspend", Type: "boolean", Description: batchv1beta1.CronJobSpec{}.SwaggerDoc()["suspend"]},
		{Name: "Active", Type: "integer", Description: batchv1beta1.CronJobStatus{}.SwaggerDoc()["active"]},
		{Name: "Last Schedule", Type: "string", Description: batchv1beta1.CronJobStatus{}.SwaggerDoc()["lastScheduleTime"]},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
		{Name: "Containers", Type: "string", Priority: 1, Description: "Names of each container in the template."},
		{Name: "Images", Type: "string", Priority: 1, Description: "Images referenced by each container in the template."},
		{Name: "Selector", Type: "string", Priority: 1, Description: batchv1.JobSpec{}.SwaggerDoc()["selector"]},
	}
	_ = h.TableHandler(cronJobColumnDefinitions, printCronJobList)

	serviceColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Type", Type: "string", Description: corev1.ServiceSpec{}.SwaggerDoc()["type"]},
		{Name: "Cluster-IP", Type: "string", Description: corev1.ServiceSpec{}.SwaggerDoc()["clusterIP"]},
		{Name: "External-IP", Type: "string", Description: corev1.ServiceSpec{}.SwaggerDoc()["externalIPs"]},
		{Name: "Port(s)", Type: "string", Description: corev1.ServiceSpec{}.SwaggerDoc()["ports"]},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
		{Name: "Selector", Type: "string", Priority: 1, Description: corev1.ServiceSpec{}.SwaggerDoc()["selector"]},
	}
	_ = h.TableHandler(serviceColumnDefinitions, printServiceList)

	ingressColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Class", Type: "string", Description: "The name of the IngressClass resource that should be used for additional configuration"},
		{Name: "Hosts", Type: "string", Description: "Hosts that incoming requests are matched against before the ingress rule"},
		{Name: "Address", Type: "string", Description: "Address is a list containing ingress points for the load-balancer"},
		{Name: "Ports", Type: "string", Description: "Ports of TLS configurations that open"},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(ingressColumnDefinitions, printIngressList)

	ingressClassColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Controller", Type: "string", Description: "Controller that is responsible for handling this class"},
		{Name: "Parameters", Type: "string", Description: "A reference to a resource with additional parameters"},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(ingressClassColumnDefinitions, printIngressClassList)

	statefulSetColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Ready", Type: "string", Description: "Number of the pod with ready state"},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
		{Name: "Containers", Type: "string", Priority: 1, Description: "Names of each container in the template."},
		{Name: "Images", Type: "string", Priority: 1, Description: "Images referenced by each container in the template."},
	}
	_ = h.TableHandler(statefulSetColumnDefinitions, printStatefulSetList)

	endpointColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Endpoints", Type: "string", Description: corev1.Endpoints{}.SwaggerDoc()["subsets"]},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(endpointColumnDefinitions, printEndpointsList)

	nodeColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Status", Type: "string", Description: "The status of the node"},
		{Name: "Roles", Type: "string", Description: "The roles of the node"},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
		{Name: "Version", Type: "string", Description: corev1.NodeSystemInfo{}.SwaggerDoc()["kubeletVersion"]},
		{Name: "Internal-IP", Type: "string", Priority: 1, Description: corev1.NodeStatus{}.SwaggerDoc()["addresses"]},
		{Name: "External-IP", Type: "string", Priority: 1, Description: corev1.NodeStatus{}.SwaggerDoc()["addresses"]},
		{Name: "OS-Image", Type: "string", Priority: 1, Description: corev1.NodeSystemInfo{}.SwaggerDoc()["osImage"]},
		{Name: "Kernel-Version", Type: "string", Priority: 1, Description: corev1.NodeSystemInfo{}.SwaggerDoc()["kernelVersion"]},
		{Name: "Container-Runtime", Type: "string", Priority: 1, Description: corev1.NodeSystemInfo{}.SwaggerDoc()["containerRuntimeVersion"]},
	}
	_ = h.TableHandler(nodeColumnDefinitions, printNodeList)

	eventColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Last Seen", Type: "string", Description: corev1.Event{}.SwaggerDoc()["lastTimestamp"]},
		{Name: "Type", Type: "string", Description: corev1.Event{}.SwaggerDoc()["type"]},
		{Name: "Reason", Type: "string", Description: corev1.Event{}.SwaggerDoc()["reason"]},
		{Name: "Object", Type: "string", Description: corev1.Event{}.SwaggerDoc()["involvedObject"]},
		{Name: "Subobject", Type: "string", Priority: 1, Description: corev1.Event{}.InvolvedObject.SwaggerDoc()["fieldPath"]},
		{Name: "Source", Type: "string", Priority: 1, Description: corev1.Event{}.SwaggerDoc()["source"]},
		{Name: "Message", Type: "string", Description: corev1.Event{}.SwaggerDoc()["message"]},
		{Name: "First Seen", Type: "string", Priority: 1, Description: corev1.Event{}.SwaggerDoc()["firstTimestamp"]},
		{Name: "Count", Type: "string", Priority: 1, Description: corev1.Event{}.SwaggerDoc()["count"]},
		{Name: "Name", Type: "string", Priority: 1, Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
	}
	_ = h.TableHandler(eventColumnDefinitions, printEventList)

	namespaceColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Status", Type: "string", Description: "The status of the namespace"},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(namespaceColumnDefinitions, printNamespaceList)

	secretColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Type", Type: "string", Description: corev1.Secret{}.SwaggerDoc()["type"]},
		{Name: "Data", Type: "string", Description: corev1.Secret{}.SwaggerDoc()["data"]},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(secretColumnDefinitions, printSecretList)

	serviceAccountColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Secrets", Type: "string", Description: corev1.ServiceAccount{}.SwaggerDoc()["secrets"]},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(serviceAccountColumnDefinitions, printServiceAccountList)

	persistentVolumeColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Capacity", Type: "string", Description: corev1.PersistentVolumeSpec{}.SwaggerDoc()["capacity"]},
		{Name: "Access Modes", Type: "string", Description: corev1.PersistentVolumeSpec{}.SwaggerDoc()["accessModes"]},
		{Name: "Reclaim Policy", Type: "string", Description: corev1.PersistentVolumeSpec{}.SwaggerDoc()["persistentVolumeReclaimPolicy"]},
		{Name: "Status", Type: "string", Description: corev1.PersistentVolumeStatus{}.SwaggerDoc()["phase"]},
		{Name: "Claim", Type: "string", Description: corev1.PersistentVolumeSpec{}.SwaggerDoc()["claimRef"]},
		{Name: "StorageClass", Type: "string", Description: "StorageClass of the pv"},
		{Name: "VolumeAttributesClass", Type: "string", Description: "VolumeAttributesClass of the pv"},
		{Name: "Reason", Type: "string", Description: corev1.PersistentVolumeStatus{}.SwaggerDoc()["reason"]},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
		{Name: "VolumeMode", Type: "string", Priority: 1, Description: corev1.PersistentVolumeSpec{}.SwaggerDoc()["volumeMode"]},
	}
	_ = h.TableHandler(persistentVolumeColumnDefinitions, printPersistentVolumeList)

	persistentVolumeClaimColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Status", Type: "string", Description: corev1.PersistentVolumeClaimStatus{}.SwaggerDoc()["phase"]},
		{Name: "Volume", Type: "string", Description: corev1.PersistentVolumeClaimSpec{}.SwaggerDoc()["volumeName"]},
		{Name: "Capacity", Type: "string", Description: corev1.PersistentVolumeClaimStatus{}.SwaggerDoc()["capacity"]},
		{Name: "Access Modes", Type: "string", Description: corev1.PersistentVolumeClaimStatus{}.SwaggerDoc()["accessModes"]},
		{Name: "StorageClass", Type: "string", Description: "StorageClass of the pvc"},
		{Name: "VolumeAttributesClass", Type: "string", Description: "VolumeAttributesClass of the pvc"},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
		{Name: "VolumeMode", Type: "string", Priority: 1, Description: corev1.PersistentVolumeClaimSpec{}.SwaggerDoc()["volumeMode"]},
	}
	_ = h.TableHandler(persistentVolumeClaimColumnDefinitions, printPersistentVolumeClaimList)

	deploymentColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Ready", Type: "string", Description: "Number of the pod with ready state"},
		{Name: "Up-to-date", Type: "string", Description: appsv1.DeploymentStatus{}.SwaggerDoc()["updatedReplicas"]},
		{Name: "Available", Type: "string", Description: appsv1.DeploymentStatus{}.SwaggerDoc()["availableReplicas"]},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
		{Name: "Containers", Type: "string", Priority: 1, Description: "Names of each container in the template."},
		{Name: "Images", Type: "string", Priority: 1, Description: "Images referenced by each container in the template."},
		{Name: "Selector", Type: "string", Priority: 1, Description: appsv1.DeploymentSpec{}.SwaggerDoc()["selector"]},
	}
	_ = h.TableHandler(deploymentColumnDefinitions, printDeploymentList)

	horizontalPodAutoscalerColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Reference", Type: "string", Description: autoscalingv2.HorizontalPodAutoscalerSpec{}.SwaggerDoc()["scaleTargetRef"]},
		{Name: "Targets", Type: "string", Description: autoscalingv2.HorizontalPodAutoscalerSpec{}.SwaggerDoc()["metrics"]},
		{Name: "MinPods", Type: "string", Description: autoscalingv2.HorizontalPodAutoscalerSpec{}.SwaggerDoc()["minReplicas"]},
		{Name: "MaxPods", Type: "string", Description: autoscalingv2.HorizontalPodAutoscalerSpec{}.SwaggerDoc()["maxReplicas"]},
		{Name: "Replicas", Type: "string", Description: autoscalingv2.HorizontalPodAutoscalerStatus{}.SwaggerDoc()["currentReplicas"]},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(horizontalPodAutoscalerColumnDefinitions, printHorizontalPodAutoscalerList)

	configMapColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Data", Type: "string", Description: corev1.ConfigMap{}.SwaggerDoc()["data"]},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(configMapColumnDefinitions, printConfigMapList)

	networkPolicyColumnDefinitioins := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Pod-Selector", Type: "string", Description: networkingv1.NetworkPolicySpec{}.SwaggerDoc()["podSelector"]},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(networkPolicyColumnDefinitioins, printNetworkPolicyList)

	roleBindingsColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Role", Type: "string", Description: rbacv1beta1.RoleBinding{}.SwaggerDoc()["roleRef"]},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
		{Name: "Users", Type: "string", Priority: 1, Description: "Users in the roleBinding"},
		{Name: "Groups", Type: "string", Priority: 1, Description: "Groups in the roleBinding"},
		{Name: "ServiceAccounts", Type: "string", Priority: 1, Description: "ServiceAccounts in the roleBinding"},
	}
	_ = h.TableHandler(roleBindingsColumnDefinitions, printRoleBindingList)

	clusterRoleBindingsColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Role", Type: "string", Description: rbacv1beta1.ClusterRoleBinding{}.SwaggerDoc()["roleRef"]},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
		{Name: "Users", Type: "string", Priority: 1, Description: "Users in the clusterRoleBinding"},
		{Name: "Groups", Type: "string", Priority: 1, Description: "Groups in the clusterRoleBinding"},
		{Name: "ServiceAccounts", Type: "string", Priority: 1, Description: "ServiceAccounts in the clusterRoleBinding"},
	}
	_ = h.TableHandler(clusterRoleBindingsColumnDefinitions, printClusterRoleBindingList)

	certificateSigningRequestColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
		{Name: "SignerName", Type: "string", Description: certificatesv1.CertificateSigningRequestSpec{}.SwaggerDoc()["signerName"]},
		{Name: "Requestor", Type: "string", Description: certificatesv1.CertificateSigningRequestSpec{}.SwaggerDoc()["request"]},
		{Name: "RequestedDuration", Type: "string", Description: certificatesv1.CertificateSigningRequestSpec{}.SwaggerDoc()["expirationSeconds"]},
		{Name: "Condition", Type: "string", Description: certificatesv1.CertificateSigningRequestStatus{}.SwaggerDoc()["conditions"]},
	}
	_ = h.TableHandler(certificateSigningRequestColumnDefinitions, printCertificateSigningRequestList)

	leaseColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Holder", Type: "string", Description: coordinationv1.LeaseSpec{}.SwaggerDoc()["holderIdentity"]},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(leaseColumnDefinitions, printLeaseList)

	storageClassColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Provisioner", Type: "string", Description: storagev1.StorageClass{}.SwaggerDoc()["provisioner"]},
		{Name: "ReclaimPolicy", Type: "string", Description: storagev1.StorageClass{}.SwaggerDoc()["reclaimPolicy"]},
		{Name: "VolumeBindingMode", Type: "string", Description: storagev1.StorageClass{}.SwaggerDoc()["volumeBindingMode"]},
		{Name: "AllowVolumeExpansion", Type: "string", Description: storagev1.StorageClass{}.SwaggerDoc()["allowVolumeExpansion"]},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(storageClassColumnDefinitions, printStorageClassList)

	controllerRevisionColumnDefinition := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Controller", Type: "string", Description: "Controller of the object"},
		{Name: "Revision", Type: "string", Description: appsv1.ControllerRevision{}.SwaggerDoc()["revision"]},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(controllerRevisionColumnDefinition, printControllerRevisionList)

	resourceQuotaColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
		{Name: "Request", Type: "string", Description: "Request represents a minimum amount of cpu/memory that a container may consume."},
		{Name: "Limit", Type: "string", Description: "Limits control the maximum amount of cpu/memory that a container may use independent of contention on the node."},
	}
	_ = h.TableHandler(resourceQuotaColumnDefinitions, printResourceQuotaList)

	priorityClassColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Value", Type: "integer", Description: schedulingv1.PriorityClass{}.SwaggerDoc()["value"]},
		{Name: "Global-Default", Type: "boolean", Description: schedulingv1.PriorityClass{}.SwaggerDoc()["globalDefault"]},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
		{Name: "PreemptionPolicy", Type: "string", Description: schedulingv1.PriorityClass{}.SwaggerDoc()["preemptionPolicy"]},
	}
	_ = h.TableHandler(priorityClassColumnDefinitions, printPriorityClassList)

	runtimeClassColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Handler", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["handler"]},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(runtimeClassColumnDefinitions, printRuntimeClassList)

	volumeAttachmentColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Attacher", Type: "string", Format: "name", Description: storagev1.VolumeAttachmentSpec{}.SwaggerDoc()["attacher"]},
		{Name: "PV", Type: "string", Description: storagev1.VolumeAttachmentSource{}.SwaggerDoc()["persistentVolumeName"]},
		{Name: "Node", Type: "string", Description: storagev1.VolumeAttachmentSpec{}.SwaggerDoc()["nodeName"]},
		{Name: "Attached", Type: "boolean", Description: storagev1.VolumeAttachmentStatus{}.SwaggerDoc()["attached"]},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(volumeAttachmentColumnDefinitions, printVolumeAttachmentList)

	endpointSliceColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "AddressType", Type: "string", Description: discoveryv1.EndpointSlice{}.SwaggerDoc()["addressType"]},
		{Name: "Ports", Type: "string", Description: discoveryv1.EndpointSlice{}.SwaggerDoc()["ports"]},
		{Name: "Endpoints", Type: "string", Description: discoveryv1.EndpointSlice{}.SwaggerDoc()["endpoints"]},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(endpointSliceColumnDefinitions, printEndpointSliceList)

	csiNodeColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Drivers", Type: "integer", Description: "Drivers indicates the number of CSI drivers registered on the node"},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(csiNodeColumnDefinitions, printCSINodeList)

	csiDriverColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "AttachRequired", Type: "boolean", Description: storagev1.CSIDriverSpec{}.SwaggerDoc()["attachRequired"]},
		{Name: "PodInfoOnMount", Type: "boolean", Description: storagev1.CSIDriverSpec{}.SwaggerDoc()["podInfoOnMount"]},
		{Name: "StorageCapacity", Type: "boolean", Description: storagev1.CSIDriverSpec{}.SwaggerDoc()["storageCapacity"]},
		{Name: "TokenRequests", Type: "string", Description: storagev1.CSIDriverSpec{}.SwaggerDoc()["tokenRequests"]},
		{Name: "RequiresRepublish", Type: "boolean", Description: storagev1.CSIDriverSpec{}.SwaggerDoc()["requiresRepublish"]},
		{Name: "Modes", Type: "string", Description: storagev1.CSIDriverSpec{}.SwaggerDoc()["volumeLifecycleModes"]},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(csiDriverColumnDefinitions, printCSIDriverList)

	csiStorageCapacityColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "StorageClassName", Type: "string", Description: storagev1.CSIStorageCapacity{}.SwaggerDoc()["storageClassName"]},
		{Name: "Capacity", Type: "string", Description: storagev1.CSIStorageCapacity{}.SwaggerDoc()["capacity"]},
	}
	_ = h.TableHandler(csiStorageCapacityColumnDefinitions, printCSIStorageCapacityList)

	mutatingWebhookColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Webhooks", Type: "integer", Description: "Webhooks indicates the number of webhooks registered in this configuration"},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(mutatingWebhookColumnDefinitions, printMutatingWebhookList)

	validatingWebhookColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Webhooks", Type: "integer", Description: "Webhooks indicates the number of webhooks registered in this configuration"},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(validatingWebhookColumnDefinitions, printValidatingWebhookList)

	flowSchemaColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "PriorityLevel", Type: "string", Description: flowcontrolv1.PriorityLevelConfigurationReference{}.SwaggerDoc()["name"]},
		{Name: "MatchingPrecedence", Type: "string", Description: flowcontrolv1.FlowSchemaSpec{}.SwaggerDoc()["matchingPrecedence"]},
		{Name: "DistinguisherMethod", Type: "string", Description: flowcontrolv1.FlowSchemaSpec{}.SwaggerDoc()["distinguisherMethod"]},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
		{Name: "MissingPL", Type: "string", Description: "references a broken or non-existent PriorityLevelConfiguration"},
	}
	_ = h.TableHandler(flowSchemaColumnDefinitions, printFlowSchemaList)

	priorityLevelColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Type", Type: "string", Description: flowcontrolv1.PriorityLevelConfigurationSpec{}.SwaggerDoc()["type"]},
		{Name: "NominalConcurrencyShares", Type: "string", Description: flowcontrolv1.LimitedPriorityLevelConfiguration{}.SwaggerDoc()["nominalConcurrencyShares"]},
		{Name: "Queues", Type: "string", Description: flowcontrolv1.QueuingConfiguration{}.SwaggerDoc()["queues"]},
		{Name: "HandSize", Type: "string", Description: flowcontrolv1.QueuingConfiguration{}.SwaggerDoc()["handSize"]},
		{Name: "QueueLengthLimit", Type: "string", Description: flowcontrolv1.QueuingConfiguration{}.SwaggerDoc()["queueLengthLimit"]},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(priorityLevelColumnDefinitions, printPriorityLevelConfigurationList)

	resourceClaimColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "State", Type: "string", Description: "A summary of the current state (allocated, pending, reserved, etc.)."},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(resourceClaimColumnDefinitions, printResourceClaimList)

	nodeResourceSliceColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Node", Type: "string", Description: resourcev1beta1.ResourceSliceSpec{}.SwaggerDoc()["nodeName"]},
		{Name: "Driver", Type: "string", Description: resourcev1beta1.ResourceSliceSpec{}.SwaggerDoc()["driver"]},
		{Name: "Pool", Type: "string", Description: resourcev1beta1.ResourcePool{}.SwaggerDoc()["name"]},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	_ = h.TableHandler(nodeResourceSliceColumnDefinitions, printResourceSliceList)
}

// Pass ports=nil for all ports.
func formatEndpoints(endpoints *corev1.Endpoints, ports sets.Set[string]) string {
	if len(endpoints.Subsets) == 0 {
		return "<none>"
	}
	var list []string
	maximum := 3
	more := false
	count := 0
	for i := range endpoints.Subsets {
		ss := &endpoints.Subsets[i]
		if len(ss.Ports) == 0 {
			// It's possible to have headless services with no ports.
			count += len(ss.Addresses)
			for i := range ss.Addresses {
				if len(list) == maximum {
					more = true
					// the next loop is redundant
					break
				}
				list = append(list, ss.Addresses[i].IP)
			}
			// avoid nesting code too deeply
			continue
		}

		// "Normal" services with ports defined.
		for i := range ss.Ports {
			port := &ss.Ports[i]
			if ports == nil || ports.Has(port.Name) {
				count += len(ss.Addresses)
				for i := range ss.Addresses {
					if len(list) == maximum {
						more = true
						// the next loop is redundant
						break
					}
					addr := &ss.Addresses[i]
					hostPort := net.JoinHostPort(addr.IP, strconv.Itoa(int(port.Port)))
					list = append(list, hostPort)
				}
			}
		}
	}

	ret := strings.Join(list, ",")
	if more {
		return fmt.Sprintf("%s + %d more...", ret, count-maximum)
	}
	return ret
}

func formatDiscoveryPorts(ports []discoveryv1.EndpointPort) string {
	var list []string
	maximum := 3
	more := false
	count := 0
	for _, port := range ports {
		if len(list) < maximum {
			portNum := "*"
			if port.Port != nil {
				portNum = strconv.Itoa(int(*port.Port))
			} else if port.Name != nil {
				portNum = *port.Name
			}
			list = append(list, portNum)
		} else if len(list) == maximum {
			more = true
		}
		count++
	}
	return listWithMoreString(list, more, count, maximum)
}

func formatDiscoveryEndpoints(endpoints []discoveryv1.Endpoint) string {
	var list []string
	maximum := 3
	more := false
	count := 0
	for _, endpoint := range endpoints {
		for _, address := range endpoint.Addresses {
			if len(list) < maximum {
				list = append(list, address)
			} else if len(list) == maximum {
				more = true
			}
			count++
		}
	}
	return listWithMoreString(list, more, count, maximum)
}

func listWithMoreString(list []string, more bool, count, max int) string {
	ret := strings.Join(list, ",")
	if more {
		return fmt.Sprintf("%s + %d more...", ret, count-max)
	}
	if ret == "" {
		ret = "<unset>"
	}
	return ret
}

// translateMicroTimestampSince returns the elapsed time since timestamp in
// human-readable approximation.
func translateMicroTimestampSince(timestamp metav1.MicroTime) string {
	if timestamp.IsZero() {
		return "<unknown>"
	}

	return duration.HumanDuration(time.Since(timestamp.Time))
}

// translateTimestampSince returns the elapsed time since timestamp in
// human-readable approximation.
func translateTimestampSince(timestamp metav1.Time) string {
	if timestamp.IsZero() {
		return "<unknown>"
	}

	return duration.HumanDuration(time.Since(timestamp.Time))
}

var (
	podSuccessConditions = []metav1.TableRowCondition{{Type: metav1.RowCompleted, Status: metav1.ConditionTrue, Reason: string(corev1.PodSucceeded), Message: "The pod has completed successfully."}}
	podFailedConditions  = []metav1.TableRowCondition{{Type: metav1.RowCompleted, Status: metav1.ConditionTrue, Reason: string(corev1.PodFailed), Message: "The pod failed."}}
)

func printPodList(podList *corev1.PodList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(podList.Items))
	for i := range podList.Items {
		r, err := printPod(&podList.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printPod(pod *corev1.Pod) ([]metav1.TableRow, error) {
	restarts := 0
	restartableInitContainerRestarts := 0
	totalContainers := len(pod.Spec.Containers)
	readyContainers := 0
	lastRestartDate := metav1.NewTime(time.Time{})
	lastRestartableInitContainerRestartDate := metav1.NewTime(time.Time{})

	podPhase := pod.Status.Phase
	reason := string(podPhase)
	if pod.Status.Reason != "" {
		reason = pod.Status.Reason
	}

	// If the Pod carries {type:PodScheduled, reason:SchedulingGated}, set reason to 'SchedulingGated'.
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodScheduled && condition.Reason == corev1.PodReasonSchedulingGated {
			reason = corev1.PodReasonSchedulingGated
		}
	}

	row := metav1.TableRow{}

	switch pod.Status.Phase {
	case corev1.PodSucceeded:
		row.Conditions = podSuccessConditions
	case corev1.PodFailed:
		row.Conditions = podFailedConditions
	}

	initContainers := make(map[string]*corev1.Container)
	for i := range pod.Spec.InitContainers {
		initContainers[pod.Spec.InitContainers[i].Name] = &pod.Spec.InitContainers[i]
		if IsRestartableInitContainer(&pod.Spec.InitContainers[i]) {
			totalContainers++
		}
	}

	initializing := false
	for i := range pod.Status.InitContainerStatuses {
		container := pod.Status.InitContainerStatuses[i]
		restarts += int(container.RestartCount)
		if container.LastTerminationState.Terminated != nil {
			terminatedDate := container.LastTerminationState.Terminated.FinishedAt
			if lastRestartDate.Before(&terminatedDate) {
				lastRestartDate = terminatedDate
			}
		}
		if IsRestartableInitContainer(initContainers[container.Name]) {
			restartableInitContainerRestarts += int(container.RestartCount)
			if container.LastTerminationState.Terminated != nil {
				terminatedDate := container.LastTerminationState.Terminated.FinishedAt
				if lastRestartableInitContainerRestartDate.Before(&terminatedDate) {
					lastRestartableInitContainerRestartDate = terminatedDate
				}
			}
		}
		switch {
		case container.State.Terminated != nil && container.State.Terminated.ExitCode == 0:
			continue
		case IsRestartableInitContainer(initContainers[container.Name]) &&
			container.Started != nil && *container.Started:
			if container.Ready {
				readyContainers++
			}
			continue
		case container.State.Terminated != nil:
			// initialization is failed
			if len(container.State.Terminated.Reason) == 0 {
				if container.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Init:Signal:%d", container.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("Init:ExitCode:%d", container.State.Terminated.ExitCode)
				}
			} else {
				reason = "Init:" + container.State.Terminated.Reason
			}
			initializing = true
		case container.State.Waiting != nil && len(container.State.Waiting.Reason) > 0 && container.State.Waiting.Reason != "PodInitializing":
			reason = "Init:" + container.State.Waiting.Reason
			initializing = true
		default:
			reason = fmt.Sprintf("Init:%d/%d", i, len(pod.Spec.InitContainers))
			initializing = true
		}
		break
	}

	if !initializing || isPodInitializedConditionTrue(&pod.Status) {
		restarts = restartableInitContainerRestarts
		lastRestartDate = lastRestartableInitContainerRestartDate
		hasRunning := false
		for i := len(pod.Status.ContainerStatuses) - 1; i >= 0; i-- {
			container := pod.Status.ContainerStatuses[i]

			restarts += int(container.RestartCount)
			if container.LastTerminationState.Terminated != nil {
				terminatedDate := container.LastTerminationState.Terminated.FinishedAt
				if lastRestartDate.Before(&terminatedDate) {
					lastRestartDate = terminatedDate
				}
			}
			if container.State.Waiting != nil && container.State.Waiting.Reason != "" {
				reason = container.State.Waiting.Reason
			} else if container.State.Terminated != nil && container.State.Terminated.Reason != "" {
				reason = container.State.Terminated.Reason
			} else if container.State.Terminated != nil && container.State.Terminated.Reason == "" {
				if container.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Signal:%d", container.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("ExitCode:%d", container.State.Terminated.ExitCode)
				}
			} else if container.Ready && container.State.Running != nil {
				hasRunning = true
				readyContainers++
			}
		}

		// change pod status back to "Running" if there is at least one container still reporting as "Running" status
		if reason == "Completed" && hasRunning {
			if hasPodReadyCondition(pod.Status.Conditions) {
				reason = "Running"
			} else {
				reason = "NotReady"
			}
		}
	}

	if !pod.DeletionTimestamp.IsZero() && pod.Status.Reason == NodeUnreachablePodReason {
		reason = "Unknown"
	} else if !pod.DeletionTimestamp.IsZero() && !IsPodPhaseTerminal(podPhase) {
		reason = "Terminating"
	}

	restartsStr := strconv.Itoa(restarts)
	if restarts != 0 && !lastRestartDate.IsZero() {
		restartsStr = fmt.Sprintf("%d (%s ago)", restarts, translateTimestampSince(lastRestartDate))
	}

	row.Cells = append(row.Cells, pod.Name, fmt.Sprintf("%d/%d", readyContainers, totalContainers), reason, restartsStr, translateTimestampSince(pod.CreationTimestamp))
	nodeName := pod.Spec.NodeName
	nominatedNodeName := pod.Status.NominatedNodeName
	podIP := ""
	if len(pod.Status.PodIPs) > 0 {
		podIP = pod.Status.PodIPs[0].IP
	}

	if podIP == "" {
		podIP = "<none>"
	}
	if nodeName == "" {
		nodeName = "<none>"
	}
	if nominatedNodeName == "" {
		nominatedNodeName = "<none>"
	}

	readinessGates := "<none>"
	if len(pod.Spec.ReadinessGates) > 0 {
		trueConditions := 0
		for _, readinessGate := range pod.Spec.ReadinessGates {
			conditionType := readinessGate.ConditionType
			for _, condition := range pod.Status.Conditions {
				if condition.Type == conditionType {
					if condition.Status == corev1.ConditionTrue {
						trueConditions++
					}
					break
				}
			}
		}
		readinessGates = fmt.Sprintf("%d/%d", trueConditions, len(pod.Spec.ReadinessGates))
	}
	row.Cells = append(row.Cells, podIP, nodeName, nominatedNodeName, readinessGates)

	return []metav1.TableRow{row}, nil
}

func hasPodReadyCondition(conditions []corev1.PodCondition) bool {
	for _, condition := range conditions {
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func hasJobCondition(conditions []batchv1.JobCondition, conditionType batchv1.JobConditionType) bool {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func printPodDisruptionBudget(obj *policyv1.PodDisruptionBudget) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}

	var minAvailable string
	var maxUnavailable string
	if obj.Spec.MinAvailable != nil {
		minAvailable = obj.Spec.MinAvailable.String()
	} else {
		minAvailable = "N/A"
	}

	if obj.Spec.MaxUnavailable != nil {
		maxUnavailable = obj.Spec.MaxUnavailable.String()
	} else {
		maxUnavailable = "N/A"
	}

	row.Cells = append(row.Cells, obj.Name, minAvailable, maxUnavailable, int64(obj.Status.DisruptionsAllowed), translateTimestampSince(obj.CreationTimestamp))
	return []metav1.TableRow{row}, nil
}

func printPodDisruptionBudgetList(list *policyv1.PodDisruptionBudgetList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printPodDisruptionBudget(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printReplicaSet(obj *appsv1.ReplicaSet) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}

	desiredReplicas := obj.Spec.Replicas
	currentReplicas := obj.Status.Replicas
	readyReplicas := obj.Status.ReadyReplicas

	row.Cells = append(row.Cells, obj.Name, ptr.Deref(desiredReplicas, 0), int64(currentReplicas), int64(readyReplicas), translateTimestampSince(obj.CreationTimestamp))
	names, images := layoutContainerCells(obj.Spec.Template.Spec.Containers)
	row.Cells = append(row.Cells, names, images, metav1.FormatLabelSelector(obj.Spec.Selector))
	return []metav1.TableRow{row}, nil
}

func printReplicaSetList(list *appsv1.ReplicaSetList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printReplicaSet(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printJob(obj *batchv1.Job) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}

	var completions string
	if obj.Spec.Completions != nil {
		completions = fmt.Sprintf("%d/%d", obj.Status.Succeeded, *obj.Spec.Completions)
	} else {
		parallelism := int32(0)
		if obj.Spec.Parallelism != nil {
			parallelism = *obj.Spec.Parallelism
		}
		if parallelism > 1 {
			completions = fmt.Sprintf("%d/1 of %d", obj.Status.Succeeded, parallelism)
		} else {
			completions = fmt.Sprintf("%d/1", obj.Status.Succeeded)
		}
	}
	var jobDuration string
	switch {
	case obj.Status.StartTime == nil:
	case obj.Status.CompletionTime == nil:
		jobDuration = duration.HumanDuration(time.Since(obj.Status.StartTime.Time))
	default:
		jobDuration = duration.HumanDuration(obj.Status.CompletionTime.Sub(obj.Status.StartTime.Time))
	}
	var status string
	if hasJobCondition(obj.Status.Conditions, batchv1.JobComplete) {
		status = "Complete"
	} else if hasJobCondition(obj.Status.Conditions, batchv1.JobFailed) {
		status = "Failed"
	} else if !obj.ObjectMeta.DeletionTimestamp.IsZero() {
		status = "Terminating"
	} else if hasJobCondition(obj.Status.Conditions, batchv1.JobSuspended) {
		status = "Suspended"
	} else if hasJobCondition(obj.Status.Conditions, batchv1.JobFailureTarget) {
		status = "FailureTarget"
	} else {
		status = "Running"
	}

	row.Cells = append(row.Cells, obj.Name, status, completions, jobDuration, translateTimestampSince(obj.CreationTimestamp))
	names, images := layoutContainerCells(obj.Spec.Template.Spec.Containers)
	row.Cells = append(row.Cells, names, images, metav1.FormatLabelSelector(obj.Spec.Selector))
	return []metav1.TableRow{row}, nil
}

func printJobList(list *batchv1.JobList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printJob(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printCronJob(obj *batchv1.CronJob) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}

	lastScheduleTime := "<none>"
	if obj.Status.LastScheduleTime != nil {
		lastScheduleTime = translateTimestampSince(*obj.Status.LastScheduleTime)
	}

	timeZone := "<none>"
	if obj.Spec.TimeZone != nil {
		timeZone = *obj.Spec.TimeZone
	}

	row.Cells = append(row.Cells, obj.Name, obj.Spec.Schedule, timeZone, printBoolPtr(obj.Spec.Suspend), int64(len(obj.Status.Active)), lastScheduleTime, translateTimestampSince(obj.CreationTimestamp))
	names, images := layoutContainerCells(obj.Spec.JobTemplate.Spec.Template.Spec.Containers)
	row.Cells = append(row.Cells, names, images, metav1.FormatLabelSelector(obj.Spec.JobTemplate.Spec.Selector))
	return []metav1.TableRow{row}, nil
}

func printCronJobList(list *batchv1.CronJobList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printCronJob(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

// loadBalancerStatusStringer behaves mostly like a string interface and converts the given status to a string.
// `wide` indicates whether the returned value is meant for --o=wide output. If not, it's clipped to 16 bytes.
func loadBalancerStatusStringer(s corev1.LoadBalancerStatus, wide bool) string {
	ingress := s.Ingress
	result := sets.New[string]()
	for i := range ingress {
		if ingress[i].IP != "" {
			result.Insert(ingress[i].IP)
		} else if ingress[i].Hostname != "" {
			result.Insert(ingress[i].Hostname)
		}
	}

	r := strings.Join(result.UnsortedList(), ",")
	if !wide && len(r) > loadBalancerWidth {
		r = r[0:(loadBalancerWidth-3)] + "..."
	}
	return r
}

func getServiceExternalIP(svc *corev1.Service, wide bool) string {
	switch svc.Spec.Type {
	case corev1.ServiceTypeClusterIP:
		if len(svc.Spec.ExternalIPs) > 0 {
			return strings.Join(svc.Spec.ExternalIPs, ",")
		}
		return "<none>"
	case corev1.ServiceTypeNodePort:
		if len(svc.Spec.ExternalIPs) > 0 {
			return strings.Join(svc.Spec.ExternalIPs, ",")
		}
		return "<none>"
	case corev1.ServiceTypeLoadBalancer:
		lbIps := loadBalancerStatusStringer(svc.Status.LoadBalancer, wide)
		if len(svc.Spec.ExternalIPs) > 0 {
			var results []string
			if len(lbIps) > 0 {
				results = append(results, strings.Split(lbIps, ",")...)
			}
			results = append(results, svc.Spec.ExternalIPs...)
			return strings.Join(results, ",")
		}
		if len(lbIps) > 0 {
			return lbIps
		}
		return "<pending>"
	case corev1.ServiceTypeExternalName:
		return svc.Spec.ExternalName
	}
	return "<unknown>"
}

func makePortString(ports []corev1.ServicePort) string {
	pieces := make([]string, len(ports))
	for ix := range ports {
		port := &ports[ix]
		pieces[ix] = fmt.Sprintf("%d/%s", port.Port, port.Protocol)
		if port.NodePort > 0 {
			pieces[ix] = fmt.Sprintf("%d:%d/%s", port.Port, port.NodePort, port.Protocol)
		}
	}
	return strings.Join(pieces, ",")
}

func printService(obj *corev1.Service) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}
	svcType := obj.Spec.Type
	internalIP := "<none>"
	if len(obj.Spec.ClusterIPs) > 0 {
		internalIP = obj.Spec.ClusterIPs[0]
	}

	externalIP := getServiceExternalIP(obj, true)
	svcPorts := makePortString(obj.Spec.Ports)
	if len(svcPorts) == 0 {
		svcPorts = "<none>"
	}

	row.Cells = append(row.Cells, obj.Name, string(svcType), internalIP, externalIP, svcPorts, translateTimestampSince(obj.CreationTimestamp))
	row.Cells = append(row.Cells, labels.FormatLabels(obj.Spec.Selector))

	return []metav1.TableRow{row}, nil
}

func printServiceList(list *corev1.ServiceList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printService(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func formatHosts(rules []networkingv1.IngressRule) string {
	var list []string
	maximum := 3
	more := false
	for _, rule := range rules {
		if len(list) == maximum {
			more = true
		}
		if !more && len(rule.Host) != 0 {
			list = append(list, rule.Host)
		}
	}
	if len(list) == 0 {
		return "*"
	}
	ret := strings.Join(list, ",")
	if more {
		return fmt.Sprintf("%s + %d more...", ret, len(rules)-maximum)
	}
	return ret
}

func formatPorts(tls []networkingv1.IngressTLS) string {
	if len(tls) != 0 {
		return "80, 443"
	}
	return "80"
}

func printIngress(obj *networkingv1.Ingress) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}
	className := "<none>"
	if obj.Spec.IngressClassName != nil {
		className = *obj.Spec.IngressClassName
	}
	hosts := formatHosts(obj.Spec.Rules)
	address := ingressLoadBalancerStatusStringer(obj.Status.LoadBalancer, true)
	ports := formatPorts(obj.Spec.TLS)
	createTime := translateTimestampSince(obj.CreationTimestamp)
	row.Cells = append(row.Cells, obj.Name, className, hosts, address, ports, createTime)
	return []metav1.TableRow{row}, nil
}

// ingressLoadBalancerStatusStringer behaves mostly like a string interface and converts the given status to a string.
// `wide` indicates whether the returned value is meant for --o=wide output. If not, it's clipped to 16 bytes.
func ingressLoadBalancerStatusStringer(s networkingv1.IngressLoadBalancerStatus, wide bool) string {
	ingress := s.Ingress
	result := sets.New[string]()
	for i := range ingress {
		if ingress[i].IP != "" {
			result.Insert(ingress[i].IP)
		} else if ingress[i].Hostname != "" {
			result.Insert(ingress[i].Hostname)
		}
	}

	r := strings.Join(result.UnsortedList(), ",")
	if !wide && len(r) > loadBalancerWidth {
		r = r[0:(loadBalancerWidth-3)] + "..."
	}
	return r
}

func printIngressList(list *networkingv1.IngressList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printIngress(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printIngressClass(obj *networkingv1.IngressClass) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}
	parameters := "<none>"
	if obj.Spec.Parameters != nil {
		parameters = obj.Spec.Parameters.Kind
		if obj.Spec.Parameters.APIGroup != nil {
			parameters = parameters + "." + *obj.Spec.Parameters.APIGroup
		}
		parameters = parameters + "/" + obj.Spec.Parameters.Name
	}
	createTime := translateTimestampSince(obj.CreationTimestamp)
	row.Cells = append(row.Cells, obj.Name, obj.Spec.Controller, parameters, createTime)
	return []metav1.TableRow{row}, nil
}

func printIngressClassList(list *networkingv1.IngressClassList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printIngressClass(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printStatefulSet(obj *appsv1.StatefulSet) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}
	desiredReplicas := obj.Spec.Replicas
	readyReplicas := obj.Status.ReadyReplicas
	createTime := translateTimestampSince(obj.CreationTimestamp)
	row.Cells = append(row.Cells, obj.Name, fmt.Sprintf("%d/%d", int64(readyReplicas), ptr.Deref(desiredReplicas, 0)), createTime)
	names, images := layoutContainerCells(obj.Spec.Template.Spec.Containers)
	row.Cells = append(row.Cells, names, images)
	return []metav1.TableRow{row}, nil
}

func printStatefulSetList(list *appsv1.StatefulSetList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printStatefulSet(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printDaemonSet(obj *appsv1.DaemonSet) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}

	desiredScheduled := obj.Status.DesiredNumberScheduled
	currentScheduled := obj.Status.CurrentNumberScheduled
	numberReady := obj.Status.NumberReady
	numberUpdated := obj.Status.UpdatedNumberScheduled
	numberAvailable := obj.Status.NumberAvailable

	row.Cells = append(row.Cells, obj.Name, int64(desiredScheduled), int64(currentScheduled), int64(numberReady), int64(numberUpdated), int64(numberAvailable), labels.FormatLabels(obj.Spec.Template.Spec.NodeSelector), translateTimestampSince(obj.CreationTimestamp))
	names, images := layoutContainerCells(obj.Spec.Template.Spec.Containers)
	row.Cells = append(row.Cells, names, images, metav1.FormatLabelSelector(obj.Spec.Selector))
	return []metav1.TableRow{row}, nil
}

func printDaemonSetList(list *appsv1.DaemonSetList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printDaemonSet(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printEndpoints(obj *corev1.Endpoints) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}
	row.Cells = append(row.Cells, obj.Name, formatEndpoints(obj, nil), translateTimestampSince(obj.CreationTimestamp))
	return []metav1.TableRow{row}, nil
}

func printEndpointsList(list *corev1.EndpointsList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printEndpoints(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printEndpointSlice(obj *discoveryv1.EndpointSlice) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}
	row.Cells = append(row.Cells, obj.Name, string(obj.AddressType), formatDiscoveryPorts(obj.Ports), formatDiscoveryEndpoints(obj.Endpoints), translateTimestampSince(obj.CreationTimestamp))
	return []metav1.TableRow{row}, nil
}

func printEndpointSliceList(list *discoveryv1.EndpointSliceList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printEndpointSlice(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printCSINode(obj *storagev1.CSINode) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}
	row.Cells = append(row.Cells, obj.Name, int64(len(obj.Spec.Drivers)), translateTimestampSince(obj.CreationTimestamp))
	return []metav1.TableRow{row}, nil
}

func printCSINodeList(list *storagev1.CSINodeList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printCSINode(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printCSIDriver(obj *storagev1.CSIDriver) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}
	attachRequired := true
	if obj.Spec.AttachRequired != nil {
		attachRequired = *obj.Spec.AttachRequired
	}
	podInfoOnMount := false
	if obj.Spec.PodInfoOnMount != nil {
		podInfoOnMount = *obj.Spec.PodInfoOnMount
	}
	var allModes []string
	for _, mode := range obj.Spec.VolumeLifecycleModes {
		allModes = append(allModes, string(mode))
	}
	modes := strings.Join(allModes, ",")
	if len(modes) == 0 {
		modes = "<none>"
	}

	row.Cells = append(row.Cells, obj.Name, attachRequired, podInfoOnMount)
	storageCapacity := false
	if obj.Spec.StorageCapacity != nil {
		storageCapacity = *obj.Spec.StorageCapacity
	}
	row.Cells = append(row.Cells, storageCapacity)

	tokenRequests := "<unset>"
	if obj.Spec.TokenRequests != nil {
		var audiences []string
		for _, t := range obj.Spec.TokenRequests {
			audiences = append(audiences, t.Audience)
		}
		tokenRequests = strings.Join(audiences, ",")
	}
	requiresRepublish := false
	if obj.Spec.RequiresRepublish != nil {
		requiresRepublish = *obj.Spec.RequiresRepublish
	}
	row.Cells = append(row.Cells, tokenRequests, requiresRepublish)

	row.Cells = append(row.Cells, modes, translateTimestampSince(obj.CreationTimestamp))
	return []metav1.TableRow{row}, nil
}

func printCSIDriverList(list *storagev1.CSIDriverList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printCSIDriver(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printCSIStorageCapacity(obj *storagev1.CSIStorageCapacity) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}

	capacity := "<unset>"
	if obj.Capacity != nil {
		capacity = obj.Capacity.String()
	}

	row.Cells = append(row.Cells, obj.Name, obj.StorageClassName, capacity)
	return []metav1.TableRow{row}, nil
}

func printCSIStorageCapacityList(list *storagev1.CSIStorageCapacityList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printCSIStorageCapacity(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printMutatingWebhook(obj *admissionregistrationv1.MutatingWebhookConfiguration) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}
	row.Cells = append(row.Cells, obj.Name, int64(len(obj.Webhooks)), translateTimestampSince(obj.CreationTimestamp))
	return []metav1.TableRow{row}, nil
}

func printMutatingWebhookList(list *admissionregistrationv1.MutatingWebhookConfigurationList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printMutatingWebhook(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printValidatingWebhook(obj *admissionregistrationv1.ValidatingWebhookConfiguration) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}
	row.Cells = append(row.Cells, obj.Name, int64(len(obj.Webhooks)), translateTimestampSince(obj.CreationTimestamp))
	return []metav1.TableRow{row}, nil
}

func printValidatingWebhookList(list *admissionregistrationv1.ValidatingWebhookConfigurationList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printValidatingWebhook(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printNamespace(obj *corev1.Namespace) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}
	row.Cells = append(row.Cells, obj.Name, string(obj.Status.Phase), translateTimestampSince(obj.CreationTimestamp))
	return []metav1.TableRow{row}, nil
}

func printNamespaceList(list *corev1.NamespaceList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printNamespace(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printSecret(obj *corev1.Secret) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}
	row.Cells = append(row.Cells, obj.Name, string(obj.Type), int64(len(obj.Data)), translateTimestampSince(obj.CreationTimestamp))
	return []metav1.TableRow{row}, nil
}

func printSecretList(list *corev1.SecretList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printSecret(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printServiceAccount(obj *corev1.ServiceAccount) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}
	row.Cells = append(row.Cells, obj.Name, int64(len(obj.Secrets)), translateTimestampSince(obj.CreationTimestamp))
	return []metav1.TableRow{row}, nil
}

func printServiceAccountList(list *corev1.ServiceAccountList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printServiceAccount(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printNode(obj *corev1.Node) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}

	conditionMap := make(map[corev1.NodeConditionType]*corev1.NodeCondition)
	NodeAllConditions := []corev1.NodeConditionType{corev1.NodeReady}
	for i := range obj.Status.Conditions {
		cond := obj.Status.Conditions[i]
		conditionMap[cond.Type] = &cond
	}
	var status []string
	for _, validCondition := range NodeAllConditions {
		if condition, ok := conditionMap[validCondition]; ok {
			if condition.Status == corev1.ConditionTrue {
				status = append(status, string(condition.Type))
			} else {
				status = append(status, "Not"+string(condition.Type))
			}
		}
	}
	if len(status) == 0 {
		status = append(status, "Unknown")
	}
	if obj.Spec.Unschedulable {
		status = append(status, "SchedulingDisabled")
	}

	roles := strings.Join(findNodeRoles(obj), ",")
	if len(roles) == 0 {
		roles = "<none>"
	}

	row.Cells = append(row.Cells, obj.Name, strings.Join(status, ","), roles, translateTimestampSince(obj.CreationTimestamp), obj.Status.NodeInfo.KubeletVersion)
	osImage, kernelVersion, crVersion := obj.Status.NodeInfo.OSImage, obj.Status.NodeInfo.KernelVersion, obj.Status.NodeInfo.ContainerRuntimeVersion
	if osImage == "" {
		osImage = "<unknown>"
	}
	if kernelVersion == "" {
		kernelVersion = "<unknown>"
	}
	if crVersion == "" {
		crVersion = "<unknown>"
	}
	row.Cells = append(row.Cells, getNodeInternalIP(obj), getNodeExternalIP(obj), osImage, kernelVersion, crVersion)

	return []metav1.TableRow{row}, nil
}

// Returns first external ip of the node or "<none>" if none is found.
func getNodeExternalIP(node *corev1.Node) string {
	for _, address := range node.Status.Addresses {
		if address.Type == corev1.NodeExternalIP {
			return address.Address
		}
	}

	return "<none>"
}

// Returns the internal IP of the node or "<none>" if none is found.
func getNodeInternalIP(node *corev1.Node) string {
	for _, address := range node.Status.Addresses {
		if address.Type == corev1.NodeInternalIP {
			return address.Address
		}
	}

	return "<none>"
}

// findNodeRoles returns the roles of a given node.
// The roles are determined by looking for:
// * a node-role.kubernetes.io/<role>="" label
// * a kubernetes.io/role="<role>" label
func findNodeRoles(node *corev1.Node) []string {
	roles := sets.New[string]()
	for k, v := range node.Labels {
		switch {
		case strings.HasPrefix(k, labelNodeRolePrefix):
			if role := strings.TrimPrefix(k, labelNodeRolePrefix); len(role) > 0 {
				roles.Insert(role)
			}

		case k == nodeLabelRole && v != "":
			roles.Insert(v)
		}
	}
	return roles.UnsortedList()
}

func printNodeList(list *corev1.NodeList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printNode(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printPersistentVolume(obj *corev1.PersistentVolume) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}

	claimRefUID := ""
	if obj.Spec.ClaimRef != nil {
		claimRefUID += obj.Spec.ClaimRef.Namespace
		claimRefUID += "/"
		claimRefUID += obj.Spec.ClaimRef.Name
	}

	modesStr := GetAccessModesAsString(obj.Spec.AccessModes)
	reclaimPolicyStr := string(obj.Spec.PersistentVolumeReclaimPolicy)

	aQty := obj.Spec.Capacity[corev1.ResourceStorage]
	aSize := aQty.String()

	phase := obj.Status.Phase
	if !obj.ObjectMeta.DeletionTimestamp.IsZero() {
		phase = "Terminating"
	}
	volumeMode := "<unset>"
	if obj.Spec.VolumeMode != nil {
		volumeMode = string(*obj.Spec.VolumeMode)
	}

	volumeAttributeClass := "<unset>"
	if obj.Spec.VolumeAttributesClassName != nil {
		volumeAttributeClass = *obj.Spec.VolumeAttributesClassName
	}

	row.Cells = append(row.Cells, obj.Name, aSize, modesStr, reclaimPolicyStr,
		string(phase), claimRefUID, GetPersistentVolumeClass(obj), volumeAttributeClass,
		obj.Status.Reason, translateTimestampSince(obj.CreationTimestamp), volumeMode)
	return []metav1.TableRow{row}, nil
}

func printPersistentVolumeList(list *corev1.PersistentVolumeList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printPersistentVolume(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printPersistentVolumeClaim(obj *corev1.PersistentVolumeClaim) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}

	phase := obj.Status.Phase
	if !obj.ObjectMeta.DeletionTimestamp.IsZero() {
		phase = "Terminating"
	}

	volumeAttributeClass := "<unset>"
	storage := obj.Spec.Resources.Requests[corev1.ResourceStorage]
	capacity := ""
	accessModes := ""
	volumeMode := "<unset>"

	if obj.Spec.VolumeAttributesClassName != nil {
		volumeAttributeClass = *obj.Spec.VolumeAttributesClassName
	}

	if obj.Spec.VolumeName != "" {
		accessModes = GetAccessModesAsString(obj.Status.AccessModes)
		storage = obj.Status.Capacity[corev1.ResourceStorage]
		capacity = storage.String()
	}

	if obj.Spec.VolumeMode != nil {
		volumeMode = string(*obj.Spec.VolumeMode)
	}

	row.Cells = append(row.Cells, obj.Name, string(phase), obj.Spec.VolumeName, capacity, accessModes,
		GetPersistentVolumeClaimClass(obj), volumeAttributeClass, translateTimestampSince(obj.CreationTimestamp), volumeMode)
	return []metav1.TableRow{row}, nil
}

func printPersistentVolumeClaimList(list *corev1.PersistentVolumeClaimList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printPersistentVolumeClaim(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printEvent(obj *corev1.Event) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}

	firstTimestamp := translateTimestampSince(obj.FirstTimestamp)
	if obj.FirstTimestamp.IsZero() {
		firstTimestamp = translateMicroTimestampSince(obj.EventTime)
	}

	lastTimestamp := translateTimestampSince(obj.LastTimestamp)
	if obj.LastTimestamp.IsZero() {
		lastTimestamp = firstTimestamp
	}

	count := obj.Count
	if obj.Series != nil {
		lastTimestamp = translateMicroTimestampSince(obj.Series.LastObservedTime)
		count = obj.Series.Count
	} else if count == 0 {
		// Singleton events don't have a count set in the new API.
		count = 1
	}

	var target string
	if len(obj.InvolvedObject.Name) > 0 {
		target = fmt.Sprintf("%s/%s", strings.ToLower(obj.InvolvedObject.Kind), obj.InvolvedObject.Name)
	} else {
		target = strings.ToLower(obj.InvolvedObject.Kind)
	}
	row.Cells = append(row.Cells,
		lastTimestamp,
		obj.Type,
		obj.Reason,
		target,
		obj.InvolvedObject.FieldPath,
		formatEventSource(obj.Source, obj.ReportingController, obj.ReportingInstance),
		strings.TrimSpace(obj.Message),
		firstTimestamp,
		int64(count),
		obj.Name,
	)

	return []metav1.TableRow{row}, nil
}

// Sorts and prints the EventList in a human-friendly format.
func printEventList(list *corev1.EventList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printEvent(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printRoleBinding(obj *rbacv1.RoleBinding) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}

	roleRef := fmt.Sprintf("%s/%s", obj.RoleRef.Kind, obj.RoleRef.Name)
	row.Cells = append(row.Cells, obj.Name, roleRef, translateTimestampSince(obj.CreationTimestamp))
	users, groups, sas, _ := SubjectsStrings(obj.Subjects)
	row.Cells = append(row.Cells, strings.Join(users, ", "), strings.Join(groups, ", "), strings.Join(sas, ", "))
	return []metav1.TableRow{row}, nil
}

// Prints the RoleBinding in a human-friendly format.
func printRoleBindingList(list *rbacv1.RoleBindingList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printRoleBinding(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printClusterRoleBinding(obj *rbacv1.ClusterRoleBinding) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}

	roleRef := fmt.Sprintf("%s/%s", obj.RoleRef.Kind, obj.RoleRef.Name)
	row.Cells = append(row.Cells, obj.Name, roleRef, translateTimestampSince(obj.CreationTimestamp))
	users, groups, sas, _ := SubjectsStrings(obj.Subjects)
	row.Cells = append(row.Cells, strings.Join(users, ", "), strings.Join(groups, ", "), strings.Join(sas, ", "))
	return []metav1.TableRow{row}, nil
}

// Prints the ClusterRoleBinding in a human-friendly format.
func printClusterRoleBindingList(list *rbacv1.ClusterRoleBindingList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printClusterRoleBinding(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printCertificateSigningRequest(obj *certificatesv1.CertificateSigningRequest) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}
	status := extractCSRStatus(obj)
	signerName := "<none>"
	if obj.Spec.SignerName != "" {
		signerName = obj.Spec.SignerName
	}
	requestedDuration := "<none>"
	if obj.Spec.ExpirationSeconds != nil {
		requestedDuration = duration.HumanDuration(csr.ExpirationSecondsToDuration(*obj.Spec.ExpirationSeconds))
	}
	row.Cells = append(row.Cells, obj.Name, translateTimestampSince(obj.CreationTimestamp), signerName, obj.Spec.Username, requestedDuration, status)
	return []metav1.TableRow{row}, nil
}

func extractCSRStatus(csr *certificatesv1.CertificateSigningRequest) string {
	var approved, denied, failed bool
	for _, c := range csr.Status.Conditions {
		switch c.Type {
		case certificatesv1.CertificateApproved:
			approved = true
		case certificatesv1.CertificateDenied:
			denied = true
		case certificatesv1.CertificateFailed:
			failed = true
		}
	}
	var status string
	// must be in order of presidence
	if denied {
		status += "Denied"
	} else if approved {
		status += "Approved"
	} else {
		status += "Pending"
	}
	if failed {
		status += ",Failed"
	}
	if len(csr.Status.Certificate) > 0 {
		status += ",Issued"
	}
	return status
}

func printCertificateSigningRequestList(list *certificatesv1.CertificateSigningRequestList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printCertificateSigningRequest(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printDeployment(obj *appsv1.Deployment) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}
	desiredReplicas := obj.Spec.Replicas
	updatedReplicas := obj.Status.UpdatedReplicas
	readyReplicas := obj.Status.ReadyReplicas
	availableReplicas := obj.Status.AvailableReplicas
	age := translateTimestampSince(obj.CreationTimestamp)
	containers := obj.Spec.Template.Spec.Containers
	selector, err := metav1.LabelSelectorAsSelector(obj.Spec.Selector)
	selectorString := ""
	if err != nil {
		selectorString = "<invalid>"
	} else {
		selectorString = selector.String()
	}
	row.Cells = append(row.Cells, obj.Name, fmt.Sprintf("%d/%d", int64(readyReplicas), ptr.Deref(desiredReplicas, 0)), int64(updatedReplicas), int64(availableReplicas), age)
	containerNames, images := layoutContainerCells(containers)
	row.Cells = append(row.Cells, containerNames, images, selectorString)
	return []metav1.TableRow{row}, nil
}

func printDeploymentList(list *appsv1.DeploymentList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printDeployment(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func formatHPAMetrics(specs []autoscalingv2.MetricSpec, statuses []autoscalingv2.MetricStatus) string {
	if len(specs) == 0 {
		return "<none>"
	}
	var list []string
	maximum := 2
	more := false
	count := 0
	for i, spec := range specs {
		switch spec.Type {
		case autoscalingv2.ExternalMetricSourceType:
			if spec.External.Target.AverageValue != nil {
				current := "<unknown>"
				if len(statuses) > i && statuses[i].External != nil && statuses[i].External.Current.AverageValue != nil {
					current = statuses[i].External.Current.AverageValue.String()
				}
				list = append(list, fmt.Sprintf("%s/%s (avg)", current, spec.External.Target.AverageValue.String()))
			} else {
				current := "<unknown>"
				if len(statuses) > i && statuses[i].External != nil {
					current = statuses[i].External.Current.Value.String()
				}
				list = append(list, fmt.Sprintf("%s/%s", current, spec.External.Target.Value.String()))
			}
		case autoscalingv2.PodsMetricSourceType:
			current := "<unknown>"
			if len(statuses) > i && statuses[i].Pods != nil {
				current = statuses[i].Pods.Current.AverageValue.String()
			}
			list = append(list, fmt.Sprintf("%s/%s", current, spec.Pods.Target.AverageValue.String()))
		case autoscalingv2.ObjectMetricSourceType:
			if spec.Object.Target.AverageValue != nil {
				current := "<unknown>"
				if len(statuses) > i && statuses[i].Object != nil && statuses[i].Object.Current.AverageValue != nil {
					current = statuses[i].Object.Current.AverageValue.String()
				}
				list = append(list, fmt.Sprintf("%s/%s (avg)", current, spec.Object.Target.AverageValue.String()))
			} else {
				current := "<unknown>"
				if len(statuses) > i && statuses[i].Object != nil {
					current = statuses[i].Object.Current.Value.String()
				}
				list = append(list, fmt.Sprintf("%s/%s", current, spec.Object.Target.Value.String()))
			}
		case autoscalingv2.ResourceMetricSourceType:
			if spec.Resource.Target.AverageValue != nil {
				current := "<unknown>"
				if len(statuses) > i && statuses[i].Resource != nil {
					current = statuses[i].Resource.Current.AverageValue.String()
				}
				list = append(list, fmt.Sprintf("%s: %s/%s", spec.Resource.Name.String(), current, spec.Resource.Target.AverageValue.String()))
			} else {
				current := "<unknown>"
				if len(statuses) > i && statuses[i].Resource != nil && statuses[i].Resource.Current.AverageUtilization != nil {
					current = fmt.Sprintf("%d%%", *statuses[i].Resource.Current.AverageUtilization)
				}

				target := "<auto>"
				if spec.Resource.Target.AverageUtilization != nil {
					target = fmt.Sprintf("%d%%", *spec.Resource.Target.AverageUtilization)
				}
				list = append(list, fmt.Sprintf("%s: %s/%s", spec.Resource.Name.String(), current, target))
			}
		case autoscalingv2.ContainerResourceMetricSourceType:
			if spec.ContainerResource.Target.AverageValue != nil {
				current := "<unknown>"
				if len(statuses) > i && statuses[i].ContainerResource != nil {
					current = statuses[i].ContainerResource.Current.AverageValue.String()
				}
				list = append(list, fmt.Sprintf("%s: %s/%s", spec.ContainerResource.Name.String(), current, spec.ContainerResource.Target.AverageValue.String()))
			} else {
				current := "<unknown>"
				if len(statuses) > i && statuses[i].ContainerResource != nil && statuses[i].ContainerResource.Current.AverageUtilization != nil {
					current = fmt.Sprintf("%d%%", *statuses[i].ContainerResource.Current.AverageUtilization)
				}

				target := "<auto>"
				if spec.ContainerResource.Target.AverageUtilization != nil {
					target = fmt.Sprintf("%d%%", *spec.ContainerResource.Target.AverageUtilization)
				}
				list = append(list, fmt.Sprintf("%s: %s/%s", spec.ContainerResource.Name.String(), current, target))
			}
		default:
			list = append(list, "<unknown type>")
		}

		count++
	}

	if count > maximum {
		list = list[:maximum]
		more = true
	}

	ret := strings.Join(list, ", ")
	if more {
		return fmt.Sprintf("%s + %d more...", ret, count-maximum)
	}
	return ret
}

func printHorizontalPodAutoscaler(obj *autoscalingv2.HorizontalPodAutoscaler) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}

	reference := fmt.Sprintf("%s/%s",
		obj.Spec.ScaleTargetRef.Kind,
		obj.Spec.ScaleTargetRef.Name)
	minPods := "<unset>"
	metrics := formatHPAMetrics(obj.Spec.Metrics, obj.Status.CurrentMetrics)
	if obj.Spec.MinReplicas != nil {
		minPods = fmt.Sprintf("%d", *obj.Spec.MinReplicas)
	}
	maxPods := obj.Spec.MaxReplicas
	currentReplicas := obj.Status.CurrentReplicas
	row.Cells = append(row.Cells, obj.Name, reference, metrics, minPods, int64(maxPods), int64(currentReplicas), translateTimestampSince(obj.CreationTimestamp))
	return []metav1.TableRow{row}, nil
}

func printHorizontalPodAutoscalerList(list *autoscalingv2.HorizontalPodAutoscalerList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printHorizontalPodAutoscaler(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printConfigMap(obj *corev1.ConfigMap) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}
	row.Cells = append(row.Cells, obj.Name, int64(len(obj.Data)+len(obj.BinaryData)), translateTimestampSince(obj.CreationTimestamp))
	return []metav1.TableRow{row}, nil
}

func printConfigMapList(list *corev1.ConfigMapList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printConfigMap(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printNetworkPolicy(obj *networkingv1.NetworkPolicy) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}
	row.Cells = append(row.Cells, obj.Name, metav1.FormatLabelSelector(&obj.Spec.PodSelector), translateTimestampSince(obj.CreationTimestamp))
	return []metav1.TableRow{row}, nil
}

func printNetworkPolicyList(list *networkingv1.NetworkPolicyList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printNetworkPolicy(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printStorageClass(obj *storagev1.StorageClass) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}

	name := obj.Name
	if IsDefaultAnnotation(obj.ObjectMeta) {
		name += " (default)"
	}

	provisionerType := obj.Provisioner
	reclaimPolicy := string(corev1.PersistentVolumeReclaimDelete)
	if obj.ReclaimPolicy != nil {
		reclaimPolicy = string(*obj.ReclaimPolicy)
	}

	volumeBindingMode := string(storagev1.VolumeBindingImmediate)
	if obj.VolumeBindingMode != nil {
		volumeBindingMode = string(*obj.VolumeBindingMode)
	}

	allowVolumeExpansion := false
	if obj.AllowVolumeExpansion != nil {
		allowVolumeExpansion = *obj.AllowVolumeExpansion
	}

	row.Cells = append(row.Cells, name, provisionerType, reclaimPolicy, volumeBindingMode, allowVolumeExpansion,
		translateTimestampSince(obj.CreationTimestamp))

	return []metav1.TableRow{row}, nil
}

func printStorageClassList(list *storagev1.StorageClassList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printStorageClass(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printLease(obj *coordinationv1.Lease) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}

	var holderIdentity string
	if obj.Spec.HolderIdentity != nil {
		holderIdentity = *obj.Spec.HolderIdentity
	}
	row.Cells = append(row.Cells, obj.Name, holderIdentity, translateTimestampSince(obj.CreationTimestamp))
	return []metav1.TableRow{row}, nil
}

func printLeaseList(list *coordinationv1.LeaseList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printLease(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

// Lay out all the containers on one line if use wide output.
func layoutContainerCells(containers []corev1.Container) (names string, images string) {
	var namesBuffer bytes.Buffer
	var imagesBuffer bytes.Buffer

	for i, container := range containers {
		namesBuffer.WriteString(container.Name)
		imagesBuffer.WriteString(container.Image)
		if i != len(containers)-1 {
			namesBuffer.WriteString(",")
			imagesBuffer.WriteString(",")
		}
	}
	return namesBuffer.String(), imagesBuffer.String()
}

// formatEventSource formats EventSource as a comma separated string excluding Host when empty.
// It uses reportingController when Source.Component is empty and reportingInstance when Source.Host is empty
func formatEventSource(es corev1.EventSource, reportingController, reportingInstance string) string {
	return formatEventSourceComponentInstance(
		firstNonEmpty(es.Component, reportingController),
		firstNonEmpty(es.Host, reportingInstance),
	)
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if len(s) > 0 {
			return s
		}
	}
	return ""
}

func formatEventSourceComponentInstance(component, instance string) string {
	if len(instance) == 0 {
		return component
	}
	return component + ", " + instance
}

func printControllerRevision(obj *appsv1.ControllerRevision) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}

	controllerRef := metav1.GetControllerOf(obj)
	controllerName := "<none>"
	if controllerRef != nil {
		gv, err := schema.ParseGroupVersion(controllerRef.APIVersion)
		if err != nil {
			return nil, err
		}
		gvk := gv.WithKind(controllerRef.Kind)
		controllerName = formatResourceName(gvk.GroupKind(), controllerRef.Name, true)
	}
	revision := obj.Revision
	age := translateTimestampSince(obj.CreationTimestamp)
	row.Cells = append(row.Cells, obj.Name, controllerName, revision, age)
	return []metav1.TableRow{row}, nil
}

func printControllerRevisionList(list *appsv1.ControllerRevisionList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printControllerRevision(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

// formatResourceName receives a resource kind, name, and boolean specifying
// whether or not to update the current name to "kind/name"
func formatResourceName(kind schema.GroupKind, name string, withKind bool) string {
	if !withKind || kind.Empty() {
		return name
	}

	return strings.ToLower(kind.String()) + "/" + name
}

func printResourceQuota(resourceQuota *corev1.ResourceQuota) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}

	resources := make([]corev1.ResourceName, 0, len(resourceQuota.Status.Hard))
	for resource := range resourceQuota.Status.Hard {
		resources = append(resources, resource)
	}
	sort.Sort(SortableResourceNames(resources))

	requestColumn := bytes.NewBuffer([]byte{})
	limitColumn := bytes.NewBuffer([]byte{})
	for i := range resources {
		w := requestColumn
		resource := resources[i]
		usedQuantity := resourceQuota.Status.Used[resource]
		hardQuantity := resourceQuota.Status.Hard[resource]

		// use limitColumn writer if a resource name prefixed with "limits" is found
		if pieces := strings.Split(resource.String(), "."); len(pieces) > 1 && pieces[0] == "limits" {
			w = limitColumn
		}

		_, _ = fmt.Fprintf(w, "%s: %s/%s, ", resource, usedQuantity.String(), hardQuantity.String())
	}

	age := translateTimestampSince(resourceQuota.CreationTimestamp)
	row.Cells = append(row.Cells, resourceQuota.Name, age, strings.TrimSuffix(requestColumn.String(), ", "), strings.TrimSuffix(limitColumn.String(), ", "))
	return []metav1.TableRow{row}, nil
}

func printResourceQuotaList(list *corev1.ResourceQuotaList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printResourceQuota(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printPriorityClass(obj *schedulingv1.PriorityClass) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}

	name := obj.Name
	value := obj.Value
	globalDefault := obj.GlobalDefault
	row.Cells = append(row.Cells, name, int64(value), globalDefault, translateTimestampSince(obj.CreationTimestamp))
	var preemptionPolicy string
	if obj.PreemptionPolicy != nil {
		preemptionPolicy = string(*obj.PreemptionPolicy)
	}
	row.Cells = append(row.Cells, preemptionPolicy)
	return []metav1.TableRow{row}, nil
}

func printPriorityClassList(list *schedulingv1.PriorityClassList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printPriorityClass(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printRuntimeClass(obj *nodev1.RuntimeClass) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}

	name := obj.Name
	handler := obj.Handler
	row.Cells = append(row.Cells, name, handler, translateTimestampSince(obj.CreationTimestamp))

	return []metav1.TableRow{row}, nil
}

func printRuntimeClassList(list *nodev1.RuntimeClassList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printRuntimeClass(&list.Items[i])

		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printVolumeAttachment(obj *storagev1.VolumeAttachment) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}

	name := obj.Name
	pvName := ""
	if obj.Spec.Source.PersistentVolumeName != nil {
		pvName = *obj.Spec.Source.PersistentVolumeName
	}
	row.Cells = append(row.Cells, name, obj.Spec.Attacher, pvName, obj.Spec.NodeName, obj.Status.Attached, translateTimestampSince(obj.CreationTimestamp))

	return []metav1.TableRow{row}, nil
}

func printVolumeAttachmentList(list *storagev1.VolumeAttachmentList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printVolumeAttachment(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printFlowSchema(obj *flowcontrolv1.FlowSchema) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}

	name := obj.Name
	plName := obj.Spec.PriorityLevelConfiguration.Name
	distinguisherMethod := "<none>"
	if obj.Spec.DistinguisherMethod != nil {
		distinguisherMethod = string(obj.Spec.DistinguisherMethod.Type)
	}
	badPLRef := "?"
	for _, cond := range obj.Status.Conditions {
		if cond.Type == flowcontrolv1.FlowSchemaConditionDangling {
			badPLRef = string(cond.Status)
			break
		}
	}
	row.Cells = append(row.Cells, name, plName, int64(obj.Spec.MatchingPrecedence), distinguisherMethod, translateTimestampSince(obj.CreationTimestamp), badPLRef)

	return []metav1.TableRow{row}, nil
}

func printFlowSchemaList(list *flowcontrolv1.FlowSchemaList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	fsSeq := make(FlowSchemaSequence, len(list.Items))
	for i := range list.Items {
		fsSeq[i] = &list.Items[i]
	}
	sort.Sort(fsSeq)
	for i := range fsSeq {
		r, err := printFlowSchema(fsSeq[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printPriorityLevelConfiguration(obj *flowcontrolv1.PriorityLevelConfiguration) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}
	name := obj.Name
	ncs := interface{}("<none>")
	queues := interface{}("<none>")
	handSize := interface{}("<none>")
	queueLengthLimit := interface{}("<none>")
	if obj.Spec.Limited != nil {
		ncs = obj.Spec.Limited.NominalConcurrencyShares
		if qc := obj.Spec.Limited.LimitResponse.Queuing; qc != nil {
			queues = qc.Queues
			handSize = qc.HandSize
			queueLengthLimit = qc.QueueLengthLimit
		}
	}
	row.Cells = append(row.Cells, name, string(obj.Spec.Type), ncs, queues, handSize, queueLengthLimit, translateTimestampSince(obj.CreationTimestamp))

	return []metav1.TableRow{row}, nil
}

func printPriorityLevelConfigurationList(list *flowcontrolv1.PriorityLevelConfigurationList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printPriorityLevelConfiguration(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printResourceClaim(obj *resourcev1beta1.ResourceClaim) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}
	row.Cells = append(row.Cells, obj.Name, resourceClaimState(obj), translateTimestampSince(obj.CreationTimestamp))

	return []metav1.TableRow{row}, nil
}

func resourceClaimState(obj *resourcev1beta1.ResourceClaim) string {
	var states []string
	if !obj.DeletionTimestamp.IsZero() {
		states = append(states, "deleted")
	}
	if obj.Status.Allocation == nil {
		if obj.DeletionTimestamp.IsZero() {
			states = append(states, "pending")
		}
	} else {
		states = append(states, "allocated")
		if len(obj.Status.ReservedFor) > 0 {
			states = append(states, "reserved")
		}
	}
	return strings.Join(states, ",")
}

func printResourceClaimList(list *resourcev1beta1.ResourceClaimList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printResourceClaim(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printResourceSlice(obj *resourcev1beta1.ResourceSlice) ([]metav1.TableRow, error) {
	row := metav1.TableRow{}
	row.Cells = append(row.Cells, obj.Name, obj.Spec.NodeName, obj.Spec.Driver, obj.Spec.Pool.Name, translateTimestampSince(obj.CreationTimestamp))

	return []metav1.TableRow{row}, nil
}

func printResourceSliceList(list *resourcev1beta1.ResourceSliceList) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printResourceSlice(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printBoolPtr(value *bool) string {
	if value != nil {
		return printBool(*value)
	}

	return "<unset>"
}

func printBool(value bool) string {
	if value {
		return "True"
	}

	return "False"
}

// SortableResourceNames - An array of sortable resource names
type SortableResourceNames []corev1.ResourceName

func (list SortableResourceNames) Len() int {
	return len(list)
}

func (list SortableResourceNames) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}

func (list SortableResourceNames) Less(i, j int) bool {
	return list[i] < list[j]
}

func isPodInitializedConditionTrue(status *corev1.PodStatus) bool {
	for _, condition := range status.Conditions {
		if condition.Type != corev1.PodInitialized {
			continue
		}

		return condition.Status == corev1.ConditionTrue
	}
	return false
}
