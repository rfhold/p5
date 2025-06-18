import * as local from "@pulumi/local";
import * as pulumi from "@pulumi/pulumi";
import * as command from "@pulumi/command";
import * as random from "@pulumi/random";

const config = new pulumi.Config();

const reset = config.getBoolean("reset") ?? false;
const fail = config.getBoolean("fail") ?? false;

if (reset) {
	new local.File("delete", {
		content: "file to delete",
		filename: `./files/${pulumi.getStack()}/delete.txt`,
	});

	new local.File("discard", {
		content: "file to discard",
		filename: `./files/${pulumi.getStack()}/discard.txt`,
	}, {
		retainOnDelete: true,
	});
}

new local.File("file", {
	content: "This is a test file",
	filename: `./files/${pulumi.getStack()}/file.txt`,
});

let stackName = `${pulumi.getOrganization()}/${pulumi.getProject()}/${pulumi.getStack()}`;

if (reset) {
	new pulumi.StackReference("removed-reference", {
		name: stackName,
	});
}

let stackReference = new pulumi.StackReference("current", {
	name: stackName,
});

export const iteration = stackReference.getOutput("iteration").apply(iteration => {
	return (iteration ?? 0) + 1;
});

new local.File("iteration", {
	content: pulumi.interpolate`Iteration: ${iteration}`,
	filename: `./files/${pulumi.getStack()}/iteration.txt`,
});

if (!reset) {
	// new random.RandomString("import-good", {
	// 	length: 10,
	// 	upper: false,
	// 	numeric: false,
	// 	special: false,
	// 	overrideSpecial: "",
	// }, {
	// 	import: "abcdefghij",
	// });

	new local.File("new", {
		content: "new",
		filename: `./files/${pulumi.getStack()}/new.txt`,
	});
}

const secret = new random.RandomPassword("password", {
	length: 16,
	keepers: {
		iteration,
	},
}, {
	deleteBeforeReplace: true,
});

export const secretOutput = secret.result;

new local.File("secret", {
	content: secret.result,
	filename: `./files/${pulumi.getStack()}/secret.txt`,
});

const contentCommand = new command.local.Command(`sleep`, {
	create: pulumi.interpolate`sleep 2 && echo "${iteration}"`,
});

new local.File(`file-from-command`, {
	content: contentCommand.stdout,
	filename: `./files/${pulumi.getStack()}/file-from-command.txt`,
});

export const booleanOutput = true;
export const numberOutput = 42;
export const stringOutput = "This is a string output";
export const objectOutput = { "key": "value" };
export const arrayOutput = [1, 2, 3];

if (fail) {
	new command.local.Command("fail", {
		create: "exit 1",
	});
}
