package cli

import (
	"context"
	"fmt"
	"math/big"

	"cosmossdk.io/math"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/gogoproto/proto"
	"github.com/spf13/cobra"
)

func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the USTC staking module",
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(positionQueryCmd(), positionsByOwnerQueryCmd(), rewardStateQueryCmd(), paramsQueryCmd(), validateStateQueryCmd())
	return cmd
}

func validateStateQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate-state",
		Args:  cobra.NoArgs,
		Short: "Check USTC staking accounting invariants",
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			if clientCtx.Height <= 0 {
				status, err := clientCtx.Client.Status(cmd.Context())
				if err != nil {
					return fmt.Errorf("get validation snapshot height: %w", err)
				}
				clientCtx = clientCtx.WithHeight(status.SyncInfo.LatestBlockHeight)
			}
			queryClient := types.NewQueryClient(clientCtx)
			pageKey := []byte(nil)
			principalLiability, activeShares, rewardLiability := new(big.Int), new(big.Int), new(big.Int)
			claimableLiability := new(big.Int)
			rewardLiabilityExact := new(big.Rat)
			var principalPool, rewardPool []sdk.Coin
			var rewardState types.RewardState
			var reservedRewards math.Int
			var firstPage bool
			for {
				page, err := queryClient.ValidateState(context.Background(), &types.QueryValidateStateRequest{
					Pagination: &query.PageRequest{Key: pageKey, Limit: 500},
				})
				if err != nil {
					return err
				}
				if !page.Valid {
					if err := clientCtx.PrintProto(page); err != nil {
						return err
					}
					return validationFailure(page.Detail)
				}
				pagePrincipal, err := parseValidationInteger(page.PrincipalLiability)
				if err != nil {
					return fmt.Errorf("invalid principal liability returned by validation query: %w", err)
				}
				pageShares, err := parseValidationInteger(page.ActiveShares)
				if err != nil {
					return fmt.Errorf("invalid active shares returned by validation query: %w", err)
				}
				pageRewards, err := parseValidationInteger(page.RewardLiability)
				if err != nil {
					return fmt.Errorf("invalid reward liability returned by validation query: %w", err)
				}
				principalLiability.Add(principalLiability, pagePrincipal)
				activeShares.Add(activeShares, pageShares)
				rewardLiability.Add(rewardLiability, pageRewards)
				pageClaimable, err := parseValidationInteger(page.ClaimableLiability)
				if err != nil {
					return fmt.Errorf("invalid claimable liability returned by validation query: %w", err)
				}
				claimableLiability.Add(claimableLiability, pageClaimable)
				pageExact, ok := new(big.Rat).SetString(page.RewardLiabilityExact)
				if !ok || pageExact.Sign() < 0 {
					return fmt.Errorf("invalid exact reward liability returned by validation query")
				}
				rewardLiabilityExact.Add(rewardLiabilityExact, pageExact)
				if !page.ReservedRewards.IsNil() && page.ReservedRewards.IsNegative() {
					return fmt.Errorf("invalid reserved rewards returned by validation query")
				}
				if firstPage && (!page.ReservedRewards.Equal(reservedRewards) || !proto.Equal(&page.RewardState, &rewardState) ||
					!sdk.Coins(page.PrincipalPoolBalances).Equal(principalPool) || !sdk.Coins(page.RewardPoolBalances).Equal(rewardPool)) {
					return fmt.Errorf("validation pages do not describe one consistent snapshot")
				}
				principalPool, rewardPool, rewardState = page.PrincipalPoolBalances, page.RewardPoolBalances, page.RewardState
				reservedRewards = page.ReservedRewards
				firstPage = true
				pageKey = page.Pagination.NextKey
				if len(pageKey) == 0 {
					break
				}
			}
			principalBalance, err := validationPoolBalance(principalPool)
			if err != nil {
				return fmt.Errorf("principal pool: %w", err)
			}
			rewardBalance, err := validationPoolBalance(rewardPool)
			if err != nil {
				return fmt.Errorf("reward pool: %w", err)
			}
			rewardIndexValid := !rewardState.RewardIndex.IsNil() && !rewardState.RewardIndex.IsNegative()
			totalSharesValid := !rewardState.TotalShares.IsNil() && !rewardState.TotalShares.IsNegative() &&
				rewardState.TotalShares.BigInt().BitLen() <= math.MaxBitLen && rewardState.TotalShares.BigInt().Cmp(activeShares) == 0
			allocated := rewardState.AllocatedRewards
			protected := new(big.Int)
			allocatedValid := !rewardState.UnallocatedRewards.IsNil() && !rewardState.UnallocatedRewards.IsNegative() &&
				rewardState.UnallocatedRewards.BigInt().BitLen() <= math.MaxBitLen && !allocated.IsNil() && !allocated.IsNegative() &&
				allocated.BigInt().BitLen() <= math.MaxBitLen
			if allocatedValid {
				protected.Add(rewardState.UnallocatedRewards.BigInt(), allocated.BigInt())
			}
			claimableValid := claimableLiability.BitLen() <= math.MaxBitLen && !reservedRewards.IsNil() && !reservedRewards.IsNegative() && claimableLiability.Cmp(reservedRewards.BigInt()) == 0
			principalValid := principalLiability.BitLen() <= math.MaxBitLen && principalLiability.Cmp(principalBalance.BigInt()) <= 0
			wholeLiabilityValid := rewardLiability.BitLen() <= math.MaxBitLen && allocatedValid && rewardLiability.Cmp(allocated.BigInt()) <= 0
			exactLiabilityValid := allocatedValid && rewardLiabilityExact.Cmp(new(big.Rat).SetInt(allocated.BigInt())) <= 0
			backingValid := allocatedValid && protected.BitLen() <= math.MaxBitLen && rewardBalance.BigInt().Cmp(protected) >= 0
			valid := principalValid && claimableValid && wholeLiabilityValid && exactLiabilityValid && backingValid && rewardIndexValid && totalSharesValid
			detail := "all USTC staking accounting checks passed at the pinned height"
			if !valid {
				detail = fmt.Sprintf("accounting mismatch: principal liability=%s pool=%s; active shares=%s stored=%s; reward liability=%s exact=%s allocated=%s reserved=%s claimable=%s pool=%s; reward index valid=%t",
					principalLiability, principalBalance, activeShares, rewardState.TotalShares, rewardLiability, rewardLiabilityExact, allocated, reservedRewards, claimableLiability, rewardBalance, rewardIndexValid)
			}
			response := &types.QueryValidateStateResponse{Valid: valid, Detail: detail}
			if err := clientCtx.PrintProto(response); err != nil {
				return err
			}
			if !valid {
				return validationFailure(detail)
			}
			return nil
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func validationFailure(detail string) error {
	return fmt.Errorf("USTC staking validation failed: %s", detail)
}

func parseValidationInteger(value string) (*big.Int, error) {
	parsed, ok := new(big.Int).SetString(value, 10)
	if !ok || parsed.Sign() < 0 || parsed.BitLen() > math.MaxBitLen {
		return nil, fmt.Errorf("value must be a non-negative SDK integer")
	}
	return parsed, nil
}

func validationPoolBalance(balances []sdk.Coin) (math.Int, error) {
	balance := math.ZeroInt()
	for _, coin := range balances {
		if coin.Denom != types.BondDenom {
			return math.ZeroInt(), fmt.Errorf("unexpected denomination %s", coin.Denom)
		}
		if coin.Amount.IsNil() || coin.Amount.IsNegative() {
			return math.ZeroInt(), fmt.Errorf("invalid balance %s", coin)
		}
		balance = balance.Add(coin.Amount)
	}
	return balance, nil
}

func positionQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "position [position-id]",
		Args:  cobra.ExactArgs(1),
		Short: "Query a USTC staking position",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			id, err := parseUint64Argument(args[0], "position id")
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).Position(context.Background(), &types.QueryPositionRequest{PositionId: id})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func positionsByOwnerQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "positions [owner]",
		Aliases: []string{"positions-by-owner"},
		Args:    cobra.ExactArgs(1),
		Short:   "Query USTC staking positions owned by an address",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			if _, err := sdk.AccAddressFromBech32(args[0]); err != nil {
				return err
			}
			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).PositionsByOwner(context.Background(), &types.QueryPositionsByOwnerRequest{Owner: args[0], Pagination: pageReq})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func rewardStateQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reward-state",
		Args:  cobra.NoArgs,
		Short: "Query USTC staking reward state",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).RewardState(context.Background(), &types.QueryRewardStateRequest{})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func paramsQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Args:  cobra.NoArgs,
		Short: "Query USTC staking parameters",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).Params(context.Background(), &types.QueryParamsRequest{})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
