package chain

import (
	"context"
	"fmt"
	"os"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	resty "github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
	lens "github.com/strangelove-ventures/lens/client"
)

type Chains []*Chain

func (chains Chains) ImportMnemonic(ctx context.Context, mnemonic string) error {
	for _, info := range chains {
		err := info.ImportMnemonic(mnemonic)
		if err != nil {
			return err
		}
	}
	return nil
}

func (chains Chains) FindByPrefix(prefix string) *Chain {
	for _, info := range chains {
		if info.Prefix == prefix {
			return info
		}
	}
	return nil
}

type Chain struct {
	Prefix string            `json:"prefix"`
	RPC    string            `json:"rpc"`
	client *lens.ChainClient `json:"-"`
}

func (chain *Chain) getClient() *lens.ChainClient {
	if chain.client == nil {
		chainID, err := getChainID(chain.RPC)
		if err != nil {
			log.Fatalf("failed to get chain id for %s. err: %v", chain.Prefix, err)
		}

		// Build chain config
		chainConfig := lens.ChainClientConfig{
			Key:            "anon",
			ChainID:        chainID,
			RPCAddr:        chain.RPC,
			AccountPrefix:  chain.Prefix,
			KeyringBackend: "memory",
			GasAdjustment:  1.2,
			Debug:          true,
			Timeout:        "20s",
			OutputFormat:   "json",
			SignModeStr:    "direct",
			Modules:        lens.ModuleBasics,
		}
		chainConfig.Key = "anon"

		// Creates client object to pull chain info
		c, err := lens.NewChainClient(&chainConfig, "", os.Stdin, os.Stdout)
		if err != nil {
			log.Fatal(err)
		}

		chain.client = c
	}
	return chain.client
}

func (chain *Chain) ImportMnemonic(mnemonic string) error {
	_, err := chain.getClient().RestoreKey("anon", mnemonic)
	if err != nil {
		return err
	}

	return nil
}

func (chain Chain) Send(toAddr string, coins cosmostypes.Coins) error {
	c := chain.getClient()

	faucetRawAddr, err := c.GetKeyAddress()
	if err != nil {
		return err
	}
	faucetAddr, err := c.EncodeBech32AccAddr(faucetRawAddr)
	if err != nil {
		return err
	}

	log.Infof("Sending %s from faucet address [%s] to recipient [%s]", coins, faucetAddr, toAddr)
	//	Build transaction message
	req := &banktypes.MsgSend{
		FromAddress: faucetAddr,
		ToAddress:   toAddr,
		Amount:      coins,
	}

	// Send message and get response
	res, err := c.SendMsg(context.Background(), req)
	if err != nil {
		return err
	}
	fmt.Println(c.PrintTxResponse(res))

	return nil
}

func getChainID(rpcUrl string) (string, error) {
	rpc := resty.New().SetHostURL(rpcUrl)

	resp, err := rpc.R().
		SetResult(map[string]interface{}{}).
		SetError(map[string]interface{}{}).
		Get("/commit")
	if err != nil {
		return "", err
	}

	if resp.IsError() {
		//return "", resp.Error().(*map[string]interface{})
		return "", fmt.Errorf("could not get chain id; http error code received %d", resp.StatusCode())
	}

	respbody := resp.Result().(*map[string]interface{})
	result := (*respbody)["result"]
	signedHeader := result.(map[string]interface{})["signed_header"]
	header := signedHeader.(map[string]interface{})["header"]
	chainID := header.(map[string]interface{})["chain_id"].(string)
	return chainID, nil
}

/*
"result": {
	"signed_header": {
	  "header": {
	    "version": {
	      "block": "11"
	    },
	    "chain_id": "umee-1",
	    "height": "731426",
*/
