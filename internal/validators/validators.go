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

package validators

import (
	"encoding/json"
	"fmt"
	"reflect"
)

const (
	amgrAlertDataUnmarshalErrorMessage         = "error decoding JSON from 'message.data' for Alertmanager validator: %s"
	amgrAlertDataRequiredStructureErrorMessage = "notification field 'message.data' does not meet the syntax requirements for Alertmanager: %s"
)

// TODO
type AlertmanagerAlertList []AlertmanagerAlert

// Alert represents the structure of an alert in Alertmanager
// Ref: https://prometheus.io/docs/alerting/latest/clients/
// Ref: https://raw.githubusercontent.com/prometheus/alertmanager/main/api/v2/openapi.yaml
type AlertmanagerAlert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`

	// Optional params
	StartsAt     string `json:"startsAt,omitempty"`
	EndsAt       string `json:"endsAt,omitempty"`
	GeneratorUrl string `json:"generatorURL,omitempty"`
}

// ValidateAlertmanager checks whether the notification data meets the requirements for Alertmanager
func ValidateAlertmanager(data string) (result bool, hint string, err error) {

	alertList := AlertmanagerAlertList{AlertmanagerAlert{}}

	//
	err = json.Unmarshal([]byte(data), &alertList)
	if err != nil {
		return false, hint, fmt.Errorf(amgrAlertDataUnmarshalErrorMessage, err)
	}

	if reflect.ValueOf(alertList).IsZero() {
		return false, amgrAlertDataRequiredStructureErrorMessage, nil
	}

	//
	for _, alert := range alertList {

		// Check the main label for the alert.
		// This is required only on API v1
		_, alertNameFound := alert.Labels["alertname"]
		if !alertNameFound {
			hint = fmt.Sprintf("%s: %s", amgrAlertDataRequiredStructureErrorMessage, "label 'alertname' not found")
			return false, hint, nil
		}

		// Check whether startAt field is set
		if alert.StartsAt == "" {
			hint = fmt.Sprintf("%s: %s", amgrAlertDataRequiredStructureErrorMessage, "field 'startsAt' not found")
			return false, hint, nil
		}
	}

	return true, hint, nil
}
