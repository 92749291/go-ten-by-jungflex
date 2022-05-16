package hostrunner

import (
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/obscuronet/obscuro-playground/go/ethclient"
	"github.com/obscuronet/obscuro-playground/go/ethclient/mgmtcontractlib"
	"github.com/obscuronet/obscuro-playground/go/ethclient/wallet"
	"github.com/obscuronet/obscuro-playground/go/log"
	"github.com/obscuronet/obscuro-playground/go/obscuronode/host"
	"github.com/obscuronet/obscuro-playground/go/obscuronode/host/p2p"
)

// RunHost runs an Obscuro host as a standalone process.
func RunHost(config HostConfig) {
	setLogs()

	nodeID := common.BytesToAddress([]byte(config.NodeID))
	hostCfg := host.AggregatorCfg{
		GossipRoundDuration:  time.Duration(config.GossipRoundNanos),
		ClientRPCTimeoutSecs: config.RPCTimeoutSecs,
		HasRPC:               true,
		RPCAddress:           &config.ClientServerAddr,
	}

	nodeWallet := wallet.NewInMemoryWallet(config.PrivateKeyString)
	contractAddr := common.HexToAddress(config.ContractAddress)
	txHandler := mgmtcontractlib.NewEthMgmtContractTxHandler(contractAddr)
	l1Client, err := ethclient.NewEthClient(nodeID, config.EthClientHost, uint(config.EthClientPort), nodeWallet, contractAddr)
	if err != nil {
		panic(err)
	}
	enclaveClient := host.NewEnclaveRPCClient(config.EnclaveAddr, host.ClientRPCTimeoutSecs*time.Second, nodeID)
	aggP2P := p2p.NewSocketP2PLayer(config.OurP2PAddr, config.PeerP2PAddrs, nodeID)

	agg := host.NewObscuroAggregator(nodeID, hostCfg, nil, config.IsGenesis, aggP2P, l1Client, enclaveClient, txHandler)

	agg.Start()
}

// Sets the log file.
func setLogs() {
	logFile, err := os.Create("host_logs.txt")
	if err != nil {
		panic(err)
	}
	log.SetLog(logFile)
}
