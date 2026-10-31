package ethereum

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
	c "github.com/verity-team/dws/internal/common"
)

type EthGetBlockByNumberRequest struct {
	JsonRPC string        `json:"jsonrpc"` // nolint:revive
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

func createDirectoryIfNotExists(dirPath string) error {
	_, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		// The directory does not exist, so create it
		err = os.MkdirAll(dirPath, 0o700)
		if err != nil {
			err = fmt.Errorf("failed to create directory, %w", err)
			log.Error(err)
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}

func getFinalizedBlockFromCache(ctxt c.Context, bn uint64) ([]byte, error) {
	if ctxt.BlockCache == "" {
		return nil, nil
	}
	fp := filepath.Join(ctxt.BlockCache, fmt.Sprintf("fb-%d.json", bn))
	return os.ReadFile(fp)
}

// writeBlockToFile persists a full block body: the finalized crawler keeps it
// in the block cache, every other crawler in the debug data store. Both are
// optional and neither may fail the block -- a full disk or a bad permission
// on a directory that only saves a jsonrpc request later must not stall the
// crawler -- so failures are logged as warnings and otherwise ignored.
func writeBlockToFile(ctxt c.Context, bn uint64, json []byte) {
	if ctxt.BlockCache != "" && ctxt.CrawlerType == c.Finalized {
		if err := createDirectoryIfNotExists(ctxt.BlockCache); err != nil {
			log.Warnf("failed to create the block cache, %v", err)
			return
		}
		fp := filepath.Join(ctxt.BlockCache, fmt.Sprintf("fb-%d.json", bn))
		if err := os.WriteFile(fp, json, 0600); err != nil {
			log.Warnf("failed to cache finalized block #%d, %v", bn, err)
		}
		return
	}
	writeDebugBlock(ctxt, bn, json)
}

// writeDebugBlock writes a block response to the debug data store, errors are
// tolerated.
func writeDebugBlock(ctxt c.Context, bn uint64, json []byte) {
	if ctxt.DebugDataStore == "" {
		return
	}
	if err := createDirectoryIfNotExists(ctxt.DebugDataStore); err != nil {
		log.Warn(err)
		return
	}
	fp := filepath.Join(ctxt.DebugDataStore, fmt.Sprintf("%s-%d.json", ctxt.CrawlerType, bn))
	if err := os.WriteFile(fp, json, 0600); err != nil {
		err = fmt.Errorf("failed to write block #%d, %w", bn, err)
		log.Warn(err)
	}
}

func writeTxReceiptsToFile(ctxt c.Context, epoch int64, json []byte) {
	// write transaction receipts to debug data store, errors are tolerated
	if ctxt.DebugDataStore != "" {
		err := createDirectoryIfNotExists(ctxt.DebugDataStore)
		if err != nil {
			log.Warn(err)
			return
		}
		fp := filepath.Join(ctxt.DebugDataStore, fmt.Sprintf("txr-%d-%d.json", epoch, time.Now().UnixMilli()))
		err = os.WriteFile(fp, json, 0600)
		if err != nil {
			err = fmt.Errorf("failed to write transaction receipts for epoch #%d, %w", epoch, err)
			log.Warn(err)
		}
	}
}

func writeTxsToFile(ctxt c.Context, epoch int64, json []byte) {
	// write transaction objects to debug data store, errors are tolerated
	if ctxt.DebugDataStore != "" {
		err := createDirectoryIfNotExists(ctxt.DebugDataStore)
		if err != nil {
			log.Warn(err)
			return
		}
		fp := filepath.Join(ctxt.DebugDataStore, fmt.Sprintf("txs-%d-%d.json", epoch, time.Now().UnixMilli()))
		err = os.WriteFile(fp, json, 0600)
		if err != nil {
			err = fmt.Errorf("failed to write transaction objects, %w", err)
			log.Warn(err)
		}
	}
}
