package worker

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/umee-network/fonzie/chain"
	"time"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

type Work struct {
	ChainId        string
	Msg            banktypes.Output
	DiscordMessage *discordgo.MessageCreate
}

type Faucet struct {
	session *discordgo.Session
	clients map[string]*chain.Chain
	works   map[string]chan *Work
}

func NewFaucet(dg *discordgo.Session, chains []*chain.Chain, buf int) *Faucet {
	works := make(map[string]chan *Work)
	clients := make(map[string]*chain.Chain)

	for _, c := range chains {
		works[c.Prefix] = make(chan *Work, buf)
		clients[c.Prefix] = c
	}

	return &Faucet{
		session: dg,
		clients: clients,
		works:   works,
	}
}

func (f *Faucet) Run() {
	for chainId, workCh := range f.works {
		go f.runWorker(chainId, workCh)
	}
}

func (f *Faucet) runWorker(chainId string, workCh chan *Work) {
	tick := time.NewTicker(10 * time.Second)
	works := make([]*Work, 0)

	for {
		select {
		case <-tick.C:
			if len(works) > 0 {
				messages := make([]banktypes.Output, 0, len(works))
				for _, work := range works {
					messages = append(messages, work.Msg)
				}

				err := f.clients[chainId].SendMsgs(messages)
				if err == nil {
					for _, w := range works {
						// Everything worked, so-- respond successfully to Discord requester
						f.sendReaction(w.DiscordMessage, "✅")
						f.sendMessage(w.DiscordMessage, fmt.Sprintf("Dispensed 💸 `%d` ttnt", w.Msg.Coins.AmountOf("uttnt").Int64()/1_000_000))
					}

				}

				works = make([]*Work, 0)
			}
		case w := <-workCh:
			works = append(works, w)
		}
	}
}

func (f *Faucet) SendTask(chainId string, work *Work) {
	f.works[chainId] <- work
}

func (f *Faucet) sendMessage(m *discordgo.MessageCreate, msg string) error {
	_, err := f.session.ChannelMessageSendReply(m.ChannelID, msg, m.Reference())
	if err != nil {
		return err
	}
	return nil
}

func (f *Faucet) sendReaction(m *discordgo.MessageCreate, reaction string) error {
	err := f.session.MessageReactionAdd(m.ChannelID, m.ID, reaction)
	if err != nil {
		return err
	}
	return nil
}
