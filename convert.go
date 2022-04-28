package httpExporter

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// SpanData contains all the properties of the span.
type SpanData struct {
	TraceID                       string                    `json:"traceId"` // A unique identifier for the trace
	SpanID                        string                    `json:"spanId"`  // A unique identifier for a span within a trace
	ParentSpanID                  string                    `json:"parentSpanId"`
	Name                          string                    `json:"name"`                   // A description of the spans operation
	StartTime                     int64                     `json:"startTime"`              // Start time of the span
	EndTime                       int64                     `json:"endTime"`                // End time of the span
	Attrs                         map[attribute.Key]interface{} `json:"attrs"`                  // A collection of key-value pairs
	DroppedAttributeCount         int                       `json:"droppedAttributesCount"` // Number of attributes that were dropped due to reasons like too many attributes
	Links                         []Link                    `json:"links,omitempty"`
	DroppedLinkCount              int                       `json:"droppedLinkCount"`
	StatusCode                    string                    `json:"statusCode"` // Status code of the span. Defaults to unset
	MessageEvents                 []Event                   `json:"messageEvents,omitempty"`
	DroppedMessageEventCount      int                       `json:"droppedMessageEventCount"`
	SpanKind                      trace.SpanKind            `json:"spanKind"`                   // Type of span
	StatusMessage                 string                    `json:"statusMessage"`              // Human readable error message
	InstrumentationLibraryName    string                    `json:"instrumentationLibraryName"` // Instrumentation library used to provide instrumentation
	InstrumentationLibraryVersion string                    `json:"instrumentationLibraryVersion"`
	Resource                      map[attribute.Key]interface{} `json:"resource,omitempty"` // Contains attributes representing an entity that produced this span
}

// An event is a time-stamped annotation of the span that has user supplied text description and key-value pairs
type Event struct {
	Ts    int64                     `json:"ts"`    // The time at which the event occurred
	Name  string                    `json:"name"`  // Event name
	Attrs map[attribute.Key]interface{} `json:"attrs"` // collection of key-value pairs on the event
}

// A link contains references from this span to a span in the same or different trace
type Link struct {
	TraceID string                    `json:"traceId"`
	SpanID  string                    `json:"spanId"`
	Attrs   map[attribute.Key]interface{} `json:"attrs"`
}

func convertSpansToHttp(spans []sdktrace.ReadOnlySpan) []SpanData{
	httpSpans := []SpanData{}
	for _, span := range spans{
		httpSpan := SpanData{}
		httpSpan.TraceID = span.SpanContext().TraceID().String()
		httpSpan.SpanID = span.SpanContext().SpanID().String()
		httpSpan.ParentSpanID = span.Parent().SpanID().String()
		httpSpan.SpanKind = span.SpanKind()
		httpSpan.Name = span.Name()
		httpSpan.StatusMessage = span.Status().Description
		httpSpan.StatusCode = span.Status().Code.String()
		httpSpan.StartTime = span.StartTime().UnixNano()
		httpSpan.EndTime = span.EndTime().UnixNano()
		httpSpan.InstrumentationLibraryName = span.InstrumentationLibrary().Name
		httpSpan.InstrumentationLibraryVersion = span.InstrumentationLibrary().Version
		httpSpan.Resource = attributesToMap(span.Resource().Attributes())

		httpSpan.MessageEvents = eventsToSlice(span.Events())
		httpSpan.Attrs = attributesToMap(span.Attributes())
		httpSpan.Links = linksToSlice(span.Links())
		httpSpans = append(httpSpans, httpSpan)
	}
	return httpSpans
}


// attributesToMap converts attributes from a slice of key-values to a map for exporting
func attributesToMap(attributes []attribute.KeyValue) map[attribute.Key]interface{} {
	attrs := make(map[attribute.Key]interface{})
	for _, v := range attributes {
		attrs[v.Key] = v.Value.AsInterface()
	}
	return attrs
}

// linksToSlice converts links from the format []trace.Link to []Link for exporting
func linksToSlice(links []sdktrace.Link) []Link {
	var l []Link
	for _, v := range links {
		temp := Link{
			TraceID: v.SpanContext.TraceID().String(),
			SpanID:  v.SpanContext.SpanID().String(),
			Attrs:   attributesToMap(v.Attributes),
		}
		l = append(l, temp)
	}
	return l
}

// eventsToSlice converts events from the format []trace.Event to []Event for exporting
func eventsToSlice(events []sdktrace.Event) []Event {
	var e []Event
	for _, v := range events {
		temp := Event{
			Ts:    v.Time.UnixNano(),
			Name:  v.Name,
			Attrs: attributesToMap(v.Attributes),
		}
		e = append(e, temp)
	}
	return e
}
