// -*- coding: utf-8-unix -*-
package main

import (
	"code.google.com/p/go-uuid/uuid"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/kumagi/http_jubatus/process"
	"os"
)

type BootJSON struct {
	Name  string      `json:"name"			binding:"required"`
	Param interface{} `json:"parameter" binding:"required"`
}

type JubatusServer struct {
	Filepath string
	Proc     process.JubatusProcess
}

func (j *JubatusServer) Call(module string, method string, arg []interface{}) (interface{}, error) {
	return j.Proc.Call(module, method, arg)
}

func (j *JubatusServer) Kill() {
	os.Remove(j.Filepath)
}

func NewJubatusServer(jubatype string, arg interface{}) (*JubatusServer, error) {
	jtype := jubatype
	filename := uuid.New() + ".json"
	data, _ := json.Marshal(arg)
	filepath := "/tmp/" + filename

	fp, err := os.Create(filepath)
	if err != nil {
		return nil, err
	}
	fp.Write(data)
	fp.Close()
	fmt.Println(arg)

	new_process, err := process.NewJubatusProcess("juba"+jtype, filepath)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return &JubatusServer{filepath, *new_process}, err
}

func main() {
	router := gin.Default()

	servers := make(map[string]map[string]*JubatusServer)
	modules := []string{
		"anomaly",
		"bandit",
		"burst",
		"classifier",
		"clustering",
		"graph",
		"nearest_neighbor",
		"recommender",
		"regression",
		"stat",
	}

	for _, module := range modules {
		local_module := module

		router.POST("/"+local_module, func(c *gin.Context) {
			/*
			 Create new jubatus model
			 Name => unique name of new model
			 Param => jubatus boot parameter passed with -f option
			*/

			fmt.Println("" + local_module)
			var arg BootJSON
			c.Bind(&arg)
			if _, ok := servers[local_module][arg.Name]; ok {
				c.String(409, local_module+"/"+arg.Name+" is already exists\n")
				return
			}
			newServer, err := NewJubatusServer(local_module, arg.Param)
			if err != nil {
				fmt.Println(err)
				c.String(500, err.Error())
				return
			}

			if servers[local_module] == nil {
				servers[local_module] = make(map[string]*JubatusServer)
			}
			servers[local_module][arg.Name] = newServer

			c.String(200, "ok")
		})

		router.POST("/"+local_module+"/:name/:method", func(c *gin.Context) {
			/*
			 Do machine learning
			 you can use Jubatus via HTTP rpc
			*/
			var argument []interface{}
			c.Bind(&argument)

			name := c.Params.ByName("name")
			method := c.Params.ByName("method")

			if server, ok := servers[local_module][name]; ok {
				fmt.Println(argument)
				ret, err := server.Call(local_module, method, argument)
				fmt.Println("return: ", ret, err)
				if err == nil {
					c.JSON(200, gin.H{"result": ret})
				} else {
					c.JSON(500, gin.H{
						"result":  nil,
						"message": err.Error(),
					})
				}
			} else {
				c.String(404, "target "+name+" not found")
			}
		})

		router.GET("/"+local_module, func(c *gin.Context) {
			/*
			 get list of names of machine learning models
			*/
			ret := []string{}
			for _, local_module := range modules {
				for name, _ := range servers[local_module] {
					ret = append(ret, local_module+"/"+name)
				}
			}
			c.JSON(200, gin.H{"servers": ret})
		})

		router.DELETE("/"+local_module+"/:name", func(c *gin.Context) {
			/*
			 delete machine learning model
			*/
			name := c.Params.ByName("name")
			if server, ok := servers[local_module][name]; ok {
				server.Proc.Kill() // kill process
				server.Kill()      // delete .json
				delete(servers[local_module], name)
				c.String(200, "deleted")
			} else {
				c.String(404, "target "+name+" not found")
			}
		})
	}

	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "3000"
	}

	router.Run(":" + port)
}
