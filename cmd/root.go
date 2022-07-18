package cmd

import (
	"context"
	"fmt"

	"github.com/numary/fctl/cmd/cloud"
	"github.com/numary/fctl/cmd/config"
	"github.com/numary/fctl/cmd/ledger"
	"github.com/numary/fctl/cmd/payments"
	"github.com/numary/fctl/cmd/reconciliation"
	"github.com/numary/fctl/cmd/reports"
	"github.com/numary/fctl/cmd/search"
	"github.com/numary/fctl/cmd/stack"
	"github.com/numary/fctl/cmd/ui"
	"github.com/numary/fctl/cmd/wallets"
	fctl "github.com/numary/fctl/pkg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

var (
	flagProfile string
)

func NewRoot() *cobra.Command {
	root := &cobra.Command{
		Use:  "fctl",
		Long: "Formance Control CLI",
	}

	root.PersistentFlags().StringVar(&flagProfile, "profile", "default", "config profile to use")

	root.AddCommand(NewVersion())

	return root
}

func Execute() {
	options := fx.Options(
		fx.NopLogger,
		fx.Provide(fctl.NewClient),
		fx.Provide(fx.Annotate(
			NewRoot,
			fx.ResultTags(`name:"root"`),
		)),
		config.ConfigModule(),
		cloud.CloudModule(),
		stack.StackModule(),
		ui.UIModule(),
		ledger.LedgerModule(),
		payments.PaymentsModule(),
		reports.ReportsModule(),
		search.SearchModule(),
		reconciliation.ReconciliationModule(),
		wallets.WalletsModule(),
		fx.Provide(fx.Annotate(
			func(root *cobra.Command) (*fctl.Config, error) {
				err := viper.BindPFlags(root.PersistentFlags())

				if err != nil {
					return nil, err
				}

				viper.SetConfigName("config")
				viper.SetConfigType("yaml")
				viper.AddConfigPath("$HOME/.formance")
				viper.ReadInConfig()

				config := &fctl.Config{}
				err = viper.Unmarshal(config)

				if err != nil {
					return nil, err
				}

				return config, err
			},
			fx.ParamTags(`name:"root"`),
		)),
		fx.Provide(func(c *fctl.Config) fctl.GetCurrentProfile {
			return func() (*fctl.CurrentProfile, fctl.CurrentProfileName, error) {
				p := viper.GetString("profile")

				profile, ok := c.Profiles[p]

				if !ok {
					return nil, "", fmt.Errorf("profile %s not found", p)
				}

				return &profile, p, nil
			}
		}),
		fx.Invoke(fx.Annotate(
			func(root *cobra.Command, commands ...*cobra.Command) {
				root.AddCommand(commands...)
			},
			fx.ParamTags(`name:"root"`, `group:"root-commands"`),
		)),
		fx.Invoke(fx.Annotate(func(root *cobra.Command, lc fx.Lifecycle) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					err := root.ExecuteContext(ctx)

					return err
				},
			})
		}, fx.ParamTags(`name:"root"`))),
	)

	app := fx.New(options)
	app.Start(context.Background())
}
