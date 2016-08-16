package main

import (
	"fmt"
	"github.com/gamunu/tensor/util"
	"github.com/gin-gonic/gin"
	"github.com/gamunu/tensor/tensorui/support"
)

func main() {
	fmt.Printf("Tensor : %v\n", util.Version)
	fmt.Printf("Port : %v\n", util.Config.UiPort)

	r := gin.New()
	r.Use(gin.Recovery(), recovery, gin.Logger())

	support.Route(r)

	r.Run(util.Config.UiPort)

}

func recovery(c *gin.Context) {

	//report to bug nofiy system
	c.Next()
}