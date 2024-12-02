package controller

import (
	"prosimcorp.com/SearchRuler/internal/globals"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1alpha1 "prosimcorp.com/SearchRuler/api/v1alpha1"
)

func (r *QueryConnectorReconciler) UpdateConditionSuccess(QueryConnector *v1alpha1.QueryConnector) {

	//
	condition := globals.NewCondition(globals.ConditionTypeResourceSynced, metav1.ConditionTrue,
		globals.ConditionReasonTargetSynced, globals.ConditionReasonTargetSyncedMessage)

	globals.UpdateCondition(&QueryConnector.Status.Conditions, condition)
}

func (r *QueryConnectorReconciler) UpdateConditionKubernetesApiCallFailure(QueryConnector *v1alpha1.QueryConnector) {

	//
	condition := globals.NewCondition(globals.ConditionTypeResourceSynced, metav1.ConditionTrue,
		globals.ConditionReasonKubernetesApiCallErrorType, globals.ConditionReasonKubernetesApiCallErrorMessage)

	globals.UpdateCondition(&QueryConnector.Status.Conditions, condition)
}
