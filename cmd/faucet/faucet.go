// Copyright 2017 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

// faucet is an Ether faucet backed by a light client.
package main

//go:generate go-bindata -nometadata -o website.go faucet.html
//go:generate gofmt -w -s website.go

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"math"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethstats"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/params/types/ctypes"
	"github.com/ethereum/go-ethereum/params/types/genesisT"
	"github.com/gorilla/websocket"
)

var (
	foundationFlag = flag.Bool("chain.foundation", false, "Configure genesis and bootnodes for foundation chain defaults")
	classicFlag    = flag.Bool("chain.classic", false, "Configure genesis and bootnodes for classic chain defaults")
	mordorFlag     = flag.Bool("chain.mordor", false, "Configure genesis and bootnodes for mordor chain defaults")
	astorFlag      = flag.Bool("chain.astor", false, "Configure genesis and bootnodes for astor chain defaults")
	kottiFlag      = flag.Bool("chain.kotti", false, "Configure genesis and bootnodes for kotti chain defaults")
	testnetFlag    = flag.Bool("chain.testnet", false, "Configure genesis and bootnodes for testnet chain defaults")
	rinkebyFlag    = flag.Bool("chain.rinkeby", false, "Configure genesis and bootnodes for rinkeby chain defaults")
	goerliFlag     = flag.Bool("chain.goerli", false, "Configure genesis and bootnodes for goerli chain defaults")

	attachFlag    = flag.String("attach", "", "Attach to an IPC or WS endpoint")
	attachChainID = flag.Int64("attach.chainid", 0, "Configure fallback chain id value for use in attach mode (used if target does not have value available yet).")

	syncmodeFlag = flag.String("syncmode", "light", "Configure sync mode for faucet's client")
	datadirFlag  = flag.String("datadir", "", "Use a custom datadir")

	genesisFlag = flag.String("genesis", "", "Genesis json file to seed the chain with")
	apiPortFlag = flag.Int("apiport", 8080, "Listener port for the HTTP API connection")
	ethPortFlag = flag.Int("ethport", 30303, "Listener port for the devp2p connection")
	bootFlag    = flag.String("bootnodes", "", "Comma separated bootnode enode URLs to seed with")
	netFlag     = flag.Uint64("network", 0, "Network ID to use for the Ethereum protocol")

	statsFlag = flag.String("ethstats", "", "Ethstats network monitoring auth string")

	netnameFlag = flag.String("faucet.name", "", "Network name to assign to the faucet")
	payoutFlag  = flag.Int("faucet.amount", 1, "Number of Ethers to pay out per user request")
	minutesFlag = flag.Int("faucet.minutes", 1440, "Number of minutes to wait between funding rounds")
	tiersFlag   = flag.Int("faucet.tiers", 3, "Number of funding tiers to enable (x3 time, x2.5 funds)")

	accJSONFlag = flag.String("account.json", "", "Key json file to fund user requests with")
	accPassFlag = flag.String("account.pass", "", "Decryption password to access faucet funds")

	captchaToken  = flag.String("captcha.token", "", "Recaptcha site key to authenticate client side")
	captchaSecret = flag.String("captcha.secret", "", "Recaptcha secret key to authenticate server side")

	noauthFlag = flag.Bool("noauth", false, "Enables funding requests without authentication")
	logFlag    = flag.Int("loglevel", 3, "Log level to use for Ethereum and the faucet")

	twitterTokenFlag   = flag.String("twitter.token", "", "Bearer token to authenticate with the v2 Twitter API")
	twitterTokenV1Flag = flag.String("twitter.token.v1", "", "Bearer token to authenticate with the v1.1 Twitter API")
)

var chainFlags = []*bool{
	foundationFlag,
	classicFlag,
	mordorFlag,
	kottiFlag,
	testnetFlag,
	rinkebyFlag,
	goerliFlag,
}

var (
	ether = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
)

var (
	gitCommit = "" // Git SHA1 commit hash of the release (set via linker flags)
	gitDate   = "" // Git commit date YYYYMMDD of the release (set via linker flags)
)

func faucetDirFromChainIndicators(chainID uint64, genesisHash common.Hash) string {
	datadir := filepath.Join(os.Getenv("HOME"), ".faucet")
	if *datadirFlag != "" {
		datadir = *datadirFlag
	}
	switch genesisHash {
	case params.MainnetGenesisHash:
		if chainID == params.ClassicChainConfig.GetChainID().Uint64() {
			return filepath.Join(datadir, "classic")
		}
		return filepath.Join(datadir, "")
	case params.RopstenGenesisHash:
		return filepath.Join(datadir, "ropsten")
	case params.RinkebyGenesisHash:
		return filepath.Join(datadir, "rinkeby")
	case params.GoerliGenesisHash:
		return filepath.Join(datadir, "goerli")
	case params.KottiGenesisHash:
		return filepath.Join(datadir, "kotti")
	case params.MordorGenesisHash:
		return filepath.Join(datadir, "mordor")
	case params.AstorGenesisHash:
		return filepath.Join(datadir, "astor")
	}
	return datadir
}

func parseChainFlags() (gs *genesisT.Genesis, bs string, netid uint64) {
	var configs = []struct {
		flag bool
		gs   *genesisT.Genesis
		bs   []string
	}{
		{*foundationFlag, params.DefaultGenesisBlock(), nil},
		{*classicFlag, params.DefaultClassicGenesisBlock(), nil},
		{*mordorFlag, params.DefaultMordorGenesisBlock(), nil},
		{*astorFlag, params.DefaultAstorGenesisBlock(), nil},
		{*testnetFlag, params.DefaultRopstenGenesisBlock(), nil},
		{*rinkebyFlag, params.DefaultRinkebyGenesisBlock(), nil},
		{*kottiFlag, params.DefaultKottiGenesisBlock(), nil},
		{*goerliFlag, params.DefaultGoerliGenesisBlock(), nil},
	}

	var bss []string
	for _, conf := range configs {
		if conf.flag {
			gs, bss, netid = conf.gs, conf.bs, *conf.gs.Config.GetNetworkID()
			break
		}
	}
	if len(bss) > 0 {
		bs = strings.Join(bss, ",")
	}

	// allow overrides
	if *genesisFlag != "" {
		blob, err := ioutil.ReadFile(*genesisFlag)
		if err != nil {
			log.Crit("Failed to read genesis block contents", "genesis", *genesisFlag, "err", err)
		}
		gs = new(genesisT.Genesis)
		if err = json.Unmarshal(blob, gs); err != nil {
			log.Crit("Failed to parse genesis block json", "err", err)
		}
	}
	if *bootFlag != "" {
		bs = *bootFlag
	}
	if *netFlag != 0 {
		netid = *netFlag
	}
	return
}

// auditFlagUse ensures that exclusive/incompatible flag values are not set.
// If invalid use if found, the program exits with log.Crit.
func auditFlagUse() {
	var (
		ffalse          = false
		activeChainFlag = &ffalse
	)
	for _, f := range chainFlags {
		if *f {
			if *activeChainFlag {
				log.Crit("cannot use two -chain.* flags simultaneously")
			}
			activeChainFlag = f
		}
	}

	if !*activeChainFlag && *genesisFlag == "" && *attachFlag == "" {
		log.Crit("missing chain configuration option; use one of -chain.<identity>, -genesis, or -attach")
	}

	type fflag struct {
		name  string
		value interface{}
	}
	exclusiveOrCrit := func(a, b fflag) {
		didSetOne := false
		for _, f := range []fflag{a, b} {
			isSet := false
			switch t := f.value.(type) {
			case *string:
				isSet = *t != ""
			case *bool:
				isSet = *t
			case bool:
				isSet = t
			case *int64:
				isSet = *t != 0
			case *uint64:
				isSet = *t != 0
			case *int:
				isSet = *t != 0
			default:
				panic(fmt.Sprintf("unhandled flag type case: %v %t %s", t, t, f.name))
			}
			if didSetOne && isSet {
				var av, bv interface{}
				av = a.value
				bv = b.value
				if reflect.TypeOf(a.value).Kind() == reflect.Ptr {
					av = reflect.ValueOf(a.value).Elem()
				}
				if reflect.TypeOf(b.value).Kind() == reflect.Ptr {
					bv = reflect.ValueOf(b.value).Elem()
				}
				log.Crit("flags are exclusive", "flags", []string{a.name, b.name}, "values", []interface{}{av, bv})
			}
			didSetOne = isSet
		}
	}
	for _, exclusivePair := range [][]fflag{
		{{"attach", attachFlag}, {"chain.<identity>", activeChainFlag}},
		{{"attach", attachFlag}, {"genesis", genesisFlag}},
		{{"attach", attachFlag}, {"ethstats", statsFlag}},
		{{"attach", attachFlag}, {"network", netFlag}},
		{{"attach", attachFlag}, {"bootnodes", bootFlag}},
	} {
		exclusiveOrCrit(exclusivePair[0], exclusivePair[1])
	}

	if *attachFlag == "" && *attachChainID != 0 {
		log.Crit("must use -attach when using -attach.chainid")
	}

	if *syncmodeFlag != "" {
		allowedModes := []string{"light", "fast", "full"}
		var ok bool
		for _, mode := range allowedModes {
			if mode == *syncmodeFlag {
				ok = true
				break
			}
		}
		if !ok {
			log.Crit("invalid value for -syncmode", "value", *syncmodeFlag, "allowed", allowedModes)
		}
	}
}

func main() {
	// Parse the flags and set up the logger to print everything requested
	flag.Parse()
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(*logFlag), log.StreamHandler(os.Stderr, log.TerminalFormat(true))))

	auditFlagUse()

	// Load and parse the genesis block requested by the user
	var genesis *genesisT.Genesis
	var enodes []*enode.Node
	var blob []byte

	// client will be used if the faucet is attaching. If not it won't be touched.
	var client *ethclient.Client
	genesis, *bootFlag, *netFlag = parseChainFlags()

	// Construct the payout tiers
	amounts := make([]string, *tiersFlag)
	periods := make([]string, *tiersFlag)
	for i := 0; i < *tiersFlag; i++ {
		// Calculate the amount for the next tier and format it
		amount := float64(*payoutFlag) * math.Pow(2.5, float64(i))
		amounts[i] = fmt.Sprintf("%s Ethers", strconv.FormatFloat(amount, 'f', -1, 64))
		if amount == 1 {
			amounts[i] = strings.TrimSuffix(amounts[i], "s")
		}
		// Calculate the period for the next tier and format it
		period := *minutesFlag * int(math.Pow(3, float64(i)))
		periods[i] = fmt.Sprintf("%d mins", period)
		if period%60 == 0 {
			period /= 60
			periods[i] = fmt.Sprintf("%d hours", period)

			if period%24 == 0 {
				period /= 24
				periods[i] = fmt.Sprintf("%d days", period)
			}
		}
		if period == 1 {
			periods[i] = strings.TrimSuffix(periods[i], "s")
		}
	}
	// Load up and render the faucet website
	tmpl, err := Asset("faucet.html")
	if err != nil {
		log.Crit("Failed to load the faucet template", "err", err)
	}
	website := new(bytes.Buffer)
	err = template.Must(template.New("").Parse(string(tmpl))).Execute(website, map[string]interface{}{
		"Network":   *netnameFlag,
		"Amounts":   amounts,
		"Periods":   periods,
		"Recaptcha": *captchaToken,
		"NoAuth":    *noauthFlag,
	})
	if err != nil {
		log.Crit("Failed to render the faucet template", "err", err)
	}

	if genesis != nil {
		log.Info("Using chain/net config", "network id", *netFlag, "bootnodes", *bootFlag, "chain config", fmt.Sprintf("%v", genesis.Config))

		// Convert the bootnodes to internal enode representations
		var enodes []*enode.Node
		for _, boot := range strings.Split(*bootFlag, ",") {
			if url, err := enode.Parse(enode.ValidSchemes, boot); err == nil {
				enodes = append(enodes, url)
			} else {
				log.Error("Failed to parse bootnode URL", "url", boot, "err", err)
			}
		}
	} else {
		log.Info("Attaching faucet to running client")
		client, err = ethclient.DialContext(context.Background(), *attachFlag)
		if err != nil {
			log.Crit("Failed to connect to client", "error", err)
		}
	}

	// Load up the account key and decrypt its password
	if blob, err = ioutil.ReadFile(*accPassFlag); err != nil {
		log.Crit("Failed to read account password contents", "file", *accPassFlag, "err", err)
	}
	pass := strings.TrimSuffix(string(blob), "\n")

	// Get the chain id from the genesis or the designated API.
	// We'll use this to infer the keystore and chain data directories.
	// NOTE: Relying on having chain id immediately available may be fragile.
	// IRC eth_chainId can possibly not return the configured value until the block activating the chain id EIP155 is reached.
	// See the difference in implementation between ethapi/api.go and eth/api.go #ChainID() methods.
	// There's an issue open about this somewhere at ethereum/xxx.
	// This could be resolved by creating a -chainid flag to use as a fallback.
	// NOTE(meowsbits): chainID and genesisHash are ONLY used as input for configuring the
	// default data directory. This logic must be bypassed when and if a -datadir flag were in use.
	chainID := uint64(0)
	var genesisHash common.Hash
	if genesis != nil {
		chainID = genesis.GetChainID().Uint64()
		genesisHash = core.GenesisToBlock(genesis, nil).Hash()
	} else {
		cid, err := client.ChainID(context.Background())
		if err != nil {
			log.Crit("Failed to get chain id from client", "error", err)
		}
		genesisBlock, err := client.BlockByNumber(context.Background(), big.NewInt(0))
		if err != nil {
			log.Crit("Failed to get genesis block from client", "error", err)
		}
		chainID = cid.Uint64()
		genesisHash = genesisBlock.Hash()

		// ChainID is only REQUIRED to disambiguate ETH/ETC chains.
		if chainID == 0 && genesisHash == params.MainnetGenesisHash {
			if *attachChainID == 0 {
				// Exit with error if disambiguating fallback is unset.
				log.Crit("Ambiguous/unavailable chain identity", "recommended solution", "use -attach.chainid to configure a fallback or wait until target client is synced past EIP155 block height")
			}
			chainID = uint64(*attachChainID)
		}
	}

	keystorePath := filepath.Join(faucetDirFromChainIndicators(chainID, genesisHash), "keys")
	ks := keystore.NewKeyStore(keystorePath, keystore.StandardScryptN, keystore.StandardScryptP)
	if blob, err = ioutil.ReadFile(*accJSONFlag); err != nil {
		log.Crit("Failed to read account key contents", "file", *accJSONFlag, "err", err)
	}
	acc, err := ks.Import(blob, pass, pass)
	if err != nil && err != keystore.ErrAccountAlreadyExists {
		log.Crit("Failed to import faucet signer account", "err", err)
	} else if err == nil {
		log.Info("Imported faucet signer account", "address", acc.Address)
	}
	if err := ks.Unlock(acc, pass); err != nil {
		log.Crit("Failed to unlock faucet signer account", "err", err)
	}

	// Assemble and start the faucet light service
	// faucet, err := newFaucet(genesis, *ethPortFlag, enodes, *netFlag, *statsFlag, ks, website.Bytes())
	faucet, err := newFaucet(ks, website.Bytes())
	if err != nil {
		log.Crit("Failed to construct faucet", "err", err)
	}
	defer faucet.close()

	if genesis != nil {
		log.Info("Starting faucet client stack")
		err = faucet.startStack(genesis, *ethPortFlag, enodes, *netFlag)
		if err != nil {
			log.Crit("Failed to start to stack", "error", err)
		}
	} else {
		faucet.client = client
	}

	if err := faucet.listenAndServe(*apiPortFlag); err != nil {
		log.Crit("Failed to launch faucet API", "err", err)
	}
}

// request represents an accepted funding request.
type request struct {
	Avatar  string             `json:"avatar"`  // Avatar URL to make the UI nicer
	Account common.Address     `json:"account"` // Ethereum address being funded
	Time    time.Time          `json:"time"`    // Timestamp when the request was accepted
	Tx      *types.Transaction `json:"tx"`      // Transaction funding the account
}

// faucet represents a crypto faucet backed by an Ethereum light client.
type faucet struct {
	config ctypes.ChainConfigurator // Chain configurations for signing
	stack  *node.Node               // Ethereum protocol stack
	client *ethclient.Client        // Client connection to the Ethereum chain
	index  []byte                   // Index page to serve up on the web

	keystore *keystore.KeyStore // Keystore containing the single signer
	account  accounts.Account   // Account funding user faucet requests
	head     *types.Header      // Current head header of the faucet
	balance  *big.Int           // Current balance of the faucet
	nonce    uint64             // Current pending nonce of the faucet
	price    *big.Int           // Current gas price to issue funds with

	conns    []*wsConn            // Currently live websocket connections
	timeouts map[string]time.Time // History of users and their funding timeouts
	reqs     []*request           // Currently pending funding requests
	update   chan struct{}        // Channel to signal request updates

	lock sync.RWMutex // Lock protecting the faucet's internals
}

// wsConn wraps a websocket connection with a write mutex as the underlying
// websocket library does not synchronize access to the stream.
type wsConn struct {
	conn  *websocket.Conn
	wlock sync.Mutex
}

const (
	multiFaucetNodeName = "MultiFaucet"
	coreFaucetNodeName  = "CoreFaucet"
)

// migrateFaucetDirectory moves an existing "MultiFaucet" directory to the new "CoreFaucet".
// It only returns an error if the migration is attempted and fails.
// If no 'old' directory is found to migrate (ie DNE), it returns nil (no error).
func migrateFaucetDirectory(faucetDataDir string) error {
	oldFaucetNodePath := filepath.Join(faucetDataDir, multiFaucetNodeName)
	targetFaucetNodePath := filepath.Join(faucetDataDir, coreFaucetNodeName)

	log.Info("Checking datadir migration", "datadir", faucetDataDir)

	d, err := os.Stat(oldFaucetNodePath)
	if err != nil && os.IsNotExist(err) {
		// This is an expected positive error.
		return nil
	}
	if err == nil && d.IsDir() {

		// Path exists and is a directory.
		log.Warn("Found existing 'MultiFaucet' directory, migrating", "old", oldFaucetNodePath, "new", targetFaucetNodePath)
		if err := os.Rename(oldFaucetNodePath, targetFaucetNodePath); err != nil {
			return err
		}
		return nil
	}
	if err == nil && !d.IsDir() {
		return errors.New("expected a directory at faucet node datadir path, found non-directory")
	}
	// Return any error which is not expected, eg. bad permissions.
	return err
}

// startStack starts the node stack, ensures peering, and assigns the respective ethclient to the faucet.
func (f *faucet) startStack(genesis *genesisT.Genesis, port int, enodes []*enode.Node, network uint64) error {

	genesisHash := core.GenesisToBlock(genesis, nil).Hash()

	faucetDataDir := faucetDirFromChainIndicators(genesis.Config.GetChainID().Uint64(), genesisHash)

	// Handle renaming of existing directory from old name to new name.
	if err := migrateFaucetDirectory(faucetDataDir); err != nil {
		log.Crit("Migration handling of faucet datadir failed", "error", err)
	}

	// Assemble the raw devp2p protocol stack
	stack, err := node.New(&node.Config{
		Name:    coreFaucetNodeName,
		Version: params.VersionWithCommit(gitCommit, gitDate),
		DataDir: faucetDataDir,
		P2P: p2p.Config{
			NAT: nat.Any(),
			// NoDiscovery is DISABLED to allow the node the find peers without relying on manually configured bootnodes.
			// NoDiscovery:      true,
			DiscoveryV5:      true,
			ListenAddr:       fmt.Sprintf(":%d", port),
			MaxPeers:         25,
			BootstrapNodesV5: enodes,
		},
	})
	if err != nil {
		return err
	}

	// Assemble the Ethereum light client protocol
	cfg := ethconfig.Defaults
	cfg.NetworkId = network
	cfg.Genesis = genesis

	switch *syncmodeFlag {
	case "light":
		cfg.SyncMode = downloader.LightSync
	case "fast":
		cfg.SyncMode = downloader.FastSync
		cfg.ProtocolVersions = ethconfig.Defaults.ProtocolVersions
	case "full":
		cfg.SyncMode = downloader.FullSync
		cfg.ProtocolVersions = ethconfig.Defaults.ProtocolVersions
	default:
		panic("impossible to reach, this should be handled in the auditFlagUse function")
	}

	// Note that we have to set the discovery configs AFTER establishing the configuration
	// sync mode because discovery setting depend on light vs. fast/full.
	switch genesisHash {
	case params.MainnetGenesisHash:
		if genesis.GetChainID().Uint64() == params.DefaultClassicGenesisBlock().GetChainID().Uint64() {
			utils.SetDNSDiscoveryDefaults2(&cfg, params.ClassicDNSNetwork1)
		} else {
			utils.SetDNSDiscoveryDefaults(&cfg, core.GenesisToBlock(genesis, nil).Hash())
		}
	case params.KottiGenesisHash:
		utils.SetDNSDiscoveryDefaults2(&cfg, params.KottiDNSNetwork1)
	case params.MordorGenesisHash:
		utils.SetDNSDiscoveryDefaults2(&cfg, params.MordorDNSNetwork1)
	default:
		utils.SetDNSDiscoveryDefaults(&cfg, core.GenesisToBlock(genesis, nil).Hash())
	}
	log.Info("Config discovery", "urls", cfg.EthDiscoveryURLs)

	// Establish the backend and enable stats reporting if configured to do so.
	switch *syncmodeFlag {
	case "light":
		lesBackend, err := les.New(stack, &cfg)
		if err != nil {
			return fmt.Errorf("Failed to register the Ethereum service: %w", err)
		}
		if *statsFlag != "" {
			if err := ethstats.New(stack, lesBackend.ApiBackend, lesBackend.Engine(), *statsFlag); err != nil {
				return err
			}
		}
	case "fast", "full":
		ethBackend, err := eth.New(stack, &cfg)
		if err != nil {
			return fmt.Errorf("Failed to register the Ethereum service: %w", err)
		}
		if *statsFlag != "" {
			if err := ethstats.New(stack, ethBackend.APIBackend, ethBackend.Engine(), *statsFlag); err != nil {
				return err
			}
		}
	default:
		panic("impossible to reach, this should be handled in the auditFlagUse function")
	}

	// Boot up the client and ensure it connects to bootnodes
	if err := stack.Start(); err != nil {
		return err
	}
	for _, boot := range enodes {
		old, err := enode.Parse(enode.ValidSchemes, boot.String())
		if err == nil {
			log.Info("Manually adding bootnode", "enode", old.String())
			stack.Server().AddPeer(old)
		}
	}
	// Attach to the client and retrieve and interesting metadatas
	api, err := stack.Attach()
	if err != nil {
		stack.Close()
		return err
	}
	f.stack = stack
	f.client = ethclient.NewClient(api)
	return nil
}

func newFaucet(ks *keystore.KeyStore, index []byte) (*faucet, error) {
	f := &faucet{
		// config:   genesis.Config,
		// stack:    stack,
		// client:   client,
		index:    index,
		keystore: ks,
		account:  ks.Accounts()[0],
		timeouts: make(map[string]time.Time),
		update:   make(chan struct{}, 1),
	}
	return f, nil
}

// close terminates the Ethereum connection and tears down the faucet.
func (f *faucet) close() error {
	if f.stack != nil {
		return f.stack.Close()
	}
	f.client.Close()
	return nil
}

// listenAndServe registers the HTTP handlers for the faucet and boots it up
// for service user funding requests.
func (f *faucet) listenAndServe(port int) error {
	go f.loop()

	http.HandleFunc("/", f.webHandler)
	http.HandleFunc("/api", f.apiHandler)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

// webHandler handles all non-api requests, simply flattening and returning the
// faucet website.
func (f *faucet) webHandler(w http.ResponseWriter, r *http.Request) {
	w.Write(f.index)
}

// apiHandler handles requests for Ether grants and transaction statuses.
func (f *faucet) apiHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	// Start tracking the connection and drop at the end
	defer conn.Close()

	f.lock.Lock()
	wsconn := &wsConn{conn: conn}
	f.conns = append(f.conns, wsconn)
	f.lock.Unlock()

	defer func() {
		f.lock.Lock()
		for i, c := range f.conns {
			if c.conn == conn {
				f.conns = append(f.conns[:i], f.conns[i+1:]...)
				break
			}
		}
		f.lock.Unlock()
	}()
	// Gather the initial stats from the network to report
	var (
		head    *types.Header
		balance *big.Int
		nonce   uint64
	)
	for head == nil || balance == nil {
		// Retrieve the current stats cached by the faucet
		f.lock.RLock()
		if f.head != nil {
			head = types.CopyHeader(f.head)
		}
		if f.balance != nil {
			balance = new(big.Int).Set(f.balance)
		}
		nonce = f.nonce
		f.lock.RUnlock()

		if head == nil || balance == nil {
			// Report the faucet offline until initial stats are ready
			//lint:ignore ST1005 This error is to be displayed in the browser
			if err = sendError(wsconn, errors.New("Faucet offline")); err != nil {
				log.Warn("Failed to send faucet error to client", "err", err)
				return
			}
			time.Sleep(3 * time.Second)
		}
	}
	// Send over the initial stats and the latest header
	f.lock.RLock()
	reqs := f.reqs
	f.lock.RUnlock()
	peerCount, err := f.client.PeerCount(context.Background())
	if err != nil {
		log.Warn("Failed to get peer count", "error", err)
		return
	}
	if err = send(wsconn, map[string]interface{}{
		"funds":    new(big.Int).Div(balance, ether),
		"funded":   nonce,
		"peers":    peerCount,
		"requests": reqs,
	}, 3*time.Second); err != nil {
		log.Warn("Failed to send initial stats to client", "err", err)
		return
	}
	if err = send(wsconn, head, 3*time.Second); err != nil {
		log.Warn("Failed to send initial header to client", "err", err)
		return
	}
	// Keep reading requests from the websocket until the connection breaks
	for {
		// Fetch the next funding request and validate against github
		var msg struct {
			URL     string `json:"url"`
			Tier    uint   `json:"tier"`
			Captcha string `json:"captcha"`
		}
		if err = conn.ReadJSON(&msg); err != nil {
			return
		}
		if !*noauthFlag && !strings.HasPrefix(msg.URL, "https://twitter.com/") && !strings.HasPrefix(msg.URL, "https://www.facebook.com/") {
			if err = sendError(wsconn, errors.New("URL doesn't link to supported services")); err != nil {
				log.Warn("Failed to send URL error to client", "err", err)
				return
			}
			continue
		}
		if msg.Tier >= uint(*tiersFlag) {
			//lint:ignore ST1005 This error is to be displayed in the browser
			if err = sendError(wsconn, errors.New("Invalid funding tier requested")); err != nil {
				log.Warn("Failed to send tier error to client", "err", err)
				return
			}
			continue
		}
		log.Info("Faucet funds requested", "url", msg.URL, "tier", msg.Tier)

		// If captcha verifications are enabled, make sure we're not dealing with a robot
		if *captchaToken != "" {
			form := url.Values{}
			form.Add("secret", *captchaSecret)
			form.Add("response", msg.Captcha)

			res, err := http.PostForm("https://www.google.com/recaptcha/api/siteverify", form)
			if err != nil {
				if err = sendError(wsconn, err); err != nil {
					log.Warn("Failed to send captcha post error to client", "err", err)
					return
				}
				continue
			}
			var result struct {
				Success bool            `json:"success"`
				Errors  json.RawMessage `json:"error-codes"`
			}
			err = json.NewDecoder(res.Body).Decode(&result)
			res.Body.Close()
			if err != nil {
				if err = sendError(wsconn, err); err != nil {
					log.Warn("Failed to send captcha decode error to client", "err", err)
					return
				}
				continue
			}
			if !result.Success {
				log.Warn("Captcha verification failed", "err", string(result.Errors))
				//lint:ignore ST1005 it's funny and the robot won't mind
				if err = sendError(wsconn, errors.New("Beep-bop, you're a robot!")); err != nil {
					log.Warn("Failed to send captcha failure to client", "err", err)
					return
				}
				continue
			}
		}
		// Retrieve the Ethereum address to fund, the requesting user and a profile picture
		var (
			id       string
			username string
			avatar   string
			address  common.Address
		)
		switch {
		case strings.HasPrefix(msg.URL, "https://twitter.com/"):
			id, username, avatar, address, err = authTwitter(msg.URL, *twitterTokenV1Flag, *twitterTokenFlag)
		case strings.HasPrefix(msg.URL, "https://www.facebook.com/"):
			username, avatar, address, err = authFacebook(msg.URL)
			id = username
		case *noauthFlag:
			username, avatar, address, err = authNoAuth(msg.URL)
			id = username
		default:
			//lint:ignore ST1005 This error is to be displayed in the browser
			err = errors.New("Something funky happened, please open an issue at https://github.com/ethereum/go-ethereum/issues")
		}
		if err != nil {
			if err = sendError(wsconn, err); err != nil {
				log.Warn("Failed to send prefix error to client", "err", err)
				return
			}
			continue
		}
		log.Info("Faucet request valid", "url", msg.URL, "tier", msg.Tier, "user", username, "address", address)

		// Ensure the user didn't request funds too recently
		f.lock.Lock()
		var (
			fund    bool
			timeout time.Time
		)
		if timeout = f.timeouts[id]; time.Now().After(timeout) {
			// User wasn't funded recently, create the funding transaction
			amount := new(big.Int).Mul(big.NewInt(int64(*payoutFlag)), ether)
			amount = new(big.Int).Mul(amount, new(big.Int).Exp(big.NewInt(5), big.NewInt(int64(msg.Tier)), nil))
			amount = new(big.Int).Div(amount, new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(msg.Tier)), nil))

			tx := types.NewTransaction(f.nonce+uint64(len(f.reqs)), address, amount, 21000, f.price, nil)

			// FIXME(meowsbits): Getting the chain id more than once is redundant and can be optimized.
			chainId, err := f.client.ChainID(context.Background())
			if err != nil {
				log.Warn("Failed to get chain id", "error", err)
				return
			}

			signed, err := f.keystore.SignTx(f.account, tx, chainId)
			if err != nil {
				f.lock.Unlock()
				if err = sendError(wsconn, err); err != nil {
					log.Warn("Failed to send transaction creation error to client", "err", err)
					return
				}
				continue
			}
			// Submit the transaction and mark as funded if successful
			if err := f.client.SendTransaction(context.Background(), signed); err != nil {
				f.lock.Unlock()
				if err = sendError(wsconn, err); err != nil {
					log.Warn("Failed to send transaction transmission error to client", "err", err)
					return
				}
				continue
			}
			f.reqs = append([]*request{{
				Avatar:  avatar,
				Account: address,
				Time:    time.Now(),
				Tx:      signed,
			}}, f.reqs...)
			timeout := time.Duration(*minutesFlag*int(math.Pow(3, float64(msg.Tier)))) * time.Minute
			grace := timeout / 288 // 24h timeout => 5m grace

			f.timeouts[id] = time.Now().Add(timeout - grace)
			fund = true
		}
		f.lock.Unlock()

		// Send an error if too frequent funding, othewise a success
		if !fund {
			if err = sendError(wsconn, fmt.Errorf("%s left until next allowance", common.PrettyDuration(time.Until(timeout)))); err != nil { // nolint: gosimple
				log.Warn("Failed to send funding error to client", "err", err)
				return
			}
			continue
		}
		if err = sendSuccess(wsconn, fmt.Sprintf("Funding request accepted for %s into %s", username, address.Hex())); err != nil {
			log.Warn("Failed to send funding success to client", "err", err)
			return
		}
		select {
		case f.update <- struct{}{}:
		default:
		}
	}
}

// refresh attempts to retrieve the latest header from the chain and extract the
// associated faucet balance and nonce for connectivity caching.
func (f *faucet) refresh(head *types.Header) error {
	// Ensure a state update does not run for too long
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// If no header was specified, use the current chain head
	var err error
	if head == nil {
		if head, err = f.client.HeaderByNumber(ctx, nil); err != nil {
			return err
		}
	}
	// Retrieve the balance, nonce and gas price from the current head
	var (
		balance *big.Int
		nonce   uint64
		price   *big.Int
	)
	if balance, err = f.client.BalanceAt(ctx, f.account.Address, head.Number); err != nil {
		return err
	}
	if nonce, err = f.client.NonceAt(ctx, f.account.Address, head.Number); err != nil {
		return err
	}
	if price, err = f.client.SuggestGasPrice(ctx); err != nil {
		return err
	}
	// Everything succeeded, update the cached stats and eject old requests
	f.lock.Lock()
	f.head, f.balance = head, balance
	f.price, f.nonce = price, nonce
	for len(f.reqs) > 0 && f.reqs[0].Tx.Nonce() < f.nonce {
		f.reqs = f.reqs[1:]
	}
	f.lock.Unlock()

	return nil
}

// loop keeps waiting for interesting events and pushes them out to connected
// websockets.
func (f *faucet) loop() {
	// Wait for chain events and push them to clients
	heads := make(chan *types.Header, 16)
	sub, err := f.client.SubscribeNewHead(context.Background(), heads)
	if err != nil {
		log.Crit("Failed to subscribe to head events", "err", err)
	}
	defer sub.Unsubscribe()

	// Start a goroutine to update the state from head notifications in the background
	update := make(chan *types.Header)

	go func() {
		for head := range update {
			// New chain head arrived, query the current stats and stream to clients
			timestamp := time.Unix(int64(head.Time), 0)
			if age := time.Since(timestamp); age > time.Hour {
				log.Trace("Skipping faucet refresh, head too old", "number", head.Number, "hash", head.Hash(), "age", common.PrettyAge(timestamp))
				continue
			}
			if err := f.refresh(head); err != nil {
				log.Warn("Failed to update faucet state", "block", head.Number, "hash", head.Hash(), "err", err)
				continue
			}
			// Faucet state retrieved, update locally and send to clients
			f.lock.RLock()
			log.Info("Updated faucet state", "number", head.Number, "hash", head.Hash(), "age", common.PrettyAge(timestamp), "balance", f.balance, "nonce", f.nonce, "price", f.price)

			balance := new(big.Int).Div(f.balance, ether)
			peerCount, err := f.client.PeerCount(context.Background())
			if err != nil {
				log.Warn("Failed to get peer count", "error", err)
				continue
			}

			for _, conn := range f.conns {
				if err := send(conn, map[string]interface{}{
					"funds":    balance,
					"funded":   f.nonce,
					"peers":    peerCount,
					"requests": f.reqs,
				}, time.Second); err != nil {
					log.Warn("Failed to send stats to client", "err", err)
					conn.conn.Close()
					continue
				}
				if err := send(conn, head, time.Second); err != nil {
					log.Warn("Failed to send header to client", "err", err)
					conn.conn.Close()
				}
			}
			f.lock.RUnlock()
		}
	}()
	// Wait for various events and assing to the appropriate background threads
	for {
		select {
		case head := <-heads:
			// New head arrived, send if for state update if there's none running
			select {
			case update <- head:
			default:
			}

		case <-f.update:
			// Pending requests updated, stream to clients
			f.lock.RLock()
			for _, conn := range f.conns {
				if err := send(conn, map[string]interface{}{"requests": f.reqs}, time.Second); err != nil {
					log.Warn("Failed to send requests to client", "err", err)
					conn.conn.Close()
				}
			}
			f.lock.RUnlock()
		case err := <-sub.Err():
			if *attachFlag != "" {
				log.Crit("Connection with the attached client has been lost", "err", err)
			}
		}
	}
}

// sends transmits a data packet to the remote end of the websocket, but also
// setting a write deadline to prevent waiting forever on the node.
func send(conn *wsConn, value interface{}, timeout time.Duration) error {
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	conn.wlock.Lock()
	defer conn.wlock.Unlock()
	conn.conn.SetWriteDeadline(time.Now().Add(timeout))
	return conn.conn.WriteJSON(value)
}

// sendError transmits an error to the remote end of the websocket, also setting
// the write deadline to 1 second to prevent waiting forever.
func sendError(conn *wsConn, err error) error {
	return send(conn, map[string]string{"error": err.Error()}, time.Second)
}

// sendSuccess transmits a success message to the remote end of the websocket, also
// setting the write deadline to 1 second to prevent waiting forever.
func sendSuccess(conn *wsConn, msg string) error {
	return send(conn, map[string]string{"success": msg}, time.Second)
}

// authTwitter tries to authenticate a faucet request using Twitter posts, returning
// the uniqueness identifier (user id/username), username, avatar URL and Ethereum address to fund on success.
func authTwitter(url string, tokenV1, tokenV2 string) (string, string, string, common.Address, error) {
	// Ensure the user specified a meaningful URL, no fancy nonsense
	parts := strings.Split(url, "/")
	if len(parts) < 4 || parts[len(parts)-2] != "status" {
		//lint:ignore ST1005 This error is to be displayed in the browser
		return "", "", "", common.Address{}, errors.New("Invalid Twitter status URL")
	}
	// Strip any query parameters from the tweet id and ensure it's numeric
	tweetID := strings.Split(parts[len(parts)-1], "?")[0]
	if !regexp.MustCompile("^[0-9]+$").MatchString(tweetID) {
		return "", "", "", common.Address{}, errors.New("Invalid Tweet URL")
	}
	// Twitter's API isn't really friendly with direct links.
	// It is restricted to 300 queries / 15 minute with an app api key.
	// Anything more will require read only authorization from the users and that we want to avoid.

	// If Twitter bearer token is provided, use the API, selecting the version
	// the user would prefer (currently there's a limit of 1 v2 app / developer
	// but unlimited v1.1 apps).
	switch {
	case tokenV1 != "":
		return authTwitterWithTokenV1(tweetID, tokenV1)
	case tokenV2 != "":
		return authTwitterWithTokenV2(tweetID, tokenV2)
	}
	// Twiter API token isn't provided so we just load the public posts
	// and scrape it for the Ethereum address and profile URL. We need to load
	// the mobile page though since the main page loads tweet contents via JS.
	url = strings.Replace(url, "https://twitter.com/", "https://mobile.twitter.com/", 1)

	res, err := http.Get(url)
	if err != nil {
		return "", "", "", common.Address{}, err
	}
	defer res.Body.Close()

	// Resolve the username from the final redirect, no intermediate junk
	parts = strings.Split(res.Request.URL.String(), "/")
	if len(parts) < 4 || parts[len(parts)-2] != "status" {
		//lint:ignore ST1005 This error is to be displayed in the browser
		return "", "", "", common.Address{}, errors.New("Invalid Twitter status URL")
	}
	username := parts[len(parts)-3]

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", "", "", common.Address{}, err
	}
	address := common.HexToAddress(string(regexp.MustCompile("0x[0-9a-fA-F]{40}").Find(body)))
	if address == (common.Address{}) {
		//lint:ignore ST1005 This error is to be displayed in the browser
		return "", "", "", common.Address{}, errors.New("No Ethereum address found to fund")
	}
	var avatar string
	if parts = regexp.MustCompile("src=\"([^\"]+twimg.com/profile_images[^\"]+)\"").FindStringSubmatch(string(body)); len(parts) == 2 {
		avatar = parts[1]
	}
	return username + "@twitter", username, avatar, address, nil
}

// authTwitterWithTokenV1 tries to authenticate a faucet request using Twitter's v1
// API, returning the user id, username, avatar URL and Ethereum address to fund on
// success.
func authTwitterWithTokenV1(tweetID string, token string) (string, string, string, common.Address, error) {
	// Query the tweet details from Twitter
	url := fmt.Sprintf("https://api.twitter.com/1.1/statuses/show.json?id=%s", tweetID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", "", common.Address{}, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", "", common.Address{}, err
	}
	defer res.Body.Close()

	var result struct {
		Text string `json:"text"`
		User struct {
			ID       string `json:"id_str"`
			Username string `json:"screen_name"`
			Avatar   string `json:"profile_image_url"`
		} `json:"user"`
	}
	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		return "", "", "", common.Address{}, err
	}
	address := common.HexToAddress(regexp.MustCompile("0x[0-9a-fA-F]{40}").FindString(result.Text))
	if address == (common.Address{}) {
		//lint:ignore ST1005 This error is to be displayed in the browser
		return "", "", "", common.Address{}, errors.New("No Ethereum address found to fund")
	}
	return result.User.ID + "@twitter", result.User.Username, result.User.Avatar, address, nil
}

// authTwitterWithTokenV2 tries to authenticate a faucet request using Twitter's v2
// API, returning the user id, username, avatar URL and Ethereum address to fund on
// success.
func authTwitterWithTokenV2(tweetID string, token string) (string, string, string, common.Address, error) {
	// Query the tweet details from Twitter
	url := fmt.Sprintf("https://api.twitter.com/2/tweets/%s?expansions=author_id&user.fields=profile_image_url", tweetID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", "", common.Address{}, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", "", common.Address{}, err
	}
	defer res.Body.Close()

	var result struct {
		Data struct {
			AuthorID string `json:"author_id"`
			Text     string `json:"text"`
		} `json:"data"`
		Includes struct {
			Users []struct {
				ID       string `json:"id"`
				Username string `json:"username"`
				Avatar   string `json:"profile_image_url"`
			} `json:"users"`
		} `json:"includes"`
	}

	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		return "", "", "", common.Address{}, err
	}

	address := common.HexToAddress(regexp.MustCompile("0x[0-9a-fA-F]{40}").FindString(result.Data.Text))
	if address == (common.Address{}) {
		//lint:ignore ST1005 This error is to be displayed in the browser
		return "", "", "", common.Address{}, errors.New("No Ethereum address found to fund")
	}
	return result.Data.AuthorID + "@twitter", result.Includes.Users[0].Username, result.Includes.Users[0].Avatar, address, nil
}

// authFacebook tries to authenticate a faucet request using Facebook posts,
// returning the username, avatar URL and Ethereum address to fund on success.
func authFacebook(url string) (string, string, common.Address, error) {
	// Ensure the user specified a meaningful URL, no fancy nonsense
	parts := strings.Split(strings.Split(url, "?")[0], "/")
	if parts[len(parts)-1] == "" {
		parts = parts[0 : len(parts)-1]
	}
	if len(parts) < 4 || parts[len(parts)-2] != "posts" {
		//lint:ignore ST1005 This error is to be displayed in the browser
		return "", "", common.Address{}, errors.New("Invalid Facebook post URL")
	}
	username := parts[len(parts)-3]

	// Facebook's Graph API isn't really friendly with direct links. Still, we don't
	// want to do ask read permissions from users, so just load the public posts and
	// scrape it for the Ethereum address and profile URL.
	//
	// Facebook recently changed their desktop webpage to use AJAX for loading post
	// content, so switch over to the mobile site for now. Will probably end up having
	// to use the API eventually.
	crawl := strings.Replace(url, "www.facebook.com", "m.facebook.com", 1)

	res, err := http.Get(crawl)
	if err != nil {
		return "", "", common.Address{}, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", "", common.Address{}, err
	}
	address := common.HexToAddress(string(regexp.MustCompile("0x[0-9a-fA-F]{40}").Find(body)))
	if address == (common.Address{}) {
		//lint:ignore ST1005 This error is to be displayed in the browser
		return "", "", common.Address{}, errors.New("No Ethereum address found to fund")
	}
	var avatar string
	if parts = regexp.MustCompile("src=\"([^\"]+fbcdn.net[^\"]+)\"").FindStringSubmatch(string(body)); len(parts) == 2 {
		avatar = parts[1]
	}
	return username + "@facebook", avatar, address, nil
}

// authNoAuth tries to interpret a faucet request as a plain Ethereum address,
// without actually performing any remote authentication. This mode is prone to
// Byzantine attack, so only ever use for truly private networks.
func authNoAuth(url string) (string, string, common.Address, error) {
	address := common.HexToAddress(regexp.MustCompile("0x[0-9a-fA-F]{40}").FindString(url))
	if address == (common.Address{}) {
		//lint:ignore ST1005 This error is to be displayed in the browser
		return "", "", common.Address{}, errors.New("No Ethereum address found to fund")
	}
	return address.Hex() + "@noauth", "", address, nil
}
