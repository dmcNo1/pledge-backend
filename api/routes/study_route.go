package routes

import (
	"pledge-backend/api/controllers"
	"pledge-backend/api/middlewares"

	"github.com/gin-gonic/gin"
)

func InitStudyRoute(e *gin.Engine) {
	ethRouter := e.Group("/eth")
	{
		ethRouter.Use((&middlewares.IpRateMiddleware{}).Middleware)
		controller := controllers.StudyController{}
		ethRouter.GET("/block/:block_num", controller.GetBlock)
		ethRouter.GET("/tx/:tx_hash", controller.GetTxMsg)
		ethRouter.GET("/tx_receipt/:tx_hash", controller.GetReceipt)
		ethRouter.GET("/set_item", controller.SetItem)
	}
}
