package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strings"
	"sync"
	"uniswaptgbot/config"
	"uniswaptgbot/erc20"

	"database/sql"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	nodeUrl := config.Config("ETHEREUM_NODE_URL")
	dbUrl := config.Config("DB_URL")
	fmt.Println(nodeUrl)
	fmt.Println(dbUrl)
	sql, err := sql.Open("mysql", dbUrl)
	if err != nil {
		panic(err)
	}
	fmt.Println("Connected to database successfully.")

	client, err := ethclient.Dial(nodeUrl)
	if err != nil {
		panic(err)
	}
	headers := make(chan *types.Header)
	sub, err := client.SubscribeNewHead(context.Background(), headers)
	if err != nil {
		log.Printf("Failed to subscribe to new head: %v\n", err)
	}

	//monitor new blocks
	for {
		select {
		case err := <-sub.Err():
			log.Printf("Subscription Error %v!", err)
		case header := <-headers:
			block, err := client.BlockByHash(context.Background(), header.Hash())
			if err != nil {
				log.Printf("Failed to retrieve block %v ", err)
				break
			}
			// Process each transaction in the block
			for _, tx := range block.Transactions() {
				if tx.To() == nil {
					deployer, err := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
					if err != nil {
						log.Printf("Failed to retrieve sender: %v\n", err)
						continue
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
							fmt.Printf("Error getting token info: %s\n", err)
							continue
						}
						fmt.Printf("Token Name: %s", name)
						fmt.Printf("Total Supply: %s", totSupply.String())
						sql.Query("INSERT INTO ethereum (name, total_supply) VALUES (?, ?)", name, totSupply.String())
					}
				}
			}
		}
	}
}

func isERC20(contractAddr common.Address, client *ethclient.Client) bool {
	code, err := client.CodeAt(context.Background(), contractAddr, nil)
	if err != nil {
		log.Printf("Failed to retrieve contract code: %v", err)
	}
	if len(code) == 0 {
		log.Printf("no contract code at given address")
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
		log.Printf("Failed to instantiate contract: %v\n", err)
		return "", nil, err
	}
	fmt.Printf("!!!!--1")
	//Get token name
	name, err := instance.Name(&bind.CallOpts{})
	if err != nil {
		log.Printf("Failed to retrieve token name: %v\n", err)
		return "", nil, err
	}
	fmt.Printf("Token Name: %s\n", name)
	fmt.Printf("!!!!--2")
	//Get total Supply
	totalSupply, err := instance.TotalSupply(&bind.CallOpts{})
	if err != nil {
		log.Printf("Failed to retrieve total supply: %v\n", err)
		return "", nil, err
	}
	fmt.Printf("Total Supply: %s\n", totalSupply.String())
	//Get decimals
	decimals, err := instance.Decimals(&bind.CallOpts{})
	if err != nil {
		log.Printf("Failed to retrieve decimals: %v\n", err)
		return "", nil, err
	}
	fmt.Printf("Decimals: %s\n", decimals)
	//Get symbol
	symbol, err := instance.Symbol(&bind.CallOpts{})
	if err != nil {
		log.Printf("Failed to retrieve symbol: %v\n", err)
		return "", nil, err
	}
	fmt.Printf("Symbol: %s\n", symbol)
	fmt.Printf("!!!!---3")
	return name, totalSupply, nil
}
