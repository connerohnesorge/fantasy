---
title: "REST API"
source: "https://mlflow.org/docs/latest/api_reference/rest-api.html"
author:
published:
created: 2025-12-25
description:
tags:
  - "clippings"
---
The MLflow REST API allows you to create, list, and get experiments and runs, and log parameters, metrics, and artifacts. The API is hosted under the `/api` route on the MLflow tracking server. For example, to search for experiments on a tracking server hosted at `http://localhost:5000`, make a POST request to `http://localhost:5000/api/2.0/mlflow/experiments/search`.

Table of Contents

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/experiments/create` | `POST` |

Create an experiment with a name. Returns the ID of the newly created experiment. Validates that another experiment with the same name does not already exist and fails if another experiment with the same name already exists.

Throws `RESOURCE_ALREADY_EXISTS` if a experiment with the given name exists.

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| name | `STRING` | Experiment name. This field is required. |
| artifact\_location | `STRING` | Location where all artifacts for the experiment are stored. If not provided, the remote server will select an appropriate default. |
| tags | An array of | A collection of tags to set on the experiment. Maximum tag size and number of tags per request depends on the storage backend. All storage backends are guaranteed to support tag keys up to 250 bytes in size and tag values up to 5000 bytes in size. All storage backends are also guaranteed to support up to 20 tags per request. |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| experiment\_id | `STRING` | Unique identifier for the experiment. |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/experiments/search` | `POST` |

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| max\_results | `INT64` | Maximum number of experiments desired. Servers may select a desired default max\_results value. All servers are guaranteed to support a max\_results threshold of at least 1,000 but may support more. Callers of this endpoint are encouraged to pass max\_results explicitly and leverage page\_token to iterate through experiments. |
| page\_token | `STRING` | Token indicating the page of experiments to fetch |
| filter | `STRING` | A filter expression over experiment attributes and tags that allows returning a subset of experiments. The syntax is a subset of SQL that supports ANDing together binary operations between an attribute or tag, and a constant.  Example: `name LIKE 'test-%' AND tags.key = 'value'`  You can select columns with special characters (hyphen, space, period, etc.) by using double quotes or backticks.  Example: `tags."extra-key" = 'value'` or ``tags.`extra-key` = 'value'``  Supported operators are `=`, `!=`, `LIKE`, and `ILIKE`. |
| order\_by | An array of `STRING` | List of columns for ordering search results, which can include experiment name and id with an optional “DESC” or “ASC” annotation, where “ASC” is the default. Tiebreaks are done by experiment id DESC. |
| view\_type |  | Qualifier for type of experiments to be returned. If unspecified, return only active experiments. |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| experiments | An array of | Experiments that match the search criteria |
| next\_page\_token | `STRING` | Token that can be used to retrieve the next page of experiments. An empty token means that no more experiments are available for retrieval. |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/experiments/get` | `GET` |

Get metadata for an experiment. This method works on deleted experiments.

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| experiment\_id | `STRING` | ID of the associated experiment. This field is required. |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| experiment |  | Experiment details. |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/experiments/get-by-name` | `GET` |

Get metadata for an experiment.

This endpoint will return deleted experiments, but prefers the active experiment if an active and deleted experiment share the same name. If multiple deleted experiments share the same name, the API will return one of them.

Throws `RESOURCE_DOES_NOT_EXIST` if no experiment with the specified name exists.

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| experiment\_name | `STRING` | Name of the associated experiment. This field is required. |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| experiment |  | Experiment details. |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/experiments/delete` | `POST` |

Mark an experiment and associated metadata, runs, metrics, params, and tags for deletion. If the experiment uses FileStore, artifacts associated with experiment are also deleted.

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| experiment\_id | `STRING` | ID of the associated experiment. This field is required. |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/experiments/restore` | `POST` |

Restore an experiment marked for deletion. This also restores associated metadata, runs, metrics, params, and tags. If experiment uses FileStore, underlying artifacts associated with experiment are also restored.

Throws `RESOURCE_DOES_NOT_EXIST` if experiment was never created or was permanently deleted.

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| experiment\_id | `STRING` | ID of the associated experiment. This field is required. |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/experiments/update` | `POST` |

Update experiment metadata.

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| experiment\_id | `STRING` | ID of the associated experiment. This field is required. |
| new\_name | `STRING` | If provided, the experiment’s name is changed to the new name. The new name must be unique. |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/runs/create` | `POST` |

Create a new run within an experiment. A run is usually a single execution of a machine learning or data ETL pipeline. MLflow uses runs to track ,, and associated with a single execution.

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| experiment\_id | `STRING` | ID of the associated experiment. |
| user\_id | `STRING` | ID of the user executing the run. This field is deprecated as of MLflow 1.0, and will be removed in a future MLflow release. Use ‘mlflow.user’ tag instead. |
| run\_name | `STRING` | Name of the run. |
| start\_time | `INT64` | Unix timestamp in milliseconds of when the run started. |
| tags | An array of | Additional metadata for run. |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| run |  | The newly created run. |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/runs/delete` | `POST` |

Mark a run for deletion.

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| run\_id | `STRING` | ID of the run to delete. This field is required. |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/runs/restore` | `POST` |

Restore a deleted run.

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| run\_id | `STRING` | ID of the run to restore. This field is required. |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/runs/get` | `GET` |

Get metadata, metrics, params, and tags for a run. In the case where multiple metrics with the same key are logged for a run, return only the value with the latest timestamp. If there are multiple values with the latest timestamp, return the maximum of these values.

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| run\_id | `STRING` | ID of the run to fetch. Must be provided. |
| run\_uuid | `STRING` | \[Deprecated, use run\_id instead\] ID of the run to fetch. This field will be removed in a future MLflow version. |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| run |  | Run metadata (name, start time, etc) and data (metrics, params, and tags). |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/runs/log-metric` | `POST` |

Log a metric for a run. A metric is a key-value pair (string key, float value) with an associated timestamp. Examples include the various metrics that represent ML model accuracy. A metric can be logged multiple times.

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| run\_id | `STRING` | ID of the run under which to log the metric. Must be provided. |
| run\_uuid | `STRING` | \[Deprecated, use run\_id instead\] ID of the run under which to log the metric. This field will be removed in a future MLflow version. |
| key | `STRING` | Name of the metric. This field is required. |
| value | `DOUBLE` | Double value of the metric being logged. This field is required. |
| timestamp | `INT64` | Unix timestamp in milliseconds at the time metric was logged. This field is required. |
| step | `INT64` | Step at which to log the metric |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/runs/log-batch` | `POST` |

Log a batch of metrics, params, and tags for a run. If any data failed to be persisted, the server will respond with an error (non-200 status code). In case of error (due to internal server error or an invalid request), partial data may be written.

You can write metrics, params, and tags in interleaving fashion, but within a given entity type are guaranteed to follow the order specified in the request body. That is, for an API request like

```json
{
   "run_id": "2a14ed5c6a87499199e0106c3501eab8",
   "metrics": [
     {"key": "mae", "value": 2.5, "timestamp": 1552550804},
     {"key": "rmse", "value": 2.7, "timestamp": 1552550804},
   ],
   "params": [
     {"key": "model_class", "value": "LogisticRegression"},
   ]
}

```

the server is guaranteed to write metric “rmse” after “mae”, though it may write param “model\_class” before both metrics, after “mae”, or after both metrics.

The overwrite behavior for metrics, params, and tags is as follows:

- Metrics: metric values are never overwritten. Logging a metric (key, value, timestamp) appends to the set of values for the metric with the provided key.
- Tags: tag values can be overwritten by successive writes to the same tag key. That is, if multiple tag values with the same key are provided in the same API request, the last-provided tag value is written. Logging the same tag (key, value) is permitted - that is, logging a tag is idempotent.
- Params: once written, param values cannot be changed (attempting to overwrite a param value will result in an error). However, logging the same param (key, value) is permitted - that is, logging a param is idempotent.

### Request Limits

A single JSON-serialized API request may be up to 1 MB in size and contain:

- No more than 1000 metrics, params, and tags in total
- Up to 1000 metrics
- Up to 100 params
- Up to 100 tags

For example, a valid request might contain 900 metrics, 50 params, and 50 tags, but logging 900 metrics, 50 params, and 51 tags is invalid. The following limits also apply to metric, param, and tag keys and values:

- Metric, param, and tag keys can be up to 250 characters in length
- Param and tag values can be up to 250 characters in length

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| run\_id | `STRING` | ID of the run to log under |
| metrics | An array of | Metrics to log. A single request can contain up to 1000 metrics, and up to 1000 metrics, params, and tags in total. |
| params | An array of | Params to log. A single request can contain up to 100 params, and up to 1000 metrics, params, and tags in total. |
| tags | An array of | Tags to log. A single request can contain up to 100 tags, and up to 1000 metrics, params, and tags in total. |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/runs/log-model` | `POST` |

Note

Experimental: This API may change or be removed in a future release without warning.

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| run\_id | `STRING` | ID of the run to log under |
| model\_json | `STRING` | MLmodel file in json format. |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/runs/log-inputs` | `POST` |

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| run\_id | `STRING` | ID of the run to log under This field is required. |
| datasets | An array of | Dataset inputs |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/experiments/set-experiment-tag` | `POST` |

Set a tag on an experiment. Experiment tags are metadata that can be updated.

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| experiment\_id | `STRING` | ID of the experiment under which to log the tag. Must be provided. This field is required. |
| key | `STRING` | Name of the tag. Maximum size depends on storage backend. All storage backends are guaranteed to support key values up to 250 bytes in size. This field is required. |
| value | `STRING` | String value of the tag being logged. Maximum size depends on storage backend. All storage backends are guaranteed to support key values up to 5000 bytes in size. This field is required. |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/experiments/delete-experiment-tag` | `POST` |

Delete a tag on an experiment.

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| experiment\_id | `STRING` | ID of the experiment that the tag was logged under. Must be provided. This field is required. |
| key | `STRING` | Name of the tag. Maximum size is 255 bytes. Must be provided. This field is required. |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/runs/set-tag` | `POST` |

Set a tag on a run. Tags are run metadata that can be updated during a run and after a run completes.

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| run\_id | `STRING` | ID of the run under which to log the tag. Must be provided. |
| run\_uuid | `STRING` | \[Deprecated, use run\_id instead\] ID of the run under which to log the tag. This field will be removed in a future MLflow version. |
| key | `STRING` | Name of the tag. Maximum size depends on storage backend. All storage backends are guaranteed to support key values up to 250 bytes in size. This field is required. |
| value | `STRING` | String value of the tag being logged. Maximum size depends on storage backend. All storage backends are guaranteed to support key values up to 5000 bytes in size. This field is required. |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/runs/delete-tag` | `POST` |

Delete a tag on a run. Tags are run metadata that can be updated during a run and after a run completes.

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| run\_id | `STRING` | ID of the run that the tag was logged under. Must be provided. This field is required. |
| key | `STRING` | Name of the tag. Maximum size is 255 bytes. Must be provided. This field is required. |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/runs/log-parameter` | `POST` |

Log a param used for a run. A param is a key-value pair (string key, string value). Examples include hyperparameters used for ML model training and constant dates and values used in an ETL pipeline. A param can be logged only once for a run.

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| run\_id | `STRING` | ID of the run under which to log the param. Must be provided. |
| run\_uuid | `STRING` | \[Deprecated, use run\_id instead\] ID of the run under which to log the param. This field will be removed in a future MLflow version. |
| key | `STRING` | Name of the param. Maximum size is 255 bytes. This field is required. |
| value | `STRING` | String value of the param being logged. Maximum size is 6000 bytes. This field is required. |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/metrics/get-history` | `GET` |

Get a list of all values for the specified metric for a given run.

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| run\_id | `STRING` | ID of the run from which to fetch metric values. Must be provided. |
| run\_uuid | `STRING` | \[Deprecated, use run\_id instead\] ID of the run from which to fetch metric values. This field will be removed in a future MLflow version. |
| metric\_key | `STRING` | Name of the metric. This field is required. |
| page\_token | `STRING` | Token indicating the page of metric history to fetch |
| max\_results | `INT32` | Maximum number of logged instances of a metric for a run to return per call. Backend servers may restrict the value of max\_results depending on performance requirements. Requests that do not specify this value will behave as non-paginated queries where all metric history values for a given metric within a run are returned in a single response. |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| metrics | An array of | All logged values for this metric. |
| next\_page\_token | `STRING` | Token that can be used to issue a query for the next page of metric history values. A missing token indicates that no additional metrics are available to fetch. |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/runs/search` | `POST` |

Search for runs that satisfy expressions. Search expressions can use and keys.

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| experiment\_ids | An array of `STRING` | List of experiment IDs to search over. |
| filter | `STRING` | A filter expression over params, metrics, and tags, that allows returning a subset of runs. The syntax is a subset of SQL that supports ANDing together binary operations between a param, metric, or tag and a constant.  Example: `metrics.rmse < 1 and params.model_class = 'LogisticRegression'`  You can select columns with special characters (hyphen, space, period, etc.) by using double quotes:`metrics."model class" = 'LinearRegression' and tags."user-name" = 'Tomas'`  Supported operators are `=`, `!=`, `>`, `>=`, `<`, and `<=`. |
| run\_view\_type |  | Whether to display only active, only deleted, or all runs. Defaults to only active runs. |
| max\_results | `INT32` | Maximum number of runs desired. If unspecified, defaults to 1000. All servers are guaranteed to support a max\_results threshold of at least 50,000 but may support more. Callers of this endpoint are encouraged to pass max\_results explicitly and leverage page\_token to iterate through experiments. |
| order\_by | An array of `STRING` | List of columns to be ordered by, including attributes, params, metrics, and tags with an optional “DESC” or “ASC” annotation, where “ASC” is the default. Example: \[“params.input DESC”, “metrics.alpha ASC”, “metrics.rmse”\] Tiebreaks are done by start\_time DESC followed by run\_id for runs with the same start time (and this is the default ordering criterion if order\_by is not provided). |
| page\_token | `STRING` |  |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| runs | An array of | Runs that match the search criteria. |
| next\_page\_token | `STRING` |  |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/artifacts/list` | `GET` |

List artifacts for a run. Takes an optional `artifact_path` prefix which if specified, the response contains only artifacts with the specified prefix.

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| run\_id | `STRING` | ID of the run whose artifacts to list. Must be provided. |
| run\_uuid | `STRING` | \[Deprecated, use run\_id instead\] ID of the run whose artifacts to list. This field will be removed in a future MLflow version. |
| path | `STRING` | Filter artifacts matching this path (a relative path from the root artifact directory). |
| page\_token | `STRING` | Token indicating the page of artifact results to fetch |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| root\_uri | `STRING` | Root artifact directory for the run. |
| files | An array of | File location and metadata for artifacts. |
| next\_page\_token | `STRING` | Token that can be used to retrieve the next page of artifact results |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/runs/update` | `POST` |

Update run metadata.

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| run\_id | `STRING` | ID of the run to update. Must be provided. |
| run\_uuid | `STRING` | \[Deprecated, use run\_id instead\] ID of the run to update.. This field will be removed in a future MLflow version. |
| status |  | Updated status of the run. |
| end\_time | `INT64` | Unix timestamp in milliseconds of when the run ended. |
| run\_name | `STRING` | Updated name of the run. |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| run\_info |  | Updated metadata of the run. |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/scorers/list` | `GET` |

List all scorers for an experiment.

### Request Structure

List all scorers for an experiment.

| Field Name | Type | Description |
| --- | --- | --- |
| experiment\_id | `STRING` | The experiment ID. |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| scorers | An array of | List of scorer entities (latest version for each scorer name). |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/scorers/versions` | `GET` |

List all versions of a specific scorer for an experiment.

### Request Structure

List all versions of a specific scorer for an experiment.

| Field Name | Type | Description |
| --- | --- | --- |
| experiment\_id | `STRING` | The experiment ID. |
| name | `STRING` | The scorer name. |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| scorers | An array of | List of scorer entities for all versions of the scorer. |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/scorers/register` | `POST` |

Register a scorer for an experiment.

### Request Structure

Register a scorer for an experiment.

| Field Name | Type | Description |
| --- | --- | --- |
| experiment\_id | `STRING` | The experiment ID. |
| name | `STRING` | The scorer name. |
| serialized\_scorer | `STRING` | The serialized scorer string (JSON). |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| version | `INT32` | The new version number for the scorer. |
| scorer\_id | `STRING` | The unique identifier for the scorer. |
| experiment\_id | `STRING` | The experiment ID (same as request). |
| name | `STRING` | The scorer name (same as request). |
| serialized\_scorer | `STRING` | The serialized scorer string (same as request). |
| creation\_time | `INT64` | The creation time of the scorer version (in milliseconds since epoch). |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/scorers/get` | `GET` |

Get a specific scorer for an experiment.

### Request Structure

Get a specific scorer for an experiment.

| Field Name | Type | Description |
| --- | --- | --- |
| experiment\_id | `STRING` | The experiment ID. |
| name | `STRING` | The scorer name. |
| version | `INT32` | The scorer version. If not specified, returns the scorer with maximum version. |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| scorer |  | The scorer entity. |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/scorers/delete` | `DELETE` |

Delete a scorer for an experiment.

### Request Structure

Delete a scorer for an experiment.

| Field Name | Type | Description |
| --- | --- | --- |
| experiment\_id | `STRING` | The experiment ID. |
| name | `STRING` | The scorer name. |
| version | `INT32` | The scorer version to delete. If not specified, deletes all versions. |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/gateway/endpoints/create` | `POST` |

Create a new endpoint with model configurations

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| name | `STRING` | Optional user-friendly name for the endpoint |
| model\_definition\_ids | An array of `STRING` | List of model definition IDs to attach to this endpoint |
| created\_by | `STRING` | Username of the creator |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| endpoint |  | The created endpoint with all model mappings |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/gateway/endpoints/update` | `POST` |

Update an endpoint’s name

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| endpoint\_id | `STRING` | ID of the endpoint to update |
| name | `STRING` | Optional new name for the endpoint |
| updated\_by | `STRING` | Username of the updater |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| endpoint |  | The updated endpoint |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/gateway/endpoints/delete` | `DELETE` |

Delete an endpoint and all its model configurations

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| endpoint\_id | `STRING` | ID of the endpoint to delete |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/gateway/endpoints/get` | `GET` |

Get endpoint details including all model configurations

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| endpoint\_id | `STRING` | Either endpoint\_id or name must be provided |
| name | `STRING` |  |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| endpoint |  | The endpoint with all model configurations |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/gateway/endpoints/list` | `GET` |

List endpoints with optional filtering by provider or secret

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| provider | `STRING` | Optional filter by provider |
| secret\_id | `STRING` | Optional filter by secret ID |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| endpoints | An array of | List of endpoints with their model configurations |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/gateway/endpoints/bindings/create` | `POST` |

Create a binding between an endpoint and an MLflow resource

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| endpoint\_id | `STRING` | ID of the endpoint to bind |
| resource\_type | `STRING` | Type of MLflow resource |
| resource\_id | `STRING` | ID of the resource instance |
| created\_by | `STRING` | Username of the creator |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| binding |  | The created binding |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/gateway/endpoints/bindings/delete` | `DELETE` |

Delete a binding between an endpoint and a resource

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| endpoint\_id | `STRING` | ID of the endpoint |
| resource\_type | `STRING` | Type of resource bound to the endpoint |
| resource\_id | `STRING` | ID of the resource |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/gateway/endpoints/bindings/list` | `GET` |

List all bindings for an endpoint

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| endpoint\_id | `STRING` | ID of the endpoint to list bindings for |
| resource\_type | `STRING` | Type of resource to filter bindings by (e.g., “scorer\_job”) |
| resource\_id | `STRING` | ID of the resource to filter bindings by |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| bindings | An array of | List of bindings for the endpoint |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/gateway/model-definitions/create` | `POST` |

Create a reusable model definition

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| name | `STRING` | User-friendly name for the model definition (must be unique) |
| secret\_id | `STRING` | ID of the secret containing authentication credentials |
| provider | `STRING` | LLM provider (e.g., “openai”, “anthropic”) |
| model\_name | `STRING` | Provider-specific model identifier (e.g., “gpt-4o”, “claude-3-5-sonnet”) |
| created\_by | `STRING` | Username of the creator |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| model\_definition |  | The created model definition |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/gateway/model-definitions/update` | `POST` |

Update a model definition

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| model\_definition\_id | `STRING` | ID of the model definition to update |
| name | `STRING` | Optional new name |
| secret\_id | `STRING` | Optional new secret ID |
| model\_name | `STRING` | Optional new model name |
| updated\_by | `STRING` | Username of the updater |
| provider | `STRING` | Optional new provider |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| model\_definition |  | The updated model definition |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/gateway/model-definitions/delete` | `DELETE` |

Delete a model definition (fails if in use by any endpoint)

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| model\_definition\_id | `STRING` | ID of the model definition to delete (fails if in use by any endpoint) |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/gateway/model-definitions/get` | `GET` |

Get a model definition by ID

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| model\_definition\_id | `STRING` | ID of the model definition to retrieve |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| model\_definition |  | The model definition |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/gateway/model-definitions/list` | `GET` |

List all model definitions with optional filters

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| provider | `STRING` | Optional filter by provider |
| secret\_id | `STRING` | Optional filter by secret ID |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| model\_definitions | An array of | List of model definitions |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/gateway/secrets/create` | `POST` |

Create a new encrypted secret for LLM provider authentication

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| secret\_name | `STRING` | User-friendly name for the secret (must be unique) |
| secret\_value | An array of | The secret value(s) to encrypt as key-value pairs. For simple API keys: {“api\_key”: “sk-xxx”} For compound credentials: {“aws\_access\_key\_id”: “…”, “aws\_secret\_access\_key”: “…”} |
| provider | `STRING` | Optional LLM provider (e.g., “openai”, “anthropic”) |
| auth\_config\_json | `STRING` | Optional provider-specific auth configuration as JSON string. For multi-auth providers, include “auth\_mode” key (e.g., {“auth\_mode”: “access\_keys”, “aws\_region\_name”: “us- east-1”}) |
| created\_by | `STRING` | Username of the creator |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| secret |  | The created secret metadata (does not include encrypted value) |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/gateway/secrets/update` | `POST` |

Update an existing secret’s value or auth configuration

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| secret\_id | `STRING` | ID of the secret to update |
| secret\_value | An array of | Optional new secret value(s) for key rotation as key-value pairs (empty map = no change). For simple API keys: {“api\_key”: “sk-xxx”} For compound credentials: {“aws\_access\_key\_id”: “…”, “aws\_secret\_access\_key”: “…”} |
| auth\_config\_json | `STRING` | Optional new auth configuration as JSON string. For multi-auth providers, include “auth\_mode” key (e.g., {“auth\_mode”: “access\_keys”, “aws\_region\_name”: “us- east-1”}) |
| updated\_by | `STRING` | Username of the updater |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| secret |  | The updated secret metadata |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/gateway/secrets/delete` | `DELETE` |

Delete a secret

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| secret\_id | `STRING` | ID of the secret to delete |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/gateway/secrets/get` | `GET` |

Get metadata about a secret (does not include the encrypted value)

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| secret\_id | `STRING` | Either secret\_id or secret\_name must be provided |
| secret\_name | `STRING` |  |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| secret |  | Secret metadata (does not include encrypted value) |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/gateway/secrets/list` | `GET` |

List all secrets with optional filtering by provider

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| provider | `STRING` | Optional filter by provider (e.g., “openai”, “anthropic”) |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| secrets | An array of | List of secret metadata (does not include encrypted values) |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/gateway/endpoints/models/attach` | `POST` |

Attach an existing model definition to an endpoint

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| endpoint\_id | `STRING` | ID of the endpoint to attach the model to |
| model\_definition\_id | `STRING` | ID of the model definition to attach |
| weight | `FLOAT` | Optional routing weight (default 1) |
| created\_by | `STRING` | Username of the creator |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| mapping |  | The created mapping |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/gateway/endpoints/models/detach` | `POST` |

Detach a model definition from an endpoint (does not delete the model definition)

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| endpoint\_id | `STRING` | ID of the endpoint |
| model\_definition\_id | `STRING` | ID of the model definition to detach |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/gateway/endpoints/set-tag` | `POST` |

Set a tag on an endpoint

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| endpoint\_id | `STRING` | ID of the endpoint to set tag on |
| key | `STRING` | Tag key to set |
| value | `STRING` | Tag value to set |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/gateway/endpoints/delete-tag` | `DELETE` |

Delete a tag from an endpoint

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| endpoint\_id | `STRING` | ID of the endpoint to delete tag from |
| key | `STRING` | Tag key to delete |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/registered-models/create` | `POST` |

Throws `RESOURCE_ALREADY_EXISTS` if a registered model with the given name exists.

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| name | `STRING` | Register models under this name This field is required. |
| tags | An array of | Additional metadata for registered model. |
| description | `STRING` | Optional description for registered model. |
| deployment\_job\_id | `STRING` | Deployment job id for this model. |

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/registered-models/get` | `GET` |

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/registered-models/rename` | `POST` |

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/registered-models/update` | `PATCH` |

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| name | `STRING` | Registered model unique name identifier. This field is required. |
| description | `STRING` | If provided, updates the description for this `registered_model`. |
| deployment\_job\_id | `STRING` | Deployment job id for this model. |

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/registered-models/get-latest-versions` | `POST` |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| model\_versions | An array of | Latest version models for each requests stage. Only return models with current `READY` status. If no `stages` provided, returns the latest version for each stage, including `"None"`. |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/model-versions/create` | `POST` |

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| name | `STRING` | Register model under this name This field is required. |
| source | `STRING` | URI indicating the location of the model artifacts. This field is required. |
| run\_id | `STRING` | MLflow run ID for correlation, if `source` was generated by an experiment run in MLflow tracking server |
| tags | An array of | Additional metadata for model version. |
| run\_link | `STRING` | MLflow run link - this is the exact link of the run that generated this model version, potentially hosted at another instance of MLflow. |
| description | `STRING` | Optional description for model version. |
| model\_id | `STRING` | Optional model\_id for model version that is used to link the registered model to the source logged model |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| model\_version |  | Return new version number generated for this model in registry. |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/model-versions/get` | `GET` |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| model\_version |  |  |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/model-versions/update` | `PATCH` |

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| name | `STRING` | Name of the registered model This field is required. |
| version | `STRING` | Model version number This field is required. |
| description | `STRING` | If provided, updates the description for this `registered_model`. |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| model\_version |  | Return new version number generated for this model in registry. |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/model-versions/delete` | `DELETE` |

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/model-versions/search` | `GET` |

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| filter | `STRING` | String filter condition, like “name=’my-model-name’”. Must be a single boolean condition, with string values wrapped in single quotes. |
| max\_results | `INT64` | Maximum number of models desired. Max threshold is 200K. Backends may choose a lower default value and maximum threshold. |
| order\_by | An array of `STRING` | List of columns to be ordered by including model name, version, stage with an optional “DESC” or “ASC” annotation, where “ASC” is the default. Tiebreaks are done by latest stage transition timestamp, followed by name ASC, followed by version DESC. |
| page\_token | `STRING` | Pagination token to go to next page based on previous search query. |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| model\_versions | An array of | Models that match the search criteria |
| next\_page\_token | `STRING` | Pagination token to request next page of models for the same search query. |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/model-versions/get-download-uri` | `GET` |

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/model-versions/transition-stage` | `POST` |

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| name | `STRING` | Name of the registered model This field is required. |
| version | `STRING` | Model version number This field is required. |
| stage | `STRING` | Transition model\_version to new stage. This field is required. |
| archive\_existing\_versions | `BOOL` | When transitioning a model version to a particular stage, this flag dictates whether all existing model versions in that stage should be atomically moved to the “archived” stage. This ensures that at-most-one model version exists in the target stage. This field is *required* when transitioning a model versions’s stage This field is required. |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| model\_version |  | Updated model version |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/registered-models/search` | `GET` |

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| filter | `STRING` | String filter condition, like “name LIKE ‘my-model-name’”. Interpreted in the backend automatically as “name LIKE ‘%my-model-name%’”. Single boolean condition, with string values wrapped in single quotes. |
| max\_results | `INT64` | Maximum number of models desired. Default is 100. Max threshold is 1000. |
| order\_by | An array of `STRING` | List of columns for ordering search results, which can include model name and last updated timestamp with an optional “DESC” or “ASC” annotation, where “ASC” is the default. Tiebreaks are done by model name ASC. |
| page\_token | `STRING` | Pagination token to go to the next page based on a previous search query. |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| registered\_models | An array of | Registered Models that match the search criteria. |
| next\_page\_token | `STRING` | Pagination token to request the next page of models. |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/registered-models/set-tag` | `POST` |

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| name | `STRING` | Unique name of the model. This field is required. |
| key | `STRING` | Name of the tag. Maximum size depends on storage backend. If a tag with this name already exists, its preexisting value will be replaced by the specified value. All storage backends are guaranteed to support key values up to 250 bytes in size. This field is required. |
| value | `STRING` | String value of the tag being logged. Maximum size depends on storage backend. This field is required. |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/model-versions/set-tag` | `POST` |

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| name | `STRING` | Unique name of the model. This field is required. |
| version | `STRING` | Model version number. This field is required. |
| key | `STRING` | Name of the tag. Maximum size depends on storage backend. If a tag with this name already exists, its preexisting value will be replaced by the specified value. All storage backends are guaranteed to support key values up to 250 bytes in size. This field is required. |
| value | `STRING` | String value of the tag being logged. Maximum size depends on storage backend. This field is required. |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/registered-models/delete-tag` | `DELETE` |

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| name | `STRING` | Name of the registered model that the tag was logged under. This field is required. |
| key | `STRING` | Name of the tag. The name must be an exact match; wild-card deletion is not supported. Maximum size is 250 bytes. This field is required. |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/model-versions/delete-tag` | `DELETE` |

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| name | `STRING` | Name of the registered model that the tag was logged under. This field is required. |
| version | `STRING` | Model version number that the tag was logged under. This field is required. |
| key | `STRING` | Name of the tag. The name must be an exact match; wild-card deletion is not supported. Maximum size is 250 bytes. This field is required. |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/registered-models/alias` | `DELETE` |

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| name | `STRING` | Name of the registered model. This field is required. |
| alias | `STRING` | Name of the alias. The name must be an exact match; wild-card deletion is not supported. Maximum size is 256 bytes. This field is required. |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/registered-models/alias` | `GET` |

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| name | `STRING` | Name of the registered model. This field is required. |
| alias | `STRING` | Name of the alias. Maximum size is 256 bytes. This field is required. |

### Response Structure

| Field Name | Type | Description |
| --- | --- | --- |
| model\_version |  |  |

---

| Endpoint | HTTP Method |
| --- | --- |
| `2.0/mlflow/registered-models/alias` | `POST` |

### Request Structure

| Field Name | Type | Description |
| --- | --- | --- |
| name | `STRING` | Name of the registered model. This field is required. |
| alias | `STRING` | Name of the alias. Maximum size depends on storage backend. If an alias with this name already exists, its preexisting value will be replaced by the specified version. All storage backends are guaranteed to support alias name values up to 256 bytes in size. This field is required. |
| version | `STRING` | Model version number. This field is required. |

### Dataset

Dataset. Represents a reference to data used for training, testing, or evaluation during the model development process.

| Field Name | Type | Description |
| --- | --- | --- |
| name | `STRING` | The name of the dataset. E.g. “my.uc.table@2” “nyc-taxi-dataset”, “fantastic-elk-3” This field is required. |
| digest | `STRING` | Dataset digest, e.g. an md5 hash of the dataset that uniquely identifies it within datasets of the same name. This field is required. |
| source\_type | `STRING` | The type of the dataset source, e.g. ‘databricks-uc-table’, ‘DBFS’, ‘S3’, … This field is required. |
| source | `STRING` | Source information for the dataset. Note that the source may not exactly reproduce the dataset if it was transformed / modified before use with MLflow. This field is required. |
| schema | `STRING` | The schema of the dataset. E.g., MLflow ColSpec JSON for a dataframe, MLflow TensorSpec JSON for an ndarray, or another schema format. |
| profile | `STRING` | The profile of the dataset. Summary statistics for the dataset, such as the number of rows in a table, the mean / std / mode of each column in a table, or the number of elements in an array. |

### DatasetInput

DatasetInput. Represents a dataset and input tags.

| Field Name | Type | Description |
| --- | --- | --- |
| tags | An array of | A list of tags for the dataset input, e.g. a “context” tag with value “training” |
| dataset |  | The dataset being used as a Run input. This field is required. |

### Experiment

Experiment

| Field Name | Type | Description |
| --- | --- | --- |
| experiment\_id | `STRING` | Unique identifier for the experiment. |
| name | `STRING` | Human readable name that identifies the experiment. |
| artifact\_location | `STRING` | Location where artifacts for the experiment are stored. |
| lifecycle\_stage | `STRING` | Current life cycle stage of the experiment: “active” or “deleted”. Deleted experiments are not returned by APIs. |
| last\_update\_time | `INT64` | Last update time |
| creation\_time | `INT64` | Creation time |
| tags | An array of | Tags: Additional metadata key-value pairs. |

### ExperimentTag

Tag for an experiment.

| Field Name | Type | Description |
| --- | --- | --- |
| key | `STRING` | The tag key. |
| value | `STRING` | The tag value. |

### FileInfo

| Field Name | Type | Description |
| --- | --- | --- |
| path | `STRING` | Path relative to the root artifact directory run. |
| is\_dir | `BOOL` | Whether the path is a directory. |
| file\_size | `INT64` | Size in bytes. Unset for directories. |

### GatewayEndpoint

Endpoint entity representing an LLM gateway endpoint

| Field Name | Type | Description |
| --- | --- | --- |
| endpoint\_id | `STRING` | Unique identifier for the endpoint |
| name | `STRING` | User-friendly name for the endpoint |
| created\_at | `INT64` | Timestamp (milliseconds since epoch) when the endpoint was created |
| last\_updated\_at | `INT64` | Timestamp (milliseconds since epoch) when the endpoint was last updated |
| model\_mappings | An array of | List of model mappings bound to this endpoint |
| created\_by | `STRING` | User ID who created the endpoint |
| last\_updated\_by | `STRING` | User ID who last updated the endpoint |
| tags | An array of | Tags associated with the endpoint |

### GatewayEndpointBinding

Binding between an endpoint and an MLflow resource. Uses composite key (endpoint\_id, resource\_type, resource\_id) for identification.

| Field Name | Type | Description |
| --- | --- | --- |
| endpoint\_id | `STRING` | ID of the endpoint this binding references |
| resource\_type | `STRING` | Type of MLflow resource (e.g., “scorer\_job”) |
| resource\_id | `STRING` | ID of the specific resource instance |
| created\_at | `INT64` | Timestamp (milliseconds since epoch) when the binding was created |
| last\_updated\_at | `INT64` | Timestamp (milliseconds since epoch) when the binding was last updated |
| created\_by | `STRING` | User ID who created the binding |
| last\_updated\_by | `STRING` | User ID who last updated the binding |

### GatewayEndpointModelMapping

Mapping between an endpoint and a model definition

| Field Name | Type | Description |
| --- | --- | --- |
| mapping\_id | `STRING` | Unique identifier for this mapping |
| endpoint\_id | `STRING` | ID of the endpoint |
| model\_definition\_id | `STRING` | ID of the model definition |
| model\_definition |  | The full model definition (populated via JOIN) |
| weight | `FLOAT` | Routing weight for traffic distribution |
| created\_at | `INT64` | Timestamp (milliseconds since epoch) when the mapping was created |
| created\_by | `STRING` | User ID who created the mapping |

### GatewayEndpointTag

Tag associated with an endpoint

| Field Name | Type | Description |
| --- | --- | --- |
| key | `STRING` | Tag key |
| value | `STRING` | Tag value |

### GatewayModelDefinition

Reusable model definition that can be shared across endpoints

| Field Name | Type | Description |
| --- | --- | --- |
| model\_definition\_id | `STRING` | Unique identifier for this model definition |
| name | `STRING` | User-friendly name for identification and reuse |
| secret\_id | `STRING` | ID of the secret containing authentication credentials |
| secret\_name | `STRING` | Name of the secret for display purposes |
| provider | `STRING` | LLM provider (e.g., “openai”, “anthropic”, “cohere”, “bedrock”) |
| model\_name | `STRING` | Provider-specific model identifier (e.g., “gpt-4o”, “claude-3-5-sonnet”) |
| created\_at | `INT64` | Timestamp (milliseconds since epoch) when the model definition was created |
| last\_updated\_at | `INT64` | Timestamp (milliseconds since epoch) when the model definition was last updated |
| created\_by | `STRING` | User ID who created the model definition |
| last\_updated\_by | `STRING` | User ID who last updated the model definition |

### GatewaySecretInfo

Secret metadata entity (does not include the decrypted secret value)

| Field Name | Type | Description |
| --- | --- | --- |
| secret\_id | `STRING` | Unique identifier for the secret (UUID) |
| secret\_name | `STRING` | User-friendly name for the secret (must be unique) |
| masked\_values | An array of | Masked version of the secret values for display as key-value pairs. For simple API keys: {“api\_key”: “sk-…xyz123”} For compound credentials: `{"aws_access_key_id": "AKI...1234", "aws_secret_access_key": "***"}` |
| created\_at | `INT64` | Timestamp (milliseconds since epoch) when the secret was created |
| last\_updated\_at | `INT64` | Timestamp (milliseconds since epoch) when the secret was last updated |
| provider | `STRING` | LLM provider identifier (e.g., “openai”, “anthropic”, “cohere”) |
| created\_by | `STRING` | User ID who created the secret |
| last\_updated\_by | `STRING` | User ID who last updated the secret |
| auth\_config\_json | `STRING` | Provider-specific auth configuration as JSON (e.g., region, project\_id) |

### InputTag

Tag for an input.

| Field Name | Type | Description |
| --- | --- | --- |
| key | `STRING` | The tag key. This field is required. |
| value | `STRING` | The tag value. This field is required. |

### MaskedValuesEntry

| Field Name | Type | Description |
| --- | --- | --- |
| key | `STRING` |  |
| value | `STRING` |  |

### Metric

Metric associated with a run, represented as a key-value pair.

| Field Name | Type | Description |
| --- | --- | --- |
| key | `STRING` | Key identifying this metric. |
| value | `DOUBLE` | Value associated with this metric. |
| timestamp | `INT64` | The timestamp at which this metric was recorded. |
| step | `INT64` | Step at which to log the metric. |

### ModelMetric

Metric associated with a model, represented as a key-value pair. Copied from MLflow metric

| Field Name | Type | Description |
| --- | --- | --- |
| key | `STRING` | Key identifying this metric. |
| value | `DOUBLE` | Value associated with this metric. |
| timestamp | `INT64` | The timestamp at which this metric was recorded. |
| step | `INT64` | Step at which to log the metric. |

### ModelOutput

Represents a LoggedModel output of a Run.

| Field Name | Type | Description |
| --- | --- | --- |
| model\_id | `STRING` | The unique identifier of the model. This field is required. |
| step | `INT64` | Step at which the model was produced. This field is required. |

### ModelParam

Param for a model version.

| Field Name | Type | Description |
| --- | --- | --- |
| name | `STRING` | Name of the param. |
| value | `STRING` | Value of the param associated with the name, could be empty |

### ModelVersion

| Field Name | Type | Description |
| --- | --- | --- |
| name | `STRING` | Unique name of the model |
| version | `STRING` | Model’s version number. |
| creation\_timestamp | `INT64` | Timestamp recorded when this `model_version` was created. |
| last\_updated\_timestamp | `INT64` | Timestamp recorded when metadata for this `model_version` was last updated. |
| user\_id | `STRING` | User that created this `model_version`. |
| current\_stage | `STRING` | Current stage for this `model_version`. |
| description | `STRING` | Description of this `model_version`. |
| source | `STRING` | URI indicating the location of the source model artifacts, used when creating `model_version` |
| run\_id | `STRING` | MLflow run ID used when creating `model_version`, if `source` was generated by an experiment run stored in MLflow tracking server. |
| status |  | Current status of `model_version` |
| status\_message | `STRING` | Details on current `status`, if it is pending or failed. |
| tags | An array of | Tags: Additional metadata key-value pairs for this `model_version`. |
| run\_link | `STRING` | Run Link: Direct link to the run that generated this version. This field is set at model version creation time only for model versions whose source run is from a tracking server that is different from the registry server. |
| aliases | An array of `STRING` | Aliases pointing to this `model_version`. |
| model\_id | `STRING` | Optional model\_id for model version that is used to link the registered model to the source logged model |
| model\_params | An array of | Optional parameters for the model. |
| model\_metrics | An array of | Optional metrics for the model. |
| deployment\_job\_state |  | Deployment job state for this model version. |

### ModelVersionDeploymentJobState

| Field Name | Type | Description |
| --- | --- | --- |
| job\_id | `STRING` |  |
| run\_id | `STRING` |  |
| job\_state |  |  |
| run\_state |  |  |
| current\_task\_name | `STRING` |  |

### ModelVersionTag

Tag for a model version.

| Field Name | Type | Description |
| --- | --- | --- |
| key | `STRING` | The tag key. |
| value | `STRING` | The tag value. |

### Param

Param associated with a run.

| Field Name | Type | Description |
| --- | --- | --- |
| key | `STRING` | Key identifying this param. |
| value | `STRING` | Value associated with this param. |

### RegisteredModel

| Field Name | Type | Description |
| --- | --- | --- |
| name | `STRING` | Unique name for the model. |
| creation\_timestamp | `INT64` | Timestamp recorded when this `registered_model` was created. |
| last\_updated\_timestamp | `INT64` | Timestamp recorded when metadata for this `registered_model` was last updated. |
| user\_id | `STRING` | User that created this `registered_model` NOTE: this field is not currently returned. |
| description | `STRING` | Description of this `registered_model`. |
| latest\_versions | An array of | Collection of latest model versions for each stage. Only contains models with current `READY` status. |
| tags | An array of | Tags: Additional metadata key-value pairs for this `registered_model`. |
| aliases | An array of | Aliases pointing to model versions associated with this `registered_model`. |
| deployment\_job\_id | `STRING` | Deployment job id for this model. |
| deployment\_job\_state |  | Deployment job state for this model. |

### Run

A single run.

| Field Name | Type | Description |
| --- | --- | --- |
| info |  | Run metadata. |
| data |  | Run data. |
| inputs |  | Run inputs. |
| outputs |  | Run outputs. |

### RunData

Run data (metrics, params, and tags).

| Field Name | Type | Description |
| --- | --- | --- |
| metrics | An array of | Run metrics. |
| params | An array of | Run parameters. |
| tags | An array of | Additional metadata key-value pairs. |

### RunInfo

Metadata of a single run.

| Field Name | Type | Description |
| --- | --- | --- |
| run\_id | `STRING` | Unique identifier for the run. |
| run\_uuid | `STRING` | \[Deprecated, use run\_id instead\] Unique identifier for the run. This field will be removed in a future MLflow version. |
| run\_name | `STRING` | The name of the run. |
| experiment\_id | `STRING` | The experiment ID. |
| user\_id | `STRING` | User who initiated the run. This field is deprecated as of MLflow 1.0, and will be removed in a future MLflow release. Use ‘mlflow.user’ tag instead. |
| status |  | Current status of the run. |
| start\_time | `INT64` | Unix timestamp of when the run started in milliseconds. |
| end\_time | `INT64` | Unix timestamp of when the run ended in milliseconds. |
| artifact\_uri | `STRING` | URI of the directory where artifacts should be uploaded. This can be a local path (starting with “/”), or a distributed file system (DFS) path, like `s3://bucket/directory` or `dbfs:/my/directory`. If not set, the local `./mlruns` directory is chosen. |
| lifecycle\_stage | `STRING` | Current life cycle stage of the experiment: OneOf(“active”, “deleted”) |

### RunInputs

Run inputs.

| Field Name | Type | Description |
| --- | --- | --- |
| dataset\_inputs | An array of | Dataset inputs to the Run. |
| model\_inputs | An array of | Model inputs to the Run. |

### RunOutputs

Outputs of a Run.

| Field Name | Type | Description |
| --- | --- | --- |
| model\_outputs | An array of | Model outputs of the Run. |

### RunTag

Tag for a run.

| Field Name | Type | Description |
| --- | --- | --- |
| key | `STRING` | The tag key. |
| value | `STRING` | The tag value. |

### Scorer

Scorer entity representing a scorer in the database.

| Field Name | Type | Description |
| --- | --- | --- |
| experiment\_id | `INT32` | The experiment ID. |
| scorer\_name | `STRING` | The scorer name. |
| scorer\_version | `INT32` | The scorer version. |
| serialized\_scorer | `STRING` | The serialized scorer string. |
| creation\_time | `INT64` | The creation time of the scorer version (in milliseconds since epoch). |
| scorer\_id | `STRING` | The unique identifier for the scorer. |

### SecretValueEntry

| Field Name | Type | Description |
| --- | --- | --- |
| key | `STRING` |  |
| value | `STRING` |  |

### SecretValueEntry

| Field Name | Type | Description |
| --- | --- | --- |
| key | `STRING` |  |
| value | `STRING` |  |

### DeploymentJobRunState

| Name | Description |
| --- | --- |
| DEPLOYMENT\_JOB\_RUN\_STATE\_UNSPECIFIED |  |
| NO\_VALID\_DEPLOYMENT\_JOB\_FOUND |  |
| RUNNING |  |
| SUCCEEDED |  |
| FAILED |  |
| PENDING |  |
| APPROVAL |  |

### ModelVersionStatus

| Name | Description |
| --- | --- |
| PENDING\_REGISTRATION | Request to register a new model version is pending as server performs background tasks. |
| FAILED\_REGISTRATION | Request to register a new model version has failed. |
| READY | Model version is ready for use. |

### RunStatus

Status of a run.

| Name | Description |
| --- | --- |
| RUNNING | Run has been initiated. |
| SCHEDULED | Run is scheduled to run at a later time. |
| FINISHED | Run has completed. |
| FAILED | Run execution failed. |
| KILLED | Run killed by user. |

### State

| Name | Description |
| --- | --- |
| DEPLOYMENT\_JOB\_CONNECTION\_STATE\_UNSPECIFIED |  |
| NOT\_SET\_UP | default state |
| CONNECTED | connected job: job exists, owner has ACLs, and required job parameters are present |
| NOT\_FOUND | job was deleted OR owner had job ACLs removed |
| REQUIRED\_PARAMETERS\_CHANGED | required job parameters were changed |

### ViewType

View type for ListExperiments query.

| Name | Description |
| --- | --- |
| ACTIVE\_ONLY | Default. Return only active experiments. |
| DELETED\_ONLY | Return only deleted experiments. |
| ALL | Get all experiments. |
