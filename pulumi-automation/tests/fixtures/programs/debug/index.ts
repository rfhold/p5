import * as local from "@pulumi/local";
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

// These two will overwrite each other, so we can test that the refresh works correctly
const file = new local.File("file", {
	content: "This is a test file",
	filename: `./files/${pulumi.getStack()}.txt`,
});

new local.File("file-overwrite", {
	content: "This is an overwritten test file",
	filename: file.filename,
}, {
	dependsOn: [file],
});

