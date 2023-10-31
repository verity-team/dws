package ethereum

import (
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/verity-team/dws/internal/common"
)

type EthGetBlockByNumberRequest struct {
	JsonRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

func writeBlockToFile(ctxt common.Context, bn uint64, json []byte) error {
	if ctxt.BlockStorage != "" {
		fp := ctxt.BlockStorage + "/" + fmt.Sprintf("%s-%d.json", ctxt.CrawlerType, bn)
		err := os.WriteFile(fp, json, 0400)
		if err != nil {
			log.Warn(err)
		}
	}
	return nil
}

func writeTxReceiptsToFile(ctxt common.Context, bn uint64, json []byte) error {
	if ctxt.BlockStorage != "" {
		fp := ctxt.BlockStorage + "/" + fmt.Sprintf("txr-%d-%d.json", bn, time.Now().UnixMilli())
		err := os.WriteFile(fp, json, 0400)
		if err != nil {
			log.Warn(err)
		}
	}
	return nil
}

func writeTxsToFile(ctxt common.Context, epoch int64, json []byte) error {
	if ctxt.BlockStorage != "" {
		fp := ctxt.BlockStorage + "/" + fmt.Sprintf("txs-%d-%d.json", epoch, time.Now().UnixMilli())
		err := os.WriteFile(fp, json, 0400)
		if err != nil {
			log.Warn(err)
		}
	}
	return nil
}
