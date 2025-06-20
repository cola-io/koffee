package definition

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	apiv1 "k8s.io/api/core/v1"
	flowcontrolv1 "k8s.io/api/flowcontrol/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// NodeUnreachablePodReason is the reason on a pod when its state cannot be confirmed as kubelet is unresponsive
	// on the node it is (was) running.
	NodeUnreachablePodReason = "NodeLost"
	// IsDefaultStorageClassAnnotation represents a StorageClass annotation that
	// marks a class as the default StorageClass
	IsDefaultStorageClassAnnotation = "storageclass.kubernetes.io/is-default-class"

	// BetaIsDefaultStorageClassAnnotation is the beta version of IsDefaultStorageClassAnnotation.
	BetaIsDefaultStorageClassAnnotation = "storageclass.beta.kubernetes.io/is-default-class"
)

// IsRestartableInitContainer returns true if the container has ContainerRestartPolicyAlways.
// This function is not checking if the container passed to it is indeed an init container.
// It is just checking if the container restart policy has been set to always.
func IsRestartableInitContainer(initContainer *apiv1.Container) bool {
	if initContainer == nil || initContainer.RestartPolicy == nil {
		return false
	}
	return *initContainer.RestartPolicy == apiv1.ContainerRestartPolicyAlways
}

// IsPodPhaseTerminal returns true if the pod's phase is terminal.
func IsPodPhaseTerminal(phase apiv1.PodPhase) bool {
	return phase == apiv1.PodFailed || phase == apiv1.PodSucceeded
}

// GetAccessModesAsString returns a string representation of an array of access modes.
// modes, when present, are always in the same order: RWO,ROX,RWX,RWOP.
func GetAccessModesAsString(modes []apiv1.PersistentVolumeAccessMode) string {
	modes = removeDuplicateAccessModes(modes)
	modesStr := []string{}
	if ContainsAccessMode(modes, apiv1.ReadWriteOnce) {
		modesStr = append(modesStr, "RWO")
	}
	if ContainsAccessMode(modes, apiv1.ReadOnlyMany) {
		modesStr = append(modesStr, "ROX")
	}
	if ContainsAccessMode(modes, apiv1.ReadWriteMany) {
		modesStr = append(modesStr, "RWX")
	}
	if ContainsAccessMode(modes, apiv1.ReadWriteOncePod) {
		modesStr = append(modesStr, "RWOP")
	}
	return strings.Join(modesStr, ",")
}

// GetPersistentVolumeClass returns StorageClassName.
func GetPersistentVolumeClass(volume *apiv1.PersistentVolume) string {
	// Use beta annotation first
	if class, found := volume.Annotations[apiv1.BetaStorageClassAnnotation]; found {
		return class
	}
	return volume.Spec.StorageClassName
}

// GetPersistentVolumeClaimClass returns StorageClassName. If no storage class was
// requested, it returns "".
func GetPersistentVolumeClaimClass(claim *apiv1.PersistentVolumeClaim) string {
	// Use beta annotation first
	if class, found := claim.Annotations[apiv1.BetaStorageClassAnnotation]; found {
		return class
	}

	if claim.Spec.StorageClassName != nil {
		return *claim.Spec.StorageClassName
	}

	return ""
}

// removeDuplicateAccessModes returns an array of access modes without any duplicates
func removeDuplicateAccessModes(modes []apiv1.PersistentVolumeAccessMode) []apiv1.PersistentVolumeAccessMode {
	var accessModes []apiv1.PersistentVolumeAccessMode
	for _, m := range modes {
		if !ContainsAccessMode(accessModes, m) {
			accessModes = append(accessModes, m)
		}
	}
	return accessModes
}

func ContainsAccessMode(modes []apiv1.PersistentVolumeAccessMode, mode apiv1.PersistentVolumeAccessMode) bool {
	return slices.Contains(modes, mode)
}

// SubjectsStrings returns users, groups, serviceaccounts, unknown for display purposes.
func SubjectsStrings(subjects []rbacv1.Subject) ([]string, []string, []string, []string) {
	var users []string
	var groups []string
	var sas []string
	var others []string

	for _, subject := range subjects {
		switch subject.Kind {
		case rbacv1.ServiceAccountKind:
			sas = append(sas, fmt.Sprintf("%s/%s", subject.Namespace, subject.Name))

		case rbacv1.UserKind:
			users = append(users, subject.Name)

		case rbacv1.GroupKind:
			groups = append(groups, subject.Name)

		default:
			others = append(others, fmt.Sprintf("%s/%s/%s", subject.Kind, subject.Namespace, subject.Name))
		}
	}

	return users, groups, sas, others
}

const IsDefaultStorageClassValue = "true"

// IsDefaultAnnotation returns a boolean if
// the annotation is set
// TODO: remove Beta when no longer needed
func IsDefaultAnnotation(obj metav1.ObjectMeta) bool {
	if obj.Annotations[IsDefaultStorageClassAnnotation] == IsDefaultStorageClassValue {
		return true
	}
	if obj.Annotations[BetaIsDefaultStorageClassAnnotation] == IsDefaultStorageClassValue {
		return true
	}
	return false
}

var _ sort.Interface = FlowSchemaSequence{}

// FlowSchemaSequence holds sorted set of pointers to FlowSchema objects.
// FlowSchemaSequence implements `sort.Interface`
type FlowSchemaSequence []*flowcontrolv1.FlowSchema

func (s FlowSchemaSequence) Len() int {
	return len(s)
}

func (s FlowSchemaSequence) Less(i, j int) bool {
	// the flow-schema w/ lower matching-precedence is prior
	if ip, jp := s[i].Spec.MatchingPrecedence, s[j].Spec.MatchingPrecedence; ip != jp {
		return ip < jp
	}
	// sort alphabetically
	return s[i].Name < s[j].Name
}

func (s FlowSchemaSequence) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
