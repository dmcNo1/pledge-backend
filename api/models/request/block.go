package request

import "math/big"

type Block struct {
	BlockNum *big.Int
	Full     bool `form:"full"`
}
