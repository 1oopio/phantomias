package api

import (
	"fmt"

	"github.com/1oopio/phantomias/config"
	"github.com/1oopio/phantomias/database"
)

func getTXLink(s string, args ...any) string {
	if s != "" {
		return fmt.Sprintf(s, args...)
	}
	return ""
}

func getBlockLink(poolCfg *config.Pool, block *database.Block) string {
	if poolCfg.BlockLink == "" {
		return ""
	}
	switch poolCfg.Type {
	case "ergo", "kaspa", "raven":
		var hash string
		if block.Hash != nil {
			hash = *block.Hash
		}
		return fmt.Sprintf(poolCfg.BlockLink, hash)
	default:
		return fmt.Sprintf(poolCfg.BlockLink, block.BlockHeight)
	}
}

func getAddressLink(s string, args ...any) string {
	if s != "" {
		return fmt.Sprintf(s, args...)
	}
	return ""
}
