# Juria Blockchain

Juria is a high-performance consortium blockchain with [Hotstuff](https://arxiv.org/abs/1803.05069) consensus mechanism and a transaction-based state machine.

Hotstuff provides a mechanism to rotate leader (block maker) efficiently among the validator nodes. Hence it is not required to have a single trusted leader in the network.

With the use of three-chain Hotstuff commit rule, Juria ensures that the same history of blocks are commited on all nodes despite network and machine failures.

![Benchmark](docs/assets/images/benchmark_juria.png)

## Getting started
You can run the cluster tests on local machine in a few seconds.
`go 1.16` is required.

1. Prepare the repo
```sh
git clone https://github.com/aungmawjj/juria-blockchain
cd juria-blockchain
go mod tidy
```

2. Run tests
```sh
cd tests
go run .
```
The test script will compile `cmd/juria` and setup cluster of 4 nodes with different ports on local machine.
Experiments from `tests/experiments` will be run and health check will be performed throughout the tests.

***NOTE**: Network simmulation experiments are only run on the remote linux cluster.*

## Documentation
* [Key Concepts](https://aungmawjj.github.io/juria-blockchain/key-concepts)
* [Tests and Evaluation](https://aungmawjj.github.io/juria-blockchain/tests-and-evaluation)
* [Benchmark on AWS](https://aungmawjj.github.io/juria-blockchain/benchmark-on-aws)
* [Setup Juria Cluster](https://aungmawjj.github.io/juria-blockchain/setup-juria-cluster)

## About the project
### License
This project is licensed under the GPL-3.0 License.

### Contributing
When contributing to this repository, please first discuss the change you wish to make via issue, email, or any other method with the owners of this repository before making a change.
