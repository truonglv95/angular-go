package sourcemap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompareSegments(t *testing.T) {
	// Should return 0 if segments are the same
	assert.Equal(t, 0, CompareSegments(
		&SegmentMarker{Line: 0, Column: 0, Position: 0},
		&SegmentMarker{Line: 0, Column: 0, Position: 0},
	))
	assert.Equal(t, 0, CompareSegments(
		&SegmentMarker{Line: 123, Column: 0, Position: 200},
		&SegmentMarker{Line: 123, Column: 0, Position: 200},
	))

	// Should return negative if first is before second
	assert.True(t, CompareSegments(
		&SegmentMarker{Line: 0, Column: 0, Position: 0},
		&SegmentMarker{Line: 0, Column: 45, Position: 45},
	) < 0)
	assert.True(t, CompareSegments(
		&SegmentMarker{Line: 13, Column: 45, Position: 75},
		&SegmentMarker{Line: 123, Column: 45, Position: 245},
	) < 0)

	// Should return positive if first is after second
	assert.True(t, CompareSegments(
		&SegmentMarker{Line: 0, Column: 45, Position: 45},
		&SegmentMarker{Line: 0, Column: 0, Position: 0},
	) > 0)
	assert.True(t, CompareSegments(
		&SegmentMarker{Line: 123, Column: 45, Position: 245},
		&SegmentMarker{Line: 13, Column: 45, Position: 75},
	) > 0)
}

func TestOffsetSegment(t *testing.T) {
	startOfLinePositions := ComputeStartOfLinePositions("012345\n0123456789\r\n012*4567\n0123456")
	marker := &SegmentMarker{Line: 2, Column: 3, Position: 22}

	// Should return identical marker if offset is 0
	assert.Equal(t, marker, OffsetSegment(startOfLinePositions, marker, 0))

	// Should return a new marker offset by given chars
	assert.Equal(t, &SegmentMarker{Line: 2, Column: 4, Position: 23}, OffsetSegment(startOfLinePositions, marker, 1))
	assert.Equal(t, &SegmentMarker{Line: 2, Column: 5, Position: 24}, OffsetSegment(startOfLinePositions, marker, 2))
	assert.Equal(t, &SegmentMarker{Line: 2, Column: 7, Position: 26}, OffsetSegment(startOfLinePositions, marker, 4))
	assert.Equal(t, &SegmentMarker{Line: 3, Column: 0, Position: 28}, OffsetSegment(startOfLinePositions, marker, 6))
	assert.Equal(t, &SegmentMarker{Line: 3, Column: 2, Position: 30}, OffsetSegment(startOfLinePositions, marker, 8))
	assert.Equal(t, &SegmentMarker{Line: 2, Column: 2, Position: 21}, OffsetSegment(startOfLinePositions, marker, -1))
	assert.Equal(t, &SegmentMarker{Line: 2, Column: 1, Position: 20}, OffsetSegment(startOfLinePositions, marker, -2))
	assert.Equal(t, &SegmentMarker{Line: 2, Column: 0, Position: 19}, OffsetSegment(startOfLinePositions, marker, -3))
	assert.Equal(t, &SegmentMarker{Line: 1, Column: 11, Position: 18}, OffsetSegment(startOfLinePositions, marker, -4))
	assert.Equal(t, &SegmentMarker{Line: 1, Column: 9, Position: 16}, OffsetSegment(startOfLinePositions, marker, -6))
}
