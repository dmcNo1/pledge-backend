package models

import (
	"pledge-backend/db"
	"pledge-backend/log"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Transaction struct {
	Id          int             `json:"-" gorm:"column:id;primaryKey;autoIncrement"`
	Hash        string          `json:"hash" gorm:"column:hash;"`
	Value       decimal.Decimal `json:"value" gorm:"column:value;type:NUMERIC(30,0)"`
	Gas         uint64          `json:"gas" gorm:"column:gas;"`
	GasPrice    decimal.Decimal `json:"gasPrice" gorm:"column:gas_price;type:NUMERIC(30,0)"`
	Nonce       uint64          `json:"nonce" gorm:"column:nonce;"`
	ToHash      string          `json:"toHash" gorm:"column:to_hash;"`
	BlockNumber uint64          `json:"blockNumber" gorm:"column:block_number;"`
}

func (t *Transaction) TableName() string {
	return "transaction"
}

func NewTransaction(tx *types.Transaction, blockNumber uint64) *Transaction {
	return &Transaction{
		Hash:        tx.Hash().Hex(),
		Value:       decimal.NewFromBigInt(tx.Value(), 0),
		Gas:         tx.Gas(),
		GasPrice:    decimal.NewFromBigInt(tx.GasPrice(), 0),
		Nonce:       tx.Nonce(),
		ToHash:      tx.To().String(),
		BlockNumber: blockNumber,
	}
}

// 插入通过查询区块获取到的交易数据，先删后插
func (t *Transaction) InsertFromBlock(transactionList []*Transaction, blockNumber uint64) error {
	return db.Mysql.Transaction(func(tx *gorm.DB) error {
		err := db.Mysql.Exec("delete from transaction where block_number = ?", blockNumber).Debug().Error
		if err != nil {
			log.Logger.Error(err.Error())
			return err
		}
		err = db.Mysql.Table("transaction").Create(&transactionList).Debug().Error
		if err != nil {
			log.Logger.Error(err.Error())
			return err
		}
		return nil
	})
}
