package bucket

import (
	"fmt"
	"strconv"
	"strings"
)

var MasterBucketRange = "0-100"
var ZeroBucketRange = "0-0"

// DefineBucketRange returns a bucket range like this : "0:20-30:40"
func EncodeBucketRangeString(bucketsList []string, bucketRangeIDs []string) string {
	if len(bucketsList) == 0 {
		return MasterBucketRange
	} else if len(bucketRangeIDs) == 0 {
		return ZeroBucketRange
	}

	bucketRanges := []string{}
	for _, cb := range bucketRangeIDs {
		for i, b := range bucketsList {
			if b == cb {
				rangeStart := float64(i) / float64(len(bucketsList))
				rangeEnd := float64(i+1) / float64(len(bucketsList))
				bucketRanges = append(bucketRanges, fmt.Sprintf("%v-%v", rangeStart*100, rangeEnd*100))
			}
		}
	}
	return strings.Join(bucketRanges, ":")
}

// DefineBucketRange decode a bucket range string into a [][]int value
func DecodeBucketRangeString(bucketString string) [][]float64 {
	ret := [][]float64{}
	bucketRanges := strings.Split(bucketString, ":")
	for _, bucketRange := range bucketRanges {
		bucketOffset := strings.Split(bucketRange, "-")
		if len(bucketOffset) != 2 {
			continue
		}
		start, _ := strconv.ParseFloat(bucketOffset[0], 64)
		end, _ := strconv.ParseFloat(bucketOffset[1], 64)
		ret = append(ret, []float64{start, end})
	}
	// if no range, return default range
	if len(ret) == 0 {
		ret = append(ret, []float64{0., 100.})
	}
	return ret
}
