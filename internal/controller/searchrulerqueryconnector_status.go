package controller

import (
	"prosimcorp.com/SearchRuler/internal/globals"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1alpha1 "prosimcorp.com/SearchRuler/api/v1alpha1"
)

func (r *SearchRulerQueryConnectorReconciler) UpdateConditionSuccess(searchRulerQueryConnector *v1alpha1.SearchRulerQueryConnector) {

	//
	condition := globals.NewCondition(globals.ConditionTypeResourceSynced, metav1.ConditionTrue,
		globals.ConditionReasonTargetSynced, globals.ConditionReasonTargetSyncedMessage)

	globals.UpdateCondition(&searchRulerQueryConnector.Status.Conditions, condition)
}

func (r *SearchRulerQueryConnectorReconciler) UpdateConditionKubernetesApiCallFailure(searchRulerQueryConnector *v1alpha1.SearchRulerQueryConnector) {

	//
	condition := globals.NewCondition(globals.ConditionTypeResourceSynced, metav1.ConditionTrue,
		globals.ConditionReasonKubernetesApiCallErrorType, globals.ConditionReasonKubernetesApiCallErrorMessage)

	globals.UpdateCondition(&searchRulerQueryConnector.Status.Conditions, condition)
}
