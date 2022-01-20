package obscuro

import (
	"github.com/google/uuid"
	"time"
)

type Stats struct {
	nrMiners         int
	simulationTime   int
	avgBlockDuration int
	avgLatency       int
	gossipPeriod     int

	l1Height int
	totalL1  int

	l2Height           int
	totalL2            int
	maxRollupsPerBlock int
	nrEmptyBlocks      int

	totalL2Txs int
	noL1Reorgs map[NodeId]int
	noL2Reorgs map[NodeId]int
	// todo - actual avg block Duration
}

func RunSimulation(nrUsers int, nrMiners int, simulationTime int, avgBlockDuration int, avgLatency int, gossipPeriod int) Stats {

	var stats = Stats{
		nrMiners:         nrMiners,
		simulationTime:   simulationTime,
		avgBlockDuration: avgBlockDuration,
		avgLatency:       avgLatency,
		gossipPeriod:     gossipPeriod,
		noL1Reorgs:       map[NodeId]int{},
		noL2Reorgs:       map[NodeId]int{},
	}

	var network = NetworkCfg{delay: func() int {
		return RndBtw(avgLatency/10, 2*avgLatency)
	}, stats: &stats}

	l1Config := L1MiningConfig{powTime: func() int {
		return RndBtw(avgBlockDuration/nrMiners, nrMiners*avgBlockDuration)
	}}

	l2Cfg := L2Cfg{gossipPeriodMs: gossipPeriod}

	for i := 1; i <= nrMiners; i++ {
		agg := NewAgg(NodeId(i), l2Cfg, nil, &network)
		miner := NewMiner(NodeId(i), l1Config, &agg, &network)
		stats.noL1Reorgs[NodeId(i)] = 0
		agg.l1 = &miner
		network.allAgg = append(network.allAgg, agg)
		network.allMiners = append(network.allMiners, miner)
	}

	for _, m := range network.allMiners {
		//fmt.Printf("Starting Miner M%d....\n", m.id)
		t := m
		go t.Start()
		defer t.Stop()
		go t.aggregator.Start()
		defer t.aggregator.Stop()
	}

	var users = make([]Wallet, 0)
	for i := 1; i <= nrUsers; i++ {
		users = append(users, Wallet{address: uuid.New()})
	}

	go injectUserTxs(users, &network, avgBlockDuration)

	time.Sleep(Duration(simulationTime * 1000 * 1000))
	return *network.stats
}

const INITIAL_BALANCE = 5000

func injectUserTxs(users []Wallet, network *NetworkCfg, avgBlockDuration int) {
	// deposit some initial amount into every user
	for _, u := range users {
		tx := deposit(u, INITIAL_BALANCE)
		network.broadcastL1Tx(&tx)
		time.Sleep(Duration(avgBlockDuration / 3))
	}

	// generate random L2 transfers
	for {
		tx := L2Tx{
			id:     uuid.New(),
			txType: TransferTx,
			amount: RndBtw(1, 500),
			from:   users[RndBtw(0, len(users)-1)].address,
			dest:   users[RndBtw(0, len(users)-1)].address,
		}
		network.broadcastL2Tx(&tx)
		time.Sleep(Duration(RndBtw(avgBlockDuration/15, avgBlockDuration)))
	}
}

func deposit(u Wallet, amount int) L1Tx {
	return L1Tx{
		id:     uuid.New(),
		txType: DepositTx,
		amount: amount,
		dest:   u.address,
	}
}
