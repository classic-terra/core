package cli_test

import (
	"fmt"
	"strings"

	v2customcli "github.com/classic-terra/core/v3/custom/gov/client/cli"
)

func (s *CLITestSuite) TestGetCmdQueryMinimalDeposit() {
	testCases := []struct {
		name         string
		args         []string
		expCmdOutput string
	}{
		{
			"proposal with id",
			[]string{
				"1",
			},
			"1",
		},
		{
			"proposal with no id",
			[]string{
				"",
			},
			"",
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := v2customcli.GetCmdQueryMinimalDeposit()
			cmd.SetArgs(tc.args)
			s.Require().Contains(fmt.Sprint(cmd), strings.TrimSpace(tc.expCmdOutput))
		})
	}
}
