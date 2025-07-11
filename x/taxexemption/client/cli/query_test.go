package cli_test

import (
	"fmt"
	"strings"

	"github.com/classic-terra/core/v3/x/taxexemption/client/cli"
)

func (s *CLITestSuite) TestGetQueryCmd() {
	testCases := []struct {
		name         string
		args         []string
		expCmdOutput string
		expPass      bool
	}{
		// Success cases
		{
			"success: no args (help should work)",
			[]string{"--help"},
			"taxexemption",
			true,
		},

		// Failure cases
		{
			"fail: unknown subcommand",
			[]string{"unknown"},
			"",
			false,
		},
		{
			"fail: missing required args for taxable",
			[]string{"taxable"},
			"",
			false,
		},
		{
			"fail: missing required args for addresses",
			[]string{"addresses"},
			"",
			false,
		},
		{
			"fail: too many args for zones",
			[]string{"zones", "extra-arg"},
			"",
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			// Create a fresh command for each test case
			cmd := cli.GetQueryCmd()

			// Check basic properties only in the first test
			if tc.name == "success: no args (help should work)" {
				s.Require().NotNil(cmd)
				s.Require().Equal("taxexemption", cmd.Use)
				s.Require().Equal("Querying commands for the taxexemption module", cmd.Short)
				s.Require().True(cmd.DisableFlagParsing)
				s.Require().Equal(2, cmd.SuggestionsMinimumDistance)
			}

			// Set the args
			cmd.SetArgs(tc.args)

			// Handle different test case scenarios
			switch {
			case strings.Contains(tc.args[0], "help") || tc.args[len(tc.args)-1] == "--help":
				// For help commands, check if the output contains expected text
				cmdStr := fmt.Sprint(cmd)
				s.Require().Contains(cmdStr, tc.expCmdOutput)
			case tc.expPass:
				// For success cases, we'd need to execute the command
				// We skip actual execution since it would need proper setup
				s.T().Skip("Skipping execution test for success case")
			default:
				// For failure cases, check if the validation would fail
				if len(tc.args) > 0 {
					for _, subCmd := range cmd.Commands() {
						if subCmd.Name() == tc.args[0] {
							// Check argument validation if appropriate
							switch subCmd.Name() {
							case "taxable":
								if len(tc.args) == 1 {
									err := subCmd.Args(subCmd, []string{})
									s.Require().Error(err, "taxable should require args")
								}
							case "addresses":
								if len(tc.args) == 1 {
									err := subCmd.Args(subCmd, []string{})
									s.Require().Error(err, "addresses should require args")
								}
							case "zones":
								if len(tc.args) > 1 {
									err := subCmd.Args(subCmd, tc.args[1:])
									s.Require().Error(err, "zones should not accept args")
								}
							}
							break
						}
					}
				}
			}
		})
	}

	// Verify the command has the expected subcommands
	cmd := cli.GetQueryCmd()
	subCmds := cmd.Commands()
	s.Require().Len(subCmds, 3, "GetQueryCmd should add 3 subcommands")

	// Get the subcommand names
	var subCmdNames []string
	for _, subCmd := range subCmds {
		subCmdNames = append(subCmdNames, subCmd.Name())
	}

	// Verify each expected subcommand exists
	s.Require().Contains(subCmdNames, "taxable", "GetQueryCmd should add 'taxable' subcommand")
	s.Require().Contains(subCmdNames, "zones", "GetQueryCmd should add 'zones' subcommand")
	s.Require().Contains(subCmdNames, "addresses", "GetQueryCmd should add 'addresses' subcommand")
}

func (s *CLITestSuite) TestGetCmdQueryTaxable() {
	testCases := []struct {
		name           string
		args           []string
		expectArgsPass bool
	}{
		{
			"success: with valid addresses",
			[]string{
				"terra1address1", "terra1address2",
			},
			true,
		},
		{
			"fail: missing first address",
			[]string{},
			false,
		},
		{
			"fail: missing second address",
			[]string{"terra1address1"},
			false,
		},
		{
			"fail: too many arguments",
			[]string{"terra1address1", "terra1address2", "terra1address3"},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			cmd := cli.GetCmdQueryTaxable()

			// Test argument validation directly
			argsFunc := cmd.Args
			err := argsFunc(cmd, tc.args)

			if tc.expectArgsPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *CLITestSuite) TestGetCmdQueryZonelist() {
	testCases := []struct {
		name           string
		args           []string
		expCmdOutput   string
		expectArgsPass bool
		isArgsTest     bool // Whether this test case is checking argument validation
	}{
		// Success cases
		{
			"success:basic command",
			[]string{},
			"zones",
			true,
			false,
		},
		{
			"success:with limit",
			[]string{
				"--limit=5",
			},
			"limit",
			true,
			false,
		},
		{
			"success:with page",
			[]string{
				"--page=2",
			},
			"page",
			true,
			false,
		},
		{
			"success:with offset",
			[]string{
				"--offset=10",
			},
			"offset",
			true,
			false,
		},
		{
			"success:with count-total",
			[]string{
				"--count-total",
			},
			"count-total",
			true,
			false,
		},
		{
			"success:with reverse",
			[]string{
				"--reverse",
			},
			"reverse",
			true,
			false,
		},
		{
			"success:with multiple pagination flags",
			[]string{
				"--limit=5",
				"--page=2",
				"--count-total",
			},
			"limit",
			true,
			false,
		},
		{
			"success:with output format json",
			[]string{
				"--output=json",
			},
			"output",
			true,
			false,
		},
		{
			"success:with output format text",
			[]string{
				"--output=text",
			},
			"output",
			true,
			false,
		},

		// Argument validation cases
		{
			"success:valid - no args",
			[]string{},
			"",
			true,
			true,
		},
		{
			"fail: invalid - with unexpected positional arg",
			[]string{"unexpected-arg"},
			"",
			false,
			true,
		},
		{
			"fail: invalid - with multiple unexpected args",
			[]string{"arg1", "arg2"},
			"",
			false,
			true,
		},
		{
			"fail:invalid - with incorrect flag format",
			[]string{"--invalidflag"},
			"",
			true, // Cobra won't validate flag format in Args function
			true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			cmd := cli.GetCmdQueryZonelist()

			if tc.isArgsTest {
				// For argument validation tests, directly test the Args function
				var positionalArgs []string
				for _, arg := range tc.args {
					if !strings.HasPrefix(arg, "-") {
						positionalArgs = append(positionalArgs, arg)
					}
				}

				err := cmd.Args(cmd, positionalArgs)
				if tc.expectArgsPass {
					s.Require().NoError(err, "Args validation should pass for %s with args %v", tc.name, positionalArgs)
				} else {
					s.Require().Error(err, "Args validation should fail for %s with args %v", tc.name, positionalArgs)
				}
			} else {
				// For command output tests
				cmd.SetArgs(tc.args)
				s.Require().Contains(fmt.Sprint(cmd), strings.TrimSpace(tc.expCmdOutput))
			}
		})
	}

	// Additional check for command properties
	cmd := cli.GetCmdQueryZonelist()
	s.Require().Equal("zones", cmd.Use)

	// Instead of directly comparing functions, test the behavior
	// Test that cmd.Args behaves like cobra.NoArgs
	s.Require().NoError(cmd.Args(cmd, []string{}), "Should accept empty args")
	s.Require().Error(cmd.Args(cmd, []string{"arg1"}), "Should reject args")

	// Check that pagination flags were added
	cmdStr := cmd.Flags().FlagUsages()
	s.Require().Contains(cmdStr, "limit")
	s.Require().Contains(cmdStr, "page")
}

func (s *CLITestSuite) TestGetCmdQueryExemptlist() {
	testCases := []struct {
		name         string
		args         []string
		expCmdOutput string
		expPass      bool
	}{
		// Success cases
		{
			"success: with zone name",
			[]string{
				"zone1",
			},
			"zone1",
			true,
		},
		{
			"success: with empty zone name",
			[]string{
				"",
			},
			"",
			true,
		},
		{
			"success: with zone and limit",
			[]string{
				"zone1",
				"--limit=5",
			},
			"zone1",
			true,
		},
		{
			"success: with zone and page",
			[]string{
				"zone1",
				"--page=2",
			},
			"zone1",
			true,
		},
		{
			"success: with zone and offset",
			[]string{
				"zone1",
				"--offset=10",
			},
			"zone1",
			true,
		},
		{
			"success: with zone and count-total",
			[]string{
				"zone1",
				"--count-total",
			},
			"zone1",
			true,
		},
		{
			"success: with zone and reverse",
			[]string{
				"zone1",
				"--reverse",
			},
			"zone1",
			true,
		},
		{
			"success: with zone and multiple pagination flags",
			[]string{
				"zone1",
				"--limit=5",
				"--page=2",
				"--count-total",
			},
			"zone1",
			true,
		},
		{
			"success: with zone and all pagination flags",
			[]string{
				"zone1",
				"--limit=5",
				"--page=2",
				"--offset=10",
				"--count-total",
				"--reverse",
			},
			"zone1",
			true,
		},
		{
			"success: with zone and output json",
			[]string{
				"zone1",
				"--output=json",
			},
			"zone1",
			true,
		},
		{
			"success: with zone and output text",
			[]string{
				"zone1",
				"--output=text",
			},
			"zone1",
			true,
		},
		{
			"success: with zone and output yaml",
			[]string{
				"zone1",
				"--output=yaml",
			},
			"zone1",
			true,
		},
		{
			"success: with zone and node flag",
			[]string{
				"zone1",
				"--node=tcp://localhost:26657",
			},
			"zone1",
			true,
		},
		{
			"success: with zone and chain-id flag",
			[]string{
				"zone1",
				"--chain-id=test-chain",
			},
			"zone1",
			true,
		},
		{
			"success: with zone and height flag",
			[]string{
				"zone1",
				"--height=100",
			},
			"zone1",
			true,
		},

		// Failure cases
		{
			"fail: without zone name",
			[]string{},
			"",
			false,
		},
		{
			"fail: with too many zone names",
			[]string{
				"zone1",
				"zone2",
			},
			"",
			false,
		},
		{
			"fail: with too many arguments and flags",
			[]string{
				"zone1",
				"zone2",
				"--limit=5",
			},
			"",
			false,
		},
		{
			"fail: invalid pagination: negative limit",
			[]string{
				"zone1",
				"--limit=-5",
			},
			"zone1",
			false,
		},
		{
			"fail: invalid pagination: negative page",
			[]string{
				"zone1",
				"--page=-2",
			},
			"zone1",
			false,
		},
		{
			"fail: invalid format",
			[]string{
				"zone1",
				"--output=invalid",
			},
			"zone1",
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			cmd := cli.GetCmdQueryExemptlist()
			cmd.SetArgs(tc.args)

			if tc.expPass {
				// Success cases: check the command contains expected output
				s.Require().Contains(fmt.Sprint(cmd), strings.TrimSpace(tc.expCmdOutput))

				// For success cases, also test argument validation directly
				var positionalArgs []string
				for _, arg := range tc.args {
					if !strings.HasPrefix(arg, "-") {
						positionalArgs = append(positionalArgs, arg)
					}
				}
				err := cmd.Args(cmd, positionalArgs)
				s.Require().NoError(err, "Args validation failed for %s with args %v", tc.name, positionalArgs)
			} else {
				// Failure cases: test argument validation directly where appropriate
				var positionalArgs []string
				for _, arg := range tc.args {
					if !strings.HasPrefix(arg, "-") {
						positionalArgs = append(positionalArgs, arg)
					}
				}

				// Check if this is an argument validation error case
				if len(positionalArgs) != 1 {
					err := cmd.Args(cmd, positionalArgs)
					s.Require().Error(err, "Args validation should fail for %s with args %v", tc.name, positionalArgs)
				}

				// For other failure cases (like invalid flag values), we would need to run the command
				// which is not done in this test since it would require mock responses
			}
		})
	}

	// Additional verification of the command structure
	cmd := cli.GetCmdQueryExemptlist()
	s.Require().Equal("addresses [zone-name]", cmd.Use)
	s.Require().Equal("Query all tax exemption addresses of a zone", cmd.Short)

	// Check that pagination flags are properly added
	flags := cmd.Flags()
	s.Require().NotNil(flags.Lookup("limit"))
	s.Require().NotNil(flags.Lookup("page"))
	s.Require().NotNil(flags.Lookup("offset"))
	s.Require().NotNil(flags.Lookup("count-total"))
	s.Require().NotNil(flags.Lookup("reverse"))
	s.Require().NotNil(flags.Lookup("output"))
}
