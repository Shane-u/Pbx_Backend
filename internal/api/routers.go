package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"pbx_back_end/repository"
)

// shane: Response
type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"message"`
	Data interface{} `json:"data"`
}

func Routers(r *gin.Engine, repo *repository.Repository) {
	api := r.Group("/api/robots")
	{
		api.POST("/addAssistant", func(c *gin.Context) {
			var robot repository.ChatRobot
			if err := c.ShouldBindJSON(&robot); err != nil {
				c.JSON(http.StatusBadRequest, Response{Code: 0, Msg: "参数错误"})
				return
			}
			if err := repo.CreateRobot(&robot); err != nil {
				c.JSON(http.StatusInternalServerError, Response{Code: 0, Msg: "创建失败"})
				return
			}
			c.JSON(http.StatusOK, Response{Code: 1, Msg: "创建成功"})
		})
		api.POST("deleteAssistant", func(c *gin.Context) {
			var robot repository.ChatRobot
			if err := c.ShouldBindJSON(&robot); err != nil {
				c.JSON(http.StatusBadRequest, Response{Code: 0, Msg: "获取参数错误"})
				return
			}
			if robot.Id == 0 {
				c.JSON(http.StatusBadRequest, Response{Code: 0, Msg: "缺少ID参数"})
				return
			}
			if err := repo.DeleteRobot(robot.Id); err != nil {
				c.JSON(http.StatusInternalServerError, Response{Code: 0, Msg: "删除失败"})
				return
			}
			c.JSON(http.StatusOK, Response{Code: 1, Msg: "删除成功"})
		})

		api.POST("/updateAssistant", func(c *gin.Context) {
			var robot repository.ChatRobot
			if err := c.ShouldBindJSON(&robot); err != nil {
				c.JSON(http.StatusBadRequest, Response{Code: 0, Msg: "参数错误"})
				return
			}
			if err := repo.UpdateRobot(&robot); err != nil {
				c.JSON(http.StatusInternalServerError, Response{Code: 0, Msg: "更新失败"})
			}
			c.JSON(http.StatusOK, Response{Code: 1, Msg: "更新成功"})
		})

		api.GET("/getAssistant", func(c *gin.Context) {
			robots, err := repo.GetAllRobots()
			if err != nil {
				c.JSON(http.StatusInternalServerError, Response{Code: 0, Msg: "查询失败"})
				return
			}
			c.JSON(http.StatusOK, Response{Code: 1, Msg: "查询成功", Data: robots})
		})
	}
}
