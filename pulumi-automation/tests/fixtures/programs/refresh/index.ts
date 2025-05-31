import * as local from "@pulumi/local";
import * as pulumi from "@pulumi/pulumi";

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

