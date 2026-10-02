// TypeScript example: point an HTTP client at fake-jev and check the answer.
//
// Run it against a started server (see examples/README.md):
//
//   ./fake-jev run --config examples/fake-jev.yaml -- node examples/typescript/systemone.ts
//
// No npm install, no node_modules, no package.json dependency, no provider
// SDK, no credential, no model, and no network beyond the loopback server.
// Node runs the file directly by stripping the type annotations, so the
// annotations stay erasable (no enum, namespace, or parameter properties).

type JsonObject = Record<string, unknown>;

type RequestBody = {
  state: JsonObject;
  model: string;
  questions: JsonObject;
};

type ModelsDocument = {
  models: Array<{ name: string; description: string; release_date: string }>;
};

type AnswerDocument = {
  model: string;
  answers: JsonObject;
  usage: { input_tokens: number; output_tokens: number };
};

type VerifyDocument = {
  passed: boolean;
  failures: Array<{ code: string; message: string }>;
};

type ChoiceAnswer = {
  type: string;
  choice: string;
  confidence: number;
  probabilities: JsonObject;
};

type NoulAnswer = { type: string; noul: number };

const baseUrl: string = process.env.FAKE_JEV_URL ?? "http://127.0.0.1:8787";

const failures: string[] = [];

function check(condition: boolean, message: string): void {
  if (!condition) {
    failures.push(message);
  }
}

function show(label: string, value: unknown): void {
  console.log(`${label}: ${JSON.stringify(value)}`);
}

async function getJson<T>(path: string): Promise<T> {
  const response = await fetch(baseUrl + path);
  if (!response.ok) {
    throw new Error(`GET ${path} returned ${response.status}`);
  }
  return (await response.json()) as T;
}

async function postJson<T>(path: string, body: unknown): Promise<T> {
  const response = await fetch(baseUrl + path, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      // §39.5: fake-jev accepts any bearer value and never logs it. A real
      // key is neither needed nor read.
      Authorization: "Bearer not-a-real-key",
    },
    body: JSON.stringify(body),
  });
  if (!response.ok) {
    throw new Error(`POST ${path} returned ${response.status}`);
  }
  return (await response.json()) as T;
}

async function main(): Promise<void> {
  console.log(`fake-jev base URL: ${baseUrl}`);

  // 1. Model listing. §39.1 returns the configured (here: default) model list.
  const models = await getJson<ModelsDocument>("/v1/models");
  show("GET /v1/models", models);
  check(models.models.length === 1, "expected exactly one listed model");
  check(models.models[0]?.name === "jev-latest", "expected the model jev-latest");

  // 2. One System One call with a choice and a noul question.
  const request: RequestBody = {
    state: { task: "triage" },
    model: "jev-latest",
    questions: {
      route: {
        type: "choice",
        criteria: {
          frontend: "the view layer",
          backend: "the api layer",
          infra: "the deployment",
        },
      },
      urgent: { type: "noul", criteria: { deadline: "today" } },
    },
  };
  const answers = await postJson<AnswerDocument>("/v1/systemone", request);
  show("POST /v1/systemone", answers);

  const route = answers.answers.route as ChoiceAnswer;
  const urgent = answers.answers.urgent as NoulAnswer;
  check(answers.model === "jev-latest", "response model should echo the request model");
  check(route?.type === "choice", "route answer should be a choice");
  check(route?.choice === "backend", "route choice should be backend");
  check(route?.confidence === 0.91, "route confidence should be 0.91");
  check(route?.probabilities?.backend === 0.91, "backend probability should be 0.91");
  check(urgent?.type === "noul", "urgent answer should be a noul");
  check(urgent?.noul === 0.94, "urgent noul should be 0.94");
  check(
    answers.usage.input_tokens === 0 && answers.usage.output_tokens === 0,
    "usage should be the zero default",
  );

  // 3. Control-plane verification: the server decides whether the exchanges
  // above matched the fixture. §41.9.
  const verification = await getJson<VerifyDocument>("/__fake/v1/verify");
  show("GET /__fake/v1/verify", verification);
  check(verification.passed === true, "server verification should pass");

  // The same check the CLI performs:
  //   ./fake-jev verify --url <base URL>
  if (failures.length > 0) {
    for (const failure of failures) {
      console.error(`FAIL: ${failure}`);
    }
    process.exit(1);
  }
  console.log("OK: the fixture answers matched the expected values");
}

main().catch((error: unknown) => {
  console.error(`ERROR: ${error instanceof Error ? error.message : String(error)}`);
  process.exit(1);
});
