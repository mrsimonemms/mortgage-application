// Application-level workflow version derived from the Worker Build ID that
// the Temporal server reports against the workflow execution. The API
// normalises raw Build IDs into this small enum at the boundary so the UI
// (and any other consumer) only ever has to deal with v1, v2 or unknown.
//
// The convention is `<deployment>-<version>` (e.g. `mortgage-worker-v1`).
// Anything that does not match a known suffix collapses to `unknown` so the
// UI never has to handle a value it does not recognise.
export type WorkflowVersion = 'v1' | 'v2' | 'unknown';

export const WORKFLOW_VERSIONS: WorkflowVersion[] = ['v1', 'v2', 'unknown'];

// Build IDs are conventionally `<deployment>-v<n>`. Match the trailing
// `-v<n>` segment so the deployment-name prefix can change without breaking
// the mapping.
const BUILD_ID_VERSION_REGEX = /-(v\d+)$/i;

// deriveWorkflowVersion takes the Worker Build ID Temporal records on the
// workflow execution and maps it onto the application-level version. Missing
// or unrecognised build IDs return `unknown`.
//
// Examples:
//   mortgage-worker-v1 -> v1
//   mortgage-worker-v2 -> v2
//   foo-worker-v3      -> unknown (only v1 / v2 are known to this PoC)
//   undefined          -> unknown
export function deriveWorkflowVersion(
  buildId: string | undefined | null,
): WorkflowVersion {
  if (!buildId) return 'unknown';
  const match = buildId.match(BUILD_ID_VERSION_REGEX);
  const suffix = match?.[1]?.toLowerCase();
  if (suffix === 'v1') return 'v1';
  if (suffix === 'v2') return 'v2';
  return 'unknown';
}
