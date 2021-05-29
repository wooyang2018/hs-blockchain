// Copyright (C) 2021 Aung Maw
// Licensed under the GNU General Public License v3.0

package main

import (
	"fmt"
	"log"
	"os"
	"path"

	"github.com/aungmawjj/juria-blockchain/node"
	"github.com/aungmawjj/juria-blockchain/tests/cluster"
	"github.com/aungmawjj/juria-blockchain/tests/experiments"
)

const (
	JuriaPath    = "./juria"
	WorkDir      = "./workdir"
	NodeCount    = 7
	ClusterDebug = true
)

func setupExperiments() []Experiment {
	expms := make([]Experiment, 0)
	expms = append(expms, &experiments.RestartCluster{})
	expms = append(expms, &experiments.MajorityKeepRunning{})
	expms = append(expms, &experiments.RestartMajority{})
	return expms
}

func main() {
	fmt.Println()
	fmt.Println("NodeCount =", NodeCount)
	clustersDir := path.Join(WorkDir, "clusters")

	os.Mkdir(WorkDir, 0755)
	os.Mkdir(clustersDir, 0755)

	cftry, err := cluster.NewLocalFactory(cluster.LocalFactoryParams{
		JuriaPath: JuriaPath,
		WorkDir:   clustersDir,
		NodeCount: NodeCount,
		PortN0:    node.DefaultConfig.Port,
		ApiPortN0: node.DefaultConfig.APIPort,
		Debug:     ClusterDebug,
	})
	check(err)

	expms := setupExperiments()
	pass, fail := runExperiments(cftry, expms)
	fmt.Printf("\nTotal: %d\t|\tPass: %d\t|\tFail: %d\n", len(expms), pass, fail)
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
