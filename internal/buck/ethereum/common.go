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

func writeBlockToFile(ctxt c.Context, bn uint64, json []byte) error {
	var err error
	if ctxt.BlockCache != "" && ctxt.CrawlerType == c.Finalized {
		// write block to finalized block cache
		// failures are returned as errors
		err = createDirectoryIfNotExists(ctxt.BlockCache)
		if err != nil {
			log.Error(err)
			return err
		}
		fp := filepath.Join(ctxt.BlockCache, fmt.Sprintf("fb-%d.json", bn))
		err := os.WriteFile(fp, json, 0600)
		if err != nil {
			err = fmt.Errorf("failed to write finalized block #%d, %w", bn, err)
			log.Error(err)
			return err
		}
		return nil
	}
	// write block to debug data store, errors are tolerated
	if ctxt.DebugDataStore != "" {
		err = createDirectoryIfNotExists(ctxt.DebugDataStore)
		if err != nil {
			log.Warn(err)
			return nil
		}
		fp := filepath.Join(ctxt.DebugDataStore, fmt.Sprintf("%s-%d.json", ctxt.CrawlerType, bn))
		err := os.WriteFile(fp, json, 0600)
		if err != nil {
			err = fmt.Errorf("failed to write block #%d, %w", bn, err)
			log.Warn(err)
		}
	}
	return nil
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
