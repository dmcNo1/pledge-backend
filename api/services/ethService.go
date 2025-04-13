package services

import (
	"context"
	"errors"
	"math/big"
	"pledge-backend/api/common/statecode"
	"pledge-backend/api/models"
	"pledge-backend/api/models/request"
	"pledge-backend/api/models/response"
	"pledge-backend/config"
	"pledge-backend/db"
	"pledge-backend/log"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"gorm.io/gorm"
)

type EthService struct {
}

func NewEthService() *EthService {
	return &EthService{}
}

func (s *EthService) GetTxMsg(txHash string) (*models.Transaction, error) {
	// 查询数据库，如果数据库不存在交易信息，则从链上获取
	transaction := &models.Transaction{}
	err := db.Mysql.Table(transaction.TableName()).Where("hash = ?", txHash).First(&transaction).Debug().Error
	if nil != err {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Logger.Error(err.Error())
			return nil, err
		}
	} else if transaction.Hash != "" { // 数据库存在交易信息
		return transaction, nil
	}

	// 从链上获取数据
	client, err := ethclient.Dial(config.Config.TestNet.TestEthUrl)
	if nil != err {
		log.Logger.Error(err.Error())
		return nil, err
	}
	defer client.Close()

	tx, _, err := client.TransactionByHash(context.Background(), common.HexToHash(txHash))
	if nil != err {
		log.Logger.Error(err.Error())
		return nil, err
	}

	receipt, err := client.TransactionReceipt(context.Background(), common.HexToHash(txHash))

	// 封装对象，落库
	transaction = models.NewTransaction(tx, receipt.BlockNumber.Uint64())
	err = db.Mysql.Table(transaction.TableName()).Create(transaction).Debug().Error
	if nil != err {
		log.Logger.Error(err.Error())
		return nil, err
	}

	return transaction, nil
}

func (s *EthService) GetReceipt(txHash string) (*models.Receipt, error) {
	// 查询数据库，如果数据库不存在Receipt信息，则从链上获取
	receiptDB := &models.Receipt{}
	err := db.Mysql.Table(receiptDB.TableName()).Where("transaction_hash = ?", txHash).First(&receiptDB).Debug().Error
	if nil != err {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Logger.Error(err.Error())
			return nil, err
		}
	} else if receiptDB.TransactionHash != "" { // 数据库存在交易信息
		return receiptDB, nil
	}

	// 从链上获取数据
	client, err := ethclient.Dial(config.Config.TestNet.TestEthUrl)
	if nil != err {
		log.Logger.Error(err.Error())
		return nil, err
	}
	defer client.Close()

	receipt, err := client.TransactionReceipt(context.Background(), common.HexToHash(txHash))
	if nil != err {
		log.Logger.Error(err.Error())
		return nil, err
	}

	// 封装对象，落库
	receiptDB = &models.Receipt{
		Status:          receipt.Status,
		TransactionHash: receipt.TxHash.String(),
		GasUsed:         receipt.GasUsed,
		ContractAddress: receipt.ContractAddress.String(),
		BlockNumber:     receipt.BlockNumber.Uint64(),
		BlockHash:       receipt.BlockHash.String(),
		Type:            receipt.Type,
	}
	err = db.Mysql.Table(receiptDB.TableName()).Create(receiptDB).Debug().Error
	if nil != err {
		log.Logger.Error(err.Error())
		return nil, err
	}

	return receiptDB, nil
}

// 获取区块信息
func (s *EthService) GetBlock(param *request.Block) (*response.Block, int) {
	// 从库里获取Block信息
	blockDO := models.Block{}
	err := db.Mysql.Table(blockDO.TableName()).Where("number = ?", param.BlockNum.Uint64()).First(&blockDO).Debug().Error
	// 如果err为nil，说明查询到了数据，直接返回即可
	if err == nil {
		blockResp := response.NewBlock(&blockDO)
		if param.Full {
			client, err := ethclient.Dial(config.Config.TestNet.TestEthUrl)
			defer client.Close()
			if nil != err {
				log.Logger.Error(err.Error())
				return nil, statecode.CommonErrServerErr
			}
			s.GetTransaction(blockResp, client)
		}
		return blockResp, statecode.CommonSuccess
	} else if !errors.Is(err, gorm.ErrRecordNotFound) { // sql报错，直接返回错误
		log.Logger.Error(err.Error())
		return nil, statecode.CommonErrServerErr
	}

	// 库里没有数据，从链上获取数据
	client, err := ethclient.Dial(config.Config.TestNet.TestEthUrl)
	if nil != err {
		log.Logger.Error(err.Error())
		return nil, statecode.CommonErrServerErr
	}
	defer client.Close()
	block, err := client.BlockByNumber(context.Background(), param.BlockNum)
	if nil != err {
		log.Logger.Error(err.Error())
		return nil, statecode.CommonErrServerErr
	}

	// 落库
	blockDO = models.Block{
		Hash:         block.Hash().String(),
		Nonce:        block.Nonce(),
		Number:       block.Number().Uint64(),
		Time:         block.Time(),
		Transactions: uint64(block.Transactions().Len()),
	}
	err = db.Mysql.Table(blockDO.TableName()).Create(&blockDO).Debug().Error
	if err != nil {
		log.Logger.Error(err.Error())
		return nil, statecode.CommonErrServerErr
	}
	blockResp := response.NewBlock(&blockDO)

	// 查询交易信息
	if param.Full {
		transactionRespList := make([]*models.Transaction, 0)
		for _, tx := range block.Transactions() {
			transactionDB := models.NewTransaction(tx, blockResp.Number)
			transactionRespList = append(transactionRespList, transactionDB)
		}
		// 数据落库（优化点：先删后插）
		err = db.Mysql.Table("transaction").Create(&transactionRespList).Debug().Error
		if err != nil {
			log.Logger.Error(err.Error())
			return nil, statecode.CommonErrServerErr
		}
	}

	// 封装返回值
	return blockResp, statecode.CommonSuccess
}

func (s *EthService) GetTransaction(blockResp *response.Block, client *ethclient.Client) int {
	// 如果block的交易数量为0，直接返回
	if blockResp.Transactions == 0 {
		return statecode.CommonSuccess
	}

	var count int64 = 0
	transactionRespList := make([]*models.Transaction, 0)
	err := db.Mysql.Table("transaction").Where("block_number = ?", blockResp.Number).Count(&count).Debug().Error
	if err != nil {
		log.Logger.Error(err.Error())
		return statecode.CommonErrServerErr
	}
	// 数据库已有交易信息，且数量相等，直接返回
	if count == int64(blockResp.Transactions) {
		err := db.Mysql.Table("transaction").Where("block_number = ?", blockResp.Number).Find(transactionRespList).Debug().Error
		if err != nil {
			log.Logger.Error(err.Error())
			return statecode.CommonErrServerErr
		}
	} else { // 库中没有数据，从链上获取
		block, err := client.BlockByNumber(context.Background(), big.NewInt(int64(blockResp.Number)))
		if err != nil {
			log.Logger.Error(err.Error())
			return statecode.CommonErrServerErr
		}
		for _, tx := range block.Transactions() {
			transactionDB := models.NewTransaction(tx, blockResp.Number)
			transactionRespList = append(transactionRespList, transactionDB)
		}
		// 数据落库（优化点：先删后插）
		err = db.Mysql.Table("transaction").Create(&transactionRespList).Debug().Error
		if err != nil {
			log.Logger.Error(err.Error())
			return statecode.CommonErrServerErr
		}
	}
	blockResp.TransactionList = transactionRespList

	return statecode.CommonSuccess
}
