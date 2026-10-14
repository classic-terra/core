//nolint:revive
package ustc_staking

import (
	"fmt"
	"github.com/classic-terra/core/v4/app/keepers"
	types "github.com/classic-terra/core/v4/x/ustcstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	auth "github.com/cosmos/cosmos-sdk/x/auth/types"
	distr "github.com/cosmos/cosmos-sdk/x/distribution/types"
)

// normalizePoolAccounts runs only for first activation, before InitGenesis.
// Coins sent to these deterministic addresses before activation are donations,
// not staking liabilities. Only USTC is swept; other denoms stay inert so an
// attacker cannot force an unbounded denomination scan at activation.
// Both the bank transfer and FeePool credit are required.
func normalizePoolAccounts(ctx sdk.Context, k *keepers.AppKeepers) error {
	for _, name := range []string{types.PrincipalPoolName, types.RewardPoolName} {
		addr := auth.NewModuleAddress(name)
		existing := k.AccountKeeper.GetAccount(ctx, addr)
		base := auth.NewBaseAccountWithAddress(addr)
		account := auth.NewModuleAccount(base, name)
		if existing == nil {
			account = k.AccountKeeper.NewAccount(ctx, account).(*auth.ModuleAccount)
		} else {
			base.AccountNumber = existing.GetAccountNumber()
			base.Sequence = existing.GetSequence()
		}
		k.AccountKeeper.SetModuleAccount(ctx, account)
		coins := sdk.NewCoins(k.BankKeeper.GetBalance(ctx, addr, types.BondDenom))
		if coins.IsZero() {
			continue
		}
		pool, err := k.DistrKeeper.FeePool.Get(ctx)
		if err != nil {
			return fmt.Errorf("read community pool for %s: %w", name, err)
		}
		if err = k.BankKeeper.SendCoinsFromModuleToModule(ctx, name, distr.ModuleName, coins); err != nil {
			return fmt.Errorf("sweep %s: %w", name, err)
		}
		pool.CommunityPool = pool.CommunityPool.Add(sdk.NewDecCoinsFromCoins(coins...)...)
		if err = k.DistrKeeper.FeePool.Set(ctx, pool); err != nil {
			return fmt.Errorf("credit community pool for %s: %w", name, err)
		}
		ctx.EventManager().EmitEvent(sdk.NewEvent("ustcstaking_activation_sweep", sdk.NewAttribute("module", name), sdk.NewAttribute("amount", coins.String())))
	}
	return nil
}
