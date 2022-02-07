package bucket

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeBucketRangeString(t *testing.T) {
	// 10 buckets
	buckets := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
	assert.Equal(t, "0-10:20-30", EncodeBucketRangeString(buckets, []string{"a", "c"}))
	assert.Equal(t, "80-90:90-100", EncodeBucketRangeString(buckets, []string{"i", "j"}))
	assert.Equal(t, "10-20:50-60:80-90", EncodeBucketRangeString(buckets, []string{"b", "f", "i"}))

	// 5 buckets
	buckets = []string{"a", "b", "c", "d", "e"}
	assert.Equal(t, "0-20:40-60", EncodeBucketRangeString(buckets, []string{"a", "c"}))

	// 3 buckets
	buckets = []string{"a", "b", "c"}
	assert.Equal(t, "33.33333333333333-66.66666666666666", EncodeBucketRangeString(buckets, []string{"b"}))
	assert.Equal(t, "0-0", EncodeBucketRangeString(buckets, []string{}))

	// 0 buckets
	buckets = []string{}
	assert.Equal(t, "0-100", EncodeBucketRangeString(buckets, []string{}))
}

func TestDecodeBucketRangeString(t *testing.T) {
	assert.Equal(t, [][]float64{{33.33333333333333, 66.66666666666666}}, DecodeBucketRangeString("33.33333333333333-66.66666666666666"))
	assert.Equal(t, [][]float64{{10., 20.}, {50., 60.}, {80., 90.}}, DecodeBucketRangeString("10-20:50-60:80-90"))
	assert.Equal(t, [][]float64{{0., 0.}}, DecodeBucketRangeString("0-0"))
	assert.Equal(t, [][]float64{{0., 100.}}, DecodeBucketRangeString(""))
}
