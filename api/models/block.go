package models

type Block struct {
	Id           uint64 `json:"-" gorm:"primary_key;AUTO_INCREMENT"`
	Hash         string `json:"hash" gorm:"column:hash"`
	Number       uint64 `json:"number" gorm:"column:number"`
	Time         uint64 `json:"time" gorm:"column:time"`
	Nonce        uint64 `json:"nonce" gorm:"column:nonce"`
	Transactions uint64 `json:"transactions" gorm:"column:transactions"`
}

func (b *Block) TableName() string {
	return "block"
}
