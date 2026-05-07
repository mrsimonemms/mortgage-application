import type { RawWorkflowExecutionInfo } from '@temporalio/client/lib/types';

// extractWorkerBuildId reads the Worker Build ID associated with a workflow
// execution from the raw Temporal proto. The Temporal client does not yet
// surface this on the typed WorkflowExecutionInfo, so this helper centralises
// the proto traversal in one place.
//
// Preference order (newest API first):
//   1. raw.versioningInfo.deploymentVersion.buildId — Worker Deployment
//      Versioning, which is what this PoC uses.
//   2. raw.assignedBuildId — earlier Worker Versioning API, kept as a
//      fallback so unversioned-but-tagged workers still report something.
//
// Returns undefined if no build ID can be found.
export function extractWorkerBuildId(
  raw: RawWorkflowExecutionInfo | undefined,
): string | undefined {
  if (!raw) return undefined;

  const deploymentVersionBuildId =
    raw.versioningInfo?.deploymentVersion?.buildId;
  if (
    typeof deploymentVersionBuildId === 'string' &&
    deploymentVersionBuildId
  ) {
    return deploymentVersionBuildId;
  }

  const assignedBuildId = raw.assignedBuildId;
  if (typeof assignedBuildId === 'string' && assignedBuildId) {
    return assignedBuildId;
  }

  return undefined;
}
