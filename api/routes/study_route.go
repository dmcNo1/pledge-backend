package routes

import (
	"pledge-backend/api/controllers"

	"github.com/gin-gonic/gin"
)

func InitStudyRoute(e *gin.Engine) {
	ethRouter := e.Group("/eth")
	{
		controller := controllers.StudyController{}
		ethRouter.GET("/tx/:tx_hash", controller.GetTxMsg)
		ethRouter.GET("/receipt/:tx_hash", controller.GetReceipt)
		ethRouter.GET("/block/:block_num", controller.GetBlock)
	}
}
