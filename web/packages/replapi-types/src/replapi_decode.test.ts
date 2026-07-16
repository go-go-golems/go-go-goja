import { deepStrictEqual, equal } from "node:assert/strict";
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

import { fromJson, toJson } from "@bufbuild/protobuf";
import type { JsonValue } from "@bufbuild/protobuf";

import {
	ErrorResponseSchema,
	EvaluateResponseSchema,
	SessionExportSchema,
} from "./generated/proto/goja/replapi/v1/replapi_pb.ts";

const testdataDir = join(dirname(fileURLToPath(import.meta.url)), "testdata");

function readFixture(name: string): JsonValue {
	return JSON.parse(readFileSync(join(testdataDir, name), "utf8")) as JsonValue;
}

function asRecord(value: unknown): Record<string, unknown> {
	if (typeof value !== "object" || value === null || Array.isArray(value)) {
		throw new Error(`expected object, got ${JSON.stringify(value)}`);
	}
	return value as Record<string, unknown>;
}

function asArray(value: unknown): unknown[] {
	if (!Array.isArray(value)) {
		throw new Error(`expected array, got ${JSON.stringify(value)}`);
	}
	return value;
}

function decodeEvaluateResponse(): void {
	const decoded = fromJson(
		EvaluateResponseSchema,
		readFixture("evaluate_response.golden.json"),
	);

	equal(decoded.schemaVersion, 1);
	equal(decoded.session?.id, "sess-1");
	equal(decoded.session?.policy?.eval?.timeoutMs, 5000n);
	equal(decoded.cell?.execution?.status, "ok");
	equal(decoded.cell?.execution?.result, "42");
	equal(decoded.cell?.execution?.durationMs, 7n);
	equal(decoded.cell?.execution?.console[0]?.message, "hello");
	equal(decoded.session?.bindings[0]?.name, "answer");
}

function decodeErrorResponse(): void {
	const decoded = fromJson(
		ErrorResponseSchema,
		readFixture("error_response.golden.json"),
	);

	equal(decoded.schemaVersion, 1);
	equal(decoded.code, "session_owned");
	equal(decoded.message, "session is owned by another app");
	equal(decoded.requestId, "request-test-1");
}

function decodeSessionExport(): void {
	const decoded = fromJson(
		SessionExportSchema,
		readFixture("session_export.golden.json"),
	);

	equal(decoded.session?.sessionId, "sess-1");
	equal(decoded.evaluations[0]?.evaluationId, 1n);
	equal(decoded.evaluations[0]?.cellId, 1);
	equal(decoded.evaluations[0]?.bindingVersions[0]?.name, "answer");

	const encodedJson = asRecord(toJson(SessionExportSchema, decoded));
	const session = asRecord(encodedJson.session);
	const metadata = asRecord(session.metadataJson);
	deepStrictEqual(metadata.features, ["history", "docs"]);
	equal(metadata.profile, "persistent");
	equal(metadata.active, true);

	const evaluations = asArray(encodedJson.evaluations);
	const evaluation = asRecord(evaluations[0]);
	const resultJson = asRecord(evaluation.resultJson);
	deepStrictEqual(resultJson.labels, ["ok"]);
	equal(resultJson.result, 42);

	const bindingVersions = asArray(evaluation.bindingVersions);
	const bindingVersion = asRecord(bindingVersions[0]);
	equal(bindingVersion.exportJson, 42);

	const bindingDocs = asArray(evaluation.bindingDocs);
	const bindingDoc = asRecord(bindingDocs[0]);
	const normalizedJson = asRecord(bindingDoc.normalizedJson);
	deepStrictEqual(normalizedJson.tags, ["number"]);
}

decodeEvaluateResponse();
decodeErrorResponse();
decodeSessionExport();
