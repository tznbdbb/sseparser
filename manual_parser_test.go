package sseparser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	defaultTestCapacity = 8192
	customTestCapacity  = 1024
	defaultResizeFactor = 1.5
	customResizeFactor  = 2.0
)

func TestNewManualParser_Default(t *testing.T) {
	parser := NewManualParser()
	assert.NotNil(t, parser, "Parser should not be nil")
	assert.Equal(t, 0, parser.Len(), "Initial length should be 0")
	assert.Equal(t, defaultTestCapacity, parser.Cap(), "Initial capacity should be the default")
	assert.Equal(t, 0, parser.parsedOffset, "Initial parsed offset should be 0")
	assert.Equal(t, 0, parser.lastParseEnd, "Initial last parse end should be 0")
	assert.Equal(t, defaultResizeFactor, parser.resizeFactor, "Default resize factor should be set")
}

func TestNewManualParser_WithCapacity(t *testing.T) {
	parser := NewManualParser(WithCapacity(customTestCapacity))
	assert.NotNil(t, parser)
	assert.Equal(t, customTestCapacity, parser.Cap(), "Capacity should be set to the custom value")
	assert.Equal(t, customTestCapacity, parser.capacity, "Internal capacity field should be updated")

	// Test with invalid capacity
	parser = NewManualParser(WithCapacity(-100))
	assert.Equal(t, defaultTestCapacity, parser.Cap(), "Negative capacity should be ignored, fallback to default")
}

func TestNewManualParser_WithResizeFactor(t *testing.T) {
	parser := NewManualParser(WithResizeFactor(customResizeFactor))
	assert.NotNil(t, parser)
	assert.Equal(t, customResizeFactor, parser.resizeFactor, "Resize factor should be set to the custom value")

	// Test with invalid resize factor
	parser = NewManualParser(WithResizeFactor(0.5))
	assert.Equal(t, defaultResizeFactor, parser.resizeFactor, "Resize factor <= 1 should be ignored, fallback to default")
}

func TestNewManualParser_MultipleOptions(t *testing.T) {
	parser := NewManualParser(
		WithCapacity(customTestCapacity),
		WithResizeFactor(customResizeFactor),
	)
	assert.NotNil(t, parser)
	assert.Equal(t, customTestCapacity, parser.Cap(), "Custom capacity should be set")
	assert.Equal(t, customResizeFactor, parser.resizeFactor, "Custom resize factor should be set")
}

func TestTryParse_EmptyBuffer(t *testing.T) {
	parser := NewManualParser()
	event, err := parser.TryParse()
	assert.Nil(t, event, "Event should be nil for an empty buffer")
	assert.NoError(t, err, "Error should be nil for an empty buffer")
}

func TestTryParse_SingleCompleteEvent(t *testing.T) {
	parser := NewManualParser()
	sseData := []byte("id: 1\ndata: test message\n\n")
	parser.Append(sseData)

	event, err := parser.TryParse()
	require.NoError(t, err)
	require.NotNil(t, event)

	fields := event.Fields()
	require.Len(t, fields, 2)
	assert.Equal(t, "id", fields[0].Name)
	assert.Equal(t, "1", fields[0].Value)
	assert.Equal(t, "data", fields[1].Name)
	assert.Equal(t, "test message", fields[1].Value)

	// TryParse should return the same event when called again (no consumption)
	event2, err := parser.TryParse()
	require.NoError(t, err)
	require.NotNil(t, event2)
	assert.Equal(t, event, event2, "TryParse should return the same event without consuming")

	// After consuming, TryParse should return nil
	consumed := parser.Consume()
	assert.Greater(t, consumed, 0, "Should consume some bytes")

	nextEvent, err := parser.TryParse()
	assert.Nil(t, nextEvent, "There should be no more events after consuming")
	assert.NoError(t, err)
}

func TestTryParse_MultipleEvents(t *testing.T) {
	parser := NewManualParser()
	sseData := []byte("data: first\n\ndata: second\n\n")
	parser.Append(sseData)

	// First event
	event1, err1 := parser.TryParse()
	require.NoError(t, err1)
	require.NotNil(t, event1)
	assert.Equal(t, "data", event1.Fields()[0].Name)
	assert.Equal(t, "first", event1.Fields()[0].Value)

	// Consume first event
	consumed1 := parser.Consume()
	assert.Greater(t, consumed1, 0, "Should consume first event")

	// Second event
	event2, err2 := parser.TryParse()
	require.NoError(t, err2)
	require.NotNil(t, event2)
	assert.Equal(t, "data", event2.Fields()[0].Name)
	assert.Equal(t, "second", event2.Fields()[0].Value)

	// Consume second event
	consumed2 := parser.Consume()
	assert.Greater(t, consumed2, 0, "Should consume second event")

	// No more events
	event3, err3 := parser.TryParse()
	assert.Nil(t, event3)
	assert.NoError(t, err3)
}

func TestTryParse_IncompleteEvent(t *testing.T) {
	parser := NewManualParser()
	parser.Append([]byte("data: incomplete")) // Missing final \n\n

	event, err := parser.TryParse()
	assert.Nil(t, event, "Event should be nil for incomplete data")
	assert.NoError(t, err, "Error should be nil for incomplete data")
	assert.Equal(t, 0, parser.parsedOffset, "Offset should not advance for incomplete data")
	assert.Equal(t, 0, parser.lastParseEnd, "Last parse end should not advance for incomplete data")
}

func TestParse_DirectConsumption(t *testing.T) {
	parser := NewManualParser()
	sseData := []byte("id: 1\ndata: test message\n\n")
	parser.Append(sseData)

	// Parse should consume the event immediately
	event, err := parser.Parse()
	require.NoError(t, err)
	require.NotNil(t, event)

	fields := event.Fields()
	require.Len(t, fields, 2)
	assert.Equal(t, "id", fields[0].Name)
	assert.Equal(t, "1", fields[0].Value)

	// Next TryParse should return nil since event was consumed
	nextEvent, err := parser.TryParse()
	assert.Nil(t, nextEvent, "No more events should be available after Parse")
	assert.NoError(t, err)
}

func TestConsume_WithoutTryParse(t *testing.T) {
	parser := NewManualParser()
	sseData := []byte("id: 1\ndata: test message\n\n")
	parser.Append(sseData)

	// Consume without TryParse should return 0
	consumed := parser.Consume()
	assert.Equal(t, 0, consumed, "Consume without TryParse should return 0")
}

func TestConsume_Multiple(t *testing.T) {
	parser := NewManualParser()
	sseData := []byte("id: 1\ndata: test message\n\n")
	parser.Append(sseData)

	// TryParse first
	event, err := parser.TryParse()
	require.NoError(t, err)
	require.NotNil(t, event)

	// First consume should work
	consumed1 := parser.Consume()
	assert.Greater(t, consumed1, 0, "First consume should work")

	// Second consume should return 0
	consumed2 := parser.Consume()
	assert.Equal(t, 0, consumed2, "Second consume should return 0")
}

func TestCompact_Basic(t *testing.T) {
	parser := NewManualParser()
	event1Data := []byte("data: event1\n\n")
	remainingData := []byte("data: eve")
	parser.Append(event1Data)
	parser.Append(remainingData)

	// Parse and consume the first event
	event, err := parser.Parse()
	require.NoError(t, err)
	require.NotNil(t, event)

	assert.Equal(t, len(event1Data), parser.parsedOffset)
	assert.Equal(t, len(event1Data)+len(remainingData), parser.Len())

	parser.PruneParsedData()

	assert.Equal(t, 0, parser.parsedOffset, "Offset should be reset after compact")
	assert.Equal(t, 0, parser.lastParseEnd, "Last parse end should be reset after compact")
	assert.Equal(t, len(remainingData), parser.Len(), "Length should be size of remaining data")
	assert.Equal(t, string(remainingData), string(parser.buf), "Buffer should contain only the unparsed data")
}

func TestCompact_BufferShrinking(t *testing.T) {
	// Start with a small capacity to easily observe shrinking
	parser := NewManualParser(WithCapacity(16), WithResizeFactor(1.5))

	// Append data to force buffer growth
	largeData := make([]byte, 100)
	parser.Append(largeData)
	assert.True(t, parser.Cap() >= 100, "Capacity should grow to accommodate data")

	// Simulate parsing all data
	parser.parsedOffset = 100
	parser.lastParseEnd = 100

	// Compaction should leave an empty buffer and trigger shrinking
	parser.PruneParsedData()

	assert.Equal(t, 0, parser.Len())
	assert.Equal(t, 0, parser.parsedOffset)
	assert.Equal(t, 0, parser.lastParseEnd)
	// The new capacity should be smaller than the peak capacity
	assert.True(t, parser.Cap() < 100, "Capacity should shrink after compacting a large, fully parsed buffer")
}

func TestTryParse_HandlesIncompleteDataGracefully(t *testing.T) {
	parser := NewManualParser()

	// 1. Append a field without its terminating newline.
	parser.Append([]byte("data: incomplete message"))

	event, err := parser.TryParse()
	assert.Nil(t, event, "Event should be nil for incomplete data")
	assert.NoError(t, err, "Error should be nil for incomplete data")
	assert.Equal(t, 0, parser.parsedOffset, "Offset should not advance for incomplete data")

	// 2. Now, append the rest of the event.
	parser.Append([]byte("\n\n"))
	event, err = parser.TryParse()
	require.NoError(t, err, "Should not error after completing the event")
	require.NotNil(t, event, "Should parse the event after data is complete")

	fields := event.Fields()
	require.Len(t, fields, 1)
	assert.Equal(t, "data", fields[0].Name)
	assert.Equal(t, "incomplete message", fields[0].Value)
}

func TestRealWorldScenario_FragmentedNetwork(t *testing.T) {
	parser := NewManualParser()

	fragments := []string{
		"id: 123\n",
		"data: chunk 1",
		"\ndata: chunk 2\n",
		": this is a comment\n",
		"\n", // End of first event
		"event: custom\n",
		"data: final message\n\n", // End of second event
	}

	// Append first fragment
	parser.Append([]byte(fragments[0]))
	event, err := parser.TryParse()
	assert.Nil(t, event)
	assert.NoError(t, err)

	// Append next fragments until a full event is formed
	parser.Append([]byte(fragments[1]))
	parser.Append([]byte(fragments[2]))
	parser.Append([]byte(fragments[3]))
	parser.Append([]byte(fragments[4]))

	// Now, the first event should be parsable
	event1, err1 := parser.TryParse()
	require.NoError(t, err1)
	require.NotNil(t, event1)
	assert.Len(t, event1.Fields(), 3, "First event should have 3 fields (id, data, data)")
	assert.Len(t, event1.Comments(), 1, "First event should have 1 comment")
	assert.Equal(t, "123", event1.Fields()[0].Value)
	assert.Equal(t, "chunk 1chunk 2", event1.Fields()[1].Value+event1.Fields()[2].Value)

	// Consume the first event
	consumed := parser.Consume()
	assert.Greater(t, consumed, 0, "Should consume first event")

	// Append the rest of the fragments for the second event
	parser.Append([]byte(fragments[5]))
	parser.Append([]byte(fragments[6]))

	// Try to parse second event
	event2, err2 := parser.TryParse()
	require.NoError(t, err2)
	require.NotNil(t, event2)
	fields2 := event2.Fields()
	require.Len(t, fields2, 2)
	assert.Equal(t, "event", fields2[0].Name)
	assert.Equal(t, "custom", fields2[0].Value)
	assert.Equal(t, "data", fields2[1].Name)
	assert.Equal(t, "final message", fields2[1].Value)
}

func TestAuxiliaryMethods(t *testing.T) {
	parser := NewManualParser()
	event1Data := []byte("data: event1\n\n")
	unparsedData := []byte("data: event2-incomplete")
	parser.Append(event1Data)
	parser.Append(unparsedData)

	// State before parsing
	assert.Equal(t, len(event1Data)+len(unparsedData), parser.Len())
	assert.Nil(t, parser.ParsedBytes())
	assert.Equal(t, string(parser.buf), string(parser.UnparsedBytes()))

	// Parse one event (but don't consume)
	event, err := parser.TryParse()
	require.NoError(t, err)
	require.NotNil(t, event)

	// State after parsing but before consuming
	assert.Equal(t, 0, len(parser.ParsedBytes()), "ParsedBytes should return 0 before consuming")
	assert.Equal(t, string(parser.buf), string(parser.UnparsedBytes()))

	// Consume the event
	consumed := parser.Consume()
	assert.Greater(t, consumed, 0, "Should consume some bytes")

	// State after consuming
	assert.Equal(t, len(event1Data), len(parser.ParsedBytes()), "ParsedBytes should return the first event data")
	assert.Equal(t, string(event1Data), string(parser.ParsedBytes()))
	assert.Equal(t, len(unparsedData), len(parser.UnparsedBytes()), "UnparsedBytes should return the remaining data")
	assert.Equal(t, string(unparsedData), string(parser.UnparsedBytes()))

	// State after compaction
	parser.PruneParsedData()
	assert.Equal(t, len(unparsedData), parser.Len())
	assert.Nil(t, parser.ParsedBytes(), "ParsedBytes should be nil after compact")
	assert.Equal(t, string(unparsedData), string(parser.UnparsedBytes()))
}
