import * as local from "@pulumi/local";
import * as pulumi from "@pulumi/pulumi";

new local.File("file", {
	content: "This is a test file",
	filename: `./files/${pulumi.getStack()}.txt`,
});

