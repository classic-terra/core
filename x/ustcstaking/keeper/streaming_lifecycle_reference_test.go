package keeper

import (
	"context"
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// lifecycleBank adds independently observed owner credits/debits and a finite
// community budget to the existing fixture. This is a keeper accounting test;
// real SDK bank supply/permissions are covered by the app integration tests.
type lifecycleBank struct {
	*recordingBankKeeper
	wallets   map[string]*big.Int
	community *big.Int
}

func (b *lifecycleBank) SendCoinsFromAccountToModule(ctx context.Context, owner sdk.AccAddress, module string, coins sdk.Coins) error {
	amount := coins.AmountOf(types.BondDenom).BigInt()
	if b.wallets[owner.String()].Cmp(amount) < 0 {
		return fmt.Errorf("insufficient owner funds")
	}
	if err := b.recordingBankKeeper.SendCoinsFromAccountToModule(ctx, owner, module, coins); err != nil {
		return err
	}
	b.wallets[owner.String()].Sub(b.wallets[owner.String()], amount)
	return nil
}
func (b *lifecycleBank) SendCoinsFromModuleToAccount(ctx context.Context, module string, owner sdk.AccAddress, coins sdk.Coins) error {
	if err := b.recordingBankKeeper.SendCoinsFromModuleToAccount(ctx, module, owner, coins); err != nil {
		return err
	}
	b.wallets[owner.String()].Add(b.wallets[owner.String()], coins.AmountOf(types.BondDenom).BigInt())
	return nil
}
func (b *lifecycleBank) DistributeFromCommunityPoolToModule(ctx context.Context, coins sdk.Coins, module string) error {
	amount := coins.AmountOf(types.BondDenom).BigInt()
	if module != types.RewardPoolName || b.community.Cmp(amount) < 0 {
		return fmt.Errorf("invalid community transfer")
	}
	b.community.Sub(b.community, amount)
	b.rewardPoolBalance = sdk.NewCoin(types.BondDenom, b.rewardPoolBalance.Amount.Add(coins.AmountOf(types.BondDenom)))
	return nil
}
func (b *lifecycleBank) ReturnRewardDustToCommunityPool(ctx context.Context, coins sdk.Coins) error {
	b.community.Add(b.community, coins.AmountOf(types.BondDenom).BigInt())
	b.rewardPoolBalance = sdk.NewCoin(types.BondDenom, b.rewardPoolBalance.Amount.Sub(coins.AmountOf(types.BondDenom)))
	return nil
}

type lifecyclePosition struct {
	owner     string
	principal int64
	shares    *big.Int
	// Earned is updated by interval allocation to each owner, independently of
	// production reward debt. Claims subtract whole coins, retaining fractions.
	earned     *big.Rat
	claimable  *big.Int
	status     types.PositionStatus
	duration   time.Duration
	multiplier string
	end        time.Time
}
type lifecycleReference struct {
	now                                           time.Time
	rate                                          int64
	paused                                        bool
	reserve, allocated, shares, index, indexCarry *big.Int
	emission                                      *big.Rat
	positions                                     map[uint64]*lifecyclePosition
	wallets                                       map[string]*big.Int
	paid                                          map[string]*big.Int
	funded, donated, returnedPrincipal            *big.Int
}

func lifecycleFloor(r *big.Rat) *big.Int { return new(big.Int).Quo(r.Num(), r.Denom()) }
func lifecycleScale() *big.Int           { return new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil) }

// settle uses rational elapsed-day budgets and per-position interval credits.
// The only losses are the specified 18-place index quantization, discarded
// cohort carry, and a terminal fraction when a position stops earning.
func (m *lifecycleReference) settle(now time.Time) {
	if now.Equal(m.now) {
		return
	}
	elapsed := now.Sub(m.now).Nanoseconds() // Generated intervals are at most 23 days.
	m.now = now
	if m.paused || m.rate == 0 || m.shares.Sign() == 0 || m.reserve.Sign() == 0 {
		m.emission.SetInt64(0)
		return
	}
	dueExact := new(big.Rat).SetFrac(new(big.Int).Mul(big.NewInt(m.rate), big.NewInt(elapsed)), big.NewInt(86400*1000000000))
	dueExact.Add(dueExact, m.emission)
	released := lifecycleFloor(dueExact)
	m.emission.Sub(dueExact, new(big.Rat).SetInt(released))
	if released.Cmp(m.reserve) >= 0 {
		released.Set(m.reserve)
		m.emission.SetInt64(0)
	}
	m.reserve.Sub(m.reserve, released)
	m.allocated.Add(m.allocated, released)
	numerator := new(big.Int).Add(new(big.Int).Mul(released, lifecycleScale()), m.indexCarry)
	increment := new(big.Int)
	increment.QuoRem(numerator, m.shares, m.indexCarry)
	m.index.Add(m.index, increment)
	for _, p := range m.positions {
		if p.status == types.PositionStatus_POSITION_STATUS_ACTIVE {
			p.earned.Add(p.earned, new(big.Rat).SetFrac(new(big.Int).Mul(p.shares, increment), lifecycleScale()))
		}
	}
}

func TestStreamingLifecycleMatchesIndependentMultiUserReference(t *testing.T) {
	for _, seed := range []int64{20261014, 7, 4815162342} {
		t.Run(fmt.Sprint(seed), func(t *testing.T) {
			ctx, k := newKeeperTest(t)
			params := types.DefaultParams()
			duration := 21 * 24 * time.Hour
			params.LockTiers = append(params.LockTiers, types.LockTier{Id: 2, Duration: &duration, Multiplier: math.LegacyMustNewDecFromStr("1.5")})
			params.RewardRatePerDay = math.NewInt(3)
			k.SetParams(ctx, params)
			bank := &lifecycleBank{recordingBankKeeper: k.bankKeeper.(*recordingBankKeeper), wallets: map[string]*big.Int{}, community: big.NewInt(1000000000)}
			k.bankKeeper, k.communityPoolKeeper = bank, bank
			server := NewMsgServerImpl(k)
			m := &lifecycleReference{now: ctx.BlockTime(), rate: 3, reserve: new(big.Int), allocated: new(big.Int), shares: new(big.Int), index: new(big.Int), indexCarry: new(big.Int), emission: new(big.Rat), positions: map[uint64]*lifecyclePosition{}, wallets: map[string]*big.Int{}, paid: map[string]*big.Int{}, funded: new(big.Int), donated: new(big.Int), returnedPrincipal: new(big.Int)}
			owners := make([]string, 4)
			for i := range owners {
				raw := make([]byte, 20)
				raw[0] = byte(i + 1)
				owners[i] = sdk.AccAddress(raw).String()
				bank.wallets[owners[i]] = big.NewInt(1000000000)
				m.wallets[owners[i]] = big.NewInt(1000000000)
				m.paid[owners[i]] = new(big.Int)
			}
			rng := rand.New(rand.NewSource(seed))
			counts := map[string]int{}
			check := func(label string) {
				t.Helper()
				state := k.GetRewardState(ctx)
				for _, pair := range []struct {
					name string
					want *big.Int
					got  string
				}{
					{"reserve", m.reserve, state.UnallocatedRewards.String()}, {"allocated", m.allocated, state.AllocatedRewards.String()},
					{"shares", m.shares, state.TotalShares.String()}, {"index", m.index, state.RewardIndex.BigInt().String()},
					{"index carry", m.indexCarry, state.IndexRemainder.String()},
				} {
					require.Equal(t, pair.want.String(), pair.got, "%s: %s", label, pair.name)
				}
				emission := new(big.Rat).Mul(m.emission, new(big.Rat).SetInt64(86400*1000000000))
				require.True(t, emission.IsInt(), label)
				require.Equal(t, emission.Num().String(), state.EmissionRemainder.String(), label)
				require.True(t, m.now.Equal(*state.LastRewardTime), label)
				principal, explicit, totalPaid := new(big.Int), new(big.Int), new(big.Int)
				exact := new(big.Rat)
				stored := 0
				for id, p := range m.positions {
					got, found := k.GetPosition(ctx, id)
					pruned := p.status == types.PositionStatus_POSITION_STATUS_WITHDRAWN && p.claimable.Sign() == 0
					require.Equal(t, !pruned, found, "%s: position %d pruning", label, id)
					principal.Add(principal, big.NewInt(p.principal))
					explicit.Add(explicit, p.claimable)
					exact.Add(exact, new(big.Rat).SetInt(p.claimable))
					exact.Add(exact, p.earned)
					if !found {
						continue
					}
					stored++
					require.Equal(t, p.owner, got.Owner, label)
					require.Equal(t, p.status, got.Status, label)
					require.Equal(t, fmt.Sprint(p.principal), got.Principal.Amount.String(), label)
					require.Equal(t, p.shares.String(), got.Shares.String(), label)
					require.Equal(t, p.claimable.String(), got.ClaimableRewards.Amount.String(), label)
					require.Equal(t, p.duration, *got.LockDuration, label)
					require.Equal(t, p.multiplier, got.ShareMultiplier.String(), label)
					debt := new(big.Rat)
					if p.status == types.PositionStatus_POSITION_STATUS_ACTIVE {
						debt.SetInt(new(big.Int).Mul(p.shares, m.index))
						debt.Sub(debt, new(big.Rat).Mul(p.earned, new(big.Rat).SetInt(lifecycleScale())))
					} else {
						require.True(t, p.end.Equal(*got.UnbondingEndTime), label)
					}
					require.True(t, debt.IsInt(), label)
					require.Equal(t, debt.Num().String(), got.RewardDebt.BigInt().String(), label)
				}
				actualCount := 0
				k.IteratePositions(ctx, func(types.Position) bool { actualCount++; return false })
				require.Equal(t, stored, actualCount, label)
				require.Equal(t, explicit.String(), k.GetReservedRewards(ctx).String(), label)
				require.True(t, exact.Cmp(new(big.Rat).SetInt(m.allocated)) <= 0, "%s: exact liabilities exceed protection", label)
				require.Equal(t, principal.String(), intOrZero(bank.principalPoolBalance.Amount).String(), label)
				supply := new(big.Int).Add(bank.community, bank.rewardPoolBalance.Amount.BigInt())
				supply.Add(supply, principal)
				for _, owner := range owners {
					require.Equal(t, m.wallets[owner].String(), bank.wallets[owner].String(), "%s: wallet %s", label, owner)
					supply.Add(supply, bank.wallets[owner])
					totalPaid.Add(totalPaid, m.paid[owner])
				}
				require.Equal(t, "5000000000", supply.String(), label)
				require.Equal(t, new(big.Int).Sub(big.NewInt(1000000000), m.funded).String(), bank.community.String(), label)
				expectedPool := new(big.Int).Sub(new(big.Int).Add(m.funded, m.donated), totalPaid)
				require.Equal(t, expectedPool.String(), bank.rewardPoolBalance.Amount.String(), label)
				require.Equal(t, m.funded.String(), new(big.Int).Add(new(big.Int).Add(m.reserve, m.allocated), totalPaid).String(), "%s: funded budget conservation", label)
				require.NoError(t, k.ValidateState(ctx), label)
			}
			advance := func(d time.Duration) { ctx = ctx.WithBlockTime(ctx.BlockTime().Add(d)) }
			fund := func(amount int64) {
				m.settle(ctx.BlockTime())
				_, err := server.FundRewards(sdk.WrapSDKContext(ctx), &types.MsgFundRewards{Authority: k.GetAuthority(), Amount: sdk.NewInt64Coin(types.BondDenom, amount)})
				require.NoError(t, err)
				m.reserve.Add(m.reserve, big.NewInt(amount))
				m.funded.Add(m.funded, big.NewInt(amount))
				counts["fund"]++
				check("fund")
			}
			update := func(rate int64, paused bool) {
				m.settle(ctx.BlockTime())
				params.RewardRatePerDay = math.NewInt(rate)
				params.Paused = paused
				_, err := server.UpdateParams(sdk.WrapSDKContext(ctx), &types.MsgUpdateParams{Authority: k.GetAuthority(), Params: params})
				require.NoError(t, err)
				m.rate, m.paused = rate, paused
				if paused || rate == 0 {
					m.emission.SetInt64(0)
				}
				counts["params"]++
				check("params")
			}
			stake := func(owner string, amount int64, tier uint32) uint64 {
				m.settle(ctx.BlockTime())
				result, err := server.Stake(sdk.WrapSDKContext(ctx), &types.MsgStake{Owner: owner, Amount: sdk.NewInt64Coin(types.BondDenom, amount), LockTierId: tier})
				require.NoError(t, err)
				shares := big.NewInt(amount)
				multiplier := math.LegacyOneDec().String()
				if tier == 2 {
					shares.Quo(new(big.Int).Mul(shares, big.NewInt(3)), big.NewInt(2))
					multiplier = math.LegacyMustNewDecFromStr("1.5").String()
				}
				m.positions[result.PositionId] = &lifecyclePosition{owner: owner, principal: amount, shares: shares, earned: new(big.Rat), claimable: new(big.Int), status: types.PositionStatus_POSITION_STATUS_ACTIVE, duration: duration, multiplier: multiplier}
				m.shares.Add(m.shares, shares)
				m.indexCarry.SetInt64(0)
				m.wallets[owner].Sub(m.wallets[owner], big.NewInt(amount))
				counts["stake"]++
				check("stake")
				return result.PositionId
			}
			claim := func(id uint64) {
				p := m.positions[id]
				m.settle(ctx.BlockTime())
				whole := lifecycleFloor(p.earned)
				pay := new(big.Int).Add(whole, p.claimable)
				result, err := server.ClaimRewards(sdk.WrapSDKContext(ctx), &types.MsgClaimRewards{Owner: p.owner, PositionId: id})
				require.NoError(t, err)
				require.Equal(t, pay.String(), result.Amount.Amount.String())
				p.earned.Sub(p.earned, new(big.Rat).SetInt(whole))
				p.claimable.SetInt64(0)
				m.allocated.Sub(m.allocated, pay)
				m.paid[p.owner].Add(m.paid[p.owner], pay)
				m.wallets[p.owner].Add(m.wallets[p.owner], pay)
				counts["claim"]++
				check("claim")
			}
			unbond := func(id uint64) {
				p := m.positions[id]
				m.settle(ctx.BlockTime())
				_, err := server.BeginUnbonding(sdk.WrapSDKContext(ctx), &types.MsgBeginUnbonding{Owner: p.owner, PositionId: id})
				require.NoError(t, err)
				p.claimable.Add(p.claimable, lifecycleFloor(p.earned))
				p.earned.SetInt64(0)
				m.shares.Sub(m.shares, p.shares)
				p.shares.SetInt64(0)
				p.status = types.PositionStatus_POSITION_STATUS_UNBONDING
				p.end = ctx.BlockTime().Add(p.duration)
				m.indexCarry.SetInt64(0)
				if m.shares.Sign() == 0 {
					m.emission.SetInt64(0)
				}
				counts["unbond"]++
				check("unbond")
			}
			withdraw := func(id uint64) {
				p := m.positions[id]
				if ctx.BlockTime().Before(p.end) {
					before := k.GetRewardState(ctx)
					_, err := server.Withdraw(sdk.WrapSDKContext(ctx), &types.MsgWithdraw{Owner: p.owner, PositionId: id})
					require.ErrorIs(t, err, types.ErrNotMatured)
					require.Equal(t, before, k.GetRewardState(ctx), "premature withdrawal must roll back settlement")
					counts["premature"]++
					return
				}
				m.settle(ctx.BlockTime())
				_, err := server.Withdraw(sdk.WrapSDKContext(ctx), &types.MsgWithdraw{Owner: p.owner, PositionId: id})
				require.NoError(t, err)
				m.wallets[p.owner].Add(m.wallets[p.owner], big.NewInt(p.principal))
				m.returnedPrincipal.Add(m.returnedPrincipal, big.NewInt(p.principal))
				p.principal = 0
				p.status = types.PositionStatus_POSITION_STATUS_WITHDRAWN
				counts["withdraw"]++
				check("withdraw")
			}
			// Adversarial prefix guarantees idle funding, fractional claims, tiny and
			// weighted entrants, same-timestamp operations, pause, exhaustion/restart,
			// terminal fractions, maturity equality, and post-withdraw reward claims.
			fund(6)
			advance(24 * time.Hour)
			require.NoError(t, k.AccrueRewards(ctx))
			m.settle(ctx.BlockTime())
			check("idle")
			a := stake(owners[0], 1, 1)
			b := stake(owners[1], 1, 1)
			advance(24 * time.Hour)
			claim(a)
			advance(24 * time.Hour)
			claim(a)
			claim(b)
			advance(48 * time.Hour)
			fund(31)
			c := stake(owners[2], 3, 2)
			advance(12 * time.Hour)
			update(17, true)
			advance(48 * time.Hour)
			fund(19)
			claim(a)
			update(19, false)
			advance(25*time.Hour + time.Nanosecond)
			unbond(a)
			unbond(b)
			unbond(c)
			withdraw(a)
			advance(21 * 24 * time.Hour)
			withdraw(a)
			claim(a)
			claim(b)
			withdraw(b)
			withdraw(c)
			claim(c)
			// Random operation choice is independent of production state. Stable ID
			// order avoids Go map iteration changing the supposedly seeded history.
			for step := 0; step < 1200; step++ {
				if rng.Intn(4) != 0 {
					advance(time.Duration(rng.Int63n(int64(3*24*time.Hour)) + 1))
				}
				ids := make([]uint64, 0, len(m.positions))
				active := []uint64{}
				unbonding := []uint64{}
				for id := uint64(0); id < k.GetNextPositionID(ctx); id++ {
					p, ok := m.positions[id]
					if !ok {
						continue
					}
					if p.status != types.PositionStatus_POSITION_STATUS_WITHDRAWN || p.claimable.Sign() > 0 {
						ids = append(ids, id)
					}
					if p.status == types.PositionStatus_POSITION_STATUS_ACTIVE {
						active = append(active, id)
					}
					if p.status == types.PositionStatus_POSITION_STATUS_UNBONDING {
						unbonding = append(unbonding, id)
					}
				}
				switch rng.Intn(8) {
				case 0:
					if !m.paused {
						stake(owners[rng.Intn(len(owners))], int64(rng.Intn(97)+1), uint32(rng.Intn(2)+1))
					}
				case 1:
					if len(ids) > 0 {
						claim(ids[rng.Intn(len(ids))])
					}
				case 2:
					if len(active) > 0 {
						unbond(active[rng.Intn(len(active))])
					}
				case 3:
					if len(unbonding) > 0 {
						withdraw(unbonding[rng.Intn(len(unbonding))])
					}
				case 4:
					fund(int64(rng.Intn(200) + 1))
				case 5:
					update(int64(rng.Intn(301)), rng.Intn(3) == 0)
				case 6:
					// Unsolicited coins are physical surplus only; donation does not settle.
					owner := owners[rng.Intn(len(owners))]
					amount := big.NewInt(int64(rng.Intn(19) + 1))
					bank.wallets[owner].Sub(bank.wallets[owner], amount)
					m.wallets[owner].Sub(m.wallets[owner], amount)
					m.donated.Add(m.donated, amount)
					bank.rewardPoolBalance = sdk.NewCoin(types.BondDenom, bank.rewardPoolBalance.Amount.Add(math.NewIntFromBigInt(amount)))
					counts["donate"]++
					check("donate")
				case 7:
					m.settle(ctx.BlockTime())
					require.NoError(t, k.AccrueRewards(ctx))
					counts["advance"]++
					check("advance")
				}
			}
			update(29, false)
			for id := uint64(0); id < k.GetNextPositionID(ctx); id++ {
				if p, ok := m.positions[id]; ok && p.status == types.PositionStatus_POSITION_STATUS_ACTIVE {
					unbond(id)
				}
			}
			advance(23 * 24 * time.Hour)
			for id := uint64(0); id < k.GetNextPositionID(ctx); id++ {
				if p, ok := m.positions[id]; ok {
					if p.status == types.PositionStatus_POSITION_STATUS_UNBONDING {
						withdraw(id)
					}
					if p.claimable.Sign() > 0 {
						claim(id)
					}
				}
			}
			for _, op := range []string{"stake", "claim", "unbond", "withdraw", "premature", "fund", "params", "donate", "advance"} {
				require.Positive(t, counts[op], "seed %d never exercised %s", seed, op)
			}
			require.Positive(t, m.returnedPrincipal.Sign())
			check("drained lifecycle")
		})
	}
}
