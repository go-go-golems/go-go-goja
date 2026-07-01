import { copyFile, readFile, writeFile } from "node:fs/promises";
import { resolve } from "node:path";

const sourcePackage = JSON.parse(await readFile(resolve("package.json"), "utf8"));

const distPackage = {
	name: sourcePackage.name,
	version: sourcePackage.version,
	private: false,
	type: sourcePackage.type,
	license: sourcePackage.license,
	description: sourcePackage.description,
	keywords: sourcePackage.keywords,
	homepage: sourcePackage.homepage,
	bugs: sourcePackage.bugs,
	repository: sourcePackage.repository,
	publishConfig: sourcePackage.publishConfig,
	dependencies: sourcePackage.dependencies,
	exports: {
		".": {
			types: "./index.d.ts",
			import: "./index.js",
		},
		"./generated/proto/goja/replapi/v1/replapi_pb": {
			types: "./generated/proto/goja/replapi/v1/replapi_pb.d.ts",
			import: "./generated/proto/goja/replapi/v1/replapi_pb.js",
		},
	},
	files: ["**/*.js", "**/*.d.ts", "**/*.js.map", "**/*.d.ts.map", "**/*.json", "README.md"],
};

await writeFile(resolve("dist/package.json"), `${JSON.stringify(distPackage, null, 2)}\n`);
await copyFile(resolve("README.md"), resolve("dist/README.md"));
