package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/umee-network/fonzie/worker"
	"regexp"
	"strings"
	"time"

	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/cosmos/btcutil/bech32"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	log "github.com/sirupsen/logrus"
	chain "github.com/umee-network/fonzie/chain"
)

//go:generate bash -c "if [ \"$CI\" = true ] ; then echo -n $GITHUB_REF_NAME > VERSION; fi"
var (
	//go:embed VERSION
	Version string
)

type CoinsStr = string
type ChainPrefix = string
type Username = string
type ChainFunding = map[ChainPrefix]CoinsStr

type FundingReceipt struct {
	ChainPrefix ChainPrefix
	Username    Username
	FundedAt    time.Time
	Amount      cosmostypes.Coins
}
type FundingReceipts []FundingReceipt

func (receipts *FundingReceipts) Add(newReceipt FundingReceipt) {
	*receipts = append(*receipts, newReceipt)
}

func (receipts *FundingReceipts) FindByChainPrefixAndUsername(prefix ChainPrefix, username Username) *FundingReceipt {
	for _, receipt := range *receipts {
		if receipt.ChainPrefix == prefix && receipt.Username == username {
			return &receipt
		}
	}
	return nil
}

func (receipts *FundingReceipts) Prune(maxAge time.Duration) {
	pruned := FundingReceipts{}

	for _, receipt := range *receipts {
		if time.Now().Before(receipt.FundedAt.Add(maxAge)) {
			pruned = append(pruned, receipt)
		}
	}
	*receipts = pruned
}

var (
	//mnemonic   = os.Getenv("MNEMONIC")
	//keyringDir = os.Getenv("KEYRING_DIR")
	botToken       = os.Getenv("BOT_TOKEN")
	discordChannel = os.Getenv("DISCORD_CHANNEL")
	rawChains      = os.Getenv("CHAINS")
	rawFunding     = os.Getenv("FUNDING")
	chains         chain.Chains
	funding        ChainFunding
	receipts       FundingReceipts
	faucet         worker.Faucet
)

func init() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println(Version)
		os.Exit(0)
	}

	log.SetFormatter(&log.JSONFormatter{})

	//if mnemonic == "" {
	//	log.Fatal("MNEMONIC is invalid")
	//}
	if botToken == "" {
		log.Fatal("BOT_TOKEN is invalid")
	}
	if rawChains == "" {
		log.Fatal("CHAINS config cannot be blank (json array)")
	}
	if discordChannel == "" {
		log.Fatal("DISCORD_CHANNEL is invalid")
	}

	// parse chains config
	err := json.Unmarshal([]byte(rawChains), &chains)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("CHAINS: %#v", chains)
	// parse chains config
	err = json.Unmarshal([]byte(rawFunding), &funding)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("CHAIN_FUNDING: %#v", funding)
}

func main() {

	//ctx := context.Background()
	//err := chains.ImportMnemonic(ctx, mnemonic)
	//if err != nil {
	//	log.Fatal(err)
	//}

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + botToken)
	if err != nil {
		log.Fatal(err)
	}
	// Cleanly close down the Discord session.
	defer dg.Close()

	f := worker.NewFaucet(dg, chains, 100)
	f.Run()

	faucet = *f

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	// In this example, we only care about receiving message events.
	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentDirectMessages

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		log.Fatal(err)
	}

	// Wait here until CTRL-C or other term signal is received.
	log.Info("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	channel, err := s.Channel(m.ChannelID)
	if err != nil {
		return
	}

	if channel.Name != discordChannel {
		return
	}

	// Do we support this command?
	re, err := regexp.Compile(`\$(request|help)(.*)`)
	if err != nil {
		log.Fatal(err)
	}

	// Do we support this bech32 prefix?
	matches := re.FindAllStringSubmatch(m.Content, -1)
	log.Info("%#v\n", matches)
	if len(matches) > 0 {
		cmd := strings.TrimSpace(matches[0][1])
		args := strings.TrimSpace(matches[0][2])
		switch cmd {
		case "request":
			dstAddr := args
			prefix, _, err := bech32.Decode(dstAddr, 1023)
			if err != nil {
				reportError(s, m, err)
				return
			}

			chain := chains.FindByPrefix(prefix)
			if chain == nil {
				reportError(s, m, fmt.Errorf("%s chain prefix is not supported", prefix))
				return
			}

			maxAge := time.Hour * 2
			receipts.Prune(maxAge)
			receipt := receipts.FindByChainPrefixAndUsername(prefix, m.Author.Username)
			if receipt != nil {
				reportError(s, m, fmt.Errorf("You must wait %s until you can get %s funding again", receipt.FundedAt.Add(maxAge).Sub(time.Now()), prefix))
				return
			}

			// Immediately respond to Discord
			sendReaction(s, m, "👍")

			// Sip on the faucet by dstAddr
			coins, err := cosmostypes.ParseCoinsNormalized(funding[prefix])
			if err != nil {
				// fatal because the coins should have been valid in the first place at process start
				log.Fatal(err)
			}

			//err = chain.Send(dstAddr, coins)
			//if err != nil {
			//	reportError(s, m, err)
			//	return
			//}

			faucet.SendTask(chain.Prefix, &worker.Work{
				ChainId: chain.Prefix,
				Msg: banktypes.Output{
					Address: dstAddr,
					Coins:   coins,
				},
				DiscordMessage: m,
			})

			receipts.Add(FundingReceipt{
				ChainPrefix: prefix,
				Username:    m.Author.Username,
				FundedAt:    time.Now(),
				Amount:      coins,
			})

		default:
			help(s, m)
		}
	} else if m.GuildID == "" {
		// If message is DM, respond with help
		help(s, m)
	}
}

func reportError(s *discordgo.Session, m *discordgo.MessageCreate, errToReport error) {
	sendReaction(s, m, "❌")
	sendMessage(s, m, fmt.Sprintf("Error:\n `%s`", errToReport))
}

//go:embed help.md
var helpMsg string

func help(s *discordgo.Session, m *discordgo.MessageCreate) error {
	acc := []string{}
	for _, chain := range chains {
		acc = append(acc, chain.Prefix)
	}
	return sendMessage(s, m, fmt.Sprintf("**Supported address prefixes**: %s.\n\n%s", strings.Join(acc, ", "), helpMsg))
}

func sendMessage(s *discordgo.Session, m *discordgo.MessageCreate, msg string) error {
	_, err := s.ChannelMessageSendReply(m.ChannelID, msg, m.Reference())
	if err != nil {
		log.Error(err)
	}
	return err
}

func sendReaction(s *discordgo.Session, m *discordgo.MessageCreate, reaction string) error {
	err := s.MessageReactionAdd(m.ChannelID, m.ID, reaction)
	if err != nil {
		log.Error(err)
	}
	return err
}
