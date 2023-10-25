package ethereum

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/verity-team/dws/internal/common"
)

func writeBlockToFile(ctxt common.Context, blockNumber uint64, json []byte, final bool) error {
	if ctxt.BlockStorage != "" {
		var fp string
		if final {
			fp = ctxt.BlockStorage + "/" + fmt.Sprintf("fb-%d.json", blockNumber)
		} else {
			fp = ctxt.BlockStorage + "/" + fmt.Sprintf("%d.json", blockNumber)
		}
		err := os.WriteFile(fp, json, 0600)
		if err != nil {
			log.Error(err)
			return err
		}
	}
	return nil
}
