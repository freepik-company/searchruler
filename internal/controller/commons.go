package controller

const (
	SearchRuleResourceType                = "SearchRule"
	SearchRulerActionResourceType         = "SearchRulerAction"
	SearchRulerQueryConnectorResourceType = "SearchRulerQueryConnector"

	// Sync interval to check if secrets of SearchRuleAction and SearchRuleQueryConnector are up to date
	defaultSyncInterval = "1m"

	// Error messages
	resourceNotFoundError          = "%s '%s' resource not found. Ignoring since object must be deleted."
	resourceRetrievalError         = "Error getting the %s '%s' from the cluster: %s"
	resourceTargetsDeleteError     = "Failed to delete targets of %s '%s': %s"
	resourceFinalizersUpdateError  = "Failed to update finalizer of %s '%s': %s"
	resourceConditionUpdateError   = "Failed to update the condition on %s '%s': %s"
	resourceSyncTimeRetrievalError = "Can not get synchronization time from the %s '%s': %s"
	syncTargetError                = "Can not sync the target for the %s '%s': %s"

	// Finalizer
	resourceFinalizer = "searchruler.prosimcorp.com/finalizer"
)
