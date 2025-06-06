import * as local from "@pulumi/local";
import * as pulumi from "@pulumi/pulumi";

const file = new local.File("file", {
	content: "This is a test file",
	filename: `./files/${pulumi.getStack()}.txt`,
});

export const fileName = file.filename;
export const fileContent = file.contentMd5;
