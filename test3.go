package main

import (
	"fmt"
	"unsafe"
)

type test struct {
	name string
}

func main() {
	var test1 = test{}
	var test2 *test
	var test3 *test
	var test4 = test{}
	fmt.Println(unsafe.Sizeof(test1))
	fmt.Println(unsafe.Sizeof(test2))
	fmt.Println(unsafe.Sizeof(test3))
	fmt.Println(unsafe.Sizeof(test4))
	fmt.Println(test2 == test3)
	fmt.Println(test1 == test4)
}
