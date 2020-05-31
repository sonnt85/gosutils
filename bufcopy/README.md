# go-bufcopy
golang copy io stream optimised by using sync.Pool

# Benchmark
```
BenchmarkBufCopy-12    	 5000000	       430 ns/op	    3168 B/op	       3 allocs/op
BenchmarkIoCopy-12     	 3000000	       433 ns/op	    3168 B/op	       3 allocs/op
```
