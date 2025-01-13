/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ruleraction

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	//

	"prosimcorp.com/SearchRuler/internal/controller"
	"prosimcorp.com/SearchRuler/internal/globals"
)

// UpdateConditionSuccess updates the status of the RulerAction resource with a success condition
func (r *RulerActionReconciler) UpdateConditionSuccess(resource *CompoundRulerActionResource, resourceType string) {

	// Create the new condition with the success status
	condition := globals.NewCondition(globals.ConditionTypeResourceSynced, metav1.ConditionTrue,
		globals.ConditionReasonTargetSynced, globals.ConditionReasonTargetSyncedMessage)

	// Update the status of the RulerAction resource
	switch resourceType {
	case controller.ClusterRulerActionResourceType:
		globals.UpdateCondition(&resource.ClusterRulerActionResource.Status.Conditions, condition)
	default:
		globals.UpdateCondition(&resource.RulerActionResource.Status.Conditions, condition)
	}
}

// UpdateConditionKubernetesApiCallFailure updates the status of the RulerAction resource with a failure condition
func (r *RulerActionReconciler) UpdateConditionKubernetesApiCallFailure(resource *CompoundRulerActionResource, resourceType string) {

	// Create the new condition with the failure status
	condition := globals.NewCondition(globals.ConditionTypeResourceSynced, metav1.ConditionTrue,
		globals.ConditionReasonKubernetesApiCallErrorType, globals.ConditionReasonKubernetesApiCallErrorMessage)

	// Update the status of the RulerAction resource
	switch resourceType {
	case controller.ClusterRulerActionResourceType:
		globals.UpdateCondition(&resource.ClusterRulerActionResource.Status.Conditions, condition)
	default:
		globals.UpdateCondition(&resource.RulerActionResource.Status.Conditions, condition)
	}
}

// UpdateStateSuccess updates the status of the RulerAction resource with a Success condition
func (r *RulerActionReconciler) UpdateStateSuccess(resource *CompoundRulerActionResource, resourceType string) {

	// Create the new condition with the success status
	condition := globals.NewCondition(globals.ConditionTypeState, metav1.ConditionTrue,
		globals.ConditionReasonStateSuccessType, globals.ConditionReasonStateSuccessMessage)

	// Update the status of the RulerAction resource
	switch resourceType {
	case controller.ClusterRulerActionResourceType:
		globals.UpdateCondition(&resource.ClusterRulerActionResource.Status.Conditions, condition)
	default:
		globals.UpdateCondition(&resource.RulerActionResource.Status.Conditions, condition)
	}
}

// UpdateConditionConnectionError updates the status of the RulerAction resource with a ConnectionError condition
func (r *RulerActionReconciler) UpdateConditionConnectionError(resource *CompoundRulerActionResource, resourceType string) {

	// Create the new condition with the failure status
	condition := globals.NewCondition(globals.ConditionTypeState, metav1.ConditionTrue,
		globals.ConditionReasonConnectionErrorType, globals.ConditionReasonConnectionErrorMessage)

	// Update the status of the RulerAction resource
	switch resourceType {
	case controller.ClusterRulerActionResourceType:
		globals.UpdateCondition(&resource.ClusterRulerActionResource.Status.Conditions, condition)
	default:
		globals.UpdateCondition(&resource.RulerActionResource.Status.Conditions, condition)
	}
}

// UpdateConditionEvaluateTemplateError updates the status of the RulerAction resource with a EvaluateTemplateError condition
func (r *RulerActionReconciler) UpdateConditionEvaluateTemplateError(resource *CompoundRulerActionResource, resourceType string) {

	// Create the new condition with the failure status
	condition := globals.NewCondition(globals.ConditionTypeState, metav1.ConditionTrue,
		globals.ConditionReasonEvaluateTemplateErrorType, globals.ConditionReasonEvaluateTemplateErrorMessage)

	// Update the status of the RulerAction resource
	switch resourceType {
	case controller.ClusterRulerActionResourceType:
		globals.UpdateCondition(&resource.ClusterRulerActionResource.Status.Conditions, condition)
	default:
		globals.UpdateCondition(&resource.RulerActionResource.Status.Conditions, condition)
	}
}

// UpdateConditionNoCredsFound updates the status of the RulerAction resource with a NoCreds condition
func (r *RulerActionReconciler) UpdateConditionNoCredsFound(resource *CompoundRulerActionResource, resourceType string) {

	// Create the new condition with the success status
	condition := globals.NewCondition(globals.ConditionTypeState, metav1.ConditionTrue,
		globals.ConditionReasonNoCredsFoundType, globals.ConditionReasonNoCredsFoundMessage)

	// Update the status of the RulerAction resource
	switch resourceType {
	case controller.ClusterRulerActionResourceType:
		globals.UpdateCondition(&resource.ClusterRulerActionResource.Status.Conditions, condition)
	default:
		globals.UpdateCondition(&resource.RulerActionResource.Status.Conditions, condition)
	}
}
