import { execFileSync } from "node:child_process";
import { mkdtemp, mkdir, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";

const cwd = process.cwd();
const smokeDir = await mkdtemp(join(tmpdir(), "go-go-goja-replapi-types-consumer-"));
const tarballOutput = execFileSync("npm", ["pack", "./dist", "--pack-destination", smokeDir], {
	cwd,
	encoding: "utf8",
	stdio: ["ignore", "pipe", "inherit"],
})
	.trim()
	.split("\n")
	.at(-1);

if (!tarballOutput) {
	throw new Error("npm pack did not report a tarball filename");
}

const tarball = join(smokeDir, tarballOutput);

await writeFile(
	join(smokeDir, "package.json"),
	JSON.stringify(
		{
			private: true,
			type: "module",
			scripts: {
				typecheck: "tsc --noEmit",
				test: "tsx src/main.ts",
			},
			dependencies: {
				"@bufbuild/protobuf": "^2.12.1",
				"@go-go-golems/go-go-goja-replapi-types": `file:${tarball}`,
			},
			devDependencies: {
				"@types/node": "^24.10.1",
				tsx: "^4.21.0",
				typescript: "~6.0.2",
			},
		},
		null,
		2,
	),
);

await writeFile(
	join(smokeDir, "tsconfig.json"),
	JSON.stringify(
		{
			compilerOptions: {
				target: "ES2022",
				lib: ["ES2022"],
				types: ["node"],
				module: "ESNext",
				moduleResolution: "bundler",
				strict: true,
				skipLibCheck: true,
				noEmit: true,
			},
			include: ["src"],
		},
		null,
		2,
	),
);

await mkdir(join(smokeDir, "src"));
await writeFile(
	join(smokeDir, "src/main.ts"),
	`
import { strictEqual } from 'node:assert/strict';
import { fromJson } from '@bufbuild/protobuf';
import { EvaluateResponseSchema, SessionExportSchema } from '@go-go-golems/go-go-goja-replapi-types';
import { EvaluateResponseSchema as DirectEvaluateResponseSchema } from '@go-go-golems/go-go-goja-replapi-types/generated/proto/goja/replapi/v1/replapi_pb';

const response = fromJson(EvaluateResponseSchema, {
  schemaVersion: 1,
  cell: { execution: { status: 'ok', result: '42', durationMs: '7' } }
});
strictEqual(response.cell?.execution?.durationMs, 7n);

const direct = fromJson(DirectEvaluateResponseSchema, { schemaVersion: 1 });
strictEqual(direct.schemaVersion, 1);

const exported = fromJson(SessionExportSchema, {
  session: { sessionId: 's', metadataJson: { nested: ['ok'] } },
  evaluations: [{ evaluationId: '1', resultJson: { answer: 42 } }]
});
strictEqual(exported.evaluations[0]?.evaluationId, 1n);
`,
);

execFileSync("npm", ["install", "--silent"], { cwd: smokeDir, stdio: "inherit" });
execFileSync("npm", ["run", "typecheck", "--silent"], { cwd: smokeDir, stdio: "inherit" });
execFileSync("npm", ["run", "test", "--silent"], { cwd: smokeDir, stdio: "inherit" });

console.log(`clean consumer smoke passed in ${resolve(smokeDir)}`);
