package main

import (
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// RandomBundle is a component resource that bundles all random resource types
type RandomBundle struct {
	pulumi.ResourceState

	Bytes    *random.RandomBytes
	Id       *random.RandomId
	Integer  *random.RandomInteger
	Password *random.RandomPassword
	Pet      *random.RandomPet
	Shuffle  *random.RandomShuffle
	String   *random.RandomString
	Uuid     *random.RandomUuid
}

// NewRandomBundle creates a new RandomBundle component resource
func NewRandomBundle(ctx *pulumi.Context, name string, opts ...pulumi.ResourceOption) (*RandomBundle, error) {
	bundle := &RandomBundle{}
	err := ctx.RegisterComponentResource("p5:test:RandomBundle", name, bundle, opts...)
	if err != nil {
		return nil, err
	}

	// Create child resources with the component as parent
	childOpts := []pulumi.ResourceOption{pulumi.Parent(bundle)}

	bytes, err := random.NewRandomBytes(ctx, name+"-bytes", &random.RandomBytesArgs{
		Length: pulumi.Int(32),
	}, childOpts...)
	if err != nil {
		return nil, err
	}
	bundle.Bytes = bytes

	id, err := random.NewRandomId(ctx, name+"-id", &random.RandomIdArgs{
		ByteLength: pulumi.Int(8),
		Prefix:     pulumi.String("id-"),
	}, childOpts...)
	if err != nil {
		return nil, err
	}
	bundle.Id = id

	integer, err := random.NewRandomInteger(ctx, name+"-integer", &random.RandomIntegerArgs{
		Min: pulumi.Int(1),
		Max: pulumi.Int(100),
	}, childOpts...)
	if err != nil {
		return nil, err
	}
	bundle.Integer = integer

	password, err := random.NewRandomPassword(ctx, name+"-password", &random.RandomPasswordArgs{
		Length:     pulumi.Int(24),
		Special:    pulumi.Bool(true),
		MinLower:   pulumi.Int(2),
		MinUpper:   pulumi.Int(2),
		MinNumeric: pulumi.Int(2),
		MinSpecial: pulumi.Int(2),
	}, childOpts...)
	if err != nil {
		return nil, err
	}
	bundle.Password = password

	pet, err := random.NewRandomPet(ctx, name+"-pet", &random.RandomPetArgs{
		Length:    pulumi.Int(3),
		Separator: pulumi.String("-"),
	}, childOpts...)
	if err != nil {
		return nil, err
	}
	bundle.Pet = pet

	shuffle, err := random.NewRandomShuffle(ctx, name+"-shuffle", &random.RandomShuffleArgs{
		Inputs: pulumi.StringArray{
			pulumi.String("apple"),
			pulumi.String("banana"),
			pulumi.String("cherry"),
			pulumi.String("date"),
			pulumi.String("elderberry"),
		},
		ResultCount: pulumi.Int(3),
	}, childOpts...)
	if err != nil {
		return nil, err
	}
	bundle.Shuffle = shuffle

	str, err := random.NewRandomString(ctx, name+"-string", &random.RandomStringArgs{
		Length:  pulumi.Int(16),
		Special: pulumi.Bool(false),
		Upper:   pulumi.Bool(true),
		Lower:   pulumi.Bool(true),
		Numeric: pulumi.Bool(true),
	}, childOpts...)
	if err != nil {
		return nil, err
	}
	bundle.String = str

	uuid, err := random.NewRandomUuid(ctx, name+"-uuid", nil, childOpts...)
	if err != nil {
		return nil, err
	}
	bundle.Uuid = uuid

	// Register outputs
	ctx.RegisterResourceOutputs(bundle, pulumi.Map{
		"bytesHex":       bytes.Hex,
		"idHex":          id.Hex,
		"integerResult":  integer.Result,
		"passwordResult": password.Result,
		"petId":          pet.ID(),
		"shuffleResults": shuffle.Results,
		"stringResult":   str.Result,
		"uuidResult":     uuid.Result,
	})

	return bundle, nil
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// RandomBytes - generates random bytes
		bytes, err := random.NewRandomBytes(ctx, "myBytes", &random.RandomBytesArgs{
			Length: pulumi.Int(32),
		})
		if err != nil {
			return err
		}
		ctx.Export("bytesHex", bytes.Hex)

		// RandomId - generates random identifiers
		id, err := random.NewRandomId(ctx, "myId", &random.RandomIdArgs{
			ByteLength: pulumi.Int(8),
			Prefix:     pulumi.String("id-"),
		})
		if err != nil {
			return err
		}
		ctx.Export("idHex", id.Hex)

		// RandomInteger - generates random integer in range
		integer, err := random.NewRandomInteger(ctx, "myInteger", &random.RandomIntegerArgs{
			Min: pulumi.Int(1),
			Max: pulumi.Int(100),
		})
		if err != nil {
			return err
		}
		ctx.Export("integerResult", integer.Result)

		// RandomPassword - generates random password (sensitive)
		password, err := random.NewRandomPassword(ctx, "myPassword", &random.RandomPasswordArgs{
			Length:     pulumi.Int(24),
			Special:    pulumi.Bool(true),
			MinLower:   pulumi.Int(2),
			MinUpper:   pulumi.Int(2),
			MinNumeric: pulumi.Int(2),
			MinSpecial: pulumi.Int(2),
		})
		if err != nil {
			return err
		}
		ctx.Export("passwordResult", password.Result)

		// RandomPet - generates random pet names
		pet, err := random.NewRandomPet(ctx, "myPet", &random.RandomPetArgs{
			Length:    pulumi.Int(3),
			Separator: pulumi.String("-"),
		})
		if err != nil {
			return err
		}
		ctx.Export("petId", pet.ID())

		// RandomShuffle - shuffles a list of strings
		shuffle, err := random.NewRandomShuffle(ctx, "myShuffle", &random.RandomShuffleArgs{
			Inputs: pulumi.StringArray{
				pulumi.String("apple"),
				pulumi.String("banana"),
				pulumi.String("cherry"),
				pulumi.String("date"),
				pulumi.String("elderberry"),
			},
			ResultCount: pulumi.Int(3),
		})
		if err != nil {
			return err
		}
		ctx.Export("shuffleResults", shuffle.Results)

		// RandomString - generates random string
		str, err := random.NewRandomString(ctx, "myString", &random.RandomStringArgs{
			Length:  pulumi.Int(16),
			Special: pulumi.Bool(false),
			Upper:   pulumi.Bool(true),
			Lower:   pulumi.Bool(true),
			Numeric: pulumi.Bool(true),
		})
		if err != nil {
			return err
		}
		ctx.Export("stringResult", str.Result)

		// RandomUuid - generates random UUID
		uuid, err := random.NewRandomUuid(ctx, "myUuid", nil)
		if err != nil {
			return err
		}
		ctx.Export("uuidResult", uuid.Result)

		// RandomBundle - component resource containing all random types
		bundle, err := NewRandomBundle(ctx, "myBundle")
		if err != nil {
			return err
		}
		ctx.Export("bundleBytesHex", bundle.Bytes.Hex)
		ctx.Export("bundleIdHex", bundle.Id.Hex)

		return nil
	})
}
