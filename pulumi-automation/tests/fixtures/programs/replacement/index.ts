import * as random from "@pulumi/random";
import * as pulumi from "@pulumi/pulumi";

const stackReference = `${pulumi.getOrganization()}/${pulumi.getProject()}/${pulumi.getStack()}`;

const previousConstant = new pulumi.StackReference(stackReference, {
	name: stackReference,
}).getOutput("constant");

export const constant = previousConstant.apply((prev) => Boolean(prev) ? prev + 1 : 1);
const numberOfIds = 2;

let previousId: random.RandomId | null = null;

// for each id depend on the previous id to force a cascade of replacements
for (let i = 0; i < numberOfIds; i++) {
	previousId = new random.RandomId(`id-${i}`, {
		byteLength: 8,
		keepers: { constant },
	}, {
		dependsOn: previousId ? [previousId] : undefined,
		replaceOnChanges: ["keepers"],
	});
}


