import * as local from "@pulumi/local";
import * as pulumi from "@pulumi/pulumi";
import * as command from "@pulumi/command";

// const file = new local.File("file", {
// 	content: "This is a test file",
// 	filename: `./files/${pulumi.getStack()}.txt`,
// });
//
// export const fileName = file.filename;
// export const fileContent = file.contentMd5;

let stackName = `${pulumi.getOrganization()}/${pulumi.getProject()}/${pulumi.getStack()}`;

let stackReference = new pulumi.StackReference("current", {
	name: stackName,
});

const prev = stackReference.getOutput("lastFileContent")

let files = [];
for (let i = 0; i < 7; i++) {
	const contentCommand = new command.local.Command(`sleep-${i}`, {
		create: pulumi.interpolate`sleep ${i} && echo "${prev}"`,
		triggers: [prev],
	});

	const file = new local.File(`file-${i}`, {
		content: contentCommand.stdout,
		filename: `./files/${pulumi.getStack()}-${i}.txt`,
	});
	files.push(file);
}

export const fileNames = files.map(f => f.filename);
export const lastFileContent = files[files.length - 1].contentMd5;
