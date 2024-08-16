package main

import (
	"context"
	"fmt"
	"log"
	"uniswaptgbot/config"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	nodeUrl := config.Config("ETHEREUM_NODE_URL")
	fmt.Println(nodeUrl)
	client, err := ethclient.Dial(nodeUrl)
	if err != nil {
		panic(err)
	}
	headers := make(chan *types.Header)
	sub, err := client.SubscribeNewHead(context.Background(), headers)
	if err != nil {
		log.Fatalf("Failed to subscribe to new head: %v", err)
	}

	//monitor new blocks
	for {
		select {
		case err := <-sub.Err():
			log.Fatalf("Subscription Error %v!", err)
		case header := <-headers:
			block, err := client.BlockByHash(context.Background(), header.Hash())
			if err != nil {
				log.Fatalf("Failed to retrieve block %v ", err)
			}
			// Process each transaction in the block
			for _, tx := range block.Transactions() {
				if tx.To() == nil {
					deployer, err := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
					if err != nil {
						log.Fatalf("Failed to retrieve sender: %v", err)
					}
					contractAddr := crypto.CreateAddress(deployer, tx.Nonce())
					//Check wheter it's ERC20 token
					if isERC20(contractAddr, client) {
						// Get token information
						fmt.Println("New Token Deployed!")
						fmt.Printf("Deployer Address: %s\n", deployer.Hex())
						fmt.Printf("Contract Address: %s\n", contractAddr.Hex())
						// log.Printf("Token Name: %s", tokenName)
						// log.Printf("Total Supply: %s", totalSupply.String())
					}

				}
			}
		}
	}
}

func isERC20(contractAddr common.Address, client *ethclient.Client) bool {
	code, err := client.CodeAt(context.Background(), contractAddr, nil)
	if err != nil {
		log.Fatalf("Failed to retrieve contract code: %v", err)
	}
	return len(code) > 0
}
