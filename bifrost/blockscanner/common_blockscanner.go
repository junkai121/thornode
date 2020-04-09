package blockscanner

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"gitlab.com/thorchain/thornode/bifrost/config"
	"gitlab.com/thorchain/thornode/bifrost/metrics"
)

type CommonBlockScannerSupplemental interface {
	BlockRequest(rpcHost string, height int64) (string, string)
	UnmarshalBlock(buf []byte) ([]string, error)
}

// CommonBlockScanner is used to discover block height
// since both binance and thorchain use cosmos, so this part logic should be the same
type CommonBlockScanner struct {
	cfg            config.BlockScannerConfiguration
	rpcHost        string
	logger         zerolog.Logger
	wg             *sync.WaitGroup
	scanChan       chan Block
	stopChan       chan struct{}
	httpClient     *http.Client
	scannerStorage ScannerStorage
	blockRequest   func(rpcHost string, height int64) (string, string)
	unmarshalBlock func(buf []byte) (string, []string, error)
	metrics        *metrics.Metrics
	previousBlock  int64
	errorCounter   *prometheus.CounterVec
	supplemental   CommonBlockScannerSupplemental
}

type Block struct {
	Height int64
	Txs    []string
}

// NewCommonBlockScanner create a new instance of CommonBlockScanner
func NewCommonBlockScanner(cfg config.BlockScannerConfiguration, startBlockHeight int64, scannerStorage ScannerStorage, m *metrics.Metrics, supplemental CommonBlockScannerSupplemental) (*CommonBlockScanner, error) {
	if len(cfg.RPCHost) == 0 {
		return nil, errors.New("host is empty")
	}
	rpcHost := cfg.RPCHost
	if !strings.HasPrefix(rpcHost, "http") {
		rpcHost = fmt.Sprintf("http://%s", rpcHost)
	}

	// check that we can parse our host url
	_, err := url.Parse(rpcHost)
	if err != nil {
		return nil, err
	}

	if scannerStorage == nil {
		return nil, errors.New("scannerStorage is nil")
	}
	if m == nil {
		return nil, errors.New("metrics instance is nil")
	}
	return &CommonBlockScanner{
		cfg:      cfg,
		logger:   log.Logger.With().Str("module", "commonblockscanner").Logger(),
		rpcHost:  rpcHost,
		wg:       &sync.WaitGroup{},
		stopChan: make(chan struct{}),
		scanChan: make(chan Block, cfg.BlockScanProcessors),
		httpClient: &http.Client{
			Timeout: cfg.HttpRequestTimeout,
		},
		scannerStorage: scannerStorage,
		metrics:        m,
		previousBlock:  startBlockHeight,
		errorCounter:   m.GetCounterVec(metrics.CommonBlockScannerError),
		supplemental:   supplemental,
	}, nil
}

// GetHttpClient return the http client used internal to ourside world
// right now we need to use this for test
func (b *CommonBlockScanner) GetHttpClient() *http.Client {
	return b.httpClient
}

// GetMessages return the channel
func (b *CommonBlockScanner) GetMessages() <-chan Block {
	return b.scanChan
}

// Start block scanner
func (b *CommonBlockScanner) Start() {
	b.wg.Add(1)
	go b.scanBlocks()
}

// scanBlocks
func (b *CommonBlockScanner) scanBlocks() {
	b.logger.Debug().Msg("start to scan blocks")
	defer b.logger.Debug().Msg("stop scan blocks")
	defer b.wg.Done()
	currentPos, err := b.scannerStorage.GetScanPos()
	if err != nil {
		b.errorCounter.WithLabelValues("fail_get_scan_pos", "").Inc()
		b.logger.Error().Err(err).Msgf("fail to get current block scan pos, %s will start from %d", b.cfg.ChainID, b.previousBlock)
	} else {
		b.previousBlock = currentPos
	}
	b.metrics.GetCounter(metrics.CurrentPosition).Add(float64(currentPos))
	// start up to grab those blocks that we didn't finished
	for {
		select {
		case <-b.stopChan:
			return
		default:
			currentBlock, rawTxs, err := b.getRPCBlock(b.previousBlock + 1)
			if err != nil {
				// don't log an error if its because the block doesn't exist yet
				if !strings.Contains(err.Error(), "Height must be less than or equal to the current blockchain height") {

					b.errorCounter.WithLabelValues("fail_get_block", "").Inc()
					b.logger.Error().Err(err).Msg("fail to get RPCBlock")
				}
				continue
			}
			block := Block{Height: currentBlock, Txs: rawTxs}
			b.logger.Debug().Int64("current block height", currentBlock).Int64("block height", b.previousBlock).Msgf("Chain %s get block height", b.cfg.ChainID)
			b.previousBlock++
			b.metrics.GetCounter(metrics.TotalBlockScanned).Inc()
			select {
			case <-b.stopChan:
				return
			case b.scanChan <- block:
			}
			b.metrics.GetCounter(metrics.CurrentPosition).Inc()
			if err := b.scannerStorage.SetScanPos(b.previousBlock); err != nil {
				b.errorCounter.WithLabelValues("fail_save_block_pos", strconv.FormatInt(b.previousBlock, 10)).Inc()
				b.logger.Error().Err(err).Msg("fail to save block scan pos")
				// alert!!
				return
			}
		}
	}
}

func (b *CommonBlockScanner) getFromHttp(url, body string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, strings.NewReader(body))
	if err != nil {
		b.errorCounter.WithLabelValues("fail_create_http_request", url).Inc()
		return nil, errors.Wrap(err, "fail to create http request")
	}
	if len(body) > 0 {
		req.Header.Add("Content-Type", "application/json")
	}
	resp, err := b.httpClient.Do(req)
	if err != nil {
		b.errorCounter.WithLabelValues("fail_send_http_request", url).Inc()
		return nil, errors.Wrapf(err, "fail to get from %s ", url)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			b.logger.Error().Err(err).Msg("fail to close http response body.")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		b.errorCounter.WithLabelValues("unexpected_status_code", resp.Status).Inc()
		return nil, errors.Errorf("unexpected status code:%d from %s", resp.StatusCode, url)
	}
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// test if our response body is an error block json format
	errorBlock := struct {
		Error struct {
			Code    int64  `json:"code"`
			Message string `json:"message"`
			Data    string `json:"data"`
		} `json:"error"`
	}{}

	_ = json.Unmarshal(buf, &errorBlock) // ignore error
	if errorBlock.Error.Code != 0 {
		return nil, fmt.Errorf(
			"%s (%d): %s",
			errorBlock.Error.Message,
			errorBlock.Error.Code,
			errorBlock.Error.Data,
		)
	}

	return buf, nil
}

func (b *CommonBlockScanner) getRPCBlock(height int64) (int64, []string, error) {
	start := time.Now()
	defer func() {
		if err := recover(); err != nil {
			b.logger.Error().Msgf("fail to get RPCBlock:%s", err)
		}
		duration := time.Since(start)
		b.metrics.GetHistograms(metrics.BlockDiscoveryDuration).Observe(duration.Seconds())
	}()
	url, body := b.supplemental.BlockRequest(b.rpcHost, height)
	buf, err := b.getFromHttp(url, body)
	if err != nil {
		b.errorCounter.WithLabelValues("fail_get_block", url).Inc()
		time.Sleep(300 * time.Millisecond)
		return 0, nil, err
	}

	rawTxns, err := b.supplemental.UnmarshalBlock(buf)
	if err != nil {
		b.errorCounter.WithLabelValues("fail_unmarshal_block", url).Inc()
	}
	return height, rawTxns, err
}

func (b *CommonBlockScanner) Stop() error {
	b.logger.Debug().Msg("receive stop request")
	defer b.logger.Debug().Msg("common block scanner stopped")
	close(b.stopChan)
	b.wg.Wait()
	return nil
}
