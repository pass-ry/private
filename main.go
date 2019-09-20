package main

import (
        "github.com/gin-gonic/gin"
        "net/http"
)

func main () {
        r := gin.Default()
        r.Get(relativePath:"/test",func (c *gin.Context){
                firstName := c.Query(key:"firstName")
                lastName := c.DefaultQuery(key:"lastName",defaultValue:"lastOne")
                c.String(http.StatusOK,format:"%s%s",firstName, lastName)

        })

        r.Run()
}