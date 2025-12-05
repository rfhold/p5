package main

import (
	"fmt"
	"os"
	"time"

	"github.com/pulumi/pulumi-command/sdk/go/command/local"
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Create a random pet name
		pet, err := random.NewRandomPet(ctx, "my-pet", &random.RandomPetArgs{
			Length: pulumi.Int(2),
		})
		if err != nil {
			return err
		}

		// Create a command that sleeps for 3 seconds to test progress updates
		sleepCmd, err := local.NewCommand(ctx, "sleep-command", &local.CommandArgs{
			Create: pulumi.String("echo 'Starting sleep...' && sleep 3 && echo 'Done sleeping!'"),
		})
		if err != nil {
			return err
		}

		// Create a command with delete that always replaces due to timestamp trigger
		timestamp := fmt.Sprintf("%d", time.Now().Unix())
		replaceCmd, err := local.NewCommand(ctx, "replace-command", &local.CommandArgs{
			Create: pulumi.String("echo 'Creating resource...' && sleep 2 && echo 'Created!'"),
			Delete: pulumi.String("echo 'Deleting resource...' && sleep 2 && echo 'Deleted!'"),
			Triggers: pulumi.Array{
				pulumi.String(timestamp),
			},
		}, pulumi.ReplaceOnChanges([]string{"triggers"}))
		if err != nil {
			return err
		}

		// Create a command with nested environment variables that change (for testing object diffing)
		_, err = local.NewCommand(ctx, "env-command", &local.CommandArgs{
			Create: pulumi.String("echo \"Config: $CONFIG_JSON\""),
			Environment: pulumi.StringMap{
				"STATIC_VALUE": pulumi.String("this-never-changes"),
				"TIMESTAMP":    pulumi.String(timestamp),
				"CONFIG_JSON":  pulumi.Sprintf(`{"version":"1.0","updated":"%s","nested":{"time":"%s"}}`, time.Now().Format(time.RFC3339), timestamp),
			},
			Triggers: pulumi.Array{
				pulumi.String(timestamp),
			},
		}, pulumi.ReplaceOnChanges([]string{"triggers"}))
		if err != nil {
			return err
		}

		// Create a nested map with changing timestamp for testing object diffing
		// Note: Using pulumi.Map directly (not JSONMarshal) so Pulumi stores it as a structured object
		metadata := pulumi.Map{
			"version": pulumi.String("1.0.0"),
			"config": pulumi.Map{
				"enabled":   pulumi.Bool(true),
				"maxRetry":  pulumi.Int(3),
				"timestamp": pulumi.String(fmt.Sprintf("%d", time.Now().Unix())),
				"nested": pulumi.Map{
					"updatedAt": pulumi.String(time.Now().Format(time.RFC3339)),
					"tags":      pulumi.ToStringArray([]string{"test", "simple", "pulumi"}),
				},
			},
			"static": pulumi.String("this value does not change"),
		}

		// Export env vars to confirm they get set and merged correctly
		envVars := pulumi.Map{
			"BASE_VAR":     pulumi.String(os.Getenv("BASE_VAR")),
			"STACK_VAR":    pulumi.String(os.Getenv("STACK_VAR")),
			"OVERRIDE_VAR": pulumi.String(os.Getenv("OVERRIDE_VAR")),
		}

		ctx.Export("petName", pet.ID())
		ctx.Export("sleepOutput", sleepCmd.Stdout)
		ctx.Export("replaceOutput", replaceCmd.Stdout)
		ctx.Export("message", pulumi.String("Hello from p5 test project!"))
		ctx.Export("metadata", metadata)
		ctx.Export("envVars", envVars)
		return nil
	})
}
