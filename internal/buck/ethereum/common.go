package ethereum

import (
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/verity-team/dws/internal/common"
)

func writeBlockToFile(ctxt common.Context, bn uint64, json []byte, final bool) error {
	if ctxt.BlockStorage != "" {
		var fp string
		if final {
			fp = ctxt.BlockStorage + "/" + fmt.Sprintf("fb-%d.json", bn)
		} else {
			fp = ctxt.BlockStorage + "/" + fmt.Sprintf("%d.json", bn)
		}
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

func writeTxsToFile(ctxt common.Context, bn uint64, json []byte) error {
	if ctxt.BlockStorage != "" {
		fp := ctxt.BlockStorage + "/" + fmt.Sprintf("txs-%d-%d.json", bn, time.Now().UnixMilli())
		err := os.WriteFile(fp, json, 0400)
		if err != nil {
			log.Warn(err)
		}
	}
	return nil
}
