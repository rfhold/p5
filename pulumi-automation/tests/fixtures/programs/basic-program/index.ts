import * as pulumi from "@pulumi/pulumi";
import * as random from "@pulumi/random";

new random.RandomPet("first", {
	separator: "-",
});

const config = new pulumi.Config();

export const secret = config.getSecret("secret")
export const json = config.getObject<{
	key: string;
	array: string[];
}>("json");
