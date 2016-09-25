package main

import (
	"io/ioutil"
	"os"
)

func main() {
	err := ioutil.WriteFile(os.Args[1], []byte("hello there"), 0755)
	if err != nil {
		panic(err)
	}
}
