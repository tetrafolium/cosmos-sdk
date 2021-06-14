package module

import (
	"github.com/cosmos/cosmos-sdk/app"
	"github.com/cosmos/cosmos-sdk/app/cli"
	"github.com/cosmos/cosmos-sdk/app/compat"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/container"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	"github.com/cosmos/cosmos-sdk/x/auth/simulation"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var (
	_ app.TypeProvider = Module{}
	_ app.Provisioner  = Module{}
)

type Inputs struct {
	container.StructArgs

	Codec      codec.Codec
	Key        *sdk.KVStoreKey
	ParamStore paramtypes.Subspace
}

type Outputs struct {
	container.StructArgs

	ViewKeeper types.ViewKeeper
	Keeper     types.Keeper `security-role:"admin"`
}

func (m Module) RegisterTypes(registry codectypes.InterfaceRegistry) {
	types.RegisterInterfaces(registry)
}

func (m Module) Provision(registrar container.Registrar) error {
	// provide AccountRetriever
	err := registrar.Provide(func() client.AccountRetriever {
		return types.AccountRetriever{}
	})
	if err != nil {
		return err
	}

	// provide CLI handlers
	err = registrar.Provide(func(configurator cli.Configurator) {
		configurator.RootQueryCommand().AddCommand(
			authcmd.GetAccountCmd(),
			authcmd.QueryTxsByEventsCmd(),
			authcmd.QueryTxCmd(),
		)

		configurator.RootTxCommand().AddCommand(
			authcmd.GetSignCommand(),
			authcmd.GetSignBatchCommand(),
			authcmd.GetMultiSignCommand(),
			authcmd.GetMultiSignBatchCmd(),
			authcmd.GetValidateSignaturesCommand(),
			flags.LineBreak,
			authcmd.GetBroadcastCommand(),
			authcmd.GetEncodeCommand(),
			authcmd.GetDecodeCommand(),
			flags.LineBreak,
		)

		compat.RegisterAppModuleBasic(configurator, auth.AppModuleBasic{})
	})
	if err != nil {
		return err
	}

	// provide app handlers
	return registrar.Provide(
		func(configurator app.Configurator, inputs Inputs) (Outputs, error) {
			var accCtr types.AccountConstructor
			if m.AccountConstructor != nil {
				err := inputs.Codec.UnpackAny(m.AccountConstructor, &accCtr)
				if err != nil {
					return Outputs{}, err
				}
			} else {
				accCtr = DefaultAccountConstructor{}
			}

			perms := map[string][]string{}
			for _, perm := range m.Permissions {
				perms[perm.Address] = perm.Permissions
			}

			var randomGenesisAccountsProvider types.RandomGenesisAccountsProvider
			if m.RandomGenesisAccountsProvider != nil {
				err := inputs.Codec.UnpackAny(m.RandomGenesisAccountsProvider, &randomGenesisAccountsProvider)
				if err != nil {
					return Outputs{}, err
				}
			} else {
				randomGenesisAccountsProvider = DefaultRandomGenesisAccountsProvider{}
			}

			keeper := authkeeper.NewAccountKeeper(inputs.Codec, inputs.Key, inputs.ParamStore, func() types.AccountI {
				return accCtr.NewAccount()
			}, perms)
			appMod := auth.NewAppModule(inputs.Codec, keeper, func(simState *module.SimulationState) types.GenesisAccounts {
				return randomGenesisAccountsProvider.RandomGenesisAccounts(simState)
			})

			compat.RegisterAppModule(configurator, appMod)

			return Outputs{
				ViewKeeper: viewOnlyKeeper{keeper},
				Keeper:     keeper,
			}, nil
		},
	)
}

func (m DefaultAccountConstructor) NewAccount() types.AccountI {
	return &types.BaseAccount{}
}

func (m DefaultRandomGenesisAccountsProvider) RandomGenesisAccounts(simState *module.SimulationState) types.GenesisAccounts {
	return simulation.RandomGenesisAccounts(simState)
}