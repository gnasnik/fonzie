package chain

import (
	"context"
	"errors"
	"fmt"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	resty "github.com/go-resty/resty/v2"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosaccount"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"
	log "github.com/sirupsen/logrus"
	"os"
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
	Prefix string               `json:"prefix"`
	RPC    string               `json:"rpc"`
	client *cosmosclient.Client `json:"-"`
}

func (chain *Chain) getClient() *cosmosclient.Client {
	if chain.client == nil {

		keyringDir := os.Getenv("KEYRING_DIR")
		if keyringDir == "" {
			keyringDir = "/root/.titan"
		}

		client, err := cosmosclient.New(context.Background(),
			cosmosclient.WithAddressPrefix(chain.Prefix),
			cosmosclient.WithNodeAddress(chain.RPC),
			cosmosclient.WithGasPrices("0.0025uttnt"),
			cosmosclient.WithKeyringServiceName("titan"),
			cosmosclient.WithKeyringDir(keyringDir),
		)
		if err != nil {
			log.Fatal(err)
		}

		chain.client = &client
	}
	return chain.client
}

func (chain *Chain) getAccount() *cosmosaccount.Account {
	acc, err := chain.client.Account(chain.Prefix)

	if err != nil {
		return nil
	}

	return &acc
}

func (chain *Chain) ImportMnemonic(mnemonic string) error {

	return nil
}

func (chain *Chain) SendMsgs(outputs []banktypes.Output) error {
	c := chain.getClient()
	a := chain.getAccount()

	if a == nil {
		return errors.New("no account found")
	}

	inputCoins := cosmostypes.NewCoins()
	for _, o := range outputs {
		inputCoins = inputCoins.Add(o.Coins...)
	}

	faucetAddr, err := a.Address(chain.Prefix)
	if err != nil {
		return err
	}

	log.Infof("Sending %s from faucet address [%s] to recipient [%s]", inputCoins, faucetAddr, outputs)
	//	Build transaction message
	req := &banktypes.MsgMultiSend{
		Inputs: []banktypes.Input{{
			Address: faucetAddr,
			Coins:   inputCoins,
		}},
		Outputs: outputs,
	}

	// Send message and get response
	res, err := c.BroadcastTx(context.Background(), *a, req)
	if err != nil {
		return err
	}
	fmt.Println(res)

	return nil
}

func (chain *Chain) Send(toAddr string, coins cosmostypes.Coins) error {
	c := chain.getClient()
	a := chain.getAccount()

	if a == nil {
		return errors.New("no account found")
	}

	faucetAddr, err := a.Address(chain.Prefix)
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
	res, err := c.BroadcastTx(context.Background(), *a, req)
	if err != nil {
		return err
	}
	fmt.Println(res)

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
