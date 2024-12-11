package controller

const (

	// Resource types
	SearchRuleResourceType            = "SearchRule"
	RulerActionResourceType           = "RulerAction"
	QueryConnectorResourceType        = "QueryConnector"
	ClusterQueryConnectorResourceType = "ClusterQueryConnector"

	// Sync interval to check if secrets of SearchRuleAction and SearchRuleQueryConnector are up to date
	defaultSyncInterval = "1m"

	// Error messages
	resourceNotFoundError                  = "%s '%s' resource not found. Ignoring since object must be deleted."
	resourceRetrievalError                 = "Error getting the %s '%s' from the cluster: %s"
	resourceTargetsDeleteError             = "Failed to delete targets of %s '%s': %s"
	resourceFinalizersUpdateError          = "Failed to update finalizer of %s '%s': %s"
	resourceConditionUpdateError           = "Failed to update the condition on %s '%s': %s"
	resourceSyncTimeRetrievalError         = "can not get synchronization time from the %s '%s': %s"
	syncTargetError                        = "can not sync the target for the %s '%s': %s"
	ValidatorNotFoundErrorMessage          = "validator %s not found"
	ValidationFailedErrorMessage           = "validation failed: %s"
	HttpRequestCreationErrorMessage        = "error creating http request: %s"
	HttpRequestSendingErrorMessage         = "error sending http request: %s"
	AlertFiringInfoMessage                 = "alert firing for searchRule with namespaced name %s/%s. Description: %s"
	SecretNotFoundErrorMessage             = "error fetching secret %s: %v"
	MissingCredentialsMessage              = "missing credentials in secret %s"
	GetRulerActionErrorMessage             = "error getting RulerAction from event: %v"
	EvaluateTemplateErrorMessage           = "error evaluating template message: %v"
	AlertsPoolErrorMessage                 = "error getting alerts pool: %v"
	QueryConnectorNotFoundMessage          = "queryConnector %s not found in the resource namespace %s"
	QueryNotDefinedErrorMessage            = "query not defined in resource %s"
	QueryDefinedInBothErrorMessage         = "both query and queryJSON are defined in resource %s. Only one of them must be defined"
	JSONMarshalErrorMessage                = "error marshaling json: %v"
	ElasticsearchQueryErrorMessage         = "error executing elasticsearch request %s: %v"
	ResponseBodyReadErrorMessage           = "error reading response body: %v"
	ElasticsearchQueryResponseErrorMessage = "error response from Elasticsearch executing request %s: %s"
	ConditionFieldNotFoundMessage          = "conditionField %s not found in the response: %s"
	EvaluatingConditionErrorMessage        = "error evaluating condition: %v"
	ForValueParseErrorMessage              = "error parsing `for` time: %v"
	KubeEventCreationErrorMessage          = "error creating kube event: %v"

	// Finalizer
	resourceFinalizer = "searchruler.prosimcorp.com/finalizer"

	// HTTP event pattern
	HttpEventPattern = `{"data":"%s","timestamp":"%s"}`
)
