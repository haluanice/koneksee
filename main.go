// main
package main

import "runtime"

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	Routes()
	Run()
}
