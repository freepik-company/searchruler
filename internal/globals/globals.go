package globals

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// https://github.com/external-secrets/external-secrets/blob/80545f4f183795ef193747fc959558c761b51c99/apis/externalsecrets/v1alpha1/externalsecret_types.go#L168
const (
	// ConditionTypeResourceSynced indicates that the target was synced or not
	ConditionTypeResourceSynced = "ResourceSynced"

	// Kubernetes error type
	ConditionReasonKubernetesApiCallErrorType    = "KubernetesApiCallError"
	ConditionReasonKubernetesApiCallErrorMessage = "Call to Kubernetes API failed. More info in logs."

	// Success
	ConditionReasonTargetSynced        = "TargetSynced"
	ConditionReasonTargetSyncedMessage = "Target was successfully synced"
)

// NewCondition a set of default options for creating a Condition.
func NewCondition(condType string, status metav1.ConditionStatus, reason, message string) metav1.Condition {
	return metav1.Condition{
		Type:               condType,
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

func getCondition(conditions *[]metav1.Condition, condType string) *metav1.Condition {
	for i, v := range *conditions {
		if v.Type == condType {
			return &(*conditions)[i]
		}
	}
	return nil
}

func UpdateCondition(conditions *[]metav1.Condition, condition metav1.Condition) {

	// Get the condition
	currentCondition := getCondition(conditions, condition.Type)

	if currentCondition == nil {
		// Create the condition when not existent
		*conditions = append(*conditions, condition)
	} else {
		// Update the condition when existent.
		currentCondition.Status = condition.Status
		currentCondition.Reason = condition.Reason
		currentCondition.Message = condition.Message
		currentCondition.LastTransitionTime = metav1.Now()
	}
}
