package models

import "github.com/ethereum/go-ethereum/core/types"

type Receipt struct {
	Id              int64  `json:"-" gorm:"primary_key;AUTO_INCREMENT"`
	Status          uint64 `json:"status" gorm:"column:status"`
	TransactionHash string `json:"transactionHash" gorm:"column:transaction_hash"`
	GasUsed         uint64 `json:"gasUsed" gorm:"column:gas_used"`
	ContractAddress string `json:"contractAddress" gorm:"column:contract_address"`
	BlockNumber     uint64 `json:"blockNumber" gorm:"column:block_number"`
	BlockHash       string `json:"blockHash" gorm:"column:block_hash"`
	Type            uint8  `json:"type" gorm:"column:type"`
}

func (r *Receipt) TableName() string {
	return "receipt"
}

func NewReceipt(receipt *types.Receipt) *Receipt {
	return &Receipt{
		Status:          receipt.Status,
		TransactionHash: receipt.TxHash.String(),
		GasUsed:         receipt.GasUsed,
		ContractAddress: receipt.ContractAddress.String(),
		BlockNumber:     receipt.BlockNumber.Uint64(),
		BlockHash:       receipt.BlockHash.String(),
		Type:            receipt.Type,
	}
}
