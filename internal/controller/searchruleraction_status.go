package controller

import (
	"prosimcorp.com/SearchRuler/api/v1alpha1"
	"prosimcorp.com/SearchRuler/internal/globals"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *SearchRulerActionReconciler) UpdateConditionSuccess(searchRulerAction *v1alpha1.SearchRulerAction) {

	//
	condition := globals.NewCondition(globals.ConditionTypeResourceSynced, metav1.ConditionTrue,
		globals.ConditionReasonTargetSynced, globals.ConditionReasonTargetSyncedMessage)

	globals.UpdateCondition(&searchRulerAction.Status.Conditions, condition)
}

func (r *SearchRulerActionReconciler) UpdateConditionKubernetesApiCallFailure(searchRulerAction *v1alpha1.SearchRulerAction) {

	//
	condition := globals.NewCondition(globals.ConditionTypeResourceSynced, metav1.ConditionTrue,
		globals.ConditionReasonKubernetesApiCallErrorType, globals.ConditionReasonKubernetesApiCallErrorMessage)

	globals.UpdateCondition(&searchRulerAction.Status.Conditions, condition)
}
