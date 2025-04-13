package response

import "pledge-backend/api/models"

type Block struct {
	Hash            string
	Nonce           uint64
	Number          uint64
	Time            uint64
	Transactions    uint64
	TransactionList []*models.Transaction
}

func NewBlock(block *models.Block) *Block {
	return &Block{
		Hash:         block.Hash,
		Nonce:        block.Nonce,
		Number:       block.Number,
		Time:         block.Time,
		Transactions: block.Transactions,
	}
}
