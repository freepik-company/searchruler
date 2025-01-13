package controller

const (

	// Resource types
	SearchRuleResourceType            = "SearchRule"
	RulerActionResourceType           = "RulerAction"
	QueryConnectorResourceType        = "QueryConnector"
	ClusterQueryConnectorResourceType = "ClusterQueryConnector"
	ClusterRulerActionResourceType    = "ClusterRulerAction"

	// Sync interval to check if secrets of SearchRuleAction and SearchRuleQueryConnector are up to date
	DefaultSyncInterval = "1m"

	// Error messages
	ResourceNotFoundError                  = "%s '%s' resource not found. Ignoring since object must be deleted."
	CanNotGetResourceError                 = "%s '%s' resource not found. Error: %v"
	ResourceFinalizersUpdateError          = "Failed to update finalizer of %s '%s': %s"
	ResourceConditionUpdateError           = "Failed to update the condition on %s '%s': %s"
	ResourceSyncTimeRetrievalError         = "can not get synchronization time from the %s '%s': %s"
	SyncTargetError                        = "can not sync the target for the %s '%s': %s"
	ValidatorNotFoundErrorMessage          = "validator %s not found"
	ValidationFailedErrorMessage           = "validation failed: %s"
	HttpRequestCreationErrorMessage        = "error creating http request: %s"
	HttpRequestSendingErrorMessage         = "error sending http request: %s"
	AlertFiringInfoMessage                 = "alert firing for searchRule with namespaced name %s/%s. Description: %s"
	SecretNotFoundErrorMessage             = "error fetching secret %s: %v"
	MissingCredentialsMessage              = "missing credentials in secret %s"
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
	ResourceFinalizer = "searchruler.prosimcorp.com/finalizer"
)
