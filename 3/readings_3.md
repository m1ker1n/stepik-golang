Рефлексия и кодогенерация:
* https://blog.golang.org/laws-of-reflection
* https://habrahabr.ru/post/269887/
* https://golang.org/src/go/ast/example_test.go
* https://github.com/golang/tools/blob/master/cmd/stringer/stringer.go
* https://golang.org/pkg/reflect/
* http://blog.burntsushi.net/type-parametric-functions-golang/
* https://habrahabr.ru/post/269887/
* https://medium.com/kokster/go-reflection-creating-objects-from-types-part-i-primitive-types-6119e3737f5d
* https://medium.com/kokster/go-reflection-creating-objects-from-types-part-ii-composite-types-69a0e8134f20

Производительность:

Материалы на русском:
* https://habrahabr.ru/company/badoo/blog/301990/
* https://habrahabr.ru/company/badoo/blog/324682/
* https://habrahabr.ru/company/badoo/blog/332636/
* https://habrahabr.ru/company/mailru/blog/331784/ - статья про то как Почта@Mail.ru держит 3 миллиона вебсокет-соединений

Материалы на английском:
* https://blog.golang.org/profiling-go-programs
* https://about.sourcegraph.com/go/an-introduction-to-go-tool-trace-rhys-hiltner/ - большая статья, посвященная go tool trace
* https://www.goinggo.net/2017/05/language-mechanics-on-stacks-and-pointers.html
* https://www.rzaluska.com/blog/important-go-interfaces/
* https://docs.google.com/document/d/1CxgUBPlx9iJzkz9JWkb6tIpTe5q32QDmz8l0BouG0Cw/preview
* https://segment.com/blog/allocation-efficiency-in-high-performance-go-services/
* https://lwn.net/Articles/250967/ - не про го, но знать полезно
* https://github.com/golang/go/wiki/Performance - много про то что можно вытащить из pprof-а
* https://golang.org/doc/gdb
* https://about.sourcegraph.com/go/advanced-testing-in-go/
* https://about.sourcegraph.com/go/generating-better-machine-code-with-ssa/
* https://about.sourcegraph.com/go/evolutionary-optimization-peter-bourgon/
* https://signalfx.com/blog/a-pattern-for-optimizing-go-2/
* http://go-talks.appspot.com/github.com/davecheney/presentations/performance-without-the-event-loop.slide#1
* https://dave.cheney.net/2013/06/30/how-to-write-benchmarks-in-go
* https://dave.cheney.net/2014/06/07/five-things-that-make-go-fast - вообще в блоге Дейва очень много полезной инфы по го
* https://github.com/dgryski/go-perfbook/blob/master/performance.md
* https://www.youtube.com/watch?v=NS1hmEWv4Ac - Make your Go go faster! Optimising performance through reducing memory allocations + слайды * https://fosdem.org/2018/schedule/event/faster/attachments/slides/2510/export/events/attachments/faster/slides/2510/BryanBorehamGoOptimisation.pdf
* https://www.youtube.com/watch?v=N3PWzBeLX2M - Profiling and Optimizing Go
* https://www.youtube.com/watch?v=Lxt8Vqn4JiQ - Golang UK Conference 2017 | Filippo Valsorda - Fighting latency: the CPU profiler is not your ally
* https://www.youtube.com/watch?v=ydWFpcoYraU - Finding Memory Leaks in Go Programs
* http://www.integralist.co.uk/posts/profiling-go/
* https://bravenewgeek.com/so-you-wanna-go-fast/


Полезные инструменты:
* https://mholt.github.io/json-to-go - позволяет по json сформировать структуру на go, в которую он может быть распакован
* https://github.com/mailru/easyjson - кодогенератор для json от mail.ru
