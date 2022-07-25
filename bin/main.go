package main

import (
	"abbserver/src/abbserver"
	"flag"
)

func main() {
	isPostgres := flag.Bool("ps", false, 
		"using postgres instead local database")
	flag.Parse()
	abbserver.Connect(*isPostgres)
}