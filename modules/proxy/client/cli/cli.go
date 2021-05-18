package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
)

// GetQueryCmd returns the query commands for IBC proxy
func GetQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        "ibc-proxy",
		Short:                      "IBC proxy query subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
	}

	return queryCmd
}

// NewTxCmd returns the transaction commands for IBC proxy
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        "ibc-proxy",
		Short:                      "IBC proxy transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	return txCmd
}
