# searchruler
### 🔥 Meet searchruler: The Log Alerting Engine You Didn't Know You Needed!

> **📢 Migration Notice**: Starting from version [v0.5.0](https://github.com/freepik-company/searchruler/releases/tag/v0.5.0), this project has been migrated from the `prosimcorp` organization to `freepik-company`. Please update your references and dependencies to use the new repository location.

<img src="https://raw.githubusercontent.com/prosimcorp/searchruler/master/docs/img/logo.png" alt="searchruler Logo (Main) logo." width="150">

![GitHub Release](https://img.shields.io/github/v/release/prosimcorp/searchruler)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/prosimcorp/searchruler)
[![Go Report Card](https://goreportcard.com/badge/github.com/prosimcorp/searchruler)](https://goreportcard.com/report/github.com/prosimcorp/searchruler)
![image pulls](https://img.shields.io/badge/+2k-brightgreen?label=image%20pulls)
![GitHub License](https://img.shields.io/github/license/prosimcorp/searchruler)

![GitHub User's stars](https://img.shields.io/github/stars/prosimcorp?label=Prosimcorp%20Stars)
![GitHub followers](https://img.shields.io/github/followers/prosimcorp?label=Prosimcorp%20Followers)

Ever wished Prometheus Ruler had a cool cousin for log searches? Say hello to searchruler! This Kubernetes operator lets you define, run, and manage log search rules (alerts) for platforms like Elasticsearch or Opensearch—all from the comfort of your K8s cluster. 🚀

Think of it as the rule engine your logs have been craving.

And here’s the best part: defining alerts with searchruler is totally free (ehem, ehem…) and, yes, everything is as code! You get to send webhook notifications wherever you want, just like Alertmanager. Flexibility, power, and no sneaky fees.

Your logs are about to get a whole lot smarter. 💡

## Motivation

### 🕵️‍♂️ Say Goodbye to Expensive Log Alert Subscriptions!
Tired of shelling out big bucks for premium log alerting features? You know the drill: Want to set up rules or get notified? Pay up. Want to avoid endless click, click in a fancy UI? Too bad.

Well, no more! searchruler is here to save the day. This Kubernetes operator lets you define connectors, webhooks, rules, and alerts—right in your own cluster. And the best part? It’s free and code-driven! Finally, you can version-control your alerts like a pro. 🎉

### 🛠️ How It Works
Setting up searchruler is a breeze. Here are the three main building blocks that’ll make your log life so much easier:

* 🔗 **QueryConnector**: This is where the magic starts. Connect to your log source—whether it’s Elasticsearch, Opensearch, or something cool we’re cooking up for the future. The clustered scope solution is named **ClusterQueryConnector**.

* 🚀 **RulerAction**: When a rule is triggered, where should the alert go? Set up webhooks, Slack channels, or anything else you need. We keep it simple, starting with a generic webhook (because everyone loves webhooks). The clustered scope solution is named **ClusterRulerAction**.

* 📜 **SearchRule**: The heart of it all! Define your rules, set the conditions, and craft the message to send when something’s off. This is where you turn log data into actionable alerts.

### 🎉 Ready to Rule Your Logs?
No more hidden fees. No more manual clicks. Just pure, versioned, code-driven log alerting—right in Kubernetes. 🚀

## Deployment

We have designed the deployment of this project to allow remote deployment using Kustomize or Helm. This way it is possible
to use it with a GitOps approach, using tools such as ArgoCD or FluxCD.

If you prefer Kustomize, just make a Kustomization manifest referencing
the tag of the version you want to deploy as follows:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- https://github.com/prosimcorp/searchruler/releases/download/v0.1.0/install.yaml
```

> 🧚🏼 **Hey, listen! If you prefer to deploy using Helm, go to the [Helm registry](https://prosimcorp.github.io/helm-charts/)**

### Helm chart upgrade notes

Starting with the version that introduces this layout, the chart manages CRDs through `templates/crds/` (controlled by `crds.install` and `crds.keep` in `values.yaml`) instead of the legacy `crds/` directory. This makes `helm upgrade` actually update the CRDs so new fields land on existing clusters — Helm never updates CRDs that live in the special `crds/` folder.

If you installed a previous version of this chart, the CRDs already in your cluster were not created by Helm and will conflict on the first upgrade with `Error: rendered manifests contain a resource that already exists`. Adopt them into the release once before upgrading:

```bash
RELEASE=searchruler          # your helm release name
NS=searchruler-system        # the namespace where you installed it

for crd in clusterqueryconnectors clusterruleractions queryconnectors ruleractions searchrules; do
  kubectl annotate crd "$crd.searchruler.freepik.com" \
    meta.helm.sh/release-name="$RELEASE" \
    meta.helm.sh/release-namespace="$NS" --overwrite
  kubectl label crd "$crd.searchruler.freepik.com" \
    app.kubernetes.io/managed-by=Helm --overwrite
done

helm upgrade "$RELEASE" searchruler/searchruler -n "$NS"
```

By default the chart sets `helm.sh/resource-policy: keep` on every CRD, so `helm uninstall` does not cascade-delete your `SearchRule`/`QueryConnector`/`RulerAction` instances. Set `crds.keep=false` if you want full teardown on uninstall.


## Flags

Some configuration parameters can be defined by flags that can be passed to the controller.
They are described in the following table:

| Name                           | Description                                                                  | Default |
|:-------------------------------|:-----------------------------------------------------------------------------|:-------:|
| `--metrics-bind-address`       | The address the metric endpoint binds to. </br> 0 disables the server        |   `0`   |
| `--health-probe-bind-address`  | he address the probe endpoint binds to                                       | `:8081` |
| `--leader-elect`               | Enable leader election for controller manager                                | `false` |
| `--metrics-secure`             | If set the metrics endpoint is served securely                               | `false` |
| `--enable-http2`               | If set, HTTP/2 will be enabled for the metrics                               | `false` |
| `--webserver-address`          | Webserver listen address.  </br> 0 disables the webserver                    |   `0`   |
| `--rules-metrics-bind-address` | The address the custom metric endpoint binds to. </br> 0 disables the server | `false` |
| `--rules-metrics-refresh-rate` | Refresh rate of the custom metrics.                                          |  `10`   |


## Examples

After deploying this operator, you will have new resources available. Let's talk about them.
> [!TIP]
> You can find the spec samples for all the versions of the resource in the [examples directory](./config/samples)

### 🔗 QueryConnector

A `QueryConnector` is where it all starts! It defines the "source" where your log search rules (defined in SearchRules) will run. Right now, searchruler supports Elasticsearch-like sources, but we’re cooking up more integrations—stay tuned! 👀

Here’s a quick example to show you how it works:

```yaml
apiVersion: searchruler.freepik.com/v1alpha1
kind: QueryConnector
metadata:
  labels:
    app.kubernetes.io/name: search-ruler
    app.kubernetes.io/managed-by: kustomize
  name: queryconnector-sample
spec:

  # URL for the query connector. We will execute the queries in this URL
  url: "https://127.0.0.1:9200"

  # Additional headers if needed for the connection
  headers: {}

  # Skip certificate verification if the connection is HTTPS
  tlsSkipVerify: true

  # Interval to check secret credentials for any changes
  # Default value is 1m
  syncInterval: 15s

  # Certificates to use for the connection
  certificates:
    secretRef:
      name: elasticsearch-main-certificates
      namespace: default
      keyCA: ca.crt
      keyCert: cert.crt
      keyKey: key.pem

  # Secret reference to get the credentials if needed for the connection
  credentials:

    secretRef:
      name: elasticsearch-main-credentials
      namespace: default
      keyUsername: username
      keyPassword: password
```

For cluster scope just change **QueryConnector** for **ClusterQueryConenctor**.
### 🚀 RulerAction

A RulerAction defines where your alerts will be sent when a SearchRule is triggered (a.k.a. "firing"). Whether it’s a Slack channel, a webhook endpoint, alertmanager or another notification service—you’re in control! 🛠️

Here’s a quick example:
```yaml
apiVersion: searchruler.freepik.com/v1alpha1
kind: RulerAction
metadata:
  labels:
    app.kubernetes.io/name: search-ruler
    app.kubernetes.io/managed-by: kustomize
  name: ruleraction-sample
spec:

  # Webhook integration configuration to send alerts.
  # Note: The webhook integration is the only one implemented yet.
  webhook:

    # URL to send the webhook message
    url: http://127.0.0.1:8080

    # HTTP method to send the webhook message
    verb: POST

    # Skip certificate verification if the connection is HTTPS
    tlsSkipVerify: false

    # Additional headers if needed for the connection
    headers: {}

    # Validator configuration to validate the response of the webhook
    # Just alertmanager validation available yet.
    # If you use alertmanager validator, message data must be in alertmanager format:
    # https://prometheus.io/docs/alerting/latest/clients/
    # validator: alertmanager

    # Credentials to authenticate in the webhook if needed
    # credentials:   
    #   secretRef:
    #     name: alertmanager-credentials
    #     namespace: default
    #     keyUsername: username
    #     keyPassword: password
```

For cluster scope just change **QueryConnector** for **ClusterRulerAction**.
### 📜 SearchRule

This is where the magic happens! SearchRules define the conditions to check in your log sources (via queryconnectors) and specify where to send alerts (using ruleractions). You get to decide what matters and how to act on it. 🎯

The SearchRule supports two modes for sending alerts:
- **raw**: The default mode. Allows complete freedom in the message format using Go templating.
- **alertmanager**: Structured mode that generates Alertmanager-compatible alerts with templated labels and annotations.

Here are examples showing both modes:

1️⃣ **Raw Mode Example** - Simple Match Count Alert with custom message format:

```yaml
apiVersion: searchruler.freepik.com/v1alpha1
kind: SearchRule
metadata:
  name: searchrule-sample
spec:
  description: "Alert when there are a high error rate in the application."
  
  # ... other fields remain the same ...

  actionRef:
    name: ruleraction-sample
    namespace: "default"
    # Raw mode (default) allows any message format
    mode: "raw"  # This is optional as raw is the default
    data: |
      {{- $object := .object -}}
      {{- $value := .value -}}
      {{ printf "Hi, I'm on fire!" }}
      {{ printf "Name: %s" $object.Name }}
      {{ printf "Description: %s" $object.Spec.Description }}
      {{ printf "Current value: %v" $value }}
```

2️⃣ **Alertmanager Mode Example** - Using Alertmanager's format with templated labels and annotations:

```yaml
apiVersion: searchruler.freepik.com/v1alpha1
kind: SearchRule
metadata:
  name: searchrule-alertmanager-sample
spec:
  description: "Alert on high latency with Alertmanager format"
  
  # ... other fields remain the same ...

  actionRef:
    name: ruleraction-sample
    namespace: "default"
    # Use Alertmanager mode to generate compatible alerts
    mode: "alertmanager"
    # Define labels that will be included in the alert
    labels:
      alertname: "high_latency"
      severity: "warning"
      service: "{{ .object.Spec.Description }}"
      value: "{{ .value }}"
    # Define annotations for additional alert context
    annotations:
      summary: "High latency detected"
      description: |
        Service {{ .object.Spec.Description }} is experiencing high latency
        Current value: {{ .value }}
        Check interval: {{ .object.Spec.CheckInterval }}
```

#### 🔄 Mode Differences

1. **Raw Mode (`mode: "raw"` or omitted)**
   - Complete freedom in message format
   - Uses the `data` field for the entire message template
   - Perfect for custom webhooks or simple notifications

2. **Alertmanager Mode (`mode: "alertmanager"`)**
   - Generates Alertmanager-compatible alert format
   - Uses `labels` and `annotations` fields for templating
   - Automatically sets required fields like `startsAt` and `endsAt`
   - `endsAt` is calculated as 2 times the `checkInterval`
   - Ensures the required `alertname` label exists
   - Compatible with Alertmanager validation in RulerAction

#### 🎯 When to Use Each Mode

- Use **raw mode** when:
  - Sending to custom webhooks
  - Need complete control over message format
  - Integrating with systems that aren't Alertmanager

- Use **alertmanager mode** when:
  - Sending alerts to Alertmanager
  - Need structured alerts with labels and annotations
  - Want automatic validation of alert format
  - Need proper alert lifecycle management (start/end times)

#### 📩 Customizing Alert Messages for Alertmanager
In the `actionRef.data` field, you define the message that gets sent to your webhook. If your webhook is Alertmanager, you'll need to structure the message according to Alertmanager's format. Plus, you can enable the validator in the RulerAction to ensure everything’s correctly formatted.

Here’s an example to show how to configure an Alertmanager-compatible message:
```yaml
  # RuleAction reference to execute when the condition is true.
  actionRef:
    name: ruleraction-sample
    # Message template to send in the RuleAction execution. It is a Go template with the
    # object, value and, if exists, elasticsearch aggregations field variables. The object 
    # variable is the SearchRule object and the value variable is the value of the conditionField.

    # If the ruleaction is a alertmanager webhook, the message must be in alertmanager format:
    # https://prometheus.io/docs/alerting/latest/clients/
    data: |
        {{- $now := now | date "2006-01-02T15:04:05Z07:00" }}
        {{- $object := .object -}}
        {{- $value := .value -}}
        {{- $aggregations := .aggregations -}}

        {{- $alertList := list }}

        {{- $bucketsStr := "" }}
        {{- range .aggregations.hosts.buckets }}
          {{- $bucketInfo := printf "Number of documents -> %v, Response time -> %v" .doc_count .total_response_time.value }}
          {{- $bucketsStr = printf "%s\n%s" $bucketsStr $bucketInfo }}
        {{- end }}

        {{- $description := printf `

        Description: %s
        Value: %v
        Aggregations values: %s

        -------------------------------
        Name: %s
        Namespace: %s
        -------------------------------
        ` .object.Spec.Description .value $bucketStr .object.Name .object.Namespace }}

        {{- $description = ((regexReplaceAll "(?m)^[ \\t]+" $description "") | trim) }}

        {{- $annotations := dict
        "sent_by" "searchruler"
        "summary" "There are rules firing"
        "description" $description }}

        {{- $labels := dict
        "alertname" .object.Name
        "namespace" .object.Namespace
        "name" .object.Name
        "severity" "warning"
        "type" "searchruler-alert" }}

        {{- $alert := dict "startsAt" $now "annotations" $annotations "labels" $labels "generatorURL" "string-placeholder" }}
        {{- $alertList = append $alertList $alert }}

        {{- $alertJson := toJson $alertList }}
        {{- $alertJson }}
```
> [!TIP]
> 🔍 **Why Use This?**: By customizing the alert message to fit Alertmanager’s structure, you ensure seamless integration and make sure your alerts get delivered exactly the way you need. Plus, with validation enabled, you won’t have to worry about > formatting errors—everything’s checked before it’s sent! 🚀

2️⃣ **Average Field Value Alert**. Alert if the average value of a field exceeds a limit (e.g., high response times):
```yaml
apiVersion: searchruler.freepik.com/v1alpha1
kind: SearchRule
metadata:
  labels:
    app.kubernetes.io/name: search-ruler
    app.kubernetes.io/managed-by: kustomize
  name: searchrule-sample
spec:

  # Description for the Rule. It is not used in the rule execution, but is useful for the
  # message template in the RuleAction.
  description: "Alert when there are a high latency in the application."

  # QueryConnector reference to execute the queries for the rule evaluation.
  queryConnectorRef:
    name: queryconnector-sample
    namespace: default

  # Interval time for checking the value of the query. For example, every 30s we will
  # execute the query value to elasticsearch
  checkInterval: 30s

  # Elasticsearch configuration for the query execution.
  # Just elasticsearch is implemented yet.
  elasticsearch:

    # Index, index pattern or alias where the query will be executed
    # It will be appended to <URL>/<index>/_search endpoint
    index: "kibana_sample_data_logs"

    # Another example for queries with aggregations
    query:
      _source: [""]
      query:
        bool:
          must:
            - range:
                timestamp:
                  gte: "now-5m/m"
                  lte: "now/m"
        aggs:
          average_response_time:
            avg:
              field: "upstream_response_time_f"
    conditionField: "aggregations.average_response_time.value"

  # Condition for the rule evaluation. It will check the conditionField value with the
  # operator and threshold. If the condition is true, the RuleAction will be executed.
  condition:
    # Available options: greaterThan, greaterThanOrEqual, lessThan, lessThanOrEqual or equal
    operator: "greaterThan"
    # Threshold value to check the condition
    threshold: "5"
    # Time window to check the condition. For example, if the condition is greaterThan 100 for 1m
    for: "1m"

  # RuleAction reference to execute when the condition is true.
  actionRef:
    name: ruleraction-sample
    namespace: default
    # Message template to send in the RuleAction execution. It is a Go template with the
    # object, value and, if exists, elasticsearch aggregations field variables. The object 
    # variable is the SearchRule object and the value variable is the value of the conditionField.

    # If the ruleaction is a alertmanager webhook, the message must be in alertmanager format:
    # https://prometheus.io/docs/alerting/latest/clients/
    data: |
      {{- $object := .object -}}
      {{- $value := .value -}}
      {{ printf "Hi, I'm on fire!" }}
      {{ printf "Name: %s" $object.Name }}
      {{ printf "Description: %s" $object.Spec.Description }}
      {{ printf "Current value: %v" $value }}

```

#### 📡 Auto-generate a PrometheusRule

If your stack already runs the [prometheus-operator](https://github.com/prometheus-operator/prometheus-operator) plus Alertmanager, you don't need a `RulerAction` for the alert to land in Alertmanager — the operator can generate a `PrometheusRule` resource for you that mirrors the SearchRule's condition. The Prometheus Operator picks it up automatically and Prometheus evaluates the alert against the `searchrule_value` metric exposed by this operator.

```yaml
apiVersion: searchruler.freepik.com/v1alpha1
kind: SearchRule
metadata:
  name: searchrule-prometheusrule-sample
spec:
  description: "High 5xx rate"
  # ... queryConnectorRef, checkInterval, elasticsearch, condition as usual ...

  prometheusRule:
    enabled: true
    # Optional. Defaults to the SearchRule's name.
    alertName: HighErrorRate
    labels:
      severity: warning
      team: platform
    annotations:
      summary: "High error rate on the application"
      runbook_url: "https://runbooks.example.com/high-errors"

  # actionRef is optional when prometheusRule is enabled. At least one of
  # actionRef or prometheusRule MUST be defined.
```

Behavior:
- The generated `PrometheusRule` is named after the SearchRule, lives in the same namespace, and is owned by the SearchRule (deleting the SearchRule garbage-collects the alert).
- The PromQL expression is derived from `spec.condition`, e.g. `searchrule_value{searchrule_namespace="apps", rule="searchrule-prometheusrule-sample"} > 100`. The namespace label is exported as `searchrule_namespace` (not `namespace`) so it does not collide with the target label Prometheus injects when scraping via a ServiceMonitor.
- The `for` window comes from `spec.condition.for`.
- A `searchrule` label is added automatically to the alert so multiple SearchRules can share the same group in Alertmanager dashboards.

Prerequisites and caveats:
- The `monitoring.coreos.com/v1` PrometheusRule CRD must exist in the cluster. If it is missing, the SearchRule reports `PrometheusRule.Unsupported` in its `status.conditions` and no resource is created. The operator boot logs a warning at startup.
- The custom-metrics endpoint must be enabled with `--rules-metrics-bind-address`, otherwise the underlying `searchrule_value` metric is never exported and the alert can never fire. In that case the SearchRule still gets a PrometheusRule, but it is marked with `PrometheusRule.MetricsNotExposed` in `status.conditions` so the misconfiguration is visible.
- A `ServiceMonitor` (or `PodMonitor`) targeting the operator's metrics service must exist for Prometheus to actually scrape the metric. The chart's `controller.customMetrics.service` enables the Service; you provision the ServiceMonitor according to your Prometheus deployment.

#### 🧮 Custom metrics with bucket dimensions

By default, the generated PrometheusRule alerts on `searchrule_value` — a single gauge that holds the resolved `conditionField`. That works when the value already represents the whole rule (e.g. a count), but it loses information when the underlying query is a `terms` aggregation reduced to a scalar via `max_bucket` or similar. The classic case: an aggregation that groups 5xx percentages **by host** so you can act on the dominant offender.

`spec.customMetrics` lets the operator emit one Prometheus sample per bucket of the response, with labels mapped from the bucket fields:

```yaml
spec:
  # …queryConnectorRef, checkInterval, elasticsearch, condition unchanged…

  customMetrics:
    - name: akamai_5xx_by_host          # exposes searchrule_akamai_5xx_by_host
      help: "Akamai 5xx percentage by host (last 5m)"
      aggregation_map: by_domain.buckets
      labels:
        - name: host
          value: key                    # gjson path inside each bucket
      value: error_percentage.value     # defaults to `doc_count` when empty

  prometheusRule:
    enabled: true
    alertName: Akamai5xxByHostHighRate
    # metricName: akamai_5xx_by_host    # optional selector when several customMetrics declared
    labels:
      severity: critical
    annotations:
      summary: "High 5xx percentage by host (last 5m)"
      description: "Host {{ $labels.host }} at {{ $value | humanizePercentage }} 5xx"
```

What the operator publishes on `/metrics`:

```
searchrule_akamai_5xx_by_host{searchrule_namespace="searchruler",rule="…",host="cp.freepik.com"} 22.43
searchrule_akamai_5xx_by_host{searchrule_namespace="searchruler",rule="…",host="www.freepik.com"} 0.40
```

The auto-generated PrometheusRule targets that metric instead of `searchrule_value`, so Alertmanager receives one alert per bucket above the threshold with the bucket labels attached:

```
ALERTS{alertname="Akamai5xxByHostHighRate", host="cp.freepik.com", severity="critical", …}
```

Notes:
- The series set is reset every refresh tick (default 10s). When a bucket disappears from the query response (host stops failing), the series is deleted from `/metrics` automatically — no stale firing alerts.
- `searchrule_namespace` and `rule` are added implicitly so multiple SearchRules can publish under the same metric name without colliding.
- The bucket count is capped at 1000 per refresh; truncations are observable through the `searchrule_custom_metrics_truncated_total{rule="…",metric="…"}` counter.
- A SearchRule may declare up to 10 entries in `customMetrics`. The very first one is the default target of the generated PrometheusRule expression; pick another with `prometheusRule.metricName`.

## Templating engine

❤️ Special mention to [Notifik](https://github.com/freepik-company/notifik/tree/master)

### What you can use

In the actionRef.Data you can use everything you
already know from [Helm Template](https://helm.sh/docs/chart_template_guide/functions_and_pipelines/)

### How to use collected data

When a rule is firing, the data field is the one which the `RulerAction` will fire to the webhook. You can access many data for creating the message template like:
* `.object`: The `SearchRule` manifest.
* `.value`: The value of the query which detonates the alert firing.
* `.aggregations`: The value of elasticsearch aggregation response if exists. We transform the JSON response of elasticsearch into an structure to be queried in your template. For example, for queries with aggregations, the value of this field will be like:
  ```
  aggregationName:
    doc_count_error_upper_bound: 2
    sum_other_doc_count: 120
    buckets:
    - key: key1
      doc_count: 100
    - key: key2
      doc_count: 200
  ```

This means that the objects can be accessed or stored in variables in the following way:
```yaml
apiVersion: notifik.freepik.com/v1alpha1
kind: SearchRule
metadata:
  name: searchrule-sample-simple
spec:
  .
  .
  .
  actionRef:
    name: ruleraction-sample
    data: |
      {{- $object := .object -}}
      {{- $value := .value -}}    
      {{- $aggregationValues := .aggregations.buckets }}   
      {{ printf "Name: %s" $object.Name }}
      {{ printf "Description: %s" $object.Spec.Description }}
      {{ printf "Current value: %v" $value }}
      {{- range $aggregationValues }}
      {{ printf "%s: %s", .key .doc_count }}
      {{- end }}
```

> Remember: with a big power comes a big responsibility
> ```gotemplate
> {{- $source := . -}}
> ```

### How to debug

Templating issues are thrown on controller logs, but you also can see the `State` of your `searchruler` in `EvaluateTemplateError` state if there is any error evaluating the template.

To debug templates easy, we recommend using [helm-playground](https://helm-playground.com). 
You can create a template on the left side, put your manifests in the middle, and the result is shown on the right side.

## Metrics

With the custom metrics feature flag enabled (`--rules-metrics-bind-address` and `--rules-metrics-refresh-rate`) a
Prometheus metrics server will be available to scrape default metrics and the custom metrics defined in the 
`SearchRule` manifest.

### Default metrics
Default metrics are the following:
* `searchrule_value`: The value of the condition field of the `SearchRule` manifest.
* `searchrule_state`: The state of the `SearchRule` manifest.
```
# HELP searchrule_state State of the search rule
# TYPE searchrule_state gauge
searchrule_state{rule="searchrule-sample",state="Firing"} 0
searchrule_state{rule="searchrule-sample",state="Normal"} 0
searchrule_state{rule="searchrule-sample",state="PendingFiring"} 1
searchrule_state{rule="searchrule-sample",state="PendingResolving"} 0
# HELP searchrule_value Value of the search rule
# TYPE searchrule_value gauge
searchrule_value{rule="searchrule-sample"} 3401
```

## How to develop

### Prerequisites
- Kubebuilder v4.0.0+
- go version v1.22.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### The process

> We recommend you to use a development tool like [Kind](https://kind.sigs.k8s.io/) or [Minikube](https://minikube.sigs.k8s.io/docs/start/)
> to launch a lightweight Kubernetes on your local machine for development purposes

For learning purposes, we will suppose you are going to use Kind. So the first step is to create a Kubernetes cluster
on your local machine executing the following command:

```console
kind create cluster
```

Once you have launched a safe play place, execute the following command. It will install the custom resource definitions
(CRDs) in the cluster configured in your ~/.kube/config file and run Kuberbac locally against the cluster:

```console
make install run
```

If you would like to test the operator against some resources, our examples can be applied to see the result in
your Kind cluster

```sh
kubectl apply -k config/samples/
```

> Remember that your `kubectl` is pointing to your Kind cluster. However, you should always review the context your
> kubectl CLI is pointing to



## How releases are created

Each release of this operator is done following several steps carefully in order not to break the things for anyone.
Reliability is important to us, so we automated all the process of launching a release. For a better understanding of
the process, the steps are described in the following recipe:

1. Test the changes on the code:

    ```console
    make test
    ```

   > A release is not done if this stage fails


2. Define the package information

    ```console
    export VERSION="0.0.1"
    export IMG="ghcr.io/prosimcorp/searchruler:v$VERSION"
    ```

3. Generate and push the Docker image (published on Docker Hub).

    ```console
    make docker-build docker-push
    ```

4. Generate the manifests for deployments using Kustomize

   ```console
    make build-installer
    ```



## How to collaborate

This project is done on top of [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder), so read about that project
before collaborating. Of course, we are open to external collaborations for this project. For doing it you must fork the
repository, make your changes to the code and open a PR. The code will be reviewed and tested (always)

> We are developers and hate bad code. For that reason we ask you the highest quality on each line of code to improve
> this project on each iteration.



## License

Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.