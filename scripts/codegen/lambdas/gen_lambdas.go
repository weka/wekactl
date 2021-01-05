package main

import (
	"fmt"
	. "github.com/dave/jennifer/jen"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

func main() {
	if len(os.Args) != 4 {
		panic("Only 3 arguments supported,0: Buckets map 1:Lambdas ID 2:Output file")
	}

	regionsMap := os.Args[1]
	lambdasId := os.Args[2]
	outputFile := os.Args[3]
	pairs := strings.Split(regionsMap, ",")
	assigments := []Code{}
	assigments = append(assigments, Id("LambdasID").Op("=").Lit(lambdasId))
	for _, pair := range pairs {
		parts := strings.Split(pair, "=")
		region := parts[0]
		bucket := parts[1]
		assigments = append(assigments, Id(fmt.Sprintf("LambdasSource[\"%s\"]", region)).Op("=").Lit(bucket))
	}

	f := NewFile("dist")
	f.HeaderComment("auto-generated with upload_lambdas.sh/gen_lambdas.go code")

	f.Func().Id("init").Params().Block(
		assigments...,
	)

	fmt.Printf(fmt.Sprintf("%#v", f))
	err := ioutil.WriteFile(outputFile, []byte(fmt.Sprintf("%#v", f)), 0660)
	if err != nil {
		log.Fatalln(err)
	}
}
