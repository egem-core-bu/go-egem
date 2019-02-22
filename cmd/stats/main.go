package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	ethereum "git.egem.io/team/go-egem"
	"git.egem.io/team/go-egem/common/hexutil"
	"git.egem.io/team/go-egem/core/types"
	"git.egem.io/team/go-egem/ethclient"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
)

var (
	gem   *ethclient.Client
	egem  *GethInfo
	delay int
	// Connect to a specific node using -connect flag
	Connect string
	// Port pick one or set it as -port "none"
	Port string
	// Polling
	Polling int
)

func init() {
	egem = new(GethInfo)
	egem.TotalEth = big.NewInt(0)

	flag.StringVar(&Connect, "connect", "", "Connect to a specific chains rcp endpoint")
	flag.StringVar(&Port, "port", "", `Define the port to connect to, if no port just use -port "none"`)
	flag.IntVar(&Polling, "polling", 0, `Define the polling rate for new info.`)
	flag.Parse()
}

// GethInfo structure
type GethInfo struct {
	Server           string
	ContractsCreated int64
	TokenTransfers   int64
	ContractCalls    int64
	EthTransfers     int64
	BlockSize        float64
	LoadTime         float64
	TotalEth         *big.Int
	CurrentBlock     *types.Block
	Sync             *ethereum.SyncProgress
	LastBlockUpdate  time.Time
	SugGasPrice      *big.Int
	PendingTx        uint
	NetworkID        *big.Int
}

// Address struture
type Address struct {
	Balance *big.Int
	Address string
	Nonce   uint64
}

func main() {
	var err error
	egem.Server = Connect + ":" + Port
	if Polling == 0 {
		delay = 300
	}
	if Polling != 0 {
		delay = Polling
	}
	if Connect == "" && Port == "" {
		egem.Server = "http://localhost:8895"
	}

	if Connect != "" && Port == "none" {
		egem.Server = Connect
	}

	log.Printf("Connecting to node: %v\n", egem.Server)
	gem, err = ethclient.Dial(egem.Server)
	if err != nil {
		panic(err)
	}

	egem.CurrentBlock, err = gem.BlockByNumber(context.TODO(), nil)
	if err != nil {
		panic(err)
	}

	go Routine()

	log.Printf("Stats running on http://localhost:8897/stats\n")

	http.HandleFunc("/stats", MetricsHTTP)
	err = http.ListenAndServe(":8897", nil)
	if err != nil {
		panic(err)
	}
}

// CalculateTotals will do some work.
func CalculateTotals(block *types.Block) {
	egem.TotalEth = big.NewInt(0)
	egem.ContractsCreated = 0
	egem.TokenTransfers = 0
	egem.EthTransfers = 0
	for _, b := range block.Transactions() {

		if b.To() == nil {
			egem.ContractsCreated++
		}

		if len(b.Data()) >= 4 {
			method := hexutil.Encode(b.Data()[:4])
			if method == "0xa9059cbb" {
				egem.TokenTransfers++
			}
		}

		if b.Value().Sign() == 1 {
			egem.EthTransfers++
		}

		egem.TotalEth.Add(egem.TotalEth, b.Value())
	}

	size := strings.Split(egem.CurrentBlock.Size().String(), " ")
	egem.BlockSize = stringToFloat(size[0]) * 1000
}

// Routine function
func Routine() {
	var lastBlock *types.Block
	ctx := context.Background()
	for {
		t1 := time.Now()
		var err error
		egem.CurrentBlock, err = gem.BlockByNumber(ctx, nil)
		if err != nil {
			log.Printf("issue with response from server: %v\n", egem.CurrentBlock)
			time.Sleep(time.Duration(delay) * time.Millisecond)
			continue
		}
		egem.SugGasPrice, _ = gem.SuggestGasPrice(ctx)
		egem.PendingTx, _ = gem.PendingTransactionCount(ctx)
		egem.NetworkID, _ = gem.NetworkID(ctx)
		egem.Sync, _ = gem.SyncProgress(ctx)

		if lastBlock == nil || egem.CurrentBlock.NumberU64() > lastBlock.NumberU64() {
			log.Printf("Received block #%v with %v transactions (%v)\n", egem.CurrentBlock.NumberU64(), len(egem.CurrentBlock.Transactions()), egem.CurrentBlock.Hash().String())
			egem.LastBlockUpdate = time.Now()
			egem.LoadTime = time.Now().Sub(t1).Seconds()
		}

		lastBlock = egem.CurrentBlock
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}
}

// MetricsHTTP HTTP response handler for /stats
func MetricsHTTP(w http.ResponseWriter, r *http.Request) {
	var allOut []string
	block := egem.CurrentBlock
	if block == nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("issue receiving block from URL: %v", egem.Server)))
		return
	}
	CalculateTotals(block)

	v, _ := mem.VirtualMemory()
	s, _ := mem.SwapMemory()
	h, _ := host.Info()
	l, _ := load.Avg()

	type EGEMOut struct {
		LastBlock        float64    `json:"seconds_last_block"`
		Block            uint64     `json:"current_block"`
		NetworkID        *big.Int   `json:"network_id"`
		PendingTx        uint       `json:"pending_transactions"`
		BlockValue       *big.Float `json:"block_value"`
		BlockGasUsed     uint64     `json:"block_gas_used"`
		BlockGasLimit    uint64     `json:"block_gas_limit"`
		BlockGasPrice    *big.Int   `json:"block_gas_price"`
		BlockNonce       uint64     `json:"block_nonce"`
		BlockDifficulty  *big.Int   `json:"block_difficulty"`
		BlockUncles      int        `json:"block_uncles"`
		BlockSize        float64    `json:"block_size_bytes"`
		ContractsCreated int64      `json:"contracts_created"`
		TokenTransfers   int64      `json:"token_transfers"`
		Transfers        int64      `json:"transfers"`
		RPCLoadTime      float64    `json:"rpc_load_time"`
		PhyMemTotal      uint64     `json:"phy_mem_total"`
		PhyMemFree       uint64     `json:"phy_mem_free"`
		PhyUsedPercent   float64    `json:"phy_mem_used_percent"`
		SwpMemTotal      uint64     `json:"swp_mem_total"`
		SwpMemFree       uint64     `json:"swp_mem_free"`
		SwpUsedPercent   float64    `json:"swp_mem_used_percent"`
		HostName         string     `json:"hostname"`
		Platform         string     `json:"platform"`
		HostOS           string     `json:"host_os"`
		HostUptime       uint64     `json:"host_uptime_mins"`
		ServerLoad       []float64  `json:"server_load_1m_5m_15m"`
	}

	EGEMOutData := &EGEMOut{
		LastBlock:        time.Now().Sub(egem.LastBlockUpdate).Seconds(),
		Block:            block.NumberU64(),
		NetworkID:        egem.NetworkID,
		PendingTx:        egem.PendingTx,
		BlockValue:       ToEther(egem.TotalEth),
		BlockGasUsed:     block.GasUsed(),
		BlockGasLimit:    block.GasLimit(),
		BlockGasPrice:    egem.SugGasPrice,
		BlockNonce:       block.Nonce(),
		BlockDifficulty:  block.Difficulty(),
		BlockUncles:      len(block.Uncles()),
		BlockSize:        egem.BlockSize,
		ContractsCreated: egem.ContractsCreated,
		TokenTransfers:   egem.TokenTransfers,
		Transfers:        egem.EthTransfers,
		RPCLoadTime:      egem.LoadTime,
		PhyMemTotal:      (v.Total / 1024 / 1024),
		PhyMemFree:       (v.Available / 1024 / 1024),
		PhyUsedPercent:   v.UsedPercent,
		SwpMemTotal:      (s.Total / 1024 / 1024),
		SwpMemFree:       (s.Free / 1024 / 1024),
		SwpUsedPercent:   s.UsedPercent,
		HostName:         h.Hostname,
		Platform:         h.Platform,
		HostOS:           h.OS,
		HostUptime:       (h.Uptime / 60),
		ServerLoad:       []float64{l.Load1, l.Load5, l.Load15},
	}

	EGEMOutDataFinal, _ := json.Marshal(EGEMOutData)

	// Output to http://localhost:8897/stats
	allOut = append(allOut, fmt.Sprintf(string(EGEMOutDataFinal)))
	w.Write([]byte(strings.Join(allOut, "\n")))
}

// stringToFloat will simply convert a string to a float
func stringToFloat(s string) float64 {
	amount, _ := strconv.ParseFloat(s, 10)
	return amount
}

// ToEther CONVERTS WEI TO ETH
func ToEther(o *big.Int) *big.Float {
	pul, int := big.NewFloat(0), big.NewFloat(0)
	int.SetInt(o)
	pul.Mul(big.NewFloat(0.000000000000000001), int)
	return pul
}
