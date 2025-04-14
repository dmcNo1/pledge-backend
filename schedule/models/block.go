package models

import "github.com/ethereum/go-ethereum/core/types"

type Block struct {
	Hash         string
	Nonce        uint64
	Number       uint64
	Time         uint64
	Transactions uint64
}

func NewBlock(block *types.Block) *Block {
	return &Block{
		Hash:         block.Hash().String(),
		Nonce:        block.Nonce(),
		Number:       block.Number().Uint64(),
		Time:         block.Time(),
		Transactions: uint64(block.Transactions().Len()),
	}
}
