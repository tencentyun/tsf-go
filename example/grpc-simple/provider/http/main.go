package main

import "github.com/gin-gonic/gin"

func main() {
	engine := gin.New()
	engine.POST("/tsf.test.helloworld.Greeter/SayHello", func(c *gin.Context) {
		var req struct {
			Name string `json:"name"`
		}
		err := c.Bind(&req)
		if err != nil {
			c.JSON(500, map[string]string{"status": "down"})
		}
		c.JSON(200, map[string]string{"message": "hello " + req.Name + "!"})
	})
	engine.Run(":8080")
}
