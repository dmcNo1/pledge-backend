package services

import (
	"context"
	"math/big"
	"pledge-backend/api/common"
	"pledge-backend/config"
	"pledge-backend/db"
	"pledge-backend/log"
	"pledge-backend/schedule/models"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

type EthService struct {
}

func NewEthService() *EthService {
	return &EthService{}
}

func (s *EthService) GetBlock() {

	client, err := ethclient.Dial(config.Config.TestNet.TestEthUrl)
	if err != nil {
		log.Logger.Error(err.Error())
		return
	}
	defer client.Close()

	// 建立定时任务
	ticker := time.NewTicker(time.Minute * 1)
	// 需要遍历获取的区块
	blockNumberList := [...]*big.Int{big.NewInt(rpc.LatestBlockNumber.Int64()),
		big.NewInt(rpc.FinalizedBlockNumber.Int64()), big.NewInt(rpc.SafeBlockNumber.Int64())}

	for {
		select {
		case <-ticker.C:
			// 开启一个协程获取，查找链上数据比较慢
			go func() {
				for _, blockNum := range blockNumberList {
					getSpecialBlock(blockNum, client)
				}
			}()
		}
	}

}

// func GetSpecialBlockTask(headCh <-chan string, finalizedCh <-chan string, safeCh <-chan string) {
// 	client, err := ethclient.Dial(config.Config.TestNet.TestEthUrl)
// 	if err != nil {
// 		log.Logger.Error(err.Error())
// 		return
// 	}
// 	defer client.Close()

// 	for {
// 		select {
// 		case <-headCh:
// 			go getSpecialBlock(nil, client)
// 		case <-finalizedCh:
// 			go getSpecialBlock(big.NewInt(rpc.FinalizedBlockNumber.Int64()), client)
// 		case <-safeCh:
// 			go getSpecialBlock(big.NewInt(rpc.SafeBlockNumber.Int64()), client)
// 		}
// 	}
// }

func getSpecialBlock(blockNum *big.Int, client *ethclient.Client) {
	block, err := client.BlockByNumber(context.Background(), blockNum)
	if err != nil {
		log.Logger.Error(err.Error())
		return
	}

	blockResp := models.NewBlock(block)
	db.RedisSet(common.SPECIAL_BLOCK_KEY_PREFIX+blockNum.String(), blockResp, 60)
}
