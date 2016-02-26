package main

import (
	"strconv"
	"sync"
	"testing"
)

func benchmarkStrconvComparison(i int, b *testing.B) {
	var IDMutex sync.Mutex
	highestID := "122222222"

	if i == 0 {
		for n := 0; n < b.N; n++ {
			postIDint, _ := strconv.Atoi("110312919")

			IDMutex.Lock()
			highestIDint, _ := strconv.Atoi(highestID)
			if postIDint >= highestIDint {
				highestID = "110312919"
			}
			IDMutex.Unlock()
		}
	} else {
		b.SetParallelism(i)
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				postIDint, _ := strconv.Atoi("110312919")

				IDMutex.Lock()
				highestIDint, _ := strconv.Atoi(highestID)
				if postIDint >= highestIDint {
					highestID = "110312919"
				}
				IDMutex.Unlock()
			}
		})
	}
}

func benchmarkStrIntComparison(i int, b *testing.B) {
	var IDMutex sync.Mutex
	highestID := "122222222"

	if i == 0 {
		for n := 0; n < b.N; n++ {
			strNew := "132145174"
			IDMutex.Lock()
			if strIntLess(highestID, strNew) {
				highestID = strNew
			}
			IDMutex.Unlock()
		}
	} else {
		b.SetParallelism(i)
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				strNew := "132145174"
				IDMutex.Lock()
				if strIntLess(highestID, strNew) {
					highestID = strNew
				}
				IDMutex.Unlock()
			}
		})
	}

}

// === Benchmarks ===

func BenchmarkStrconvComparisonSingle(b *testing.B) {
	benchmarkStrconvComparison(0, b)
}

func BenchmarkStrconvComparisonParallel1(b *testing.B) {
	benchmarkStrconvComparison(1, b)
}

func BenchmarkStrconvComparisonParallel10(b *testing.B) {
	benchmarkStrconvComparison(10, b)
}

func BenchmarkStringintComparisonSingle(b *testing.B) {
	benchmarkStrIntComparison(0, b)
}

func BenchmarkStringintComparisonParallel1(b *testing.B) {
	benchmarkStrIntComparison(1, b)
}

func BenchmarkStringintComparisonParallel10(b *testing.B) {
	benchmarkStrIntComparison(10, b)
}
