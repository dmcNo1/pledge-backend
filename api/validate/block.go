package validate

import (
	"math/big"
	"pledge-backend/api/common/statecode"
	"pledge-backend/api/models/request"
	"pledge-backend/log"
	"strconv"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/gin-gonic/gin"
)

type Block struct {
}

func NewBlock() *Block {
	return &Block{}
}

func (b *Block) Block(ctx *gin.Context, blockParam *request.Block) int {
	err := ctx.ShouldBind(blockParam)
	if nil != err {
		log.Logger.Error(err.Error())
		return statecode.ParameterEmptyErr
	}

	// 处理参数block_num：nil，查询最新的区块；head、finalized、safe；数字；否则参数无效
	var blockNum *big.Int = nil
	returnCode := statecode.CommonSuccess
	blockNumStr := ctx.Param("block_num")

	switch blockNumStr {
	case "":
		log.Logger.Error("block_num参数为空")
		return statecode.ParameterEmptyErr
	case "nil":
	case "head":
	case "finalized":
		blockNum = big.NewInt(rpc.FinalizedBlockNumber.Int64())
	case "safe":
		blockNum = big.NewInt(rpc.SafeBlockNumber.Int64())
	default:
		// 判断是否为数字
		blockNumInt, err := strconv.Atoi(blockNumStr)
		if err != nil {
			log.Logger.Error("err")
			return statecode.ParameterNotIllegal
		}
		blockNum = big.NewInt(int64(blockNumInt))
	}
	blockParam.BlockNum = blockNum

	return returnCode
}
