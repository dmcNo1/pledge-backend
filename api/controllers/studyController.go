package controllers

import (
	"pledge-backend/api/common/statecode"
	"pledge-backend/api/models/request"
	"pledge-backend/api/models/response"
	"pledge-backend/api/services"
	"pledge-backend/api/validate"

	"github.com/gin-gonic/gin"
)

type StudyController struct {
}

// /eth/tx/{tx_hash}
// 获取交易数据
func (c *StudyController) GetTxMsg(ctx *gin.Context) {
	// 初始化response
	response := response.Gin{Res: ctx}

	// 参数验证
	txHash := ctx.Param("tx_hash")
	if txHash == "" {
		response.Response(ctx, statecode.ParameterEmptyErr, nil)
		return
	}

	ethService := services.NewEthService()
	txMsg, returnCode := ethService.GetTxMsg(txHash)
	if returnCode != statecode.CommonSuccess {
		response.Response(ctx, returnCode, nil)
		return
	}

	response.Response(ctx, statecode.CommonSuccess, txMsg)
}

// /eth/tx_receipt/{tx_hash}
// 获取交易数据
func (c *StudyController) GetReceipt(ctx *gin.Context) {
	// 初始化response
	response := response.Gin{Res: ctx}

	// 参数验证
	txHash := ctx.Param("tx_hash")
	if txHash == "" {
		response.Response(ctx, statecode.ParameterEmptyErr, nil)
		return
	}

	ethService := services.NewEthService()
	receipt, returnCode := ethService.GetReceipt(txHash)
	if returnCode != statecode.CommonSuccess {
		response.Response(ctx, returnCode, nil)
		return
	}

	response.Response(ctx, statecode.CommonSuccess, receipt)
}

// /eth/block/:block_num?full=true
func (c *StudyController) GetBlock(ctx *gin.Context) {
	response := response.Gin{Res: ctx}

	// 验证参数，并封装成param
	blockParam := request.Block{}
	returnCode := validate.NewBlock().Block(ctx, &blockParam)
	if returnCode != statecode.CommonSuccess {
		response.Response(ctx, returnCode, nil)
		return
	}

	ethService := services.NewEthService()
	block, returnCode := ethService.GetBlock(&blockParam)
	if statecode.CommonSuccess != returnCode {
		response.Response(ctx, returnCode, nil)
		return
	}

	response.Response(ctx, statecode.CommonSuccess, block)
}
