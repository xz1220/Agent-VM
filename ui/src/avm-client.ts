import {spawn} from "node:child_process";
import {AgentDetailSchema, AgentSchema, AgentSummarySchema, AvmErrorEnvelopeSchema, CapabilityCandidateSchema, DoctorReportSchema, type Agent, type AgentDetail, type AgentSummary, type AgentWriteInput, type CapabilityCandidate, type CapabilityKind, type RuntimeCheck} from "./protocol.js";
import {mockCapabilityCandidates} from "./mock-capabilities.js";
import {z} from "zod";

export class AvmCommandError extends Error {
  readonly code: string;
  readonly details?: Record<string, unknown>;
  readonly stderr: string;

  constructor(input: {code: string; message: string; details?: Record<string, unknown>; stderr?: string}) {
    super(input.message);
    this.name = "AvmCommandError";
    this.code = input.code;
    this.details = input.details;
    this.stderr = input.stderr ?? "";
  }
}

export class AvmClient {
  constructor(readonly binary: string) {}

  async listAgents(): Promise<AgentSummary[]> {
    return this.runJson(["agent", "list"], z.array(AgentSummarySchema));
  }

  async showAgent(name: string): Promise<AgentDetail> {
    return this.runJson(["agent", "show", name], AgentDetailSchema);
  }

  async createAgent(input: AgentWriteInput): Promise<Agent> {
    return this.runJson(["agent", "create", ...agentCreateArgs(input)], AgentSchema);
  }

  async editAgent(name: string, input: AgentWriteInput): Promise<Agent> {
    return this.runJson(["agent", "edit", name, ...agentEditArgs(input)], AgentSchema);
  }

  async deleteAgent(name: string): Promise<void> {
    await this.runVoid(["agent", "delete", name, "--yes"]);
  }

  async listRuntimes(): Promise<RuntimeCheck[]> {
    try {
      const report = await this.runJson(["doctor"], DoctorReportSchema);
      if (report.runtimes.length > 0) {
        return report.runtimes;
      }
    } catch {
      // The UI can still scaffold Agent CRUD while doctor is unavailable.
    }
    return [
      {runtime: "codex", available: true, issues: []},
      {runtime: "claude-code", available: true, issues: []},
      {runtime: "opencode", available: true, issues: []}
    ];
  }

  async discoverCapabilities(input: {
    kinds?: CapabilityKind[];
    runtimes?: string[];
  }): Promise<CapabilityCandidate[]> {
    // TODO(ui-integration): replace mock candidates once the Go CLI exposes
    // `avm capability discover --json` and runtime-global import commands.
    return z.array(CapabilityCandidateSchema).parse(mockCapabilityCandidates(input));
  }

  private async runVoid(args: string[]): Promise<void> {
    await run(this.binary, ["--json", ...args], undefined);
  }

  private async runJson<T>(args: string[], schema: z.ZodType<T>): Promise<T> {
    const stdout = await run(this.binary, ["--json", ...args], schema);
    return stdout;
  }
}

function agentCreateArgs(input: AgentWriteInput): string[] {
  return [
    "--name", input.name,
    ...optionalStringFlag("--description", input.description),
    ...optionalStringFlag("--role", input.role),
    ...optionalStringFlag("--system", input.system),
    ...runtimeArgs(input),
    ...capabilityArgs("--skill", input.skills),
    ...capabilityArgs("--mcp", input.mcp)
  ];
}

function agentEditArgs(input: AgentWriteInput): string[] {
  return [
    "--description", input.description,
    "--role", input.role,
    "--system", input.system,
    ...runtimeArgs(input),
    ...capabilityArgs("--skill", input.skills),
    ...capabilityArgs("--mcp", input.mcp)
  ];
}

function optionalStringFlag(flag: string, value: string): string[] {
  return value.trim() === "" ? [] : [flag, value];
}

function runtimeArgs(input: AgentWriteInput): string[] {
  const args = input.runtimes.flatMap((pref) => ["--runtime", pref.runtime]);
  const defaultRuntime = input.runtimes.find((pref) => pref.default)?.runtime;
  return defaultRuntime ? [...args, "--default-runtime", defaultRuntime] : args;
}

function capabilityArgs(flag: string, refs: {id: string}[]): string[] {
  return refs.flatMap((ref) => [flag, ref.id]);
}

async function run<T>(binary: string, args: string[], schema: z.ZodType<T> | undefined): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    const child = spawn(binary, args, {
      stdio: ["ignore", "pipe", "pipe"]
    });

    let stdout = "";
    let stderr = "";
    child.stdout.on("data", (chunk) => {
      stdout += String(chunk);
    });
    child.stderr.on("data", (chunk) => {
      stderr += String(chunk);
    });
    child.on("error", (error) => {
      reject(new AvmCommandError({
        code: "SPAWN_FAILED",
        message: `failed to spawn ${binary}: ${error.message}`,
        stderr
      }));
    });
    child.on("close", (code) => {
      if (code !== 0) {
        reject(parseError(stdout, stderr, code ?? 1));
        return;
      }
      if (!schema) {
        resolve(undefined as T);
        return;
      }
      try {
        resolve(schema.parse(JSON.parse(stdout)));
      } catch (error) {
        reject(new AvmCommandError({
          code: "INVALID_JSON",
          message: `invalid JSON from avm: ${error instanceof Error ? error.message : String(error)}`,
          stderr
        }));
      }
    });
  });
}

function parseError(stdout: string, stderr: string, exitCode: number): AvmCommandError {
  try {
    const env = AvmErrorEnvelopeSchema.parse(JSON.parse(stdout));
    return new AvmCommandError({
      code: env.error.code,
      message: env.error.message,
      details: env.error.details,
      stderr
    });
  } catch {
    return new AvmCommandError({
      code: `EXIT_${exitCode}`,
      message: stderr.trim() || stdout.trim() || `avm exited with code ${exitCode}`,
      stderr
    });
  }
}
