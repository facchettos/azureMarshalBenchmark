# azureMarshalBenchmark

```

goos: linux
goarch: amd64
pkg: vmtest
cpu: Intel(R) Core(TM) i7-10850H CPU @ 2.70GHz
BenchmarkGetAndMarshal-12     	      12	  95639134 ns/op	   43536 B/op	     449 allocs/op
BenchmarkCurrentVersion-12    	      12	 117216317 ns/op	  351987 B/op	    4906 allocs/op
BenchmarkMarshalOnlyNew-12    	   44630	     28621 ns/op	    9760 B/op	     165 allocs/op
BenchmarkMarshalOnlyOld-12    	     888	   1439959 ns/op	  312638 B/op	    4630 allocs/op
PASS
ok  	vmtest	7.628s
```
