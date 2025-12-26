// Package mlflowclient provides a thin REST client for the MLflow API.
//
// This client uses protojson marshaling with generated proto types from
// the MLflow API definitions. It supports both legacy v2 endpoints
// (experiments, runs, metrics) and modern v3 endpoints (traces, assessments, scorers).
//
// # Basic Usage
//
//	client := mlflowclient.New("http://localhost:5000",
//	    mlflowclient.WithToken("your-token"),
//	    mlflowclient.WithTimeout(60*time.Second),
//	)
//
//	// Create an experiment
//	expID, err := client.CreateExperiment(ctx, "my-experiment", nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Create a run
//	run, err := client.CreateRun(ctx, expID, mlflowclient.WithRunName("my-run"))
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Start a trace
//	trace := &mlflowclient.Trace{
//	    TraceInfoV3: &mlflow.TraceInfoV3{
//	        TraceId: stringPtr("tr-123"),
//	        State:   mlflow.TraceInfoV3_IN_PROGRESS.Enum(),
//	    },
//	}
//	result, err := client.StartTrace(ctx, trace)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # API Versioning
//
// The client uses dual API version strategy:
//   - `/api/2.0/mlflow/` - Legacy endpoints (experiments, runs, metrics)
//   - `/api/3.0/mlflow/` - Modern endpoints (traces, assessments, scorers)
//
// # Error Handling
//
// The client provides structured error types:
//   - APIError: HTTP errors from the MLflow API (4xx, 5xx)
//   - ValidationError: Client-side validation failures
//   - TimeoutError: Request timeout errors
//   - ConnectionError: Network-level failures
//
// Example:
//
//	_, err := client.GetExperiment(ctx, "non-existent")
//	if apiErr, ok := err.(*mlflowclient.APIError); ok {
//	    if apiErr.IsNotFound() {
//	        log.Println("Experiment not found")
//	    }
//	}
//
// # Retry Logic
//
// The client automatically retries failed requests with exponential backoff:
//   - Maximum retries: 3 (configurable)
//   - Initial delay: 1 second
//   - Backoff multiplier: 2x (exponential)
//   - Maximum delay: 10 seconds
//   - Jitter: ±10% randomization
//
// Retryable errors include:
//   - 5xx server errors
//   - 429 rate limit errors
//   - Connection errors
//
// # Authentication
//
// The client supports Bearer token authentication:
//
//	client := mlflowclient.New("http://localhost:5000",
//	    mlflowclient.WithToken("your-mlflow-token"),
//	)
package mlflowclient
