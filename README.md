### HTTP exporter 

This exporter lets you export OpenTelemetry traces to an http endpoint.

To use it, register the exporter with the span processor and globally set the trace provider initialized with this span processor. The following snippet should be sufficient to add the exporter, it will take care of exporting the traces once you instrument the code.

```go
func initTracer(url string) func() {
exporter, err := httpExporter.New(
		url,
		httpExporter.WithLogger(logger),
	)
	if err != nil {
		log.Fatal(err)
	}

	batcher := sdktrace.NewBatchSpanProcessor(exporter)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(batcher),
	)
	otel.SetTracerProvider(tp)

	return func() {
		_ = tp.Shutdown(context.Background())
	}
}
```
