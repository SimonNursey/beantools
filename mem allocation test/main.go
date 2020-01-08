package main

import (
	"fmt"
	"time"
)

func main() {
	var dat []byte //:= make([]byte, 1e3)
	t := time.Now()
	for i := 1; i < 1000; i++ {
		dat = append(dat, make([]byte, 1e3)...)
		//		dat2 := make([]byte, i*1e3)
		//		copy(dat2, dat)
		//		dat = dat2
	}
	fmt.Printf("%v Took %f s\n", len(dat), time.Now().Sub(t).Seconds())
}
