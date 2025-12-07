package main

import (
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		id, err := random.NewRandomId(ctx, "base-id", &random.RandomIdArgs{
			ByteLength: pulumi.Int(8),
		})
		if err != nil {
			return err
		}

		_, err = random.NewRandomString(ctx, "derived-string", &random.RandomStringArgs{
			Length: pulumi.Int(16),
			Keepers: pulumi.StringMap{
				"base": id.Hex,
			},
		})
		if err != nil {
			return err
		}

		ctx.Export("baseId", id.Hex)
		return nil
	})
}
