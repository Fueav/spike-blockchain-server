package price

import (
	"errors"
	"spike-blockchain-server/chain"
)

type TokenPriceService struct {
	Token string `json:"token" binding:"required"`
}

func GetTokenContractAddrByTokenSymbol(token string) (string, error) {
	switch token {
	case "skk":
		return chain.SKKContractAddress, nil
	case "sks":
		return chain.SKSContractAddress, nil
	case "test":
		return "0x3EE2200Efb3400fAbB9AacF31297cBdD1d435D47", nil
	default:
		return "", errors.New("token type is not supported")
	}
}
