package services

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	consts "pledge-backend/api/common"
	"pledge-backend/api/common/statecode"
	"pledge-backend/api/models"
	"pledge-backend/api/models/request"
	"pledge-backend/api/models/response"
	"pledge-backend/config"
	"pledge-backend/contract/store"
	"pledge-backend/db"
	"pledge-backend/log"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"gorm.io/gorm"
)

type EthService struct {
}

func NewEthService() *EthService {
	return &EthService{}
}

func (s *EthService) GetTxMsg(txHash string) (*models.Transaction, int) {
	// 查询数据库，如果数据库不存在交易信息，则从链上获取
	transaction := &models.Transaction{}
	err := db.Mysql.Table(transaction.TableName()).Where("hash = ?", txHash).First(&transaction).Debug().Error
	if err == nil {
		return transaction, statecode.CommonSuccess
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Logger.Error(err.Error())
		return nil, statecode.CommonErrServerErr
	}

	// 从链上获取数据
	client, err := ethclient.Dial(config.Config.TestNet.TestEthUrl)
	if nil != err {
		log.Logger.Error(err.Error())
		return nil, statecode.CommonErrServerErr
	}
	defer client.Close()

	tx, _, err := client.TransactionByHash(context.Background(), common.HexToHash(txHash))
	if nil != err {
		log.Logger.Error(err.Error())
		return nil, statecode.TxNotFound
	}

	// 通过receipt回写blockNumber
	receipt, err := client.TransactionReceipt(context.Background(), common.HexToHash(txHash))
	if nil != err {
		log.Logger.Error(err.Error())
		return nil, statecode.CommonErrServerErr
	}
	transaction = models.NewTransaction(tx, receipt.BlockNumber.Uint64())

	// 封装对象，落库
	_ = db.Mysql.Table(transaction.TableName()).Create(transaction)
	// 优化点：并发场景下可能会出现唯一键冲突，gorm没有封装对应的错误信息，需要根据原始的错误信息的错误码去判断
	// err = db.Mysql.Table(transaction.TableName()).Create(transaction).Error
	// if err != nil {
	// 	if mySqlErr, ok := err.(*mysql.MySQLError); ok {
	// 		if mySqlErr.Number == 1062 {
	// 			// 处理冲突
	// 		}
	// 	}
	// }

	return transaction, statecode.CommonSuccess
}

func (s *EthService) GetReceipt(txHash string) (*models.Receipt, int) {
	// 查询数据库，如果数据库不存在交易信息，则从链上获取
	receiptDO := &models.Receipt{}
	err := db.Mysql.Table(receiptDO.TableName()).Where("transaction_hash = ?", txHash).First(&receiptDO).Debug().Error
	if err == nil {
		return receiptDO, statecode.CommonSuccess
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Logger.Error(err.Error())
		return nil, statecode.CommonErrServerErr
	}

	// 从链上获取数据
	client, err := ethclient.Dial(config.Config.TestNet.TestEthUrl)
	if nil != err {
		log.Logger.Error(err.Error())
		return nil, statecode.CommonErrServerErr
	}
	defer client.Close()

	receipt, err := client.TransactionReceipt(context.Background(), common.HexToHash(txHash))
	if nil != err {
		log.Logger.Error(err.Error())
		return nil, statecode.ReceiptNotFound
	}

	receiptDO = models.NewReceipt(receipt)

	// 封装对象，落库
	// 优化点：并发场景下可能会出现唯一键冲突，gorm没有封装对应的错误信息，需要根据原始的错误信息的错误码去判断
	_ = db.Mysql.Table(receiptDO.TableName()).Create(receiptDO)

	return receiptDO, statecode.CommonSuccess
}

// 获取区块信息
func (s *EthService) GetBlock(param *request.Block) (*response.Block, int) {
	blockNum := param.BlockNum
	blockDO := models.Block{}

	// 如果是head、finalize、safe节点，先尝试从Redis获取
	if checkSpecialBlock(blockNum) {
		key := consts.SPECIAL_BLOCK_KEY_PREFIX + blockNum.String()
		blockByte, _ := db.RedisGet(key)
		if len(blockByte) > 0 {
			blockResp := &response.Block{}
			// 反序列化
			err := json.Unmarshal(blockByte, blockResp)
			if err != nil {
				return nil, statecode.CommonErrServerErr
			}

			if param.Full {
				s.GetTransaction(blockResp)
			}
			return blockResp, statecode.CommonSuccess
		}
		// Redis中没有，那就从链上获取
	} else {
		// 从库里获取Block信息
		err := db.Mysql.Table(blockDO.TableName()).Where("number = ?", blockNum.Uint64()).First(&blockDO).Debug().Error
		// 如果err为nil，说明查询到了数据，直接返回即可
		if err == nil {
			blockResp := response.NewBlock(&blockDO)
			if param.Full {
				s.GetTransaction(blockResp)
			}
			return blockResp, statecode.CommonSuccess
		} else if !errors.Is(err, gorm.ErrRecordNotFound) { // sql报错，直接返回错误
			log.Logger.Error(err.Error())
			return nil, statecode.CommonErrServerErr
		}
	}

	// 库里没有数据，从链上获取数据
	client, err := ethclient.Dial(config.Config.TestNet.TestEthUrl)
	if nil != err {
		log.Logger.Error(err.Error())
		return nil, statecode.CommonErrServerErr
	}
	defer client.Close()
	block, err := client.BlockByNumber(context.Background(), blockNum)
	if nil != err {
		log.Logger.Error(err.Error())
		return nil, statecode.BlockNotFound
	}

	// 落库
	blockDO = models.Block{
		Hash:         block.Hash().String(),
		Nonce:        block.Nonce(),
		Number:       block.Number().Uint64(),
		Time:         block.Time(),
		Transactions: uint64(block.Transactions().Len()),
	}
	// 两边同时插入同一条数据，如果存在唯一索引（比如Number）可能会报错，但是肯定有一条正确是数据能落库，不需要处理这种异常
	db.Mysql.Table(blockDO.TableName()).Create(&blockDO)
	blockResp := response.NewBlock(&blockDO)

	// 三个特殊区块需要实时存入Redis
	if checkSpecialBlock(blockNum) {
		db.RedisSet(consts.SPECIAL_BLOCK_KEY_PREFIX+blockNum.String(), blockResp, 60)
	}

	// 查询交易信息
	if param.Full && blockResp.Transactions != 0 {
		transactionRespList := make([]*models.Transaction, 0)
		for _, tx := range block.Transactions() {
			transactionDB := models.NewTransaction(tx, blockResp.Number)
			transactionRespList = append(transactionRespList, transactionDB)
		}
		// 数据落库
		err = (&models.Transaction{}).InsertFromBlock(transactionRespList, blockResp.Number)
		if err != nil {
			log.Logger.Error(err.Error())
			return nil, statecode.CommonErrServerErr
		}
		blockResp.TransactionList = transactionRespList
	}

	// 封装返回值
	return blockResp, statecode.CommonSuccess
}

func (s *EthService) GetTransaction(blockResp *response.Block) int {
	// 如果block的交易数量为0，直接返回
	if blockResp.Transactions == 0 {
		return statecode.CommonSuccess
	}

	// 查询库中数据条数是否匹配
	var count int64 = 0
	transactionRespList := make([]*models.Transaction, 0)
	err := db.Mysql.Table("transaction").Where("block_number = ?", blockResp.Number).Count(&count).Debug().Error
	if err != nil {
		log.Logger.Error(err.Error())
		return statecode.CommonErrServerErr
	}
	// 数据库已有交易信息，且数量相等，直接返回
	if count == int64(blockResp.Transactions) {
		err := db.Mysql.Table("transaction").Where("block_number = ?", blockResp.Number).Find(&transactionRespList).Debug().Error
		if err != nil {
			log.Logger.Error(err.Error())
			return statecode.CommonErrServerErr
		}
	} else { // 库中没有数据，或者数据条数不对，从链上获取
		client, err := ethclient.Dial(config.Config.TestNet.TestEthUrl)
		if nil != err {
			log.Logger.Error(err.Error())
			return statecode.CommonErrServerErr
		}
		defer client.Close()
		block, err := client.BlockByNumber(context.Background(), big.NewInt(int64(blockResp.Number)))
		if err != nil {
			log.Logger.Error(err.Error())
			return statecode.CommonErrServerErr
		}
		for _, tx := range block.Transactions() {
			transactionDB := models.NewTransaction(tx, blockResp.Number)
			transactionRespList = append(transactionRespList, transactionDB)
		}
		// 数据落库，先删后插
		err = (&models.Transaction{}).InsertFromBlock(transactionRespList, blockResp.Number)
		if err != nil {
			log.Logger.Error(err.Error())
			return statecode.CommonErrServerErr
		}
	}
	blockResp.TransactionList = transactionRespList

	return statecode.CommonSuccess
}

func (s *EthService) SetItem(key string, value string) (interface{}, int) {
	// 建立连接
	client, err := ethclient.Dial(config.Config.TestNet.TestEthUrl)
	if nil != err {
		log.Logger.Error(err.Error())
		return nil, statecode.CommonErrServerErr
	}
	defer client.Close()

	// 生成合约实例
	storeAddress := common.HexToAddress(config.Config.TestNet.StoreAddress)
	storeInstance, err := store.NewStore(storeAddress, client)
	if nil != err {
		log.Logger.Error(err.Error())
		return nil, statecode.CommonErrServerErr
	}

	// 获取私钥
	privateKey, err := crypto.HexToECDSA(config.Config.TestNet.PrivateKey)
	if nil != err {
		log.Logger.Error(err.Error())
		return nil, statecode.CommonErrServerErr
	}

	// 封装参数
	var keyBytes [32]byte
	var valueBytes [32]byte
	copy(keyBytes[:], []byte(key))
	copy(valueBytes[:], []byte(value))

	chainId, err := client.ChainID(context.Background())
	if nil != err {
		log.Logger.Error(err.Error())
		return nil, statecode.CommonErrServerErr
	}
	// 创建事务签名者
	opts, err := bind.NewKeyedTransactorWithChainID(privateKey, chainId)
	if nil != err {
		log.Logger.Error(err.Error())
		return nil, statecode.CommonErrServerErr
	}

	// 生成交易
	tx, err := storeInstance.SetItem(opts, keyBytes, valueBytes)
	if nil != err {
		log.Logger.Error(err.Error())
		return nil, statecode.CommonErrServerErr
	}

	// 等待交易结果
	receipt, err := bind.WaitMined(context.Background(), client, tx)
	if nil != err {
		log.Logger.Error(err.Error())
		return nil, statecode.CommonErrServerErr
	}

	callOpts := bind.CallOpts{Context: context.Background()}
	trueValue, err := storeInstance.Items(&callOpts, keyBytes)

	res := map[string]interface{}{
		"receipt": &receipt,
		"value":   trueValue,
	}
	return res, statecode.CommonSuccess
}

func checkSpecialBlock(blockNum *big.Int) bool {
	return blockNum == nil || blockNum.Int64() == rpc.LatestBlockNumber.Int64() ||
		blockNum.Int64() == rpc.FinalizedBlockNumber.Int64() || blockNum.Int64() == rpc.SafeBlockNumber.Int64()
}
