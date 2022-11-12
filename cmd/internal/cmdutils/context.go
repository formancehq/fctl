package cmdutils

import (
	"context"

	"github.com/spf13/viper"
)

type viperContextKeyType string

const viperContextKey viperContextKeyType = "viper"

func Viper(ctx context.Context) *viper.Viper {
	return ctx.Value(viperContextKey).(*viper.Viper)
}

func ContextWithViper(ctx context.Context, v *viper.Viper) context.Context {
	return context.WithValue(ctx, viperContextKey, v)
}
