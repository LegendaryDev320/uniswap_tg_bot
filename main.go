package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strings"
	"uniswaptgbot/config"
	"uniswaptgbot/erc20"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
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
					bERC20 := isERC20(contractAddr, client)
					if bERC20 {
						// Get token information
						fmt.Println("New Token Deployed!")
						fmt.Printf("Deployer Address: %s\n", deployer.Hex())
						fmt.Printf("Contract Address: %s\n", contractAddr.Hex())
						name, totSupply, err := getTokenInfo(contractAddr, client)
						if err != nil {
							fmt.Println("Error getting token info")
						}
						fmt.Printf("Token Name: %s", name)
						fmt.Printf("Total Supply: %s", totSupply.String())
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
	if len(code) == 0 {
		log.Fatalf("no contract code at given address")
		return false
	}

	hexCode := hex.EncodeToString(code)

	var erc20Signatures = []string{
		"18160ddd", // totalSupply()
		"70a08231", // balanceOf(address)
		"a9059cbb", // transfer(address,uint256)
		"23b872dd", // transferFrom(address,address,uint256)
		"095ea7b3", // approve(address,uint256)
		"dd62ed3e", // allowance(address,address)
	}

	for _, sig := range erc20Signatures {
		if !strings.Contains(hexCode, sig) {
			return false
		}
	}

	return true
}

func getTokenInfo(contractAddr common.Address, client *ethclient.Client) (string, *big.Int, error) {
	instance, err := erc20.NewGGToken(contractAddr, client)
	if err != nil {
		log.Fatal(err)
	}

	//Get token name
	name, err := instance.Name(&bind.CallOpts{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Token Name: %s\n", name)

	//Get total Supply
	totalSupply, err := instance.TotalSupply(&bind.CallOpts{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Total Supply: %s\n", totalSupply.String())

	return name, totalSupply, nil
}
