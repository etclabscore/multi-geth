package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/params"
)

// TestConsoleCmdNetworkIdentities tests network identity variables at runtime for a geth instance.
// This provides a "production equivalent" integration test for consensus-relevant chain identity values which
// cannot be adequately unit tested because of reliance on cli context variables.
// These tests should cover expected default values and possible flag-interacting values, like --<chain> with --networkid=n.
func TestConsoleCmdNetworkIdentities(t *testing.T) {
	chainIdentityCases := []struct {
		flags       []string
		networkId   int
		chainId     int
		genesisHash string
	}{
		// Default chain value, without and with --networkid flag set.
		{[]string{}, 1, 1, params.MainnetGenesisHash.Hex()},
		{[]string{"--networkid", "42"}, 42, 1, params.MainnetGenesisHash.Hex()},

		// Non-default chain value, without and with --networkid flag set.
		{[]string{"--classic"}, 1, 61, params.MainnetGenesisHash.Hex()},
		{[]string{"--classic", "--networkid", "42"}, 42, 61, params.MainnetGenesisHash.Hex()},

		// All other possible --<chain> values.
		{[]string{"--mainnet"}, 1, 1, params.MainnetGenesisHash.Hex()},
		{[]string{"--ropsten"}, 3, 3, params.RopstenGenesisHash.Hex()},
		{[]string{"--rinkeby"}, 4, 4, params.RinkebyGenesisHash.Hex()},
		{[]string{"--goerli"}, 5, 5, params.GoerliGenesisHash.Hex()},
		{[]string{"--kotti"}, 6, 6, params.KottiGenesisHash.Hex()},
		{[]string{"--mordor"}, 7, 63, params.MordorGenesisHash.Hex()},
		{[]string{"--yolov3"}, int(params.YoloV3ChainConfig.ChainID.Uint64()), int(params.YoloV3ChainConfig.ChainID.Uint64()), params.YoloV3GenesisHash.Hex()},
		{[]string{"--astor"}, 212, 212, params.AstorGenesisHash.Hex()},
		{[]string{"--mintme"}, 37480, 24734, params.MintMeGenesisHash.Hex()},
 	}
	for i, p := range chainIdentityCases {

		// Disable networking, preventing false-negatives if in an environment without networking service
		// or collisions with an existing geth service.
		p.flags = append(p.flags, "--port", "0", "--maxpeers", "0", "--nodiscover", "--nat", "none")

		t.Run(fmt.Sprintf("%d/%v/networkid", i, p.flags),
			consoleCmdStdoutTest(p.flags, "admin.nodeInfo.protocols.eth.network", p.networkId))
		t.Run(fmt.Sprintf("%d/%v/chainid", i, p.flags),
			consoleCmdStdoutTest(p.flags, "admin.nodeInfo.protocols.eth.config.chainId", p.chainId))
		t.Run(fmt.Sprintf("%d/%v/genesis_hash", i, p.flags),
			consoleCmdStdoutTest(p.flags, "eth.getBlock(0, false).hash", strconv.Quote(p.genesisHash)))
	}
}

func consoleCmdStdoutTest(flags []string, execCmd string, want interface{}) func(t *testing.T) {
	return func(t *testing.T) {
		flags = append(flags, "--exec", execCmd, "console")
		geth := runGeth(t, flags...)
		geth.Expect(fmt.Sprintf(`%v
`, want))
		geth.ExpectExit()
		if status := geth.ExitStatus(); status != 0 {
			t.Errorf("expected exit status 0, got: %d", status)
		}
	}
}

// TestGethFailureToLaunch tests that geth fail immediately when given invalid run parameters (ie CLI args).
func TestGethFailureToLaunch(t *testing.T) {
	cases := []struct {
		flags            []string
		expectErrorReStr string
	}{
		{
			flags:            []string{"--badnet"},
			expectErrorReStr: "(?ism)incorrect usage.*",
		},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("TestIncorrectUsage: %v", c.flags), func(t *testing.T) {
			geth := runGeth(t, c.flags...)
			geth.ExpectRegexp(c.expectErrorReStr)
			geth.ExpectExit()
			if status := geth.ExitStatus(); status == 0 {
				t.Errorf("expected exit status != 0, got: %d", status)
			}
		})
	}
}

// TestGethStartupLogs tests that geth logs certain things (given some set of flags).
// In these cases, geth is run with a console command to print its name (and tests that it does).
func TestGethStartupLogs(t *testing.T) {
	// semiPersistentDatadir is used to house an adhoc datadir for co-dependent geth test cases.
	semiPersistentDatadir := filepath.Join(os.TempDir(), fmt.Sprintf("geth-startup-logs-test-%d", time.Now().Unix()))
	defer os.RemoveAll(semiPersistentDatadir)

	type matching struct {
		pattern string // pattern is the pattern to match against geth's stderr log.
		matches bool   // matches defines if the pattern should succeed or fail, ie. if the pattern should exist or should not exist.
	}
	cases := []struct {
		flags    []string
		matchers []matching

		// callback is run after the geth run case completes.
		// It can be used to reset any persistent state to provide a clean slate for the subsequent cases.
		callback func() error
	}{
		{
			// --<chain> flag is NOT given and datadir does not exist, representing a first tabula-rasa run.
			// Use without a --<chain> flag is deprecated. User will be warned.
			flags: []string{},
			matchers: []matching{
				{pattern: "(?ism).+WARN.+Not specifying a chain flag is deprecated.*", matches: true},
			},
		},
		{
			// Network flag is given.
			// --<chain> flag is NOT given. This is deprecated. User will be warned.
			// Same same but different as above.
			flags: []string{"--networkid=42"},
			matchers: []matching{
				{pattern: "(?ism).+WARN.+Not specifying a chain flag is deprecated.*", matches: true},
			},
		},
		// Little bit of a HACK.
		// This is a co-dependent sequence of two test cases.
		// First, startup a geth instance that will create a database, storing the genesis block.
		// This is a basic use case and has no errors.
		// The subsequent case then run geth re-using that datadir which has an existing chain database
		// and contains a stored genesis block.
		// Since the database contains a genesis block, the chain identity and config can (and will) be deduced from it;
		// this causes no need for a --<chain> CLI flag to be passed again. The user will not be warned of a missing --<chain> flag.
		{
			// --<chain> flag is given. All is well. Database (storing genesis) is initialized.
			flags: []string{"--datadir", semiPersistentDatadir, "--mainnet"},
			matchers: []matching{
				{pattern: "(?ism).*", matches: true},
			},
		},
		{
			// --<chain> flag is NOT given, BUT geth is being run on top of an existing
			// datadir. Geth will use the existing (stored) genesis found in it.
			// User should NOT be warned.
			flags: []string{"--datadir", semiPersistentDatadir},
			matchers: []matching{
				{pattern: "(?ism).+WARN.+Not specifying a chain flag is deprecated.*", matches: false},
				{pattern: "(?ism).+INFO.+Found stored genesis block.*", matches: true},
			},
			callback: func() error {
				// Clean up this mini-suite.
				return os.RemoveAll(semiPersistentDatadir)
			},
		},
	}
	for i, c := range cases {
		t.Run(fmt.Sprintf("TestGethStartupLogs/%d: %v", i, c.flags), func(t *testing.T) {
			geth := runGeth(t, append(c.flags, "--exec", "admin.nodeInfo.name", "console")...)
			geth.KillTimeout = 10 * time.Second
			geth.ExpectRegexp("(?ism).*CoreGeth.*")
			geth.ExpectExit()
			if status := geth.ExitStatus(); status != 0 {
				t.Errorf("expected exit status == 0, got: %d", status)
			}
			for _, match := range c.matchers {
				if matched := regexp.MustCompile(match.pattern).MatchString(geth.StderrText()); matched != match.matches {
					t.Errorf("unexpected stderr output; want: %s (matching?=%v) got: %s", match.pattern, match.matches, geth.StderrText())
				}
			}
			if c.callback != nil {
				if err := c.callback(); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
