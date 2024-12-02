package controller

import (
	"prosimcorp.com/SearchRuler/api/v1alpha1"
	"prosimcorp.com/SearchRuler/internal/globals"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *RulerActionReconciler) UpdateConditionSuccess(RulerAction *v1alpha1.RulerAction) {

	//
	condition := globals.NewCondition(globals.ConditionTypeResourceSynced, metav1.ConditionTrue,
		globals.ConditionReasonTargetSynced, globals.ConditionReasonTargetSyncedMessage)

	globals.UpdateCondition(&RulerAction.Status.Conditions, condition)
}

func (r *RulerActionReconciler) UpdateConditionKubernetesApiCallFailure(RulerAction *v1alpha1.RulerAction) {

	//
	condition := globals.NewCondition(globals.ConditionTypeResourceSynced, metav1.ConditionTrue,
		globals.ConditionReasonKubernetesApiCallErrorType, globals.ConditionReasonKubernetesApiCallErrorMessage)

	globals.UpdateCondition(&RulerAction.Status.Conditions, condition)
}
