package price

import (
	"errors"
	"spike-blockchain-server/constants"
)

type TokenPriceService struct {
	Token string `json:"token" binding:"required"`
}

func GetTokenContractAddrByTokenSymbol(token string) (string, error) {
	switch token {
	case "skk":
		return constants.GOVERNANCE_TOKEN_ADDRESS, nil
	case "sks":
		return constants.GAME_TOKEN_ADDRESS, nil
	case "test":
		return "0x3EE2200Efb3400fAbB9AacF31297cBdD1d435D47", nil
	default:
		return "", errors.New("token type is not supported")
	}
}
