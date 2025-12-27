// Package mlflowclient provides a REST client for MLflow API operations.
//
// The client supports both MLflow API v2 (experiments, runs) and v3 (traces, assessments, scorers).
//
// Basic usage:
//
//	client, err := mlflowclient.New("http://localhost:5000")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Create an experiment
//	experimentID, err := client.CreateExperiment(ctx, "my-experiment")
//
//	// Upload a trace
//	trace := &mlflowclient.Trace{...}
//	traceID, err := client.StartTrace(ctx, trace)
//
// Authentication:
//
//	client, err := mlflowclient.New("http://localhost:5000",
//	    mlflowclient.WithToken("your-token"),
//	)
//
// Custom timeout and retries:
//
//	client, err := mlflowclient.New("http://localhost:5000",
//	    mlflowclient.WithTimeout(60 * time.Second),
//	    mlflowclient.WithRetries(5),
//	)
package mlflowclient
