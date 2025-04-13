package models

import (
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/shopspring/decimal"
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

func (t *Transaction) TableName() string {
	return "transaction"
}
